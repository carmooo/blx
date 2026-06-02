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
	Total   int          `json:"total,omitempty"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
	Results []SearchItem `json:"results"`
}

// SearchItem is a single item in search results (lightweight).
type SearchItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Item is a full catalog item with all metadata.
type Item struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Authors        []Author  `json:"authors,omitempty"`
	Publisher      string    `json:"publisher,omitempty"`
	Place          string    `json:"place,omitempty"`
	Year           string    `json:"year,omitempty"`
	Edition        string    `json:"edition,omitempty"`
	Physical       string    `json:"physical,omitempty"`
	Subjects       []string  `json:"subjects,omitempty"`
	Classification string    `json:"classification,omitempty"`
	Language       string    `json:"language,omitempty"`
	ISBN           string    `json:"isbn,omitempty"`
	Type           string    `json:"type,omitempty"`
	Holdings       []Holding `json:"holdings,omitempty"`
}

// Author represents an item's author.
type Author struct {
	Name  string `json:"name"`
	Dates string `json:"dates,omitempty"`
	Role  string `json:"role"`
}

// Holding represents a copy of an item at a branch.
type Holding struct {
	Branch     string `json:"branch"`
	BranchCode string `json:"branch_code,omitempty"`
	CallNumber string `json:"call_number"`
	Collection string `json:"collection"`
	Status     string `json:"status"`
	LoanDays   int    `json:"loan_days,omitempty"`
}

// Branch represents a library branch.
type Branch struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// FilterOption is a selectable filter value.
type FilterOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// CatalogRepository is the interface for accessing catalog data.
type CatalogRepository interface {
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)
	GetItem(ctx context.Context, id string) (*Item, error)
	GetHoldings(ctx context.Context, id string) ([]Holding, error)
}
