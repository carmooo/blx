package service

import "context"

// SearchParams represents search criteria from the caller.
type SearchParams struct {
	Query    string
	Index    string // keyword, title, author, subject, collection, publisher, place
	Page     int
	PerPage  int
	Sort     string // newest, oldest, title_az
	Branch   string // branch code e.g. CMLBC4
	Type     string // book, dvd, cd, etc.
	Language string // ISO 639-2 code e.g. por
	YearFrom string
	YearTo   string
}

// SearchResult is the response from a search.
type SearchResult struct {
	Total   int // 0 if unknown
	Page    int
	PerPage int
	Results []SearchItem
}

// SearchItem is a single item in search results (lightweight).
type SearchItem struct {
	ID    string
	Title string
}

// Item is a full catalog item with all metadata.
type Item struct {
	ID             string
	Title          string
	Authors        []Author
	Publisher      string
	Place          string
	Year           string
	Edition        string
	Physical       string
	Subjects       []string
	Classification string
	Language       string
	ISBN           string
	Type           string
	Holdings       []Holding
}

// Author represents an item's author.
type Author struct {
	Name  string
	Dates string
	Role  string
}

// Holding represents a copy of an item at a branch.
type Holding struct {
	Branch     string
	BranchCode string
	CallNumber string
	Collection string
	Status     string
	LoanDays   int
}

// Branch represents a library branch.
type Branch struct {
	Code string
	Name string
}

// FilterOption is a selectable filter value.
type FilterOption struct {
	Code string
	Name string
}

// CatalogRepository is the interface for accessing catalog data.
type CatalogRepository interface {
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)
	GetItem(ctx context.Context, id string) (*Item, error)
	GetHoldings(ctx context.Context, id string) ([]Holding, error)
}
