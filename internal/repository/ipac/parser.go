package ipac

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
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
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	// Find the header row: a <tr> whose direct <td>/<th> children contain "Cota" and "Local".
	headerRow := findHoldingsHeader(doc)
	if headerRow == nil {
		return nil, nil
	}

	// Map column positions from header cells.
	colMap := map[string]int{}
	for i, cell := range directCells(headerRow) {
		text := strings.TrimSpace(nodeText(cell))
		switch {
		case strings.Contains(text, "Local"):
			colMap["branch"] = i
		case text == "Cota" || strings.HasPrefix(text, "Cota"):
			colMap["callnumber"] = i
		case strings.Contains(text, "Cole"):
			colMap["collection"] = i
		case strings.Contains(text, "Estado") || strings.Contains(text, "Situa"):
			colMap["status"] = i
		case strings.Contains(text, "Prazo") || strings.Contains(text, "Empr"):
			colMap["loandays"] = i
		case strings.Contains(text, "Devolu"):
			colMap["duedate"] = i
		}
	}

	if len(colMap) < 2 {
		return nil, nil
	}

	// Parse sibling <tr> rows after the header.
	var holdings []service.Holding
	for sibling := headerRow.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type != html.ElementNode || sibling.Data != "tr" {
			continue
		}
		cells := directCells(sibling)
		if len(cells) < 2 {
			continue
		}

		h := service.Holding{}
		if idx, ok := colMap["branch"]; ok && idx < len(cells) {
			h.Branch = strings.TrimSpace(nodeText(cells[idx]))
		}
		if idx, ok := colMap["callnumber"]; ok && idx < len(cells) {
			h.CallNumber = strings.TrimSpace(nodeText(cells[idx]))
		}
		if idx, ok := colMap["collection"]; ok && idx < len(cells) {
			h.Collection = strings.TrimSpace(nodeText(cells[idx]))
		}
		if idx, ok := colMap["status"]; ok && idx < len(cells) {
			h.Status = strings.TrimSpace(nodeText(cells[idx]))
		}
		if idx, ok := colMap["loandays"]; ok && idx < len(cells) {
			text := strings.TrimSpace(nodeText(cells[idx]))
			text = strings.TrimSuffix(text, " dias")
			if days, err := strconv.Atoi(text); err == nil {
				h.LoanDays = days
			}
		}

		// Extract bibkey and itemkey from reserve link in the row.
		extractReservationKeys(sibling, &h)

		if h.Branch != "" || h.CallNumber != "" {
			holdings = append(holdings, h)
		}
	}

	return holdings, nil
}

// extractReservationKeys finds an <a> with href containing "menu=request"
// in the given row and extracts bibkey and itemkey query parameters.
func extractReservationKeys(row *html.Node, h *service.Holding) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.Contains(attr.Val, "menu=request") {
					if u, err := url.Parse(attr.Val); err == nil {
						h.BibKey = u.Query().Get("bibkey")
						h.ItemKey = u.Query().Get("itemkey")
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(row)
}

// findHoldingsHeader walks the HTML tree to find the <tr> header row
// for the holdings table. It checks direct child cells to avoid matching
// outer wrapper rows that contain the entire page.
func findHoldingsHeader(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "tr" {
		cells := directCells(n)
		if len(cells) >= 3 {
			var texts []string
			for _, c := range cells {
				texts = append(texts, strings.TrimSpace(nodeText(c)))
			}
			joined := strings.Join(texts, " ")
			if strings.Contains(joined, "Cota") && (strings.Contains(joined, "Local") || strings.Contains(joined, "Estado")) {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findHoldingsHeader(c); result != nil {
			return result
		}
	}
	return nil
}

// directCells returns the direct <td> and <th> child elements of a node.
func directCells(n *html.Node) []*html.Node {
	var cells []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, c)
		}
	}
	return cells
}

// nodeText recursively extracts all text content from an HTML node.
func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(nodeText(c))
	}
	return sb.String()
}
