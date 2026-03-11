package indexer_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/luminarr/luminarr/internal/core/indexer"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/ratelimit"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/testutil"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ── Mock indexer ──────────────────────────────────────────────────────────────

type mockIndexer struct {
	searchReleases []plugin.Release
	searchErr      error
	testErr        error
}

func (m *mockIndexer) Name() string              { return "mock" }
func (m *mockIndexer) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }
func (m *mockIndexer) Capabilities(_ context.Context) (plugin.Capabilities, error) {
	return plugin.Capabilities{SearchAvailable: true, MovieSearch: true}, nil
}
func (m *mockIndexer) Search(_ context.Context, _ plugin.SearchQuery) ([]plugin.Release, error) {
	return m.searchReleases, m.searchErr
}
func (m *mockIndexer) GetRecent(_ context.Context) ([]plugin.Release, error) {
	return m.searchReleases, m.searchErr
}
func (m *mockIndexer) Test(_ context.Context) error {
	return m.testErr
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// newTestReg creates an isolated registry so tests never touch registry.Default.
func newTestReg(mock *mockIndexer) *registry.Registry {
	reg := registry.New()
	reg.RegisterIndexer("mock", func(_ json.RawMessage) (plugin.Indexer, error) {
		return mock, nil
	})
	return reg
}

// newServiceFromSQL constructs an indexer.Service backed by an existing *sql.DB.
// Use this when you need to insert seed rows into the same DB the service uses.
func newServiceFromSQL(sqlDB *sql.DB, mock *mockIndexer) *indexer.Service {
	q := dbsqlite.New(sqlDB)
	return indexer.NewService(q, newTestReg(mock), nil, ratelimit.New())
}

func sampleSettings() json.RawMessage {
	b, _ := json.Marshal(map[string]string{"url": "http://indexer.example.com"})
	return b
}

func sampleCreateReq() indexer.CreateRequest {
	return indexer.CreateRequest{
		Name:     "My Indexer",
		Kind:     "mock",
		Enabled:  true,
		Priority: 10,
		Settings: sampleSettings(),
	}
}

// seedMovie inserts a minimal but FK-valid movie row so Grab can reference it.
func seedMovie(t *testing.T, sqlDB *sql.DB, movieID string) {
	t.Helper()
	ctx := context.Background()
	now := "2024-01-01T00:00:00Z"
	mustExec(t, sqlDB, ctx, `INSERT INTO quality_profiles
		(id, name, cutoff_json, qualities_json, upgrade_allowed, created_at, updated_at)
		VALUES ('qp-1','HD','{}','[]',1,?,?)`, now, now)
	mustExec(t, sqlDB, ctx, `INSERT INTO libraries
		(id, name, root_path, default_quality_profile_id, min_free_space_gb, tags_json, created_at, updated_at)
		VALUES ('lib-1','Movies','/movies','qp-1',0,'[]',?,?)`, now, now)
	mustExec(t, sqlDB, ctx, `INSERT INTO movies
		(id, tmdb_id, title, original_title, year, overview, genres_json, status,
		 monitored, library_id, quality_profile_id, added_at, updated_at)
		VALUES (?,12345,'Test Movie','Test Movie',2023,'','[]','released',1,'lib-1','qp-1',?,?)`,
		movieID, now, now)
}

func mustExec(t *testing.T, db *sql.DB, ctx context.Context, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("mustExec: %v", err)
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestService_Create(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	req := sampleCreateReq()
	cfg, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cfg.ID == "" {
		t.Error("Create() returned empty ID")
	}
	if cfg.Name != req.Name {
		t.Errorf("Name = %q, want %q", cfg.Name, req.Name)
	}
	if cfg.Kind != "mock" {
		t.Errorf("Kind = %q, want mock", cfg.Kind)
	}
	if !cfg.Enabled {
		t.Error("Enabled = false, want true")
	}
	if cfg.Priority != 10 {
		t.Errorf("Priority = %d, want 10", cfg.Priority)
	}
}

func TestService_Create_UnknownKind(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Kind = "does-not-exist"
	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("Create() with unknown kind should return error")
	}
}

func TestService_Create_DefaultPriority(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Priority = 0
	cfg, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cfg.Priority != 25 {
		t.Errorf("Priority = %d, want 25 (default)", cfg.Priority)
	}
}

func TestService_Get(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	_, err := svc.Get(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, indexer.ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestService_List_OrderedByPriorityThenName(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Name, req.Priority = "A", 5
	_, _ = svc.Create(ctx, req)
	req.Name, req.Priority = "B", 1
	_, _ = svc.Create(ctx, req)

	configs, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("List() count = %d, want 2", len(configs))
	}
	// priority 1 (B) before priority 5 (A)
	if configs[0].Name != "B" {
		t.Errorf("first config = %q, want B (lowest priority first)", configs[0].Name)
	}
}

func TestService_Update(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	updated, err := svc.Update(ctx, created.ID, indexer.CreateRequest{
		Name:     "Updated Name",
		Kind:     "mock",
		Enabled:  false,
		Priority: 50,
		Settings: created.Settings,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name = %q, want Updated Name", updated.Name)
	}
	if updated.Enabled {
		t.Error("Enabled = true, want false")
	}
	if updated.Priority != 50 {
		t.Errorf("Priority = %d, want 50", updated.Priority)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	_, err := svc.Update(ctx, "00000000-0000-0000-0000-000000000000", sampleCreateReq())
	if !errors.Is(err, indexer.ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestService_Delete(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err := svc.Get(ctx, created.ID)
	if !errors.Is(err, indexer.ErrNotFound) {
		t.Errorf("Get() after Delete error = %v, want ErrNotFound", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	err := svc.Delete(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, indexer.ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestService_Test_Success(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{testErr: nil})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Test(ctx, created.ID); err != nil {
		t.Errorf("Test() error = %v, want nil", err)
	}
}

func TestService_Test_Failure(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{testErr: errors.New("connection refused")})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Test(ctx, created.ID); err == nil {
		t.Error("Test() should return error when indexer test fails")
	}
}

func TestService_Search_ReturnsAndSortsByQuality(t *testing.T) {
	releases := []plugin.Release{
		{
			GUID:     "guid-low",
			Title:    "Movie.2023.720p.WEBRip.x264-GROUP",
			Protocol: plugin.ProtocolTorrent,
			Seeds:    50,
		},
		{
			GUID:     "guid-high",
			Title:    "Movie.2023.1080p.BluRay.x264-GROUP",
			Protocol: plugin.ProtocolTorrent,
			Seeds:    100,
		},
	}
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{searchReleases: releases})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Enabled = true
	_, _ = svc.Create(ctx, req)

	results, err := svc.Search(ctx, plugin.SearchQuery{TMDBID: 12345, Query: "Movie", Year: 2023})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}
	// 1080p BluRay (guid-high) > 720p WEBRip (guid-low)
	if results[0].GUID != "guid-high" {
		t.Errorf("first result GUID = %q, want guid-high (higher quality)", results[0].GUID)
	}
	if results[0].QualityScore <= results[1].QualityScore {
		t.Errorf("results not sorted by quality score: [0]=%d [1]=%d",
			results[0].QualityScore, results[1].QualityScore)
	}
}

func TestService_Search_NoIndexers(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	results, err := svc.Search(ctx, plugin.SearchQuery{Query: "Movie"})
	if err != nil {
		t.Errorf("Search() with no indexers error = %v, want nil", err)
	}
	if len(results) != 0 {
		t.Errorf("Search() with no indexers returned %d results, want 0", len(results))
	}
}

func TestService_Search_DisabledIndexerSkipped(t *testing.T) {
	mock := &mockIndexer{
		searchReleases: []plugin.Release{{GUID: "should-not-appear"}},
	}
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, mock)
	ctx := context.Background()

	req := sampleCreateReq()
	req.Enabled = false
	_, _ = svc.Create(ctx, req)

	results, err := svc.Search(ctx, plugin.SearchQuery{Query: "Movie"})
	if err != nil {
		t.Errorf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search() with disabled indexer returned %d results, want 0", len(results))
	}
}

func TestExtractSeedCriteria_BothFields(t *testing.T) {
	settings := json.RawMessage(`{"url":"http://example.com","seed_ratio":2.0,"seed_time_minutes":120}`)
	c := indexer.ExtractSeedCriteria(settings)
	if c.SeedRatio != 2.0 {
		t.Errorf("SeedRatio = %v, want 2.0", c.SeedRatio)
	}
	if c.SeedTimeMinutes != 120 {
		t.Errorf("SeedTimeMinutes = %v, want 120", c.SeedTimeMinutes)
	}
}

func TestExtractSeedCriteria_RatioOnly(t *testing.T) {
	settings := json.RawMessage(`{"seed_ratio":1.5}`)
	c := indexer.ExtractSeedCriteria(settings)
	if c.SeedRatio != 1.5 {
		t.Errorf("SeedRatio = %v, want 1.5", c.SeedRatio)
	}
	if c.SeedTimeMinutes != 0 {
		t.Errorf("SeedTimeMinutes = %v, want 0", c.SeedTimeMinutes)
	}
}

func TestExtractSeedCriteria_TimeOnly(t *testing.T) {
	settings := json.RawMessage(`{"seed_time_minutes":60}`)
	c := indexer.ExtractSeedCriteria(settings)
	if c.SeedRatio != 0 {
		t.Errorf("SeedRatio = %v, want 0", c.SeedRatio)
	}
	if c.SeedTimeMinutes != 60 {
		t.Errorf("SeedTimeMinutes = %v, want 60", c.SeedTimeMinutes)
	}
}

func TestExtractSeedCriteria_EmptySettings(t *testing.T) {
	settings := json.RawMessage(`{}`)
	c := indexer.ExtractSeedCriteria(settings)
	if c.SeedRatio != 0 || c.SeedTimeMinutes != 0 {
		t.Errorf("expected zero values, got ratio=%v time=%v", c.SeedRatio, c.SeedTimeMinutes)
	}
}

func TestExtractSeedCriteria_NilSettings(t *testing.T) {
	c := indexer.ExtractSeedCriteria(nil)
	if c.SeedRatio != 0 || c.SeedTimeMinutes != 0 {
		t.Errorf("expected zero values for nil, got ratio=%v time=%v", c.SeedRatio, c.SeedTimeMinutes)
	}
}

func TestService_GetSeedCriteria(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	settings := json.RawMessage(`{"url":"http://indexer.example.com","seed_ratio":1.5,"seed_time_minutes":90}`)
	req := indexer.CreateRequest{
		Name:     "Seed Test Indexer",
		Kind:     "mock",
		Enabled:  true,
		Priority: 10,
		Settings: settings,
	}
	created, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	criteria, err := svc.GetSeedCriteria(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSeedCriteria() error = %v", err)
	}
	if criteria.SeedRatio != 1.5 {
		t.Errorf("SeedRatio = %v, want 1.5", criteria.SeedRatio)
	}
	if criteria.SeedTimeMinutes != 90 {
		t.Errorf("SeedTimeMinutes = %v, want 90", criteria.SeedTimeMinutes)
	}
}

func TestService_GetSeedCriteria_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	_, err := svc.GetSeedCriteria(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, indexer.ErrNotFound) {
		t.Errorf("GetSeedCriteria() error = %v, want ErrNotFound", err)
	}
}

func TestService_Grab_RecordsHistory(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockIndexer{})
	ctx := context.Background()

	seedMovie(t, sqlDB, "movie-1")

	release := plugin.Release{
		GUID:     "rel-guid-1",
		Title:    "Test.Movie.2023.1080p.BluRay.x264-GROUP",
		Protocol: plugin.ProtocolTorrent,
		Size:     1_073_741_824,
	}

	history, err := svc.Grab(ctx, "movie-1", "", release, "", "", "")
	if err != nil {
		t.Fatalf("Grab() error = %v", err)
	}
	if history.ID == "" {
		t.Error("Grab() returned empty history ID")
	}
	if history.MovieID != "movie-1" {
		t.Errorf("MovieID = %q, want movie-1", history.MovieID)
	}
	if history.ReleaseGuid != "rel-guid-1" {
		t.Errorf("ReleaseGuid = %q, want rel-guid-1", history.ReleaseGuid)
	}

	rows, err := svc.GrabHistory(ctx, "movie-1")
	if err != nil {
		t.Fatalf("GrabHistory() error = %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("GrabHistory() count = %d, want 1", len(rows))
	}
}
