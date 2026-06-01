package ipac

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"

	"github.com/joao-carmo/blx/internal/service"
)

// parseRSS parses RSS XML bytes into a slice of SearchItem.
// It strips the XML declaration to avoid encoding issues (the feed
// declares ISO-8859-1 but Go's xml package only supports UTF-8).
func parseRSS(data []byte) ([]service.SearchItem, error) {
	cleaned := stripXMLDeclaration(data)

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
