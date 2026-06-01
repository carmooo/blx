package ipac

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/joao-carmo/blx/internal/service"
)

const (
	defaultBaseURL = "https://catalogolx.cm-lisboa.pt/ipac20"
	maxConcurrent  = 5
	rateLimit      = 5.0
	rateBurst      = 10
	maxRetries     = 3
	initialBackoff = 60 * time.Second
)

// Client handles HTTP communication with the iPAC catalog.
type Client struct {
	baseURL    string
	httpClient *http.Client
	sem        chan struct{}
	limiter    *rate.Limiter
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL overrides the default base URL (useful for testing).
func WithBaseURL(u string) ClientOption {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient creates a new iPAC HTTP client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		sem:        make(chan struct{}, maxConcurrent),
		limiter:    rate.NewLimiter(rate.Limit(rateLimit), rateBurst),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Fetch performs an HTTP GET to the given path, respecting concurrency
// limits, rate limits, and retry logic for 429 responses.
func (c *Client) Fetch(ctx context.Context, path string) ([]byte, error) {
	// Acquire semaphore slot.
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for rate limiter.
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	fullURL := c.baseURL + path

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Check context before sleeping.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("do request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			backoff := retryAfterDuration(resp, attempt)
			lastErr = fmt.Errorf("429 Too Many Requests (attempt %d/%d)", attempt+1, maxRetries+1)

			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, fullURL)
		}

		return body, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// retryAfterDuration computes the backoff duration for a 429 response.
// It uses the Retry-After header if present, otherwise exponential backoff.
func retryAfterDuration(resp *http.Response, attempt int) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(math.Pow(2, float64(attempt))) * initialBackoff
}

// Repository implements service.CatalogRepository using the iPAC system.
type Repository struct {
	client *Client
}

// NewRepository creates a new iPAC repository.
func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// indexMapping maps service-layer index names to iPAC index codes.
var indexMapping = map[string]string{
	"keyword":    ".GW",
	"title":      ".TW",
	"author":     "BAW",
	"subject":    ".SW",
	"collection": ".CW",
	"publisher":  ".EW",
	"place":      ".PW",
}

// sortMapping maps service-layer sort names to iPAC sort codes.
var sortMapping = map[string]string{
	"newest":   "3100062",
	"oldest":   "3100063",
	"title_az": "3100038",
}

// Search performs a catalog search and returns results.
func (r *Repository) Search(ctx context.Context, params service.SearchParams) (*service.SearchResult, error) {
	index := indexMapping[params.Index]
	if index == "" {
		index = indexMapping["keyword"]
	}

	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 20
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}

	q := url.Values{}
	q.Set("profile", "rbml")
	q.Set("index", index)
	q.Set("term", params.Query)
	q.Set("npp", strconv.Itoa(perPage))

	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}

	if sortCode, ok := sortMapping[params.Sort]; ok {
		q.Set("sort", sortCode)
	}

	if params.Branch != "" {
		q.Set("limitbox_1", "LOC01 = "+params.Branch)
	}

	if params.Language != "" {
		q.Set("limitbox_4", "LNG01 = "+params.Language)
	}

	path := "/rss.jsp?" + q.Encode()
	// iPAC expects spaces in limitbox values, not +.
	path = strings.ReplaceAll(path, "+", "%20")

	data, err := r.client.Fetch(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("fetch search results: %w", err)
	}

	items, err := parseRSS(data)
	if err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}

	return &service.SearchResult{
		Total:   0, // RSS feed doesn't provide total count
		Page:    page,
		PerPage: perPage,
		Results: items,
	}, nil
}

// GetItem retrieves full item metadata. Not yet implemented.
func (r *Repository) GetItem(_ context.Context, _ string) (*service.Item, error) {
	return nil, fmt.Errorf("GetItem: not implemented")
}

// GetHoldings retrieves holdings for an item. Not yet implemented.
func (r *Repository) GetHoldings(_ context.Context, _ string) ([]service.Holding, error) {
	return nil, fmt.Errorf("GetHoldings: not implemented")
}
