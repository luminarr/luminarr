package movie_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/davidfic/luminarr/internal/core/movie"
	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/metadata/tmdb"
	"github.com/davidfic/luminarr/internal/testutil"
)

// ── Mock TMDB client ──────────────────────────────────────────────────────────

type mockTMDB struct {
	searchResults []tmdb.SearchResult
	movieDetail   *tmdb.MovieDetail
	err           error
}

func (m *mockTMDB) SearchMovies(_ context.Context, _ string, _ int) ([]tmdb.SearchResult, error) {
	return m.searchResults, m.err
}

func (m *mockTMDB) GetMovie(_ context.Context, _ int) (*tmdb.MovieDetail, error) {
	return m.movieDetail, m.err
}

// ── Test helpers ─────────────────────────────────────────────────────────────

func newTestService(t *testing.T, meta movie.MetadataProvider) (*movie.Service, *dbsqlite.Queries) {
	t.Helper()
	q := testutil.NewTestDB(t)
	bus := events.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return movie.NewService(q, meta, bus, logger), q
}

// seedTestFixtures creates the FK prerequisites (quality profile + library)
// required by the movies table since migration 00010 added REFERENCES constraints.
// Returns the libraryID and profileID to use when adding movies.
func seedTestFixtures(t *testing.T, q *dbsqlite.Queries) (libraryID, profileID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	qp, err := q.CreateQualityProfile(ctx, dbsqlite.CreateQualityProfileParams{
		ID:            "qp-test",
		Name:          "Test Profile",
		CutoffJson:    `{}`,
		QualitiesJson: `[]`,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("seedTestFixtures: CreateQualityProfile: %v", err)
	}

	lib, err := q.CreateLibrary(ctx, dbsqlite.CreateLibraryParams{
		ID:                      "lib-test",
		Name:                    "Test Library",
		RootPath:                "/test",
		DefaultQualityProfileID: qp.ID,
		MinFreeSpaceGb:          0,
		TagsJson:                "[]",
		CreatedAt:               now,
		UpdatedAt:               now,
	})
	if err != nil {
		t.Fatalf("seedTestFixtures: CreateLibrary: %v", err)
	}

	return lib.ID, qp.ID
}

func sampleDetail() *tmdb.MovieDetail {
	return &tmdb.MovieDetail{
		ID:             550,
		IMDBId:         "tt0137523",
		Title:          "Fight Club",
		OriginalTitle:  "Fight Club",
		Overview:       "An insomniac office worker...",
		ReleaseDate:    "1999-10-15",
		Year:           1999,
		RuntimeMinutes: 139,
		Genres:         []string{"Drama", "Thriller"},
		PosterPath:     "/poster.jpg",
		BackdropPath:   "/backdrop.jpg",
		Status:         "released",
	}
}

func addTestMovie(t *testing.T, svc *movie.Service, libraryID, profileID string) movie.Movie {
	t.Helper()
	m, err := svc.Add(context.Background(), movie.AddRequest{
		TMDBID:           550,
		LibraryID:        libraryID,
		QualityProfileID: profileID,
		Monitored:        true,
	})
	if err != nil {
		t.Fatalf("addTestMovie: %v", err)
	}
	return m
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestAdd_Success verifies that a movie is created and returned correctly.
func TestAdd_Success(t *testing.T) {
	detail := sampleDetail()
	meta := &mockTMDB{movieDetail: detail}
	svc, q := newTestService(t, meta)
	libID, profID := seedTestFixtures(t, q)

	m, err := svc.Add(context.Background(), movie.AddRequest{
		TMDBID:           550,
		LibraryID:        libID,
		QualityProfileID: profID,
		Monitored:        true,
	})
	if err != nil {
		t.Fatalf("Add: unexpected error: %v", err)
	}

	if m.ID == "" {
		t.Error("Add: expected non-empty ID")
	}
	if m.TMDBID != 550 {
		t.Errorf("Add: TMDBID = %d, want 550", m.TMDBID)
	}
	if m.Title != "Fight Club" {
		t.Errorf("Add: Title = %q, want %q", m.Title, "Fight Club")
	}
	if m.IMDBID != "tt0137523" {
		t.Errorf("Add: IMDBID = %q, want %q", m.IMDBID, "tt0137523")
	}
	if m.Year != 1999 {
		t.Errorf("Add: Year = %d, want 1999", m.Year)
	}
	if m.RuntimeMinutes != 139 {
		t.Errorf("Add: RuntimeMinutes = %d, want 139", m.RuntimeMinutes)
	}
	if len(m.Genres) != 2 {
		t.Errorf("Add: len(Genres) = %d, want 2", len(m.Genres))
	}
	if !m.Monitored {
		t.Error("Add: Monitored = false, want true")
	}
	if m.LibraryID != libID {
		t.Errorf("Add: LibraryID = %q, want %q", m.LibraryID, libID)
	}
}

// TestAdd_DegradedMode verifies that Add succeeds without a TMDB metadata
// provider by inserting a stub record. The stub title is "tmdb:<id>" and
// MetadataRefreshedAt is nil until a refresh is performed.
func TestAdd_DegradedMode(t *testing.T) {
	svc, q := newTestService(t, nil)
	libID, profID := seedTestFixtures(t, q)

	m, err := svc.Add(context.Background(), movie.AddRequest{TMDBID: 550, LibraryID: libID, QualityProfileID: profID, Monitored: true})
	if err != nil {
		t.Fatalf("Add: unexpected error in degraded mode: %v", err)
	}
	if m.Title != "tmdb:550" {
		t.Errorf("Add (degraded): Title = %q, want %q", m.Title, "tmdb:550")
	}
	if m.TMDBID != 550 {
		t.Errorf("Add (degraded): TMDBID = %d, want 550", m.TMDBID)
	}
	if m.MetadataRefreshedAt != nil {
		t.Errorf("Add (degraded): MetadataRefreshedAt = %v, want nil", m.MetadataRefreshedAt)
	}
	if !m.Monitored {
		t.Error("Add (degraded): Monitored = false, want true")
	}
}

// TestAdd_Duplicate verifies that adding the same TMDB ID twice returns
// ErrAlreadyExists.
func TestAdd_Duplicate(t *testing.T) {
	detail := sampleDetail()
	meta := &mockTMDB{movieDetail: detail}
	svc, q := newTestService(t, meta)
	libID, profID := seedTestFixtures(t, q)

	req := movie.AddRequest{TMDBID: 550, LibraryID: libID, QualityProfileID: profID}
	if _, err := svc.Add(context.Background(), req); err != nil {
		t.Fatalf("Add (first): %v", err)
	}

	_, err := svc.Add(context.Background(), req)
	if !errors.Is(err, movie.ErrAlreadyExists) {
		t.Errorf("Add (second): got %v, want ErrAlreadyExists", err)
	}
}

// TestGet_Success verifies that a movie can be retrieved by ID.
func TestGet_Success(t *testing.T) {
	detail := sampleDetail()
	meta := &mockTMDB{movieDetail: detail}
	svc, q := newTestService(t, meta)
	libID, profID := seedTestFixtures(t, q)

	added := addTestMovie(t, svc, libID, profID)

	got, err := svc.Get(context.Background(), added.ID)
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got.ID != added.ID {
		t.Errorf("Get: ID = %q, want %q", got.ID, added.ID)
	}
	if got.Title != "Fight Club" {
		t.Errorf("Get: Title = %q, want %q", got.Title, "Fight Club")
	}
}

// TestGet_NotFound verifies that Get returns ErrNotFound for a missing ID.
func TestGet_NotFound(t *testing.T) {
	svc, _ := newTestService(t, nil)

	_, err := svc.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, movie.ErrNotFound) {
		t.Errorf("Get: got %v, want ErrNotFound", err)
	}
}

// TestList_Empty verifies that List returns an empty result on a fresh DB.
func TestList_Empty(t *testing.T) {
	svc, _ := newTestService(t, nil)

	result, err := svc.List(context.Background(), movie.ListRequest{})
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("List: Total = %d, want 0", result.Total)
	}
	if len(result.Movies) != 0 {
		t.Errorf("List: len(Movies) = %d, want 0", len(result.Movies))
	}
}

// TestList_WithResults verifies pagination and result contents.
func TestList_WithResults(t *testing.T) {
	detail := sampleDetail()
	meta := &mockTMDB{movieDetail: detail}
	svc, q := newTestService(t, meta)
	libID, profID := seedTestFixtures(t, q)

	addTestMovie(t, svc, libID, profID)

	result, err := svc.List(context.Background(), movie.ListRequest{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("List: Total = %d, want 1", result.Total)
	}
	if len(result.Movies) != 1 {
		t.Errorf("List: len(Movies) = %d, want 1", len(result.Movies))
	}
	if result.Movies[0].Title != "Fight Club" {
		t.Errorf("List: Movies[0].Title = %q, want %q", result.Movies[0].Title, "Fight Club")
	}
}

// TestDelete_Success verifies that a movie can be deleted.
func TestDelete_Success(t *testing.T) {
	detail := sampleDetail()
	meta := &mockTMDB{movieDetail: detail}
	svc, q := newTestService(t, meta)
	libID, profID := seedTestFixtures(t, q)

	added := addTestMovie(t, svc, libID, profID)

	if err := svc.Delete(context.Background(), added.ID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Confirm the movie is gone.
	_, err := svc.Get(context.Background(), added.ID)
	if !errors.Is(err, movie.ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want ErrNotFound", err)
	}
}

// TestDelete_NotFound verifies that deleting a non-existent movie returns
// ErrNotFound.
func TestDelete_NotFound(t *testing.T) {
	svc, _ := newTestService(t, nil)

	err := svc.Delete(context.Background(), "does-not-exist")
	if !errors.Is(err, movie.ErrNotFound) {
		t.Errorf("Delete: got %v, want ErrNotFound", err)
	}
}

// TestLookup_Success verifies that Lookup delegates to the metadata provider.
func TestLookup_Success(t *testing.T) {
	meta := &mockTMDB{
		searchResults: []tmdb.SearchResult{
			{ID: 550, Title: "Fight Club", Year: 1999},
			{ID: 807, Title: "Se7en", Year: 1995},
		},
	}
	svc, _ := newTestService(t, meta)

	results, err := svc.Lookup(context.Background(), movie.LookupRequest{Query: "fight"})
	if err != nil {
		t.Fatalf("Lookup: unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Lookup: len(results) = %d, want 2", len(results))
	}
	if results[0].ID != 550 {
		t.Errorf("Lookup: results[0].ID = %d, want 550", results[0].ID)
	}
}

// TestLookup_TMDBNotConfigured verifies the nil-provider guard in Lookup.
func TestLookup_TMDBNotConfigured(t *testing.T) {
	svc, _ := newTestService(t, nil)

	_, err := svc.Lookup(context.Background(), movie.LookupRequest{Query: "anything"})
	if !errors.Is(err, movie.ErrTMDBNotConfigured) {
		t.Errorf("Lookup: got %v, want ErrTMDBNotConfigured", err)
	}
}
