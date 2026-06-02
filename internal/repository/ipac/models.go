package ipac

// rssResponse is the top-level RSS XML structure.
type rssResponse struct {
	Channel rssChannel `xml:"channel"`
}

// rssChannel holds the channel metadata and items.
type rssChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

// rssItem represents a single search result in the RSS feed.
type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// marcCollection is the top-level MarcXchange XML structure.
type marcCollection struct {
	Records []marcRecord `xml:"record"`
}

// marcRecord represents a single MARC record.
type marcRecord struct {
	Type          string             `xml:"type,attr"`
	Format        string             `xml:"format,attr"`
	Leader        string             `xml:"leader"`
	ControlFields []marcControlField `xml:"controlfield"`
	DataFields    []marcDataField    `xml:"datafield"`
}

// marcControlField is a MARC control field (tags 001-009).
type marcControlField struct {
	Tag   string `xml:"tag,attr"`
	Value string `xml:",chardata"`
}

// marcDataField is a MARC data field with indicators and subfields.
type marcDataField struct {
	Tag       string         `xml:"tag,attr"`
	Ind1      string         `xml:"ind1,attr"`
	Ind2      string         `xml:"ind2,attr"`
	Subfields []marcSubfield `xml:"subfield"`
}

// marcSubfield is a single subfield within a data field.
type marcSubfield struct {
	Code  string `xml:"code,attr"`
	Value string `xml:",chardata"`
}
