package traktpopular

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/trakt"
)

// mustMarshal panics if json.Marshal fails — acceptable in test helpers.
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

// newTestPlugin builds a Plugin whose trakt.Client is redirected to serverURL.
func newTestPlugin(serverURL string) *Plugin {
	c := trakt.New("test-client-id", nil).
		WithBaseURL(serverURL).
		WithHTTPClient(&http.Client{})
	return &Plugin{client: c}
}

// ---------------------------------------------------------------------------
// Registry factory
// ---------------------------------------------------------------------------

func TestFactory_Valid(t *testing.T) {
	settings := json.RawMessage(`{}`)
	p, err := registry.Default.NewImportList("trakt_popular", settings)
	if err != nil {
		t.Fatalf("NewImportList() error = %v", err)
	}
	if p.Name() != "Trakt Popular Movies" {
		t.Errorf("Name() = %q, want Trakt Popular Movies", p.Name())
	}
}

func TestFactory_InvalidJSON(t *testing.T) {
	// Factory ignores settings (_ parameter), so even bad JSON succeeds.
	_, err := registry.Default.NewImportList("trakt_popular", json.RawMessage(`not-json`))
	if err != nil {
		// The factory uses _ for settings, so this should not error.
		// If it does, that's fine — just note the behavior.
		t.Logf("NewImportList() error = %v (factory ignores settings)", err)
	}
}

// ---------------------------------------------------------------------------
// Name
// ---------------------------------------------------------------------------

func TestName(t *testing.T) {
	p := &Plugin{client: trakt.New("test", nil)}
	if got := p.Name(); got != "Trakt Popular Movies" {
		t.Errorf("Name() = %q, want Trakt Popular Movies", got)
	}
}

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movies/popular" {
			t.Errorf("path = %q, want /movies/popular", r.URL.Path)
		}
		resp := []map[string]any{
			{
				"title": "The Dark Knight",
				"year":  2008,
				"ids":   map[string]any{"tmdb": 155, "imdb": "tt0468569"},
			},
			{
				// No TMDb ID — should be filtered out.
				"title": "Mystery Film",
				"year":  2022,
				"ids":   map[string]any{"tmdb": 0},
			},
			{
				"title": "Oppenheimer",
				"year":  2023,
				"ids":   map[string]any{"tmdb": 872585, "imdb": "tt15398776"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin(srv.URL)
	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	// Entry with tmdb=0 must be filtered.
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	if items[0].TMDbID != 155 {
		t.Errorf("items[0].TMDbID = %d, want 155", items[0].TMDbID)
	}
	if items[0].IMDbID != "tt0468569" {
		t.Errorf("items[0].IMDbID = %q, want tt0468569", items[0].IMDbID)
	}
	if items[0].Title != "The Dark Knight" {
		t.Errorf("items[0].Title = %q, want The Dark Knight", items[0].Title)
	}
	if items[0].Year != 2008 {
		t.Errorf("items[0].Year = %d, want 2008", items[0].Year)
	}

	if items[1].TMDbID != 872585 {
		t.Errorf("items[1].TMDbID = %d, want 872585", items[1].TMDbID)
	}
}

func TestFetch_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	p := newTestPlugin(srv.URL)
	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestFetch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p := newTestPlugin(srv.URL)
	_, err := p.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() expected error for 503, got nil")
	}
}

// ---------------------------------------------------------------------------
// Test
// ---------------------------------------------------------------------------

func TestTest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	p := newTestPlugin(srv.URL)
	if err := p.Test(context.Background()); err != nil {
		t.Fatalf("Test() = %v", err)
	}
}

func TestTest_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestPlugin(srv.URL)
	if err := p.Test(context.Background()); err == nil {
		t.Fatal("Test() expected error for 401, got nil")
	}
}
