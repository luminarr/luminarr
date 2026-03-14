package autosearch_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luminarr/luminarr/internal/core/autosearch"
	"github.com/luminarr/luminarr/internal/core/blocklist"
	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/quality"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/ratelimit"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/testutil"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ── mock indexer ─────────────────────────────────────────────────────────────

type mockIndexer struct {
	releases []plugin.Release
	err      error
}

func (m *mockIndexer) Name() string                 { return "mock" }
func (m *mockIndexer) Protocol() plugin.Protocol    { return plugin.ProtocolTorrent }
func (m *mockIndexer) Test(_ context.Context) error { return nil }

func (m *mockIndexer) Capabilities(_ context.Context) (plugin.Capabilities, error) {
	return plugin.Capabilities{}, nil
}

func (m *mockIndexer) Search(_ context.Context, _ plugin.SearchQuery) ([]plugin.Release, error) {
	return m.releases, m.err
}

func (m *mockIndexer) GetRecent(_ context.Context) ([]plugin.Release, error) {
	return m.releases, m.err
}

// ── mock downloader ──────────────────────────────────────────────────────────

type mockDownloader struct {
	itemID string
	err    error
}

func (m *mockDownloader) Name() string                 { return "mock-dl" }
func (m *mockDownloader) Protocol() plugin.Protocol    { return plugin.ProtocolTorrent }
func (m *mockDownloader) Test(_ context.Context) error { return nil }

func (m *mockDownloader) Add(_ context.Context, _ plugin.Release) (string, error) {
	return m.itemID, m.err
}

func (m *mockDownloader) Status(_ context.Context, _ string) (plugin.QueueItem, error) {
	return plugin.QueueItem{}, nil
}

func (m *mockDownloader) GetQueue(_ context.Context) ([]plugin.QueueItem, error) {
	return nil, nil
}

func (m *mockDownloader) Remove(_ context.Context, _ string, _ bool) error { return nil }

// ── test helpers ─────────────────────────────────────────────────────────────

type testEnv struct {
	q       *dbsqlite.Queries
	svc     *autosearch.Service
	blSvc   *blocklist.Service
	mockIdx *mockIndexer
	mockDL  *mockDownloader
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	q := testutil.NewTestDB(t)
	logger := slog.Default()
	bus := events.New(logger)

	mockIdx := &mockIndexer{}
	mockDL := &mockDownloader{itemID: "item-123"}

	reg := registry.New()
	reg.RegisterIndexer("mock", func(_ json.RawMessage) (plugin.Indexer, error) {
		return mockIdx, nil
	})
	reg.RegisterDownloader("mock-dl", func(_ json.RawMessage) (plugin.DownloadClient, error) {
		return mockDL, nil
	})

	indexerSvc := indexer.NewService(q, reg, bus, ratelimit.New())
	movieSvc := movie.NewService(q, nil, bus, logger)
	qualSvc := quality.NewService(q, bus)
	blSvc := blocklist.NewService(q)
	dlSvc := downloader.NewService(q, reg, bus)

	svc := autosearch.NewService(indexerSvc, movieSvc, dlSvc, blSvc, qualSvc, nil, bus, logger)

	return &testEnv{
		q:       q,
		svc:     svc,
		blSvc:   blSvc,
		mockIdx: mockIdx,
		mockDL:  mockDL,
	}
}

// seedWithIndexerAndDownloader creates DB rows for an enabled indexer and download
// client so the service layer can find them.
func seedWithIndexerAndDownloader(t *testing.T, env *testEnv) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := env.q.CreateIndexerConfig(ctx, dbsqlite.CreateIndexerConfigParams{
		ID:       uuid.New().String(),
		Name:     "Test Indexer",
		Kind:     "mock",
		Enabled:  1,
		Priority: 1,
		Settings: "{}",
	})
	if err != nil {
		t.Fatalf("seed indexer: %v", err)
	}

	_, err = env.q.CreateDownloadClientConfig(ctx, dbsqlite.CreateDownloadClientConfigParams{
		ID:        uuid.New().String(),
		Name:      "Test Downloader",
		Kind:      "mock-dl",
		Enabled:   1,
		Priority:  1,
		Settings:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed download client: %v", err)
	}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestSearchMovie_GrabsBestRelease(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)

	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 720p", Protocol: plugin.ProtocolTorrent, DownloadURL: "http://x/1",
			Quality: plugin.Quality{Resolution: "720p", Source: "bluray"}},
		{GUID: "r2", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent, DownloadURL: "http://x/2",
			Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}},
	}

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusGrabbed {
		t.Fatalf("got status %q, want %q (reason: %s)", result.Status, autosearch.StatusGrabbed, result.Reason)
	}
	if result.Grab == nil {
		t.Fatal("expected non-nil Grab")
	}
}

func TestSearchMovie_NoReleases(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = nil

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusNoMatch {
		t.Fatalf("got status %q, want %q", result.Status, autosearch.StatusNoMatch)
	}
}

func TestSearchMovie_MovieNotFound(t *testing.T) {
	t.Parallel()
	env := setup(t)

	_, err := env.svc.SearchMovie(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent movie")
	}
}

func TestSearchMovie_AllBlocklisted(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = []plugin.Release{
		{GUID: "blocked-1", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}},
	}

	ctx := context.Background()
	err := env.blSvc.Add(ctx, mov.ID, "blocked-1", "Inception 2010 1080p", "", "torrent", 1000, "test")
	if err != nil {
		t.Fatalf("blocklist add: %v", err)
	}

	result, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusNoMatch {
		t.Fatalf("got status %q, want %q", result.Status, autosearch.StatusNoMatch)
	}
}

func TestSearchMovie_SkipsBlocklisted_GrabsNext(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = []plugin.Release{
		{GUID: "r-bad", Title: "Inception 2010 1080p Bad", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/bad", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}},
		{GUID: "r-good", Title: "Inception 2010 1080p Good", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/good", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}},
	}

	ctx := context.Background()
	err := env.blSvc.Add(ctx, mov.ID, "r-bad", "Inception 2010 1080p Bad", "", "torrent", 1000, "bad release")
	if err != nil {
		t.Fatalf("blocklist add: %v", err)
	}

	result, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusGrabbed {
		t.Fatalf("got status %q, want %q (reason: %s)", result.Status, autosearch.StatusGrabbed, result.Reason)
	}
	if result.Grab.ReleaseTitle != "Inception 2010 1080p Good" {
		t.Fatalf("grabbed wrong release: %s", result.Grab.ReleaseTitle)
	}
}

func TestSearchMovie_ActiveGrab_Skipped(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}},
	}

	ctx := context.Background()
	result, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	if result.Status != autosearch.StatusGrabbed {
		t.Fatalf("first search: got %q, want %q", result.Status, autosearch.StatusGrabbed)
	}

	// Second search for same movie — unique index prevents duplicate active grab.
	result2, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	if result2.Status != autosearch.StatusSkipped {
		t.Fatalf("second search: got %q, want %q (reason: %s)", result2.Status, autosearch.StatusSkipped, result2.Reason)
	}
}

func TestSearchMovies_BulkCounts(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov1 := testutil.SeedMovie(t, env.q, testutil.WithTMDBID(27205))
	mov2 := testutil.SeedMovie(t, env.q, testutil.WithTMDBID(155))

	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}},
	}

	ctx := context.Background()
	bulk := env.svc.SearchMovies(ctx, []string{mov1.ID, mov2.ID})

	if bulk.Searched != 2 {
		t.Fatalf("searched: got %d, want 2", bulk.Searched)
	}
	if bulk.Grabbed < 1 {
		t.Fatalf("grabbed: got %d, want >= 1", bulk.Grabbed)
	}
	if len(bulk.Results) != 2 {
		t.Fatalf("results: got %d, want 2", len(bulk.Results))
	}
}

func TestSearchMovies_Empty(t *testing.T) {
	t.Parallel()
	env := setup(t)

	bulk := env.svc.SearchMovies(context.Background(), nil)
	if bulk.Searched != 0 {
		t.Fatalf("searched: got %d, want 0", bulk.Searched)
	}
	if bulk.Grabbed != 0 {
		t.Fatalf("grabbed: got %d, want 0", bulk.Grabbed)
	}
}

func TestSearchMovie_ProfileRejectsQuality(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)

	// Only 720p — test profile only allows 1080p bluray.
	env.mockIdx.releases = []plugin.Release{
		{GUID: "r720", Title: "Inception 2010 720p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/720", Quality: plugin.Quality{Resolution: "720p", Source: "bluray"}},
	}

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusNoMatch {
		t.Fatalf("got %q, want %q", result.Status, autosearch.StatusNoMatch)
	}
}
