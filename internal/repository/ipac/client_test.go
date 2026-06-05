package ipac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joao-carmo/blx/internal/service"
)

func TestClient_Fetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	data, err := client.Fetch(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestClient_Fetch_ConcurrencyLimit(t *testing.T) {
	var concurrent atomic.Int32
	var maxSeen atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cur := concurrent.Add(1)
		// Track max concurrent requests.
		for {
			old := maxSeen.Load()
			if cur <= old || maxSeen.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		concurrent.Add(-1)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))

	// Launch more requests than the concurrency limit.
	ctx := context.Background()
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = client.Fetch(ctx, "/test")
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.LessOrEqual(t, maxSeen.Load(), int32(maxConcurrent),
		"concurrent requests should not exceed %d", maxConcurrent)
}

func TestClient_Fetch_RetryOn429(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("success"))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	data, err := client.Fetch(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, "success", string(data))
	assert.Equal(t, int32(3), attempts.Load())
}

func TestClient_Fetch_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		_, _ = w.Write([]byte("slow"))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Fetch(ctx, "/test")
	assert.Error(t, err)
}

func TestClient_Fetch_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	_, err := client.Fetch(context.Background(), "/test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestRepository_Search_URLConstruction(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>Test</title></channel></rss>`))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	repo := NewRepository(client)

	_, err := repo.Search(context.Background(), service.SearchParams{
		Query: "Saramago",
		Index: "author",
	})
	require.NoError(t, err)

	assert.Contains(t, receivedPath, "index=BAW")
	assert.Contains(t, receivedPath, "term=Saramago")
	assert.Contains(t, receivedPath, "profile=rbml")
}

func TestRepository_GetItem(t *testing.T) {
	fixture, err := os.ReadFile("../../../testdata/marcxml_item.xml")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RequestURI(), "regiso.jsp")
		assert.Contains(t, r.URL.RequestURI(), "marcxchange=true")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	repo := NewRepository(client)

	item, err := repo.GetItem(context.Background(), "3100024~!1~!0")
	require.NoError(t, err)

	assert.Equal(t, "3100024~!1~!0", item.ID)
	assert.Equal(t, "Relação do reino de Congo e das terras circunvizinhas", item.Title)
	assert.Equal(t, "1949", item.Year)
	assert.GreaterOrEqual(t, len(item.Authors), 2)
}

func TestRepository_GetHoldings(t *testing.T) {
	fixture, err := os.ReadFile("../../../testdata/holdings_detail.html")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RequestURI(), "ipac.jsp")
		assert.Contains(t, r.URL.RequestURI(), "profile=rbml")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	repo := NewRepository(client)

	holdings, err := repo.GetHoldings(context.Background(), "3100024~!1~!0")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(holdings), 1)
	assert.NotEmpty(t, holdings[0].Branch)
	assert.NotEmpty(t, holdings[0].CallNumber)
	assert.NotEmpty(t, holdings[0].Collection)
	assert.NotEmpty(t, holdings[0].Status)
}
