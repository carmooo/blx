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
