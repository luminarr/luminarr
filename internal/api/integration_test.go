package api_test

// Integration tests exercise the full HTTP stack: real in-memory SQLite,
// real services, and the live huma/chi router. No external services are
// required. Run with: go test -run Integration ./internal/api/...

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/luminarr/luminarr/internal/api"
	"github.com/luminarr/luminarr/internal/config"
	"github.com/luminarr/luminarr/internal/core/blocklist"
	"github.com/luminarr/luminarr/internal/core/customformat"
	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/downloadhandling"
	"github.com/luminarr/luminarr/internal/core/health"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/mediamanagement"
	"github.com/luminarr/luminarr/internal/core/mediaserver"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/notification"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/core/queue"
	"github.com/luminarr/luminarr/internal/core/stats"
	"github.com/luminarr/luminarr/internal/core/tag"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/ratelimit"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/scheduler"
	"github.com/luminarr/luminarr/internal/testutil"
	"github.com/luminarr/luminarr/pkg/plugin"
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

	reg.RegisterNotifier("webhook", func(_ json.RawMessage) (plugin.Notifier, error) {
		return &testNotifier{}, nil
	})
	reg.RegisterNotifierSanitizer("webhook", func(s json.RawMessage) json.RawMessage { return s })

	reg.RegisterMediaServer("plex", func(_ json.RawMessage) (plugin.MediaServer, error) {
		return &testMediaServer{}, nil
	})
	reg.RegisterMediaServerSanitizer("plex", func(s json.RawMessage) json.RawMessage { return s })
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

type testNotifier struct{}

func (m *testNotifier) Name() string                                               { return "test-webhook" }
func (m *testNotifier) Notify(_ context.Context, _ plugin.NotificationEvent) error { return nil }
func (m *testNotifier) Test(_ context.Context) error                               { return nil }

type testMediaServer struct{}

func (m *testMediaServer) Name() string                                     { return "test-plex" }
func (m *testMediaServer) RefreshLibrary(_ context.Context, _ string) error { return nil }
func (m *testMediaServer) Test(_ context.Context) error                     { return nil }

// newIntegrationRouterFromDB builds a fully-wired router using the provided
// queries so that callers can seed data directly into the same DB.
func newIntegrationRouterFromDB(t *testing.T, q *dbsqlite.Queries, sqlDBs ...*sql.DB) http.Handler {
	t.Helper()
	logger := slog.Default()
	bus := events.New(logger)
	reg := registry.New()
	registerTestPlugins(reg)

	qualSvc := quality.NewService(q, bus)
	qualDefSvc := quality.NewDefinitionService(q)
	libSvc := library.NewService(q, bus, nil)
	movieSvc := movie.NewService(q, nil /* no TMDB */, bus, logger)
	idxSvc := indexer.NewService(q, reg, bus, ratelimit.New())
	dlSvc := downloader.NewService(q, reg, bus)
	queueSvc := queue.NewService(q, dlSvc, bus, logger)
	notifSvc := notification.NewService(q, reg)
	healthSvc := health.NewService(libSvc, dlSvc, idxSvc, logger)
	blockSvc := blocklist.NewService(q)
	statsSvc := stats.NewService(q, movieSvc)
	mmSvc := mediamanagement.NewService(q)
	dhSvc := downloadhandling.NewService(q)
	msSvc := mediaserver.NewService(q, reg)
	tagSvc := tag.NewService(q)
	cfSvc := customformat.NewService(q)
	sched := scheduler.New(logger)

	var sqlDB *sql.DB
	if len(sqlDBs) > 0 {
		sqlDB = sqlDBs[0]
	}

	return api.NewRouter(api.RouterConfig{
		Auth:                     config.Secret(testAPIKey),
		Logger:                   logger,
		StartTime:                time.Now(),
		DBType:                   "sqlite",
		DB:                       sqlDB,
		QualityService:           qualSvc,
		QualityDefinitionService: qualDefSvc,
		LibraryService:           libSvc,
		MovieService:             movieSvc,
		IndexerService:           idxSvc,
		DownloaderService:        dlSvc,
		BlocklistService:         blockSvc,
		QueueService:             queueSvc,
		Scheduler:                sched,
		NotificationService:      notifSvc,
		HealthService:            healthSvc,
		StatsService:             statsSvc,
		MediaManagementService:   mmSvc,
		DownloadHandlingService:  dhSvc,
		MediaServerService:       msSvc,
		TagService:               tagSvc,
		CustomFormatService:      cfSvc,
		Bus:                      bus,
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

// doNoAuth performs a request without the X-Api-Key header.
func doNoAuth(t *testing.T, handler http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
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

// ── Auth: Sec-Fetch-Site ─────────────────────────────────────────────────────

func TestIntegration_Auth_SecFetchSite_SameOrigin(t *testing.T) {
	h := newIntegrationRouter(t)
	// same-origin requests should be allowed without API key.
	rec := doNoAuth(t, h, http.MethodGet, "/api/v1/system/status", map[string]string{
		"Sec-Fetch-Site": "same-origin",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("Sec-Fetch-Site: same-origin = %d, want 200; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_Auth_SecFetchSite_CrossSite(t *testing.T) {
	h := newIntegrationRouter(t)
	// cross-site without API key should be rejected.
	rec := doNoAuth(t, h, http.MethodGet, "/api/v1/system/status", map[string]string{
		"Sec-Fetch-Site": "cross-site",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Sec-Fetch-Site: cross-site = %d, want 401", rec.Code)
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

func TestIntegration_SystemConfig(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/system/config", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/system/config = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if _, ok := body["tmdb_key_configured"]; !ok {
		t.Error("response missing 'tmdb_key_configured' field")
	}
}

func TestIntegration_SystemConfig_ApiKey(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/system/config/apikey", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/system/config/apikey = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if body["api_key"] != testAPIKey {
		t.Errorf("api_key = %v, want %v", body["api_key"], testAPIKey)
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

// ── /api/v1/quality-definitions ──────────────────────────────────────────────

func TestIntegration_QualityDefinitions_List(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/quality-definitions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/quality-definitions = %d; body: %s", rec.Code, rec.Body)
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

	libDir := t.TempDir()
	rec := do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name":                       "Movies",
		"root_path":                  libDir,
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

func TestIntegration_Movies_DegradedMode(t *testing.T) {
	h := newIntegrationRouter(t)

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

	libDir := t.TempDir()
	rec = do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name":                       "Movies",
		"root_path":                  libDir,
		"default_quality_profile_id": profileID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST library = %d; body: %s", rec.Code, rec.Body)
	}
	var lib map[string]any
	mustDecode(t, rec, &lib)
	libraryID, _ := lib["id"].(string)

	rec = do(t, h, http.MethodPost, "/api/v1/movies", map[string]any{
		"tmdb_id":            27205,
		"library_id":         libraryID,
		"quality_profile_id": profileID,
		"monitored":          true,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST movie (degraded) = %d; body: %s", rec.Code, rec.Body)
	}
	var mv map[string]any
	mustDecode(t, rec, &mv)
	if mv["title"] != "tmdb:27205" {
		t.Errorf("stub title = %q, want %q", mv["title"], "tmdb:27205")
	}
	if _, ok := mv["metadata_refreshed_at"]; ok {
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

func TestIntegration_Movies_GetByID(t *testing.T) {
	q := testutil.NewTestDB(t)
	h := newIntegrationRouterFromDB(t, q)
	m := testutil.SeedMovie(t, q)

	rec := do(t, h, http.MethodGet, "/api/v1/movies/"+m.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/movies/%s = %d; body: %s", m.ID, rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if body["title"] != "Inception" {
		t.Errorf("title = %v, want Inception", body["title"])
	}
}

func TestIntegration_Movies_GetNotFound(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/movies/nonexistent", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/v1/movies/nonexistent = %d, want 404", rec.Code)
	}
}

func TestIntegration_Movies_Update(t *testing.T) {
	q := testutil.NewTestDB(t)
	h := newIntegrationRouterFromDB(t, q)
	m := testutil.SeedMovie(t, q)

	rec := do(t, h, http.MethodPut, "/api/v1/movies/"+m.ID, map[string]any{
		"title":              m.Title,
		"monitored":          false,
		"library_id":         m.LibraryID,
		"quality_profile_id": m.QualityProfileID,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/v1/movies/%s = %d; body: %s", m.ID, rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if monitored, _ := body["monitored"].(bool); monitored {
		t.Error("monitored = true after update, want false")
	}
}

func TestIntegration_Movies_Delete(t *testing.T) {
	q := testutil.NewTestDB(t)
	h := newIntegrationRouterFromDB(t, q)
	m := testutil.SeedMovie(t, q)

	rec := do(t, h, http.MethodDelete, "/api/v1/movies/"+m.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/v1/movies/%s = %d; body: %s", m.ID, rec.Code, rec.Body)
	}

	rec = do(t, h, http.MethodGet, "/api/v1/movies/"+m.ID, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted movie = %d, want 404", rec.Code)
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

func TestIntegration_Notifications_CRUD(t *testing.T) {
	h := newIntegrationRouter(t)

	// Create.
	rec := do(t, h, http.MethodPost, "/api/v1/notifications", map[string]any{
		"name":    "Test Webhook",
		"kind":    "webhook",
		"enabled": true,
		"settings": map[string]any{
			"url": "http://example.com/hook",
		},
		"on_events": []string{"grab_started", "download_done"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/notifications = %d; body: %s", rec.Code, rec.Body)
	}
	var created map[string]any
	mustDecode(t, rec, &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created notification has no id")
	}
	if created["name"] != "Test Webhook" {
		t.Errorf("name = %v, want Test Webhook", created["name"])
	}

	// List — must contain the created notification.
	rec = do(t, h, http.MethodGet, "/api/v1/notifications", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/notifications = %d; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}

	// Get by ID.
	rec = do(t, h, http.MethodGet, "/api/v1/notifications/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/notifications/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Update.
	rec = do(t, h, http.MethodPut, "/api/v1/notifications/"+id, map[string]any{
		"name":    "Updated Webhook",
		"kind":    "webhook",
		"enabled": false,
		"settings": map[string]any{
			"url": "http://example.com/hook2",
		},
		"on_events": []string{"grab_started"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/v1/notifications/%s = %d; body: %s", id, rec.Code, rec.Body)
	}
	var updatedNotif map[string]any
	mustDecode(t, rec, &updatedNotif)
	if updatedNotif["name"] != "Updated Webhook" {
		t.Errorf("name = %v, want Updated Webhook", updatedNotif["name"])
	}

	// Delete.
	rec = do(t, h, http.MethodDelete, "/api/v1/notifications/"+id, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/v1/notifications/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Verify gone.
	rec = do(t, h, http.MethodGet, "/api/v1/notifications/"+id, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted notification = %d, want 404", rec.Code)
	}
}

func TestIntegration_Notifications_GetNotFound(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/notifications/nonexistent", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/v1/notifications/nonexistent = %d, want 404", rec.Code)
	}
}

func TestIntegration_Notifications_Test(t *testing.T) {
	h := newIntegrationRouter(t)

	// Create a notification first.
	rec := do(t, h, http.MethodPost, "/api/v1/notifications", map[string]any{
		"name":    "Hook for Test",
		"kind":    "webhook",
		"enabled": true,
		"settings": map[string]any{
			"url": "http://example.com/hook",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST notification = %d; body: %s", rec.Code, rec.Body)
	}
	var created map[string]any
	mustDecode(t, rec, &created)
	id, _ := created["id"].(string)

	// Test it.
	rec = do(t, h, http.MethodPost, "/api/v1/notifications/"+id+"/test", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /api/v1/notifications/%s/test = %d, want 204; body: %s", id, rec.Code, rec.Body)
	}
}

// ── /api/v1/media-servers ────────────────────────────────────────────────────

func TestIntegration_MediaServers_CRUD(t *testing.T) {
	h := newIntegrationRouter(t)

	// List — empty initially.
	rec := do(t, h, http.MethodGet, "/api/v1/media-servers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/media-servers = %d; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	// Create.
	rec = do(t, h, http.MethodPost, "/api/v1/media-servers", map[string]any{
		"name":    "My Plex",
		"kind":    "plex",
		"enabled": true,
		"settings": map[string]any{
			"url":   "http://plex:32400",
			"token": "test-token",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/media-servers = %d; body: %s", rec.Code, rec.Body)
	}
	var created map[string]any
	mustDecode(t, rec, &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created media server has no id")
	}
	if created["name"] != "My Plex" {
		t.Errorf("name = %v, want My Plex", created["name"])
	}

	// List — 1 item.
	rec = do(t, h, http.MethodGet, "/api/v1/media-servers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/media-servers = %d; body: %s", rec.Code, rec.Body)
	}
	mustDecode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 media server, got %d", len(list))
	}

	// Get by ID.
	rec = do(t, h, http.MethodGet, "/api/v1/media-servers/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/media-servers/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Update.
	rec = do(t, h, http.MethodPut, "/api/v1/media-servers/"+id, map[string]any{
		"name":    "Updated Plex",
		"kind":    "plex",
		"enabled": false,
		"settings": map[string]any{
			"url":   "http://plex:32400",
			"token": "new-token",
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/v1/media-servers/%s = %d; body: %s", id, rec.Code, rec.Body)
	}
	var updatedMS map[string]any
	mustDecode(t, rec, &updatedMS)
	if updatedMS["name"] != "Updated Plex" {
		t.Errorf("name = %v, want Updated Plex", updatedMS["name"])
	}

	// Delete.
	rec = do(t, h, http.MethodDelete, "/api/v1/media-servers/"+id, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/v1/media-servers/%s = %d; body: %s", id, rec.Code, rec.Body)
	}

	// Verify gone.
	rec = do(t, h, http.MethodGet, "/api/v1/media-servers/"+id, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted media server = %d, want 404", rec.Code)
	}
}

// ── /api/v1/blocklist ────────────────────────────────────────────────────────

func TestIntegration_Blocklist_EmptyList(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/blocklist", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/blocklist = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	items, _ := body["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("expected 0 blocklist items, got %d", len(items))
	}
}

func TestIntegration_Blocklist_ClearEmpty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodDelete, "/api/v1/blocklist", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/v1/blocklist = %d, want 204; body: %s", rec.Code, rec.Body)
	}
}

// ── /api/v1/wanted ───────────────────────────────────────────────────────────

func TestIntegration_Wanted_Missing_Empty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/wanted/missing", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/wanted/missing = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if _, ok := body["total"]; !ok {
		t.Error("response missing 'total' field")
	}
	if _, ok := body["page"]; !ok {
		t.Error("response missing 'page' field")
	}
}

func TestIntegration_Wanted_Cutoff_Empty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/wanted/cutoff", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/wanted/cutoff = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if _, ok := body["movies"]; !ok {
		t.Error("response missing 'movies' field")
	}
}

func TestIntegration_Wanted_Missing_WithMovie(t *testing.T) {
	q := testutil.NewTestDB(t)
	h := newIntegrationRouterFromDB(t, q)

	// Seed a monitored movie with no file — it should appear in wanted/missing.
	testutil.SeedMovie(t, q, testutil.WithMonitored(true))

	rec := do(t, h, http.MethodGet, "/api/v1/wanted/missing", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/wanted/missing = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	total, _ := body["total"].(float64)
	if total < 1 {
		t.Errorf("expected at least 1 missing movie, got %v", total)
	}
}

// ── /api/v1/stats ────────────────────────────────────────────────────────────

func TestIntegration_Stats_Collection(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/stats/collection", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats/collection = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if _, ok := body["total_movies"]; !ok {
		t.Error("response missing 'total_movies' field")
	}
}

func TestIntegration_Stats_Quality(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/stats/quality", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats/quality = %d; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_Stats_Storage(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/stats/storage", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats/storage = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if _, ok := body["total_bytes"]; !ok {
		t.Error("response missing 'total_bytes' field")
	}
	if _, ok := body["trend"]; !ok {
		t.Error("response missing 'trend' field")
	}
}

func TestIntegration_Stats_Grabs(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/stats/grabs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats/grabs = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if _, ok := body["total_grabs"]; !ok {
		t.Error("response missing 'total_grabs' field")
	}
}

func TestIntegration_Stats_Decades(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/stats/decades", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats/decades = %d; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_Stats_Growth(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/stats/growth", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats/growth = %d; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_Stats_Genres(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/stats/genres", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats/genres = %d; body: %s", rec.Code, rec.Body)
	}
}

// ── /api/v1/history ──────────────────────────────────────────────────────────

func TestIntegration_History_GlobalEmpty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/history", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/history = %d; body: %s", rec.Code, rec.Body)
	}
	var items []any
	mustDecode(t, rec, &items)
	if len(items) != 0 {
		t.Errorf("expected empty history, got %d items", len(items))
	}
}

func TestIntegration_History_GlobalWithEntries(t *testing.T) {
	q := testutil.NewTestDB(t)
	h := newIntegrationRouterFromDB(t, q)

	m := testutil.SeedMovie(t, q)
	testutil.SeedGrabHistory(t, q, m.ID, "Inception BluRay-1080p")
	testutil.SeedGrabHistory(t, q, m.ID, "Inception WEB-1080p")

	rec := do(t, h, http.MethodGet, "/api/v1/history", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/history = %d; body: %s", rec.Code, rec.Body)
	}
	var items []map[string]any
	mustDecode(t, rec, &items)
	if len(items) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(items))
	}
}

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

// ── /api/v1/media-management ─────────────────────────────────────────────────

func TestIntegration_MediaManagement_GetDefaults(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/media-management", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/media-management = %d; body: %s", rec.Code, rec.Body)
	}
}

// ── /api/v1/download-handling ────────────────────────────────────────────────

func TestIntegration_DownloadHandling_GetDefaults(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/download-handling", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/download-handling = %d; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_DownloadHandling_RemotePathMappings_Empty(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/download-handling/remote-path-mappings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/download-handling/remote-path-mappings = %d; body: %s", rec.Code, rec.Body)
	}
	var list []any
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("expected empty remote path mappings, got %d", len(list))
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

// ── /api/v1/parse ────────────────────────────────────────────────────────────

func TestIntegration_Parse(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/parse?filename=Inception.2010.1080p.BluRay.x264", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/parse = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if body["title"] == nil || body["title"] == "" {
		t.Error("expected non-empty title from parse")
	}
	year, _ := body["year"].(float64)
	if year != 2010 {
		t.Errorf("year = %v, want 2010", body["year"])
	}
}

// ── /api/v1/fs/browse ────────────────────────────────────────────────────────

func TestIntegration_FsBrowse(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/fs/browse?path=/tmp", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/fs/browse?path=/tmp = %d; body: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	mustDecode(t, rec, &body)
	if body["path"] != "/tmp" {
		t.Errorf("path = %v, want /tmp", body["path"])
	}
	if _, ok := body["dirs"]; !ok {
		t.Error("response missing 'dirs' field")
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

// ── Multi-step: movie lifecycle ──────────────────────────────────────────────

func TestIntegration_MovieLifecycle(t *testing.T) {
	h := newIntegrationRouter(t)
	webdl1080p := qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")

	// 1. Create quality profile.
	rec := do(t, h, http.MethodPost, "/api/v1/quality-profiles", map[string]any{
		"name":            "Lifecycle Profile",
		"cutoff":          webdl1080p,
		"qualities":       []map[string]any{webdl1080p},
		"upgrade_allowed": false,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create profile = %d; body: %s", rec.Code, rec.Body)
	}
	var profile map[string]any
	mustDecode(t, rec, &profile)
	profileID, _ := profile["id"].(string)

	// 2. Create library.
	libDir := t.TempDir()
	rec = do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name":                       "Lifecycle Movies",
		"root_path":                  libDir,
		"default_quality_profile_id": profileID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create library = %d; body: %s", rec.Code, rec.Body)
	}
	var lib map[string]any
	mustDecode(t, rec, &lib)
	libraryID, _ := lib["id"].(string)

	// 3. Add movie (degraded — no TMDB).
	rec = do(t, h, http.MethodPost, "/api/v1/movies", map[string]any{
		"tmdb_id":            12345,
		"library_id":         libraryID,
		"quality_profile_id": profileID,
		"monitored":          true,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add movie = %d; body: %s", rec.Code, rec.Body)
	}
	var movieResp map[string]any
	mustDecode(t, rec, &movieResp)
	movieID, _ := movieResp["id"].(string)

	// 4. Verify movie appears in wanted/missing (monitored, no file).
	rec = do(t, h, http.MethodGet, "/api/v1/wanted/missing", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("wanted/missing = %d; body: %s", rec.Code, rec.Body)
	}
	var wanted map[string]any
	mustDecode(t, rec, &wanted)
	total, _ := wanted["total"].(float64)
	if total < 1 {
		t.Errorf("expected at least 1 missing movie, got %v", total)
	}

	// 5. Update movie — unmonitor it.
	rec = do(t, h, http.MethodPut, "/api/v1/movies/"+movieID, map[string]any{
		"title":              "tmdb:12345",
		"monitored":          false,
		"library_id":         libraryID,
		"quality_profile_id": profileID,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update movie = %d; body: %s", rec.Code, rec.Body)
	}

	// 6. Verify stats endpoint still works with data.
	rec = do(t, h, http.MethodGet, "/api/v1/stats/collection", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats/collection = %d; body: %s", rec.Code, rec.Body)
	}
	var statsBody map[string]any
	mustDecode(t, rec, &statsBody)
	totalMovies, _ := statsBody["total_movies"].(float64)
	if totalMovies < 1 {
		t.Errorf("expected at least 1 total movie in stats, got %v", totalMovies)
	}

	// 7. Delete movie.
	rec = do(t, h, http.MethodDelete, "/api/v1/movies/"+movieID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete movie = %d; body: %s", rec.Code, rec.Body)
	}

	// 8. Verify gone.
	rec = do(t, h, http.MethodGet, "/api/v1/movies/"+movieID, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted movie = %d, want 404", rec.Code)
	}
}

// ── Radarr v3 compatibility ──────────────────────────────────────────────────

// newV3IntegrationRouter creates a router with *sql.DB wired so v3 endpoints work.
func newV3IntegrationRouter(t *testing.T) (http.Handler, *dbsqlite.Queries) {
	t.Helper()
	q, sqlDB := testutil.NewTestDBWithSQL(t)
	return newIntegrationRouterFromDB(t, q, sqlDB), q
}

func TestIntegration_V3_SystemStatus(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodGet, "/api/v3/system/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/system/status = %d; body: %s", rec.Code, rec.Body)
	}
	var status map[string]any
	mustDecode(t, rec, &status)

	if status["appName"] != "Luminarr" {
		t.Errorf("appName = %v, want Luminarr", status["appName"])
	}
	if status["authentication"] != "apiKey" {
		t.Errorf("authentication = %v, want apiKey", status["authentication"])
	}
	// Must have runtimeName and runtimeVersion.
	if status["runtimeName"] != "go" {
		t.Errorf("runtimeName = %v, want go", status["runtimeName"])
	}
	if _, ok := status["runtimeVersion"]; !ok {
		t.Error("runtimeVersion missing")
	}
}

func TestIntegration_V3_Tags_EmptyList(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodGet, "/api/v3/tag", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/tag = %d; body: %s", rec.Code, rec.Body)
	}
	var tags []map[string]any
	mustDecode(t, rec, &tags)
	if len(tags) != 0 {
		t.Errorf("expected empty tags, got %d", len(tags))
	}
}

func TestIntegration_V3_Tags_Create(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodPost, "/api/v3/tag", map[string]any{"label": "test-tag"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v3/tag = %d; body: %s", rec.Code, rec.Body)
	}
	var tag map[string]any
	mustDecode(t, rec, &tag)
	if tag["label"] != "test-tag" {
		t.Errorf("label = %v, want test-tag", tag["label"])
	}
	id, _ := tag["id"].(float64)
	if id < 1 {
		t.Errorf("id = %v, want >= 1", tag["id"])
	}
}

func TestIntegration_V3_QueueStatus(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodGet, "/api/v3/queue/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/queue/status = %d; body: %s", rec.Code, rec.Body)
	}
	var qs map[string]any
	mustDecode(t, rec, &qs)
	// totalCount should exist and be 0.
	tc, _ := qs["totalCount"].(float64)
	if tc != 0 {
		t.Errorf("totalCount = %v, want 0", qs["totalCount"])
	}
}

func TestIntegration_V3_QualityProfiles_Empty(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodGet, "/api/v3/qualityProfile", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/qualityProfile = %d; body: %s", rec.Code, rec.Body)
	}
	var profiles []map[string]any
	mustDecode(t, rec, &profiles)
	if len(profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(profiles))
	}
}

func TestIntegration_V3_QualityProfiles_WithData(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	// Create a profile via v1.
	webdl := qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")
	rec := do(t, h, http.MethodPost, "/api/v1/quality-profiles", map[string]any{
		"name":            "HD-1080p",
		"cutoff":          webdl,
		"qualities":       []map[string]any{webdl},
		"upgrade_allowed": false,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST v1 quality profile = %d; body: %s", rec.Code, rec.Body)
	}

	// List via v3.
	rec = do(t, h, http.MethodGet, "/api/v3/qualityProfile", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/qualityProfile = %d; body: %s", rec.Code, rec.Body)
	}
	var profiles []map[string]any
	mustDecode(t, rec, &profiles)
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	p := profiles[0]
	if p["name"] != "HD-1080p" {
		t.Errorf("name = %v, want HD-1080p", p["name"])
	}
	// Should have an integer id.
	id, _ := p["id"].(float64)
	if id < 1 {
		t.Errorf("id = %v, want >= 1", p["id"])
	}
}

func TestIntegration_V3_RootFolders_Empty(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodGet, "/api/v3/rootfolder", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/rootfolder = %d; body: %s", rec.Code, rec.Body)
	}
	var folders []map[string]any
	mustDecode(t, rec, &folders)
	if len(folders) != 0 {
		t.Errorf("expected empty root folders, got %d", len(folders))
	}
}

func TestIntegration_V3_RootFolders_WithLibrary(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	// Create a quality profile (required for library).
	webdl := qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")
	rec := do(t, h, http.MethodPost, "/api/v1/quality-profiles", map[string]any{
		"name":            "HD-1080p",
		"cutoff":          webdl,
		"qualities":       []map[string]any{webdl},
		"upgrade_allowed": false,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create quality profile = %d; body: %s", rec.Code, rec.Body)
	}
	var qp map[string]any
	mustDecode(t, rec, &qp)
	qpID, _ := qp["id"].(string)

	// Create a library via v1.
	libDir := t.TempDir()
	rec = do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name":                       "Movies",
		"root_path":                  libDir,
		"default_quality_profile_id": qpID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create library = %d; body: %s", rec.Code, rec.Body)
	}

	// List via v3.
	rec = do(t, h, http.MethodGet, "/api/v3/rootfolder", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/rootfolder = %d; body: %s", rec.Code, rec.Body)
	}
	var folders []map[string]any
	mustDecode(t, rec, &folders)
	if len(folders) != 1 {
		t.Fatalf("expected 1 root folder, got %d", len(folders))
	}
	if folders[0]["path"] != libDir {
		t.Errorf("path = %v, want %s", folders[0]["path"], libDir)
	}
}

func TestIntegration_V3_Movies_EmptyList(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodGet, "/api/v3/movie", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/movie = %d; body: %s", rec.Code, rec.Body)
	}
	var movies []map[string]any
	mustDecode(t, rec, &movies)
	if len(movies) != 0 {
		t.Errorf("expected empty movies, got %d", len(movies))
	}
}

func TestIntegration_V3_Movies_ListAndGet(t *testing.T) {
	h, q := newV3IntegrationRouter(t)

	// Seed data via v1: quality profile → library → movie.
	webdl := qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")
	rec := do(t, h, http.MethodPost, "/api/v1/quality-profiles", map[string]any{
		"name": "HD", "cutoff": webdl, "qualities": []map[string]any{webdl}, "upgrade_allowed": false,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create qp = %d; body: %s", rec.Code, rec.Body)
	}
	var qp map[string]any
	mustDecode(t, rec, &qp)
	qpID, _ := qp["id"].(string)

	libDir := t.TempDir()
	rec = do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name": "Movies", "root_path": libDir, "default_quality_profile_id": qpID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create library = %d; body: %s", rec.Code, rec.Body)
	}
	var lib map[string]any
	mustDecode(t, rec, &lib)
	libID, _ := lib["id"].(string)

	// Insert a movie directly via sqlc (movieSvc.Add requires TMDB which we don't have in tests).
	runtime := int64(139)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := q.CreateMovie(context.Background(), dbsqlite.CreateMovieParams{
		ID:                  "test-movie-uuid",
		TmdbID:              550,
		Title:               "Fight Club",
		OriginalTitle:       "Fight Club",
		Year:                1999,
		Overview:            "Test overview",
		RuntimeMinutes:      &runtime,
		GenresJson:          `["Drama","Thriller"]`,
		LibraryID:           libID,
		QualityProfileID:    qpID,
		Monitored:           1,
		Status:              "released",
		AddedAt:             now,
		UpdatedAt:           now,
		MinimumAvailability: "released",
		ReleaseDate:         "1999-10-15",
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}

	// List via v3.
	rec = do(t, h, http.MethodGet, "/api/v3/movie", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/movie = %d; body: %s", rec.Code, rec.Body)
	}
	var movies []map[string]any
	mustDecode(t, rec, &movies)
	if len(movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(movies))
	}
	m := movies[0]
	if m["title"] != "Fight Club" {
		t.Errorf("title = %v, want Fight Club", m["title"])
	}
	tmdbID, _ := m["tmdbId"].(float64)
	if tmdbID != 550 {
		t.Errorf("tmdbId = %v, want 550", tmdbID)
	}
	if m["monitored"] != true {
		t.Errorf("monitored = %v, want true", m["monitored"])
	}
	// Must have integer id.
	movieRowID, _ := m["id"].(float64)
	if movieRowID < 1 {
		t.Errorf("id = %v, want >= 1", m["id"])
	}
	// rootFolderPath should match the library.
	if m["rootFolderPath"] != libDir {
		t.Errorf("rootFolderPath = %v, want %s", m["rootFolderPath"], libDir)
	}
	// Tags should be an empty array, not null.
	if m["tags"] == nil {
		t.Error("tags should not be nil")
	}

	// GET by v3 ID.
	movieID := int64(movieRowID)
	rec = do(t, h, http.MethodGet, "/api/v3/movie/"+strconv.FormatInt(movieID, 10), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v3/movie/%d = %d; body: %s", movieID, rec.Code, rec.Body)
	}
	var single map[string]any
	mustDecode(t, rec, &single)
	if single["title"] != "Fight Club" {
		t.Errorf("GET single title = %v, want Fight Club", single["title"])
	}
}

func TestIntegration_V3_Movies_GetNotFound(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodGet, "/api/v3/movie/99999", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/v3/movie/99999 = %d, want 404", rec.Code)
	}
}

func TestIntegration_V3_Movies_DeleteByRowID(t *testing.T) {
	h, q := newV3IntegrationRouter(t)

	// Seed data.
	webdl := qualityBody("1080p", "webdl", "x264", "none", "WEBDL-1080p")
	rec := do(t, h, http.MethodPost, "/api/v1/quality-profiles", map[string]any{
		"name": "HD", "cutoff": webdl, "qualities": []map[string]any{webdl}, "upgrade_allowed": false,
	})
	var qp map[string]any
	mustDecode(t, rec, &qp)
	qpID, _ := qp["id"].(string)

	libDir := t.TempDir()
	rec = do(t, h, http.MethodPost, "/api/v1/libraries", map[string]any{
		"name": "Movies", "root_path": libDir, "default_quality_profile_id": qpID,
	})
	var lib map[string]any
	mustDecode(t, rec, &lib)
	libID, _ := lib["id"].(string)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := q.CreateMovie(context.Background(), dbsqlite.CreateMovieParams{
		ID: "del-test-uuid", TmdbID: 100, Title: "Delete Me", OriginalTitle: "Delete Me", Year: 2020,
		LibraryID: libID, QualityProfileID: qpID, Monitored: 1, Status: "released",
		GenresJson: "[]", AddedAt: now, UpdatedAt: now, MinimumAvailability: "released", ReleaseDate: "2020-01-01",
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}

	// Get the rowid.
	rec = do(t, h, http.MethodGet, "/api/v3/movie", nil)
	var movies []map[string]any
	mustDecode(t, rec, &movies)
	if len(movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(movies))
	}
	rowID := int64(movies[0]["id"].(float64))

	// Delete via v3.
	rec = do(t, h, http.MethodDelete, "/api/v3/movie/"+strconv.FormatInt(rowID, 10), nil)
	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/v3/movie/%d = %d; body: %s", rowID, rec.Code, rec.Body)
	}

	// Verify gone.
	rec = do(t, h, http.MethodGet, "/api/v3/movie/"+strconv.FormatInt(rowID, 10), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted movie = %d, want 404", rec.Code)
	}
}

func TestIntegration_V3_Command_RssSync(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodPost, "/api/v3/command", map[string]any{
		"name": "RssSync",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v3/command = %d; body: %s", rec.Code, rec.Body)
	}
	var resp map[string]any
	mustDecode(t, rec, &resp)
	if resp["name"] != "RssSync" {
		t.Errorf("name = %v, want RssSync", resp["name"])
	}
	if resp["status"] != "started" {
		t.Errorf("status = %v, want started", resp["status"])
	}
}

func TestIntegration_V3_Command_Unknown(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	rec := do(t, h, http.MethodPost, "/api/v3/command", map[string]any{
		"name": "SomeUnknownCommand",
	})
	// Should still return 200 — gracefully acknowledge unknown commands.
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v3/command (unknown) = %d; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_V3_Auth_ApiKeyHeader(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	// Standard X-Api-Key header — should work (already used by do()).
	rec := do(t, h, http.MethodGet, "/api/v3/system/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("X-Api-Key auth = %d, want 200", rec.Code)
	}
}

func TestIntegration_V3_Auth_QueryParam(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	// ?apikey= query param — used by Homepage, Home Assistant.
	rec := doNoAuth(t, h, http.MethodGet, "/api/v3/system/status?apikey="+testAPIKey, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("?apikey= auth = %d, want 200; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_V3_Auth_NoAuth(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	// No auth at all — should be rejected.
	rec := doNoAuth(t, h, http.MethodGet, "/api/v3/system/status", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-auth = %d, want 401", rec.Code)
	}
}

func TestIntegration_V3_Auth_SameOrigin(t *testing.T) {
	h, _ := newV3IntegrationRouter(t)

	// Browser same-origin — should be allowed without API key.
	rec := doNoAuth(t, h, http.MethodGet, "/api/v3/system/status", map[string]string{
		"Sec-Fetch-Site": "same-origin",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("same-origin auth = %d, want 200; body: %s", rec.Code, rec.Body)
	}
}

// ── Empty API key guard ──────────────────────────────────────────────────────

// newEmptyKeyRouter builds a router with an empty API key to verify that
// empty-vs-empty comparisons don't accidentally authenticate requests.
func newEmptyKeyRouter(t *testing.T) http.Handler {
	t.Helper()
	q, sqlDB := testutil.NewTestDBWithSQL(t)
	logger := slog.Default()
	bus := events.New(logger)
	reg := registry.New()
	registerTestPlugins(reg)

	qualSvc := quality.NewService(q, bus)
	qualDefSvc := quality.NewDefinitionService(q)
	libSvc := library.NewService(q, bus, nil)
	movieSvc := movie.NewService(q, nil, bus, logger)
	idxSvc := indexer.NewService(q, reg, bus, ratelimit.New())
	dlSvc := downloader.NewService(q, reg, bus)
	queueSvc := queue.NewService(q, dlSvc, bus, logger)
	notifSvc := notification.NewService(q, reg)
	healthSvc := health.NewService(libSvc, dlSvc, idxSvc, logger)
	blockSvc := blocklist.NewService(q)
	statsSvc := stats.NewService(q, movieSvc)
	mmSvc := mediamanagement.NewService(q)
	dhSvc := downloadhandling.NewService(q)
	msSvc := mediaserver.NewService(q, reg)
	sched := scheduler.New(logger)

	return api.NewRouter(api.RouterConfig{
		Auth:                     config.Secret(""), // empty key
		Logger:                   logger,
		StartTime:                time.Now(),
		DB:                       sqlDB,
		DBType:                   "sqlite",
		QualityService:           qualSvc,
		QualityDefinitionService: qualDefSvc,
		LibraryService:           libSvc,
		MovieService:             movieSvc,
		IndexerService:           idxSvc,
		DownloaderService:        dlSvc,
		BlocklistService:         blockSvc,
		QueueService:             queueSvc,
		Scheduler:                sched,
		NotificationService:      notifSvc,
		HealthService:            healthSvc,
		StatsService:             statsSvc,
		MediaManagementService:   mmSvc,
		DownloadHandlingService:  dhSvc,
		MediaServerService:       msSvc,
		Bus:                      bus,
	})
}

func TestIntegration_Auth_EmptyKey_RejectsEmptyHeader(t *testing.T) {
	h := newEmptyKeyRouter(t)
	// Send an empty X-Api-Key — must NOT authenticate even though the server
	// key is also empty (subtle.ConstantTimeCompare("","") == 1).
	rec := doNoAuth(t, h, http.MethodGet, "/api/v1/system/status", map[string]string{
		"X-Api-Key": "",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("empty key + empty header = %d, want 401; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_Auth_EmptyKey_RejectsNoHeader(t *testing.T) {
	h := newEmptyKeyRouter(t)
	rec := doNoAuth(t, h, http.MethodGet, "/api/v1/system/status", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("empty key + no header = %d, want 401; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_Auth_EmptyKey_SameOriginStillWorks(t *testing.T) {
	h := newEmptyKeyRouter(t)
	// Sec-Fetch-Site: same-origin should still authenticate — it doesn't rely on API key.
	rec := doNoAuth(t, h, http.MethodGet, "/api/v1/system/status", map[string]string{
		"Sec-Fetch-Site": "same-origin",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("empty key + same-origin = %d, want 200; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_V3_Auth_EmptyKey_RejectsEmptyHeader(t *testing.T) {
	h := newEmptyKeyRouter(t)
	rec := doNoAuth(t, h, http.MethodGet, "/api/v3/system/status", map[string]string{
		"X-Api-Key": "",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("v3 empty key + empty header = %d, want 401; body: %s", rec.Code, rec.Body)
	}
}

func TestIntegration_V3_Auth_EmptyKey_RejectsEmptyQueryParam(t *testing.T) {
	h := newEmptyKeyRouter(t)
	rec := doNoAuth(t, h, http.MethodGet, "/api/v3/system/status?apikey=", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("v3 empty key + empty apikey param = %d, want 401; body: %s", rec.Code, rec.Body)
	}
}

// ── /api/v1/custom-formats ───────────────────────────────────────────────────

func TestIntegration_CustomFormats_CRUD(t *testing.T) {
	h := newIntegrationRouter(t)

	// POST — create
	createBody := map[string]any{
		"name":                  "TrueHD ATMOS",
		"include_when_renaming": true,
		"specifications": []map[string]any{
			{
				"name":           "TrueHD",
				"implementation": "release_title",
				"negate":         false,
				"required":       true,
				"fields":         map[string]string{"value": `(?i)\bTrueHD\b`},
			},
		},
	}
	rec := do(t, h, http.MethodPost, "/api/v1/custom-formats", createBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /custom-formats = %d, want 201; body: %s", rec.Code, rec.Body)
	}

	var created map[string]any
	mustDecode(t, rec, &created)
	cfID, ok := created["id"].(string)
	if !ok || cfID == "" {
		t.Fatalf("created custom format has no id; body: %v", created)
	}
	if created["name"] != "TrueHD ATMOS" {
		t.Errorf("name = %v, want TrueHD ATMOS", created["name"])
	}

	// GET — list
	rec = do(t, h, http.MethodGet, "/api/v1/custom-formats", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /custom-formats = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var list []map[string]any
	mustDecode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("list length = %d, want 1", len(list))
	}

	// GET — single
	rec = do(t, h, http.MethodGet, "/api/v1/custom-formats/"+cfID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /custom-formats/%s = %d, want 200; body: %s", cfID, rec.Code, rec.Body)
	}

	// PUT — update
	updateBody := map[string]any{
		"name":                  "TrueHD ATMOS v2",
		"include_when_renaming": false,
		"specifications": []map[string]any{
			{
				"name":           "TrueHD",
				"implementation": "release_title",
				"negate":         false,
				"required":       false,
				"fields":         map[string]string{"value": `(?i)\bTrueHD\b`},
			},
			{
				"name":           "Atmos",
				"implementation": "release_title",
				"negate":         false,
				"required":       false,
				"fields":         map[string]string{"value": `(?i)\bAtmos\b`},
			},
		},
	}
	rec = do(t, h, http.MethodPut, "/api/v1/custom-formats/"+cfID, updateBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /custom-formats/%s = %d, want 200; body: %s", cfID, rec.Code, rec.Body)
	}
	var updated map[string]any
	mustDecode(t, rec, &updated)
	if updated["name"] != "TrueHD ATMOS v2" {
		t.Errorf("updated name = %v, want TrueHD ATMOS v2", updated["name"])
	}

	// DELETE
	rec = do(t, h, http.MethodDelete, "/api/v1/custom-formats/"+cfID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /custom-formats/%s = %d, want 204; body: %s", cfID, rec.Code, rec.Body)
	}

	// Verify deletion
	rec = do(t, h, http.MethodGet, "/api/v1/custom-formats", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /custom-formats after delete = %d; body: %s", rec.Code, rec.Body)
	}
	mustDecode(t, rec, &list)
	if len(list) != 0 {
		t.Errorf("list after delete = %d, want 0", len(list))
	}
}

func TestIntegration_CustomFormats_Schema(t *testing.T) {
	h := newIntegrationRouter(t)
	rec := do(t, h, http.MethodGet, "/api/v1/custom-formats/schema", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /custom-formats/schema = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var schema []map[string]any
	mustDecode(t, rec, &schema)
	if len(schema) != 12 {
		t.Errorf("schema length = %d, want 12", len(schema))
	}
}

func TestIntegration_CustomFormats_ImportExport(t *testing.T) {
	h := newIntegrationRouter(t)

	trashJSON := `{
		"trash_id": "abc123",
		"name": "Imported CF",
		"includeCustomFormatWhenRenaming": false,
		"specifications": [
			{
				"name": "x265",
				"implementation": "ReleaseTitleSpecification",
				"negate": false,
				"required": false,
				"fields": {"value": "(?i)x265"}
			}
		]
	}`

	// Import
	rec := doRaw(t, h, http.MethodPost, "/api/v1/custom-formats/import", []byte(trashJSON), "application/json")
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /custom-formats/import = %d, want 201; body: %s", rec.Code, rec.Body)
	}
	var imported []map[string]any
	mustDecode(t, rec, &imported)
	if len(imported) != 1 {
		t.Fatalf("imported length = %d, want 1", len(imported))
	}
	if imported[0]["name"] != "Imported CF" {
		t.Errorf("imported name = %v, want Imported CF", imported[0]["name"])
	}

	// Export all
	rec = do(t, h, http.MethodGet, "/api/v1/custom-formats/export", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /custom-formats/export = %d, want 200; body: %s", rec.Code, rec.Body)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// doRaw performs a request with a raw byte body (for import endpoints).
func doRaw(t *testing.T, handler http.Handler, method, path string, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("X-Api-Key", testAPIKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
		t.Fatalf("decoding response body: %v (body was: %s)", err, rec.Body)
	}
}
