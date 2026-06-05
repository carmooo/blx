package service

import (
	"context"
	"fmt"
)

// CatalogService orchestrates catalog operations.
type CatalogService struct {
	repo CatalogRepository
}

// NewCatalogService creates a new catalog service.
func NewCatalogService(repo CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

// Search performs a catalog search.
func (s *CatalogService) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	result, err := s.repo.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return result, nil
}

// GetItem retrieves full item metadata with holdings.
func (s *CatalogService) GetItem(ctx context.Context, id string) (*Item, error) {
	item, err := s.repo.GetItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}

	holdings, err := s.repo.GetHoldings(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get holdings: %w", err)
	}
	item.Holdings = holdings

	return item, nil
}
