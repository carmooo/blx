package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/joao-carmo/blx/internal/handler/mocks"
	"github.com/joao-carmo/blx/internal/service"
)

func TestSearchHandler(t *testing.T) {
	svc := mocks.NewMockCatalogService(t)
	svc.EXPECT().
		Search(mock.Anything, mock.MatchedBy(func(p service.SearchParams) bool {
			return p.Query == "test"
		})).
		Return(&service.SearchResult{
			Total:   1,
			Page:    1,
			PerPage: 20,
			Results: []service.SearchItem{
				{ID: "123", Title: "Test Book"},
			},
		}, nil)

	h := New(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/items/search?q=test", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result service.SearchResult
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	require.Len(t, result.Results, 1)
	assert.Equal(t, "123", result.Results[0].ID)
	assert.Equal(t, "Test Book", result.Results[0].Title)
}

func TestSearchHandlerMissingQuery(t *testing.T) {
	svc := mocks.NewMockCatalogService(t)

	h := New(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/items/search", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp apiError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "missing_query", resp.Error.Code)
}

func TestItemHandler(t *testing.T) {
	svc := mocks.NewMockCatalogService(t)
	svc.EXPECT().
		GetItem(mock.Anything, "test-id").
		Return(&service.Item{
			ID:    "test-id",
			Title: "Test Book",
			Holdings: []service.Holding{
				{
					Branch:     "Central Library",
					BranchCode: "CMLBC4",
					CallNumber: "821.134.3",
					Collection: "General",
					Status:     "Available",
				},
			},
		}, nil)

	h := New(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/items/test-id", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var item service.Item
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&item))
	assert.Equal(t, "test-id", item.ID)
	assert.Equal(t, "Test Book", item.Title)
	require.Len(t, item.Holdings, 1)
	assert.Equal(t, "Central Library", item.Holdings[0].Branch)
	assert.Equal(t, "Available", item.Holdings[0].Status)
}
