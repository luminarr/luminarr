package traktlist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/trakt"
)

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

func newTestPlugin(t *testing.T, cfg Config, serverURL string) *Plugin {
	t.Helper()
	c := trakt.New("test-client-id", nil).WithBaseURL(serverURL).WithHTTPClient(&http.Client{})
	return &Plugin{cfg: cfg, client: c}
}

// ── Factory ──────────────────────────────────────────────────────────────────

func TestFactory_Valid_Watchlist(t *testing.T) {
	settings := json.RawMessage(`{"username":"jdoe","list_type":"watchlist"}`)
	p, err := registry.Default.NewImportList("trakt_list", settings)
	if err != nil {
		t.Fatalf("NewImportList() error = %v", err)
	}
	if p.Name() != "Trakt List" {
		t.Errorf("Name() = %q, want Trakt List", p.Name())
	}
}

func TestFactory_Valid_CustomList(t *testing.T) {
	settings := json.RawMessage(`{"username":"jdoe","list_type":"custom","list_slug":"my-list"}`)
	_, err := registry.Default.NewImportList("trakt_list", settings)
	if err != nil {
		t.Fatalf("NewImportList() error = %v", err)
	}
}

func TestFactory_DefaultsToWatchlist(t *testing.T) {
	settings := json.RawMessage(`{"username":"jdoe"}`)
	_, err := registry.Default.NewImportList("trakt_list", settings)
	if err != nil {
		t.Fatalf("NewImportList() error = %v", err)
	}
}

func TestFactory_MissingUsername(t *testing.T) {
	settings := json.RawMessage(`{"list_type":"watchlist"}`)
	_, err := registry.Default.NewImportList("trakt_list", settings)
	if err == nil {
		t.Fatal("expected error for missing username")
	}
}

func TestFactory_CustomWithoutListSlug(t *testing.T) {
	settings := json.RawMessage(`{"username":"jdoe","list_type":"custom"}`)
	_, err := registry.Default.NewImportList("trakt_list", settings)
	if err == nil {
		t.Fatal("expected error for custom list_type without list_slug")
	}
}

func TestFactory_InvalidJSON(t *testing.T) {
	_, err := registry.Default.NewImportList("trakt_list", json.RawMessage(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ── Fetch ────────────────────────────────────────────────────────────────────

func TestFetch_Watchlist_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/jdoe/watchlist/movies" {
			t.Errorf("path = %q, want /users/jdoe/watchlist/movies", r.URL.Path)
		}
		resp := []map[string]any{
			{"movie": map[string]any{"title": "Inception", "year": 2010, "ids": map[string]any{"tmdb": 27205, "imdb": "tt1375666"}}},
			{"movie": map[string]any{"title": "Unknown", "year": 2020, "ids": map[string]any{"tmdb": 0}}},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin(t, Config{Username: "jdoe", ListType: "watchlist"}, srv.URL)
	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TMDbID != 27205 {
		t.Errorf("TMDbID = %d, want 27205", items[0].TMDbID)
	}
	if items[0].IMDbID != "tt1375666" {
		t.Errorf("IMDbID = %q, want tt1375666", items[0].IMDbID)
	}
}

func TestFetch_CustomList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/jdoe/lists/sci-fi/items/movies" {
			t.Errorf("path = %q, want /users/jdoe/lists/sci-fi/items/movies", r.URL.Path)
		}
		resp := []map[string]any{
			{"movie": map[string]any{"title": "Dune", "year": 2021, "ids": map[string]any{"tmdb": 438631, "imdb": "tt1160419"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin(t, Config{Username: "jdoe", ListType: "custom", ListSlug: "sci-fi"}, srv.URL)
	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TMDbID != 438631 {
		t.Errorf("TMDbID = %d, want 438631", items[0].TMDbID)
	}
}

func TestFetch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestPlugin(t, Config{Username: "jdoe", ListType: "watchlist"}, srv.URL)
	_, err := p.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

// ── Test ─────────────────────────────────────────────────────────────────────

func TestTest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	p := newTestPlugin(t, Config{Username: "jdoe", ListType: "watchlist"}, srv.URL)
	if err := p.Test(context.Background()); err != nil {
		t.Fatalf("Test() = %v", err)
	}
}

func TestTest_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := newTestPlugin(t, Config{Username: "jdoe", ListType: "watchlist"}, srv.URL)
	if err := p.Test(context.Background()); err == nil {
		t.Fatal("expected error for 403")
	}
}
