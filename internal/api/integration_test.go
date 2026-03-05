package api_test

// Integration tests exercise the full HTTP stack: real in-memory SQLite,
// real services, and the live huma/chi router. No external services are
// required. Run with: go test -run Integration ./internal/api/...

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davidfic/luminarr/internal/api"
	"github.com/davidfic/luminarr/internal/config"
	"github.com/davidfic/luminarr/internal/core/downloader"
	"github.com/davidfic/luminarr/internal/core/health"
	"github.com/davidfic/luminarr/internal/core/indexer"
	"github.com/davidfic/luminarr/internal/core/library"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/core/notification"
	"github.com/davidfic/luminarr/internal/core/quality"
	"github.com/davidfic/luminarr/internal/core/queue"
	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/internal/scheduler"
	"github.com/davidfic/luminarr/internal/testutil"
	"github.com/davidfic/luminarr/pkg/plugin"
)

const testAPIKey = "test-integration-key-abc123"

// registerTestPlugins adds lightweight mock plugins to reg so CRUD endpoints
// can be exercised without hitting real external services.
func registerTestPlugins(reg *registry.Registry) {
	reg.RegisterIndexer("torznab", func(s json.RawMessage) (plugin.Indexer, error) {
		var cfg struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(s, &cfg); err != nil || cfg.URL == "" {
			return nil, errors.New("torznab: url is required")
		}
		return &testIndexer{}, nil
	})
	reg.RegisterIndexerSanitizer("torznab", func(s json.RawMessage) json.RawMessage { return s })

	reg.RegisterDownloader("qbittorrent", func(s json.RawMessage) (plugin.DownloadClient, error) {
		var cfg struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(s, &cfg); err != nil || cfg.URL == "" {
			return nil, errors.New("qbittorrent: url is required")
		}
		return &testDownloadClient{}, nil
	})
	reg.RegisterDownloaderSanitizer("qbittorrent", func(s json.RawMessage) json.RawMessage { return s })
}

type testIndexer struct{}

func (m *testIndexer) Name() string              { return "test-indexer" }
func (m *testIndexer) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }
func (m *testIndexer) Capabilities(_ context.Context) (plugin.Capabilities, error) {
	return plugin.Capabilities{SearchAvailable: true, MovieSearch: true}, nil
}
func (m *testIndexer) Search(_ context.Context, _ plugin.SearchQuery) ([]plugin.Release, error) {
	return nil, nil
}
func (m *testIndexer) GetRecent(_ context.Context) ([]plugin.Release, error) { return nil, nil }
func (m *testIndexer) Test(_ context.Context) error                          { return nil }

type testDownloadClient struct{}

func (m *testDownloadClient) Name() string              { return "test-downloader" }
func (m *testDownloadClient) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }
func (m *testDownloadClient) Test(_ context.Context) error {
	return nil
}
func (m *testDownloadClient) Add(_ context.Context, _ plugin.Release) (string, error) {
	return "test-item-id", nil
}
func (m *testDownloadClient) Status(_ context.Context, id string) (plugin.QueueItem, error) {
	return plugin.QueueItem{ClientItemID: id, Status: plugin.StatusDownloading}, nil
}
func (m *testDownloadClient) GetQueue(_ context.Context) ([]plugin.QueueItem, error) { return nil, nil }
func (m *testDownloadClient) Remove(_ context.Context, _ string, _ bool) error       { return nil }

// newIntegrationRouterFromDB builds a fully-wired router using the provided
// queries so that callers can seed data directly into the same DB.
func newIntegrationRouterFromDB(t *testing.T, q *dbsqlite.Queries) http.Handler {
	t.Helper()
	logger := slog.Default()
	bus := events.New(logger)
	reg := registry.New()
	registerTestPlugins(reg)

	qualSvc := quality.NewService(q, bus)
	libSvc := library.NewService(q, bus, nil)
	movieSvc := movie.NewService(q, nil /* no TMDB */, bus, logger)
	idxSvc := indexer.NewService(q, reg, bus)
	dlSvc := downloader.NewService(q, reg, bus)
	queueSvc := queue.NewService(q, dlSvc, bus, logger)
	notifSvc := notification.NewService(q, reg)
	healthSvc := health.NewService(libSvc, dlSvc, idxSvc, logger)
	sched := scheduler.New(logger)

	return api.NewRouter(api.RouterConfig{
		Auth:                config.Secret(testAPIKey),
		Logger:              logger,
		StartTime:           time.Now(),
		DBType:              "sqlite",
		AIEnabled:           false,
		QualityService:      qualSvc,
		LibraryService:      libSvc,
		MovieService:        movieSvc,
		IndexerService:      idxSvc,
		DownloaderService:   dlSvc,
		QueueService:        queueSvc,
		Scheduler:           sched,
		NotificationService: notifSvc,
		HealthService:       healthSvc,
		Bus:                 bus,
	})
}

// newIntegrationRouter builds a fully-wired router backed by an in-memory DB.
func newIntegrationRouter(t *testing.T) http.Handler {
	t.Helper()
	q := testutil.NewTestDB(t)
	return newIntegrationRouterFromDB(t, q)
}

// do performs a request against the given handler.
func do(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encoding request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("X-Api-Key", testAPIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// qualityBody returns a full plugin.Quality JSON object suitable for request bodies.
func qualityBody(resolution, source, codec, hdr, name string) map[string]any {
	return map[string]any{
		"resolution": resolution,
		"source":     source,
		"codec":      codec,
		"hdr":        hdr,
		"name":       name,
	}
}

// ── /health ───────────────────────────────────────────────────────────────────

func TestIntegration_HealthProbe(t *testing.T) {
	h := newIntegrationRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, want 200", rec.Code)
	}
}

func TestIntegration_AuthRequired(t *testing.T) {
	h := newIntegrationRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	// No X-Api-Key header.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request = %d, want 401", rec.Code)
	}
}

// ── /api/v1/system ────────────────────────────────────────────────────────────

func TestIntegration_SystemStatus(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/system/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/system/status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if body["app_name"] != "Luminarr" {
		t.Errorf("app_name = %v, want Luminarr", body["app_name"])
	}
	if body["db_type"] != "sqlite" {
		t.Errorf("db_type = %v, want sqlite", body["db_type"])
	}
}

// ── /api/v1/quality-profiles ─────────────────────────────────────────────────

func TestIntegration_QualityProfiles_CRUD(t *testing.T) {
	h := newIntegrationRouter(t)

	webdl1080p := qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")

	// List — empty initially.
	rec := do(t, h, http.MethodGet, "/api/v1/quality-profiles", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/quality-profiles = %d; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d items", len(list))
	}

	// Create.
	createBody := map[string]any{
		"name":            "HD-1080p",
		"cutoff":          webdl1080p,
		"qualities":       []map[string]any{webdl1080p},
		"upgrade_allowed": false,
	}
	rec = do(t, h, http.MethodPost, "/api/v1/quality-profiles", createBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/quality-profiles = %d; body: %s", rec.Code, rec.Body)
	}
	var created map[string]any
	mustDecode(t, rec, &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created profile has no id")
	}
	if created["name"] != "HD-1080p" {
		t.Errorf("name = %v, want HD-1080p", created["name"])
	}

	// Get by ID.
	rec = do(t, h, http.MethodGet, "/api/v1/quality-profiles/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET profile = %d; body: %s", rec.Code, rec.Body)
	}

	// Update.
	updateBody := map[string]any{
		"name":            "HD-1080p-updated",
		"cutoff":          webdl1080p,
		"qualities":       []map[string]any{webdl1080p},
		"upgrade_allowed": false,
	}
	rec = do(t, h, http.MethodPut, "/api/v1/quality-profiles/"+id, updateBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT profile = %d; body: %s", rec.Code, rec.Body)
	}
	var updated map[string]any
	mustDecode(t, rec, &updated)
	if updated["name"] != "HD-1080p-updated" {
		t.Errorf("updated name = %v, want HD-1080p-updated", updated["name"])
	}

	// Delete.
	rec = do(t, h, http.MethodDelete, "/api/v1/quality-profiles/"+id, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE profile = %d; body: %s", rec.Code, rec.Body)
	}

	// Verify gone.
	rec = do(t, h, http.MethodGet, "/api/v1/quality-profiles/"+id, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted profile = %d, want 404", rec.Code)
	}
}

// ── /api/v1/libraries ────────────────────────────────────────────────────────

func TestIntegration_Libraries_CRUD(t *testing.T) {
	h := newIntegrationRouter(t)

	// Create a quality profile first — the libraries table has a FK constraint
	// on default_quality_profile_id (added in migration 00010).
	qpRec := do(t, h, http.MethodPost, "/api/v1/quality-profiles", map[string]any{
		"name":            "Test Profile",
		"cutoff":          qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p"),
		"qualities":       []map[string]any{qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")},
		"upgrade_allowed": false,
	})
	if qpRec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/quality-profiles = %d; body: %s", qpRec.Code, qpRec.Body)
	}
	var qp map[string]any
	mustDecode(t, qpRec, &qp)
	profileID, _ := qp["id"].(string)

	rec := do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name":                       "Movies",
		"root_path":                  "/movies",
		"default_quality_profile_id": profileID,
		"min_free_space_gb":          10,
		"tags":                       []string{},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/libraries = %d; body: %s", rec.Code, rec.Body)
	}
	var created map[string]any
	mustDecode(t, rec, &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("no library id returned")
	}

	// List.
	rec = do(t, h, http.MethodGet, "/api/v1/libraries", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/libraries = %d", rec.Code)
	}
	var libs []map[string]any
	mustDecode(t, rec, &libs)
	if len(libs) != 1 {
		t.Fatalf("expected 1 library, got %d", len(libs))
	}

	// Get by ID.
	rec = do(t, h, http.MethodGet, "/api/v1/libraries/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET library = %d; body: %s", rec.Code, rec.Body)
	}

	// Delete.
	rec = do(t, h, http.MethodDelete, "/api/v1/libraries/"+id, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE library = %d; body: %s", rec.Code, rec.Body)
	}
}

// ── /api/v1/movies ────────────────────────────────────────────────────────────

// TestIntegration_Movies_DegradedMode verifies that a movie can be added as a
// stub when no TMDB API key is configured. The movie is created with a
// placeholder title ("tmdb:<id>") and MetadataRefreshedAt is absent from the
// response.
func TestIntegration_Movies_DegradedMode(t *testing.T) {
	h := newIntegrationRouter(t)

	// Seed a quality profile via API.
	webdl1080p := qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")
	rec := do(t, h, http.MethodPost, "/api/v1/quality-profiles", map[string]any{
		"name":            "HD",
		"cutoff":          webdl1080p,
		"qualities":       []map[string]any{webdl1080p},
		"upgrade_allowed": false,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST quality-profile = %d; body: %s", rec.Code, rec.Body)
	}
	var profile map[string]any
	mustDecode(t, rec, &profile)
	profileID, _ := profile["id"].(string)

	// Seed a library via API.
	rec = do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name":                       "Movies",
		"root_path":                  "/movies",
		"default_quality_profile_id": profileID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST library = %d; body: %s", rec.Code, rec.Body)
	}
	var lib map[string]any
	mustDecode(t, rec, &lib)
	libraryID, _ := lib["id"].(string)

	// Add a movie — TMDB is not configured, so we expect a stub to be created.
	rec = do(t, h, http.MethodPost, "/api/v1/movies", map[string]any{
		"tmdb_id":            27205,
		"library_id":         libraryID,
		"quality_profile_id": profileID,
		"monitored":          true,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST movie (degraded) = %d; body: %s", rec.Code, rec.Body)
	}
	var movie map[string]any
	mustDecode(t, rec, &movie)
	if movie["title"] != "tmdb:27205" {
		t.Errorf("stub title = %q, want %q", movie["title"], "tmdb:27205")
	}
	if _, ok := movie["metadata_refreshed_at"]; ok {
		t.Error("stub movie should have no metadata_refreshed_at")
	}
}

func TestIntegration_Movies_ListEmpty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/movies", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/movies = %d; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_Indexers_UnknownKindRejected(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/indexers", map[string]any{
		"name": "Test",
		"kind": "no-such-kind",
		"settings": map[string]any{
			"url": "http://example.com",
		},
	})
	if rec.Code == http.StatusCreated {
		t.Fatalf("expected error creating indexer with unregistered kind, got 201")
	}
}

// ── /api/v1/indexers ─────────────────────────────────────────────────────────

func TestIntegration_Indexers_CRUD(t *testing.T) {
	h := newIntegrationRouter(t)

	// List — empty initially.
	rec := do(t, h, http.MethodGet, "/api/v1/indexers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/indexers = %d; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	// Create.
	rec = do(t, h, http.MethodPost, "/api/v1/indexers", map[string]any{
		"name": "Prowlarr",
		"kind": "torznab",
		"settings": map[string]any{
			"url":     "http://prowlarr:9696/1/api",
			"api_key": "test-key",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/indexers = %d; body: %s", rec.Code, rec.Body)
	}
	var created map[string]any
	mustDecode(t, rec, &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created indexer has no id")
	}
	if created["name"] != "Prowlarr" {
		t.Errorf("name = %v, want Prowlarr", created["name"])
	}

	// List — must contain the created indexer.
	rec = do(t, h, http.MethodGet, "/api/v1/indexers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/indexers after create = %d; body: %s", rec.Code, rec.Body)
	}
	mustDecode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 indexer after create, got %d", len(list))
	}

	// Get by ID.
	rec = do(t, h, http.MethodGet, "/api/v1/indexers/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/indexers/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Delete.
	rec = do(t, h, http.MethodDelete, "/api/v1/indexers/"+id, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/v1/indexers/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Verify gone.
	rec = do(t, h, http.MethodGet, "/api/v1/indexers/"+id, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted indexer = %d, want 404", rec.Code)
	}
}

// ── /api/v1/download-clients ──────────────────────────────────────────────────

func TestIntegration_DownloadClients_ListEmpty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/download-clients", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/download-clients = %d; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestIntegration_DownloadClients_CRUD(t *testing.T) {
	h := newIntegrationRouter(t)

	// List — empty initially.
	rec := do(t, h, http.MethodGet, "/api/v1/download-clients", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/download-clients = %d; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list initially, got %d", len(list))
	}

	// Create.
	rec = do(t, h, http.MethodPost, "/api/v1/download-clients", map[string]any{
		"name": "My qBittorrent",
		"kind": "qbittorrent",
		"settings": map[string]any{
			"url":      "http://qbittorrent:8080",
			"username": "admin",
			"password": "secret",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/download-clients = %d; body: %s", rec.Code, rec.Body)
	}
	var created map[string]any
	mustDecode(t, rec, &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created download client has no id")
	}
	if created["name"] != "My qBittorrent" {
		t.Errorf("name = %v, want My qBittorrent", created["name"])
	}
	if created["kind"] != "qbittorrent" {
		t.Errorf("kind = %v, want qbittorrent", created["kind"])
	}
	// Enabled defaults to true.
	if enabled, _ := created["enabled"].(bool); !enabled {
		t.Error("enabled = false, want true")
	}

	// List — must contain the created client.
	rec = do(t, h, http.MethodGet, "/api/v1/download-clients", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/download-clients after create = %d; body: %s", rec.Code, rec.Body)
	}
	mustDecode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 download client after create, got %d", len(list))
	}
	if list[0]["id"] != id {
		t.Errorf("list[0].id = %v, want %v", list[0]["id"], id)
	}

	// Get by ID.
	rec = do(t, h, http.MethodGet, "/api/v1/download-clients/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/download-clients/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Delete.
	rec = do(t, h, http.MethodDelete, "/api/v1/download-clients/"+id, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/v1/download-clients/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Verify gone.
	rec = do(t, h, http.MethodGet, "/api/v1/download-clients/"+id, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted download client = %d, want 404", rec.Code)
	}
}

func TestIntegration_DownloadClients_MissingURL_Rejected(t *testing.T) {
	h := newIntegrationRouter(t)
	// URL is required by the qbittorrent plugin.
	rec := do(t, h, http.MethodPost, "/api/v1/download-clients", map[string]any{
		"name":     "No URL",
		"kind":     "qbittorrent",
		"settings": map[string]any{},
	})
	if rec.Code == http.StatusCreated {
		t.Fatal("POST without URL should fail, got 201")
	}
}

// ── /api/v1/queue ────────────────────────────────────────────────────────────

func TestIntegration_Queue_ListEmpty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/queue", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/queue = %d; body: %s", rec.Code, rec.Body)
	}
}

// ── /api/v1/tasks ────────────────────────────────────────────────────────────

func TestIntegration_Tasks_List(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/tasks", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/tasks = %d; body: %s", rec.Code, rec.Body)
	}
	var tasks []map[string]any
	mustDecode(t, rec, &tasks)
	// Scheduler has no jobs in integration test — empty list is expected.
	if tasks == nil {
		t.Fatal("expected a list (possibly empty), got nil")
	}
}

func TestIntegration_Tasks_RunNonExistent(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/tasks/no-such-task/run", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /api/v1/tasks/no-such-task/run = %d, want 404", rec.Code)
	}
}

// ── /api/v1/system/health ────────────────────────────────────────────────────

func TestIntegration_SystemHealth(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/system/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/system/health = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if _, ok := body["status"]; !ok {
		t.Error("response missing 'status' field")
	}
	if _, ok := body["checks"]; !ok {
		t.Error("response missing 'checks' field")
	}
}

// ── /api/v1/notifications ────────────────────────────────────────────────────

func TestIntegration_Notifications_ListEmpty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/notifications", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/notifications = %d; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

// ── OpenAPI docs ─────────────────────────────────────────────────────────────

func TestIntegration_OpenAPIDocs(t *testing.T) {
	h := newIntegrationRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	req.Header.Set("X-Api-Key", testAPIKey)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/docs = %d, want 200", rec.Code)
	}
}

// ── /api/v1/movies/{id}/history ──────────────────────────────────────────────

func TestIntegration_MovieHistory_Empty(t *testing.T) {
	q := testutil.NewTestDB(t)
	h := newIntegrationRouterFromDB(t, q)

	m := testutil.SeedMovie(t, q)

	rec := do(t, h, http.MethodGet, "/api/v1/movies/"+m.ID+"/history", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /movies/%s/history = %d; body: %s", m.ID, rec.Code, rec.Body)
	}
	var items []any
	mustDecode(t, rec, &items)
	if len(items) != 0 {
		t.Errorf("expected empty history, got %d items", len(items))
	}
}

func TestIntegration_MovieHistory_Entries(t *testing.T) {
	q := testutil.NewTestDB(t)
	h := newIntegrationRouterFromDB(t, q)

	m := testutil.SeedMovie(t, q)

	// Seed two grab_history entries for this movie.
	testutil.SeedGrabHistory(t, q, m.ID, "Batman Begins BluRay-1080p")
	testutil.SeedGrabHistory(t, q, m.ID, "Batman Begins WEB-1080p")

	// Seed a grab for a different movie — must not appear.
	other := testutil.SeedMovie(t, q, testutil.WithTMDBID(999))
	testutil.SeedGrabHistory(t, q, other.ID, "Other Movie BluRay-1080p")

	rec := do(t, h, http.MethodGet, "/api/v1/movies/"+m.ID+"/history", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /movies/%s/history = %d; body: %s", m.ID, rec.Code, rec.Body)
	}
	var items []map[string]any
	mustDecode(t, rec, &items)
	if len(items) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(items))
	}
	// Verify required fields are present.
	for _, item := range items {
		if _, ok := item["id"]; !ok {
			t.Error("history item missing 'id'")
		}
		if _, ok := item["movie_id"]; !ok {
			t.Error("history item missing 'movie_id'")
		}
		if _, ok := item["grabbed_at"]; !ok {
			t.Error("history item missing 'grabbed_at'")
		}
		if _, ok := item["download_status"]; !ok {
			t.Error("history item missing 'download_status'")
		}
	}
}

// ── /hooks ────────────────────────────────────────────────────────────────────

func TestIntegration_HooksScan_AllLibraries(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/hooks/scan", map[string]any{})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /hooks/scan = %d, want 202; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_HooksScan_SpecificLibrary_NotFound(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/hooks/scan", map[string]any{
		"library_id": "nonexistent",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /hooks/scan with bad library = %d, want 404; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_HooksRefresh_AllMovies(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/hooks/refresh", map[string]any{})
	// refresh_metadata job is not registered in test scheduler, so RunNow returns error → 500.
	// That's expected — we're just testing the endpoint is reachable and routed correctly.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("POST /hooks/refresh = %d, want 500 (no scheduler job); body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_HooksRefresh_SpecificMovie_NotFound(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/hooks/refresh", map[string]any{
		"movie_id": "nonexistent",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /hooks/refresh with bad movie = %d, want 404; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_HooksNotify(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/hooks/notify", map[string]any{
		"type":    "custom",
		"message": "hello from test",
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /hooks/notify = %d, want 202; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_HooksNotify_MissingType(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodPost, "/api/v1/hooks/notify", map[string]any{
		"message": "no type field",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST /hooks/notify without type = %d, want 422; body: %s", rec.Code, rec.Body)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
		t.Fatalf("decoding response body: %v (body was: %s)", err, rec.Body)
	}
}
