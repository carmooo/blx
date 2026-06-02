package ipac

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"

	"github.com/joao-carmo/blx/internal/service"
)

// parseRSS parses RSS XML bytes into a slice of SearchItem.
// The iPAC feed uses ISO-8859-1 encoding, so we convert to UTF-8 first,
// then strip the XML declaration (Go's xml package only supports UTF-8).
func parseRSS(data []byte) ([]service.SearchItem, error) {
	utf8Data, err := decodeISO8859(data)
	if err != nil {
		return nil, fmt.Errorf("decode ISO-8859-1: %w", err)
	}
	cleaned := stripXMLDeclaration(utf8Data)

	var rss rssResponse
	if err := xml.Unmarshal(cleaned, &rss); err != nil {
		return nil, fmt.Errorf("unmarshal RSS: %w", err)
	}

	items := make([]service.SearchItem, 0, len(rss.Channel.Items))
	for _, ri := range rss.Channel.Items {
		id, err := extractID(ri.Link)
		if err != nil {
			continue // skip items with unparseable links
		}
		items = append(items, service.SearchItem{
			ID:    id,
			Title: ri.Title,
		})
	}

	return items, nil
}

// decodeISO8859 converts ISO-8859-1 bytes to UTF-8.
// If the data is already valid UTF-8 with multi-byte sequences, it is returned as-is
// to avoid double-encoding.
func decodeISO8859(data []byte) ([]byte, error) {
	if isValidUTF8WithMultibyte(data) {
		return data, nil
	}
	return charmap.ISO8859_1.NewDecoder().Bytes(data)
}

// isValidUTF8WithMultibyte returns true if data contains multi-byte UTF-8 sequences,
// indicating it is already UTF-8 encoded (not ISO-8859-1).
func isValidUTF8WithMultibyte(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] >= 0xC0 { // start of a multi-byte UTF-8 sequence
			// Check it's a valid continuation
			if i+1 < len(data) && data[i+1]&0xC0 == 0x80 {
				return true
			}
		}
	}
	return false
}

// stripXMLDeclaration removes the <?xml ...?> processing instruction.
func stripXMLDeclaration(data []byte) []byte {
	s := string(data)
	if idx := strings.Index(s, "?>"); idx >= 0 {
		s = s[idx+2:]
	}
	return []byte(s)
}

// extractID parses the item link URL to extract the catalog ID.
// The link contains a query parameter "uri" with value "full=3100024~!29331~!0".
// We strip the "full=" prefix and return "3100024~!29331~!0".
func extractID(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("parse link URL: %w", err)
	}

	uri := u.Query().Get("uri")
	if uri == "" {
		return "", fmt.Errorf("no uri parameter in link: %s", link)
	}

	id := strings.TrimPrefix(uri, "full=")
	if id == "" {
		return "", fmt.Errorf("empty ID after stripping prefix from: %s", uri)
	}

	return id, nil
}

// parseMarcXML parses MarcXchange XML (UNIMARC format) into a service.Item.
// Like parseRSS, always decodes from ISO-8859-1 since iPAC sends that encoding.
func parseMarcXML(data []byte) (*service.Item, error) {
	utf8Data, err := decodeISO8859(data)
	if err != nil {
		return nil, fmt.Errorf("decode ISO-8859-1: %w", err)
	}
	cleaned := stripXMLDeclaration(utf8Data)

	var collection marcCollection
	if err := xml.Unmarshal(cleaned, &collection); err != nil {
		return nil, fmt.Errorf("unmarshal MarcXML: %w", err)
	}

	if len(collection.Records) == 0 {
		return nil, fmt.Errorf("no records found in MarcXML")
	}

	rec := collection.Records[0]
	item := &service.Item{}

	for _, df := range rec.DataFields {
		subs := subfieldsMap(df.Subfields)
		switch df.Tag {
		case "010":
			item.ISBN = subs["a"]
		case "101":
			item.Language = subs["a"]
		case "200":
			item.Title = subs["a"]
		case "205":
			item.Edition = subs["a"]
		case "210":
			item.Place = subs["a"]
			item.Publisher = subs["c"]
			item.Year = subs["d"]
		case "215":
			parts := collectSubfields(df.Subfields, "a", "c", "d")
			if len(parts) > 0 {
				item.Physical = strings.Join(parts, " ; ")
			}
		case "606", "607":
			parts := collectSubfields(df.Subfields, "a", "z", "j")
			if len(parts) > 0 {
				item.Subjects = append(item.Subjects, strings.Join(parts, " -- "))
			}
		case "675":
			item.Classification = subs["a"]
		case "700", "701":
			item.Authors = append(item.Authors, buildAuthor(df.Subfields, "author"))
		case "702":
			item.Authors = append(item.Authors, buildAuthor(df.Subfields, "contributor"))
		}
	}

	// Trim trailing comma from publisher.
	item.Publisher = strings.TrimRight(item.Publisher, ", ")

	return item, nil
}

// subfieldsMap returns a map from subfield code to value (first occurrence wins).
func subfieldsMap(subs []marcSubfield) map[string]string {
	m := make(map[string]string, len(subs))
	for _, s := range subs {
		if _, exists := m[s.Code]; !exists {
			m[s.Code] = s.Value
		}
	}
	return m
}

// collectSubfields returns the values of subfields matching the given codes, in order.
func collectSubfields(subs []marcSubfield, codes ...string) []string {
	codeSet := make(map[string]bool, len(codes))
	for _, c := range codes {
		codeSet[c] = true
	}
	var parts []string
	for _, s := range subs {
		if codeSet[s.Code] && s.Value != "" {
			parts = append(parts, s.Value)
		}
	}
	return parts
}

// buildAuthor constructs an Author from MARC subfields.
func buildAuthor(subs []marcSubfield, role string) service.Author {
	m := subfieldsMap(subs)
	name := strings.TrimRight(m["a"], ", ")
	forename := strings.TrimRight(m["b"], ", ")
	if forename != "" {
		name = name + ", " + forename
	}
	return service.Author{
		Name:  name,
		Dates: m["f"],
		Role:  role,
	}
}

// parseHoldings parses iPAC HTML detail page to extract holdings information.
func parseHoldings(data []byte) ([]service.Holding, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var holdings []service.Holding

	// Find the holdings table by looking for header rows with Portuguese keywords.
	doc.Find("table").Each(func(_ int, table *goquery.Selection) {
		if len(holdings) > 0 {
			return // already found
		}

		// Try to find a header row that identifies the holdings table.
		colMap := map[string]int{}
		headerRowIdx := -1

		table.Find("tr").Each(func(rowIdx int, tr *goquery.Selection) {
			if headerRowIdx >= 0 {
				return
			}
			text := tr.Text()
			hasLocation := strings.Contains(text, "Localiza")
			hasCota := strings.Contains(text, "Cota")
			if hasLocation || hasCota {
				// Map column positions.
				tr.Find("td, th").Each(func(colIdx int, cell *goquery.Selection) {
					cellText := strings.TrimSpace(cell.Text())
					switch {
					case strings.Contains(cellText, "Localiza"):
						colMap["branch"] = colIdx
					case strings.Contains(cellText, "Cota"):
						colMap["callnumber"] = colIdx
					case strings.Contains(cellText, "Cole") || strings.Contains(cellText, "Coleção"):
						colMap["collection"] = colIdx
					case strings.Contains(cellText, "Estado") || strings.Contains(cellText, "Situa"):
						colMap["status"] = colIdx
					case strings.Contains(cellText, "Empr") || strings.Contains(cellText, "dias"):
						colMap["loandays"] = colIdx
					}
				})
				if len(colMap) >= 2 {
					headerRowIdx = rowIdx
				}
			}
		})

		if headerRowIdx < 0 {
			return
		}

		// Parse data rows after the header.
		table.Find("tr").Each(func(rowIdx int, tr *goquery.Selection) {
			if rowIdx <= headerRowIdx {
				return
			}
			cells := tr.Find("td, th")
			if cells.Length() == 0 {
				return
			}

			h := service.Holding{}
			if idx, ok := colMap["branch"]; ok && idx < cells.Length() {
				h.Branch = strings.TrimSpace(cells.Eq(idx).Text())
			}
			if idx, ok := colMap["callnumber"]; ok && idx < cells.Length() {
				h.CallNumber = strings.TrimSpace(cells.Eq(idx).Text())
			}
			if idx, ok := colMap["collection"]; ok && idx < cells.Length() {
				h.Collection = strings.TrimSpace(cells.Eq(idx).Text())
			}
			if idx, ok := colMap["status"]; ok && idx < cells.Length() {
				h.Status = strings.TrimSpace(cells.Eq(idx).Text())
			}
			if idx, ok := colMap["loandays"]; ok && idx < cells.Length() {
				text := strings.TrimSpace(cells.Eq(idx).Text())
				if days, err := strconv.Atoi(text); err == nil {
					h.LoanDays = days
				}
			}

			// Only add if we got meaningful data.
			if h.Branch != "" || h.CallNumber != "" {
				holdings = append(holdings, h)
			}
		})
	})

	return holdings, nil
}
