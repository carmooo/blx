package web

import (
	"context"
	"net/http"

	"github.com/joao-carmo/blx/internal/service"
)

// CatalogService defines the operations needed by the web handler.
type CatalogService interface {
	Search(ctx context.Context, params service.SearchParams) (*service.SearchResult, error)
	GetItem(ctx context.Context, id string) (*service.Item, error)
}

// Handler serves the BLX web frontend.
type Handler struct {
	mux *http.ServeMux
	svc CatalogService
}

// New creates a web Handler with routes registered.
func New(svc CatalogService) *Handler {
	h := &Handler{
		mux: http.NewServeMux(),
		svc: svc,
	}
	h.mux.HandleFunc("GET /{$}", h.handleHome)
	h.mux.HandleFunc("GET /search", h.handleSearch)
	h.mux.HandleFunc("GET /item/{id...}", h.handleItem)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	_ = SearchPage("").Render(r.Context(), w)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		_ = SearchPage("").Render(r.Context(), w)
		return
	}

	// If htmx request, return just the results fragment
	if r.Header.Get("HX-Request") == "true" {
		result, err := h.svc.Search(r.Context(), service.SearchParams{
			Query: query,
			Index: "keyword",
		})
		if err != nil {
			http.Error(w, "could not search catalog", http.StatusInternalServerError)
			return
		}
		_ = SearchResults(result.Results).Render(r.Context(), w)
		return
	}

	// Full page request
	_ = SearchPage(query).Render(r.Context(), w)
}

func (h *Handler) handleItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "item ID is required", http.StatusBadRequest)
		return
	}

	item, err := h.svc.GetItem(r.Context(), id)
	if err != nil {
		http.Error(w, "could not get item", http.StatusInternalServerError)
		return
	}

	_ = ItemPage(item).Render(r.Context(), w)
}
