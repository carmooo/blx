package ipac

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRSS_Fixture(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/rss_search.xml")
	require.NoError(t, err)

	items, err := parseRSS(data)
	require.NoError(t, err)

	assert.Len(t, items, 3)

	for i, item := range items {
		assert.NotEmpty(t, item.ID, "item %d should have non-empty ID", i)
		assert.NotEmpty(t, item.Title, "item %d should have non-empty Title", i)
	}

	// Verify specific IDs from the fixture.
	assert.Equal(t, "3100024~!29331~!0", items[0].ID)
	assert.Equal(t, "3100024~!38087~!1", items[1].ID)
	assert.Equal(t, "3100024~!38887~!2", items[2].ID)

	// Verify titles.
	assert.Equal(t, "Obras de José Saramago", items[0].Title)
}

func TestParseRSS_EmptyFeed(t *testing.T) {
	data := []byte(`<rss version="2.0"><channel><title>Test</title></channel></rss>`)
	items, err := parseRSS(data)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestParseRSS_InvalidXML(t *testing.T) {
	data := []byte(`not xml at all`)
	_, err := parseRSS(data)
	assert.Error(t, err)
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		name    string
		link    string
		want    string
		wantErr bool
	}{
		{
			name: "valid link",
			link: "http://bibliotecaslx.cm-lisboa.pt/ipac20/ipac.jsp?session=&profile=rbml&menu=search&aspect=basic_search&uri=full=3100024~!29331~!0",
			want: "3100024~!29331~!0",
		},
		{
			name:    "no uri param",
			link:    "http://example.com/ipac.jsp?foo=bar",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			link:    "://bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractID(tt.link)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestStripXMLDeclaration(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="ISO-8859-1"?><rss>content</rss>`)
	result := stripXMLDeclaration(input)
	assert.Equal(t, "<rss>content</rss>", string(result))
}
