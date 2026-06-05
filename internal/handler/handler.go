package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/joao-carmo/blx/internal/service"
)

// CatalogService defines the operations needed by the handler.
type CatalogService interface {
	Search(ctx context.Context, params service.SearchParams) (*service.SearchResult, error)
	GetItem(ctx context.Context, id string) (*service.Item, error)
}

// Handler serves the BLX HTTP API.
type Handler struct {
	svc CatalogService
	mux *http.ServeMux
}

// New creates a new Handler with routes registered.
func New(svc CatalogService) *Handler {
	h := &Handler{
		svc: svc,
		mux: http.NewServeMux(),
	}
	h.mux.HandleFunc("GET /api/items/search", h.handleSearch)
	h.mux.HandleFunc("GET /api/items/{id...}", h.handleItem)
	return h
}

// ServeHTTP delegates to the internal mux.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := q.Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "missing_query", "query parameter 'q' is required")
		return
	}

	params := service.SearchParams{
		Query:    query,
		Index:    q.Get("index"),
		Sort:     q.Get("sort"),
		Branch:   q.Get("branch"),
		Language: q.Get("lang"),
	}

	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Page = n
		}
	}

	if v := q.Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.PerPage = n
		}
	}

	result, err := h.svc.Search(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search_failed", "could not search catalog")
		return
	}

	writeSearchResult(w, http.StatusOK, result)
}

func (h *Handler) handleItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "item ID is required")
		return
	}

	item, err := h.svc.GetItem(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get_item_failed", "could not get item")
		return
	}

	writeItem(w, http.StatusOK, item)
}

type apiError struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiError{
		Error: errorBody{Code: code, Message: message},
	})
}

func writeSearchResult(w http.ResponseWriter, status int, result *service.SearchResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(result)
}

func writeItem(w http.ResponseWriter, status int, item *service.Item) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(item)
}
