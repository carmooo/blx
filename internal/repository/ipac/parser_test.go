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

func TestParseMarcXML(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/marcxml_item.xml")
	require.NoError(t, err)

	item, err := parseMarcXML(data)
	require.NoError(t, err)

	assert.Equal(t, "Relação do reino de Congo e das terras circunvizinhas", item.Title)
	assert.Equal(t, "1949", item.Year)
	assert.Equal(t, "Lisboa", item.Place)
	assert.Equal(t, "Agência Geral das Colónias", item.Publisher)
	assert.Equal(t, "ita", item.Language)
	assert.Equal(t, "Ed. fac-similada", item.Edition)
	assert.Equal(t, "136 p., pag. var. ; 8 grav. desdobr., 2 map. desdobr. ; 25 cm", item.Physical)
	assert.Equal(t, "910.4(673+675)\"15\"", item.Classification)

	// Authors: at least 2 (700 + 701).
	require.GreaterOrEqual(t, len(item.Authors), 2)
	assert.Equal(t, "Lopes, Duarte", item.Authors[0].Name)
	assert.Equal(t, "fl. 1578", item.Authors[0].Dates)
	assert.Equal(t, "author", item.Authors[0].Role)

	assert.Equal(t, "Pigafetta, Filippo", item.Authors[1].Name)
	assert.Equal(t, "author", item.Authors[1].Role)

	// Contributor (702).
	require.Len(t, item.Authors, 3)
	assert.Equal(t, "Capeans, Rosa", item.Authors[2].Name)
	assert.Equal(t, "contributor", item.Authors[2].Role)

	// Subjects.
	require.Len(t, item.Subjects, 1)
	assert.Contains(t, item.Subjects[0], "Reino do Congo")
	assert.Contains(t, item.Subjects[0], "Séc. 16")
	assert.Contains(t, item.Subjects[0], "[Narrativas de viagens]")
}

func TestParseMarcXML_EmptyCollection(t *testing.T) {
	data := []byte(`<collection></collection>`)
	_, err := parseMarcXML(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no records")
}

func TestParseHoldings(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/holdings_detail.html")
	require.NoError(t, err)

	holdings, err := parseHoldings(data)
	require.NoError(t, err)

	require.Len(t, holdings, 3)

	assert.Equal(t, "Biblioteca Camões", holdings[0].Branch)
	assert.Equal(t, "910.4 LOP", holdings[0].CallNumber)
	assert.Equal(t, "Fundo Geral", holdings[0].Collection)
	assert.Equal(t, "Disponível", holdings[0].Status)
	assert.Equal(t, 15, holdings[0].LoanDays)

	assert.Equal(t, "Biblioteca de Belém", holdings[1].Branch)
	assert.Equal(t, "Emprestado", holdings[1].Status)

	assert.Equal(t, "Biblioteca Palácio Galveias", holdings[2].Branch)
	assert.Equal(t, "Reservados", holdings[2].Collection)
	assert.Equal(t, "Presença", holdings[2].Status)
	assert.Equal(t, 0, holdings[2].LoanDays) // empty loan days
}

func TestParseHoldings_NoTable(t *testing.T) {
	data := []byte(`<html><body><p>No holdings here</p></body></html>`)
	holdings, err := parseHoldings(data)
	require.NoError(t, err)
	assert.Empty(t, holdings)
}
