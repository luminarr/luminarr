// Package movie manages movie records in the Luminarr library.
package movie

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/luminarr/luminarr/internal/core/renamer"
	"github.com/luminarr/luminarr/internal/db"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// Sentinel errors returned by Service methods.
var (
	ErrNotFound          = errors.New("movie not found")
	ErrAlreadyExists     = errors.New("movie already in library")
	ErrTMDBNotConfigured = errors.New("TMDB API key not configured")
	ErrFileNotFound      = errors.New("movie file not found")
)

// MetadataProvider fetches movie metadata from an external source.
type MetadataProvider interface {
	SearchMovies(ctx context.Context, query string, year int) ([]tmdb.SearchResult, error)
	GetMovie(ctx context.Context, tmdbID int) (*tmdb.MovieDetail, error)
}

// Movie is the domain representation of a movie record.
type Movie struct {
	ID                  string
	TMDBID              int
	IMDBID              string
	Title               string
	OriginalTitle       string
	Year                int
	Overview            string
	RuntimeMinutes      int
	Genres              []string
	PosterURL           string
	FanartURL           string
	Status              string
	Monitored           bool
	LibraryID           string
	QualityProfileID    string
	MinimumAvailability string
	ReleaseDate         string
	Path                string
	AddedAt             time.Time
	UpdatedAt           time.Time
	MetadataRefreshedAt *time.Time
}

// AddRequest carries the fields needed to add a movie to the library.
type AddRequest struct {
	TMDBID              int
	LibraryID           string
	QualityProfileID    string
	Monitored           bool
	MinimumAvailability string // defaults to "released" when empty
}

// AddUnmatchedRequest carries the fields needed to add a file that has no TMDB
// match. The resulting movie record has tmdb_id = 0, monitored = false, and
// uses the caller-supplied title (typically the raw filename stem).
type AddUnmatchedRequest struct {
	Title            string
	LibraryID        string
	QualityProfileID string
}

// ListRequest carries filter and pagination options for listing movies.
type ListRequest struct {
	LibraryID string // empty = all libraries
	Page      int    // 1-indexed; defaults to 1
	PerPage   int    // defaults to 50, max 250
}

// ListResult is the paginated response from List.
type ListResult struct {
	Movies  []Movie
	Total   int64
	Page    int
	PerPage int
}

// UpdateRequest carries the mutable fields for updating a movie.
type UpdateRequest struct {
	Title               string
	Monitored           bool
	LibraryID           string
	QualityProfileID    string
	MinimumAvailability string // preserved from existing when empty
}

// LookupRequest carries parameters for searching TMDB without adding to the library.
type LookupRequest struct {
	Query  string
	TMDBID int // if set, fetch exact movie; Query is ignored
	Year   int // optional year filter for query search
}

// RenamePreviewItem describes a single proposed or completed file rename.
type RenamePreviewItem struct {
	FileID  string
	OldPath string
	NewPath string
}

// RenameSettings carries the naming format options used by RenameFiles.
type RenameSettings struct {
	Format           string
	ColonReplacement renamer.ColonReplacement
}

// Service manages movie records.
type Service struct {
	q          dbsqlite.Querier
	sqlDB      *sql.DB // for transactions; nil in tests that don't need them
	meta       MetadataProvider
	mu         sync.RWMutex
	bus        *events.Bus
	logger     *slog.Logger
	renameFile func(oldPath, newPath string) error // injectable for tests; defaults to os.Rename
}

// NewService creates a new Service backed by the given querier, metadata
// provider, event bus, and logger. meta may be nil when TMDB is not configured;
// methods that require it return ErrTMDBNotConfigured.
// sqlDB may be nil in tests; when non-nil, multi-step mutations run in a
// database transaction.
func NewService(q dbsqlite.Querier, meta MetadataProvider, bus *events.Bus, logger *slog.Logger, opts ...ServiceOption) *Service {
	s := &Service{q: q, meta: meta, bus: bus, logger: logger, renameFile: os.Rename}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ServiceOption configures optional Service dependencies.
type ServiceOption func(*Service)

// WithDB provides a *sql.DB for transaction support. When set, multi-step
// mutations (AttachFile, DeleteFile) run atomically.
func WithDB(sqlDB *sql.DB) ServiceOption {
	return func(s *Service) { s.sqlDB = sqlDB }
}

// SetRenameFunc replaces the function used to rename files on disk. Intended
// for use in tests to inject a mock without touching the real filesystem.
func (s *Service) SetRenameFunc(fn func(oldPath, newPath string) error) {
	s.renameFile = fn
}

// SetMetadataProvider replaces the metadata provider at runtime without
// restarting the server. Safe for concurrent use.
func (s *Service) SetMetadataProvider(p MetadataProvider) {
	s.mu.Lock()
	s.meta = p
	s.mu.Unlock()
}

// HasMetadataProvider reports whether a metadata provider is currently wired up.
func (s *Service) HasMetadataProvider() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta != nil
}

// provider returns the current metadata provider under the read lock.
func (s *Service) provider() MetadataProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta
}

// Add fetches movie details from TMDB and inserts the movie into the library.
// When no metadata provider is configured (TMDB key absent), the movie is still
// added as a stub record with a placeholder title so that it can be monitored
// and grabbed immediately. Calling RefreshMetadata later will populate the full
// metadata once a key is configured.
// Returns ErrAlreadyExists if a movie with the same TMDB ID is already present.
func (s *Service) Add(ctx context.Context, req AddRequest) (Movie, error) {
	// Check for duplicates before hitting TMDB (or creating a stub).
	if _, err := s.q.GetMovieByTMDBID(ctx, int64(req.TMDBID)); err == nil {
		return Movie{}, ErrAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Movie{}, fmt.Errorf("checking for existing movie with tmdb_id %d: %w", req.TMDBID, err)
	}

	meta := s.provider()
	if meta == nil {
		return s.addStub(ctx, req)
	}

	detail, err := meta.GetMovie(ctx, req.TMDBID)
	if err != nil {
		return Movie{}, fmt.Errorf("fetching movie detail from TMDB: %w", err)
	}

	genresJSON, err := marshalGenres(detail.Genres)
	if err != nil {
		return Movie{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	monitored := int64(0)
	if req.Monitored {
		monitored = 1
	}

	var imdbID *string
	if detail.IMDBId != "" {
		imdbID = &detail.IMDBId
	}

	var runtimeMinutes *int64
	if detail.RuntimeMinutes > 0 {
		rt := int64(detail.RuntimeMinutes)
		runtimeMinutes = &rt
	}

	var posterURL *string
	if detail.PosterPath != "" {
		posterURL = &detail.PosterPath
	}

	var fanartURL *string
	if detail.BackdropPath != "" {
		fanartURL = &detail.BackdropPath
	}

	minAvail := req.MinimumAvailability
	if minAvail == "" {
		minAvail = "released"
	}

	row, err := s.q.CreateMovie(ctx, dbsqlite.CreateMovieParams{
		ID:                  uuid.New().String(),
		TmdbID:              int64(detail.ID),
		ImdbID:              imdbID,
		Title:               detail.Title,
		OriginalTitle:       detail.OriginalTitle,
		Year:                int64(detail.Year),
		Overview:            detail.Overview,
		RuntimeMinutes:      runtimeMinutes,
		GenresJson:          genresJSON,
		PosterUrl:           posterURL,
		FanartUrl:           fanartURL,
		Status:              detail.Status,
		Monitored:           monitored,
		LibraryID:           req.LibraryID,
		QualityProfileID:    req.QualityProfileID,
		MinimumAvailability: minAvail,
		ReleaseDate:         detail.ReleaseDate,
		Path:                nil,
		AddedAt:             now,
		UpdatedAt:           now,
		MetadataRefreshedAt: &now,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("inserting movie: %w", err)
	}

	m, err := rowToMovie(row)
	if err != nil {
		return Movie{}, err
	}

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieAdded,
		MovieID: m.ID,
		Data: map[string]any{
			"movie_id": m.ID,
			"title":    m.Title,
		},
	})

	return m, nil
}

// addStub inserts a placeholder movie record when no TMDB metadata provider is
// configured. The title is set to "tmdb:<id>" so the record is clearly
// identifiable. Calling RefreshMetadata once a key is configured will replace
// all stub fields with real data.
func (s *Service) addStub(ctx context.Context, req AddRequest) (Movie, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	monitored := int64(0)
	if req.Monitored {
		monitored = 1
	}
	minAvail := req.MinimumAvailability
	if minAvail == "" {
		minAvail = "released"
	}

	row, err := s.q.CreateMovie(ctx, dbsqlite.CreateMovieParams{
		ID:                  uuid.New().String(),
		TmdbID:              int64(req.TMDBID),
		ImdbID:              nil,
		Title:               fmt.Sprintf("tmdb:%d", req.TMDBID),
		OriginalTitle:       "",
		Year:                0,
		Overview:            "",
		RuntimeMinutes:      nil,
		GenresJson:          "[]",
		PosterUrl:           nil,
		FanartUrl:           nil,
		Status:              "unknown",
		Monitored:           monitored,
		LibraryID:           req.LibraryID,
		QualityProfileID:    req.QualityProfileID,
		MinimumAvailability: minAvail,
		Path:                nil,
		AddedAt:             now,
		UpdatedAt:           now,
		MetadataRefreshedAt: nil,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("inserting stub movie for tmdb_id %d: %w", req.TMDBID, err)
	}

	m, err := rowToMovie(row)
	if err != nil {
		return Movie{}, err
	}

	s.logger.WarnContext(ctx, "TMDB not configured — movie added as stub; configure tmdb.api_key and call /refresh to populate metadata",
		slog.String("movie_id", m.ID),
		slog.Int("tmdb_id", req.TMDBID),
	)

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieAdded,
		MovieID: m.ID,
		Data: map[string]any{
			"movie_id": m.ID,
			"title":    m.Title,
		},
	})

	return m, nil
}

// AddUnmatched inserts a placeholder movie record for a file that has no TMDB
// match. The movie is created with tmdb_id = 0 and monitored = false so it
// appears in the library but is not actively sought for download. Calling
// Update or RefreshMetadata later can associate it with a real TMDB entry.
func (s *Service) AddUnmatched(ctx context.Context, req AddUnmatchedRequest) (Movie, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	row, err := s.q.CreateMovie(ctx, dbsqlite.CreateMovieParams{
		ID:                  uuid.New().String(),
		TmdbID:              0,
		ImdbID:              nil,
		Title:               req.Title,
		OriginalTitle:       "",
		Year:                0,
		Overview:            "",
		RuntimeMinutes:      nil,
		GenresJson:          "[]",
		PosterUrl:           nil,
		FanartUrl:           nil,
		Status:              "announced",
		Monitored:           0,
		LibraryID:           req.LibraryID,
		QualityProfileID:    req.QualityProfileID,
		MinimumAvailability: "released",
		ReleaseDate:         "",
		Path:                nil,
		AddedAt:             now,
		UpdatedAt:           now,
		MetadataRefreshedAt: nil,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("inserting unmatched movie: %w", err)
	}

	m, err := rowToMovie(row)
	if err != nil {
		return Movie{}, err
	}

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieAdded,
		MovieID: m.ID,
		Data: map[string]any{
			"movie_id": m.ID,
			"title":    m.Title,
		},
	})

	return m, nil
}

// MatchToTMDB associates an existing unmatched movie record with a TMDB entry.
// It updates the movie's tmdb_id and then refreshes all metadata fields from
// TMDB in a single step. Returns ErrAlreadyExists if another movie in the
// library already owns the supplied tmdbID. Returns ErrTMDBNotConfigured when
// no metadata provider is wired up.
func (s *Service) MatchToTMDB(ctx context.Context, movieID string, tmdbID int) (Movie, error) {
	meta := s.provider()
	if meta == nil {
		return Movie{}, ErrTMDBNotConfigured
	}

	// Ensure the target movie exists.
	if _, err := s.q.GetMovie(ctx, movieID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q: %w", movieID, err)
	}

	// Check that no other movie already owns this TMDB ID.
	dup, err := s.q.GetMovieByTMDBID(ctx, int64(tmdbID))
	if err == nil && dup.ID != movieID {
		return Movie{}, ErrAlreadyExists
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Movie{}, fmt.Errorf("checking for duplicate tmdb_id %d: %w", tmdbID, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.q.UpdateMovieTMDBID(ctx, dbsqlite.UpdateMovieTMDBIDParams{
		TmdbID:    int64(tmdbID),
		UpdatedAt: now,
		ID:        movieID,
	}); err != nil {
		return Movie{}, fmt.Errorf("updating tmdb_id for movie %q: %w", movieID, err)
	}

	// RefreshMetadata reads the updated tmdb_id from the DB and fetches full
	// metadata from TMDB.
	return s.RefreshMetadata(ctx, movieID)
}

// Get returns a single movie by its internal UUID.
// Returns ErrNotFound if no movie with that ID exists.
func (s *Service) Get(ctx context.Context, id string) (Movie, error) {
	row, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q: %w", id, err)
	}
	return rowToMovie(row)
}

// List returns a paginated set of movies, optionally filtered by library.
func (s *Service) List(ctx context.Context, req ListRequest) (ListResult, error) {
	// Normalise pagination inputs.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 {
		req.PerPage = 50
	}
	if req.PerPage > 250 {
		req.PerPage = 250
	}

	limit := int64(req.PerPage)
	offset := int64((req.Page - 1) * req.PerPage)

	var (
		rows  []dbsqlite.Movie
		total int64
		err   error
	)

	if req.LibraryID != "" {
		total, err = s.q.CountMoviesByLibrary(ctx, req.LibraryID)
		if err != nil {
			return ListResult{}, fmt.Errorf("counting movies for library %q: %w", req.LibraryID, err)
		}

		rows, err = s.q.ListMoviesByLibrary(ctx, dbsqlite.ListMoviesByLibraryParams{
			LibraryID: req.LibraryID,
			Limit:     limit,
			Offset:    offset,
		})
	} else {
		total, err = s.q.CountMovies(ctx)
		if err != nil {
			return ListResult{}, fmt.Errorf("counting movies: %w", err)
		}

		rows, err = s.q.ListMovies(ctx, dbsqlite.ListMoviesParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return ListResult{}, fmt.Errorf("listing movies: %w", err)
	}

	movies := make([]Movie, 0, len(rows))
	for _, row := range rows {
		m, err := rowToMovie(row)
		if err != nil {
			return ListResult{}, err
		}
		movies = append(movies, m)
	}

	return ListResult{
		Movies:  movies,
		Total:   total,
		Page:    req.Page,
		PerPage: req.PerPage,
	}, nil
}

// Update replaces the mutable fields of an existing movie.
// Returns ErrNotFound if the movie does not exist.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Movie, error) {
	existing, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q for update: %w", id, err)
	}

	monitored := int64(0)
	if req.Monitored {
		monitored = 1
	}

	libraryID := req.LibraryID
	if libraryID == "" {
		libraryID = existing.LibraryID
	}

	qualityProfileID := req.QualityProfileID
	if qualityProfileID == "" {
		qualityProfileID = existing.QualityProfileID
	}

	title := req.Title
	if title == "" {
		title = existing.Title
	}

	minAvail := req.MinimumAvailability
	if minAvail == "" {
		minAvail = existing.MinimumAvailability
	}

	row, err := s.q.UpdateMovie(ctx, dbsqlite.UpdateMovieParams{
		ID:                  id,
		Title:               title,
		OriginalTitle:       existing.OriginalTitle,
		Year:                existing.Year,
		Overview:            existing.Overview,
		RuntimeMinutes:      existing.RuntimeMinutes,
		GenresJson:          existing.GenresJson,
		PosterUrl:           existing.PosterUrl,
		FanartUrl:           existing.FanartUrl,
		Status:              existing.Status,
		Monitored:           monitored,
		LibraryID:           libraryID,
		QualityProfileID:    qualityProfileID,
		MinimumAvailability: minAvail,
		ReleaseDate:         existing.ReleaseDate,
		UpdatedAt:           time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return Movie{}, fmt.Errorf("updating movie %q: %w", id, err)
	}

	return rowToMovie(row)
}

// Delete removes a movie by ID. It does not delete files from disk.
// Returns ErrNotFound if the movie does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	existing, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("fetching movie %q for delete: %w", id, err)
	}

	if err := s.q.DeleteMovie(ctx, id); err != nil {
		return fmt.Errorf("deleting movie %q: %w", id, err)
	}

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieDeleted,
		MovieID: id,
		Data: map[string]any{
			"movie_id": id,
			"title":    existing.Title,
		},
	})

	return nil
}

// Lookup searches TMDB without adding anything to the library.
// If req.TMDBID is non-zero, it fetches that specific movie by ID.
// Otherwise it performs a text search using req.Query and optional req.Year.
// Returns ErrTMDBNotConfigured if no metadata provider is wired up.
func (s *Service) Lookup(ctx context.Context, req LookupRequest) ([]tmdb.SearchResult, error) {
	meta := s.provider()
	if meta == nil {
		return nil, ErrTMDBNotConfigured
	}

	if req.TMDBID != 0 {
		detail, err := meta.GetMovie(ctx, req.TMDBID)
		if err != nil {
			return nil, fmt.Errorf("fetching movie by tmdb_id %d: %w", req.TMDBID, err)
		}
		return []tmdb.SearchResult{
			{
				ID:            detail.ID,
				Title:         detail.Title,
				OriginalTitle: detail.OriginalTitle,
				Overview:      detail.Overview,
				ReleaseDate:   detail.ReleaseDate,
				Year:          detail.Year,
				PosterPath:    detail.PosterPath,
				BackdropPath:  detail.BackdropPath,
			},
		}, nil
	}

	results, err := meta.SearchMovies(ctx, req.Query, req.Year)
	if err != nil {
		return nil, fmt.Errorf("searching TMDB: %w", err)
	}
	return results, nil
}

// SuggestMatches parses the stored filename/title of an unmatched movie,
// searches TMDB with the extracted title and year, and returns ranked results.
// Returns ErrNotFound if the movie does not exist.
// Returns ErrTMDBNotConfigured if no metadata provider is wired up.
func (s *Service) SuggestMatches(ctx context.Context, id string) ([]tmdb.SearchResult, ParsedFilename, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return nil, ParsedFilename{}, err
	}

	meta := s.provider()
	if meta == nil {
		return nil, ParsedFilename{}, ErrTMDBNotConfigured
	}

	// Use the actual video file path if one is attached (richer signal than the
	// stored title).  m.Path is the movie *directory*, not the filename, so we
	// query movie_files instead.  Fall back to m.Title (the raw filename stem
	// stored at import time) if no file record exists yet.
	source := m.Title
	if files, ferr := s.ListFiles(ctx, id); ferr == nil && len(files) > 0 {
		source = files[0].Path
	}
	parsed := ParseFilename(source)

	results, err := meta.SearchMovies(ctx, parsed.Title, parsed.Year)
	if err != nil {
		return nil, parsed, fmt.Errorf("searching TMDB: %w", err)
	}

	// If no results, retry by progressively dropping words from the front of
	// the title (up to 3 attempts).  This handles filenames where the parser
	// cannot split concatenated words — e.g. "The Hungergames Mockingjay Part
	// 1" yields nothing, but "Mockingjay Part 1" does.
	if len(results) == 0 {
		words := strings.Fields(parsed.Title)
		maxRetries := min(3, len(words)-1)
		for i := 1; i <= maxRetries && len(results) == 0; i++ {
			suffix := strings.Join(words[i:], " ")
			if len(strings.TrimSpace(suffix)) < 3 {
				break
			}
			results, _ = meta.SearchMovies(ctx, suffix, parsed.Year)
		}
	}

	return results, parsed, nil
}

// RefreshMetadata re-fetches TMDB data and updates the movie record in place.
// Returns ErrNotFound if the movie does not exist.
// Returns ErrTMDBNotConfigured if no metadata provider is wired up.
func (s *Service) RefreshMetadata(ctx context.Context, id string) (Movie, error) {
	meta := s.provider()
	if meta == nil {
		return Movie{}, ErrTMDBNotConfigured
	}

	existing, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q: %w", id, err)
	}

	detail, err := meta.GetMovie(ctx, int(existing.TmdbID))
	if err != nil {
		return Movie{}, fmt.Errorf("fetching updated TMDB detail for movie %q: %w", id, err)
	}

	genresJSON, err := marshalGenres(detail.Genres)
	if err != nil {
		return Movie{}, err
	}

	var runtimeMinutes *int64
	if detail.RuntimeMinutes > 0 {
		rt := int64(detail.RuntimeMinutes)
		runtimeMinutes = &rt
	}

	var posterURL *string
	if detail.PosterPath != "" {
		posterURL = &detail.PosterPath
	}

	var fanartURL *string
	if detail.BackdropPath != "" {
		fanartURL = &detail.BackdropPath
	}

	now := time.Now().UTC().Format(time.RFC3339)

	row, err := s.q.UpdateMovie(ctx, dbsqlite.UpdateMovieParams{
		ID:                  id,
		Title:               detail.Title,
		OriginalTitle:       detail.OriginalTitle,
		Year:                int64(detail.Year),
		Overview:            detail.Overview,
		RuntimeMinutes:      runtimeMinutes,
		GenresJson:          genresJSON,
		PosterUrl:           posterURL,
		FanartUrl:           fanartURL,
		Status:              detail.Status,
		Monitored:           existing.Monitored,
		LibraryID:           existing.LibraryID,
		QualityProfileID:    existing.QualityProfileID,
		MinimumAvailability: existing.MinimumAvailability,
		ReleaseDate:         detail.ReleaseDate,
		UpdatedAt:           now,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("updating movie %q after metadata refresh: %w", id, err)
	}

	// Record when the metadata was last refreshed.
	if err := s.q.UpdateMovieMetadataRefreshed(ctx, dbsqlite.UpdateMovieMetadataRefreshedParams{
		MetadataRefreshedAt: &now,
		UpdatedAt:           now,
		ID:                  id,
	}); err != nil {
		// Non-fatal: the data was updated; only the timestamp tracking failed.
		s.logger.WarnContext(ctx, "failed to update metadata_refreshed_at",
			slog.String("movie_id", id),
			slog.Any("error", err),
		)
	}

	// Re-fetch to get the final state including the refreshed timestamp.
	return s.Get(ctx, row.ID)
}

// GetByTMDBID returns a movie by its TMDB ID.
// Returns ErrNotFound if no movie with that TMDB ID exists.
func (s *Service) GetByTMDBID(ctx context.Context, tmdbID int) (Movie, error) {
	row, err := s.q.GetMovieByTMDBID(ctx, int64(tmdbID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie by tmdb_id %d: %w", tmdbID, err)
	}
	return rowToMovie(row)
}

// GetByFilePath returns the movie that owns the given file path, or
// ErrNotFound if no movie_files row exists for that path. If the
// movie_files row exists but the parent movie has been deleted
// (orphaned row), the orphan is cleaned up and ErrNotFound is returned.
func (s *Service) GetByFilePath(ctx context.Context, filePath string) (Movie, error) {
	mf, err := s.q.GetMovieFileByPath(ctx, filePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("looking up movie file %q: %w", filePath, err)
	}
	m, err := s.Get(ctx, mf.MovieID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Orphaned movie_files row — parent movie was deleted but
			// CASCADE didn't fire. Clean it up.
			_ = s.q.DeleteMovieFile(ctx, mf.ID)
		}
		return Movie{}, err
	}
	return m, nil
}

// FileInfo is the domain representation of a movie_files record.
type FileInfo struct {
	ID            string
	MovieID       string
	Path          string
	SizeBytes     int64
	Quality       plugin.Quality
	Edition       string
	ImportedAt    time.Time
	IndexedAt     time.Time
	MediainfoJSON string // raw JSON from ffprobe scan; empty if not yet scanned
}

// ListFiles returns all file records for a movie, newest import first.
func (s *Service) ListFiles(ctx context.Context, movieID string) ([]FileInfo, error) {
	rows, err := s.q.ListMovieFiles(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("listing files for movie %q: %w", movieID, err)
	}
	files := make([]FileInfo, 0, len(rows))
	for _, row := range rows {
		fi, err := rowToFileInfo(row)
		if err != nil {
			return nil, err
		}
		files = append(files, fi)
	}
	return files, nil
}

// GetFile returns the FileInfo for a single movie_file record.
func (s *Service) GetFile(ctx context.Context, fileID string) (FileInfo, error) {
	row, err := s.q.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FileInfo{}, ErrFileNotFound
		}
		return FileInfo{}, fmt.Errorf("fetching movie file %q: %w", fileID, err)
	}
	return rowToFileInfo(row)
}

// DeleteFile removes the movie_files record identified by fileID.
// If deleteFromDisk is true, the file at the stored path is also removed from
// disk (errors are logged but do not fail the operation).
// After the DB record is removed, if no files remain for the movie, the movie's
// path is cleared and its status is reset to "wanted".
func (s *Service) DeleteFile(ctx context.Context, fileID string, deleteFromDisk bool) error {
	row, err := s.q.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFileNotFound
		}
		return fmt.Errorf("fetching movie file %q: %w", fileID, err)
	}

	if deleteFromDisk {
		if removeErr := os.Remove(row.Path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.logger.WarnContext(ctx, "failed to delete movie file from disk",
				slog.String("path", row.Path),
				slog.Any("error", removeErr),
			)
		}
	}

	do := func(q dbsqlite.Querier) error {
		if err := q.DeleteMovieFile(ctx, fileID); err != nil {
			return fmt.Errorf("deleting movie file record %q: %w", fileID, err)
		}

		remaining, err := q.ListMovieFiles(ctx, row.MovieID)
		if err != nil {
			return fmt.Errorf("listing remaining files for movie %q: %w", row.MovieID, err)
		}
		if len(remaining) == 0 {
			now := time.Now().UTC().Format(time.RFC3339)
			if _, err := q.UpdateMoviePath(ctx, dbsqlite.UpdateMoviePathParams{
				Path:      nil,
				UpdatedAt: now,
				ID:        row.MovieID,
			}); err != nil {
				return fmt.Errorf("clearing path for movie %q: %w", row.MovieID, err)
			}
			if _, err := q.UpdateMovieStatus(ctx, dbsqlite.UpdateMovieStatusParams{
				Status:    "wanted",
				UpdatedAt: now,
				ID:        row.MovieID,
			}); err != nil {
				return fmt.Errorf("resetting status for movie %q: %w", row.MovieID, err)
			}
		}
		return nil
	}

	if s.sqlDB != nil {
		return db.RunInTx(ctx, s.sqlDB, do)
	}
	return do(s.q)
}

// AttachFile links a file on disk to an existing movie record. It sets
// movies.path to the full file path, creates a movie_file record,
// and marks the movie status as "downloaded".
//
// Callers are responsible for ensuring the file actually exists on disk before
// calling this method.
func (s *Service) AttachFile(ctx context.Context, movieID, filePath string, sizeBytes int64, quality plugin.Quality) error {
	qualityJSON, err := json.Marshal(quality)
	if err != nil {
		return fmt.Errorf("marshaling quality: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	do := func(q dbsqlite.Querier) error {
		if _, err := q.UpdateMoviePath(ctx, dbsqlite.UpdateMoviePathParams{
			Path:      &filePath,
			UpdatedAt: now,
			ID:        movieID,
		}); err != nil {
			return fmt.Errorf("updating movie path for %q: %w", movieID, err)
		}

		if _, err := q.CreateMovieFile(ctx, dbsqlite.CreateMovieFileParams{
			ID:          uuid.New().String(),
			MovieID:     movieID,
			Path:        filePath,
			SizeBytes:   sizeBytes,
			QualityJson: string(qualityJSON),
			Edition:     nil,
			ImportedAt:  now,
			IndexedAt:   now,
		}); err != nil {
			return fmt.Errorf("creating movie file for %q: %w", movieID, err)
		}

		if _, err := q.UpdateMovieStatus(ctx, dbsqlite.UpdateMovieStatusParams{
			Status:    "downloaded",
			UpdatedAt: now,
			ID:        movieID,
		}); err != nil {
			return fmt.Errorf("updating movie status for %q: %w", movieID, err)
		}
		return nil
	}

	if s.sqlDB != nil {
		return db.RunInTx(ctx, s.sqlDB, do)
	}
	return do(s.q)
}

// RenameFiles computes the standard-format destination filename for every file
// belonging to movieID using the supplied settings. Files whose current name
// already matches the computed name are skipped.
//
// If dryRun is true, no disk or DB operations are performed; the returned slice
// describes what would happen. If dryRun is false, each file is renamed on disk
// and the DB record is updated. Errors for individual files are logged and
// skipped; a combined error is returned if any file could not be renamed.
//
// Returns ErrNotFound if the movie does not exist.
func (s *Service) RenameFiles(ctx context.Context, movieID string, settings RenameSettings, dryRun bool) ([]RenamePreviewItem, error) {
	movie, err := s.q.GetMovie(ctx, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching movie %q: %w", movieID, err)
	}

	files, err := s.q.ListMovieFiles(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("listing files for movie %q: %w", movieID, err)
	}
	if len(files) == 0 {
		return nil, nil
	}

	format := settings.Format
	if format == "" {
		format = renamer.DefaultFileFormat
	}
	colon := settings.ColonReplacement
	if colon == "" {
		colon = renamer.ColonDelete
	}

	rm := renamer.Movie{
		Title:         movie.Title,
		OriginalTitle: movie.OriginalTitle,
		Year:          int(movie.Year),
	}

	var items []RenamePreviewItem
	for _, f := range files {
		var qual plugin.Quality
		_ = json.Unmarshal([]byte(f.QualityJson), &qual)

		ext := filepath.Ext(f.Path)
		newFilename := renamer.ApplyWithOptions(format, rm, qual, colon) + ext
		newPath := filepath.Join(filepath.Dir(f.Path), newFilename)

		if newPath == f.Path {
			continue // already correctly named
		}
		items = append(items, RenamePreviewItem{
			FileID:  f.ID,
			OldPath: f.Path,
			NewPath: newPath,
		})
	}

	if dryRun || len(items) == 0 {
		return items, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var errs []string
	var done []RenamePreviewItem

	for _, item := range items {
		if _, statErr := os.Stat(item.NewPath); statErr == nil {
			// Target already exists — skip to avoid clobbering.
			s.logger.WarnContext(ctx, "rename skipped: target path already exists",
				slog.String("old", item.OldPath),
				slog.String("new", item.NewPath),
			)
			errs = append(errs, fmt.Sprintf("target exists: %s", item.NewPath))
			continue
		}

		if renameErr := s.renameFile(item.OldPath, item.NewPath); renameErr != nil {
			s.logger.ErrorContext(ctx, "rename failed",
				slog.String("old", item.OldPath),
				slog.String("new", item.NewPath),
				slog.Any("error", renameErr),
			)
			errs = append(errs, renameErr.Error())
			continue
		}

		if dbErr := s.q.UpdateMovieFilePath(ctx, dbsqlite.UpdateMovieFilePathParams{
			Path: item.NewPath,
			ID:   item.FileID,
		}); dbErr != nil {
			s.logger.ErrorContext(ctx, "failed to update movie_files path after rename",
				slog.String("file_id", item.FileID),
				slog.Any("error", dbErr),
			)
			errs = append(errs, dbErr.Error())
			continue
		}

		// Keep movies.path in sync if this file's old path matches it.
		if movie.Path != nil && *movie.Path == item.OldPath {
			if _, pathErr := s.q.UpdateMoviePath(ctx, dbsqlite.UpdateMoviePathParams{
				Path:      &item.NewPath,
				UpdatedAt: now,
				ID:        movieID,
			}); pathErr != nil {
				s.logger.WarnContext(ctx, "failed to sync movies.path after rename",
					slog.String("movie_id", movieID),
					slog.Any("error", pathErr),
				)
			}
		}

		done = append(done, item)
	}

	if len(errs) > 0 {
		return done, fmt.Errorf("rename: %d file(s) failed: %s", len(errs), strings.Join(errs, "; "))
	}
	return done, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// marshalGenres encodes a genre slice as a JSON array string.
// A nil or empty slice becomes "[]".
func marshalGenres(genres []string) (string, error) {
	if genres == nil {
		genres = []string{}
	}
	b, err := json.Marshal(genres)
	if err != nil {
		return "", fmt.Errorf("marshaling genres: %w", err)
	}
	return string(b), nil
}

// tmdbImageURL converts a TMDB relative path (e.g. "/abc.jpg") to a full CDN
// URL using the given size string (e.g. "w500", "w1280"). Paths that already
// start with "http" are returned unchanged so the function is idempotent.
func tmdbImageURL(path, size string) string {
	if path == "" || strings.HasPrefix(path, "http") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "https://image.tmdb.org/t/p/" + size + path
}

// rowToFileInfo converts a DB movie_files row into the FileInfo domain type.
func rowToFileInfo(row dbsqlite.MovieFile) (FileInfo, error) {
	var qual plugin.Quality
	if err := json.Unmarshal([]byte(row.QualityJson), &qual); err != nil {
		// Non-fatal: return zero Quality if JSON is malformed.
		qual = plugin.Quality{}
	}

	importedAt, err := time.Parse(time.RFC3339, row.ImportedAt)
	if err != nil {
		importedAt = time.Time{}
	}
	indexedAt, err := time.Parse(time.RFC3339, row.IndexedAt)
	if err != nil {
		indexedAt = time.Time{}
	}

	edition := ""
	if row.Edition != nil {
		edition = *row.Edition
	}

	return FileInfo{
		ID:            row.ID,
		MovieID:       row.MovieID,
		Path:          row.Path,
		SizeBytes:     row.SizeBytes,
		Quality:       qual,
		Edition:       edition,
		ImportedAt:    importedAt,
		IndexedAt:     indexedAt,
		MediainfoJSON: row.MediainfoJson,
	}, nil
}

// rowToMovie converts a DB row into the domain Movie type.
func rowToMovie(row dbsqlite.Movie) (Movie, error) {
	var genres []string
	if err := json.Unmarshal([]byte(row.GenresJson), &genres); err != nil {
		return Movie{}, fmt.Errorf("unmarshaling genres for movie %q: %w", row.ID, err)
	}
	if genres == nil {
		genres = []string{}
	}

	addedAt, err := time.Parse(time.RFC3339, row.AddedAt)
	if err != nil {
		return Movie{}, fmt.Errorf("parsing added_at for movie %q: %w", row.ID, err)
	}

	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		return Movie{}, fmt.Errorf("parsing updated_at for movie %q: %w", row.ID, err)
	}

	var metadataRefreshedAt *time.Time
	if row.MetadataRefreshedAt != nil {
		t, err := time.Parse(time.RFC3339, *row.MetadataRefreshedAt)
		if err != nil {
			return Movie{}, fmt.Errorf("parsing metadata_refreshed_at for movie %q: %w", row.ID, err)
		}
		metadataRefreshedAt = &t
	}

	var imdbID string
	if row.ImdbID != nil {
		imdbID = *row.ImdbID
	}

	var runtimeMinutes int
	if row.RuntimeMinutes != nil {
		runtimeMinutes = int(*row.RuntimeMinutes)
	}

	var posterURL string
	if row.PosterUrl != nil {
		posterURL = tmdbImageURL(*row.PosterUrl, "w500")
	}

	var fanartURL string
	if row.FanartUrl != nil {
		fanartURL = tmdbImageURL(*row.FanartUrl, "w1280")
	}

	var path string
	if row.Path != nil {
		path = *row.Path
	}

	return Movie{
		ID:                  row.ID,
		TMDBID:              int(row.TmdbID),
		IMDBID:              imdbID,
		Title:               row.Title,
		OriginalTitle:       row.OriginalTitle,
		Year:                int(row.Year),
		Overview:            row.Overview,
		RuntimeMinutes:      runtimeMinutes,
		Genres:              genres,
		PosterURL:           posterURL,
		FanartURL:           fanartURL,
		Status:              row.Status,
		Monitored:           row.Monitored != 0,
		LibraryID:           row.LibraryID,
		QualityProfileID:    row.QualityProfileID,
		MinimumAvailability: row.MinimumAvailability,
		ReleaseDate:         row.ReleaseDate,
		Path:                path,
		AddedAt:             addedAt,
		UpdatedAt:           updatedAt,
		MetadataRefreshedAt: metadataRefreshedAt,
	}, nil
}

// ListMissing returns paginated monitored movies that have no associated file.
func (s *Service) ListMissing(ctx context.Context, page, perPage int) ([]Movie, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	total, err := s.q.CountMonitoredMoviesWithoutFile(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("counting missing movies: %w", err)
	}
	rows, err := s.q.ListMonitoredMoviesWithoutFile(ctx, dbsqlite.ListMonitoredMoviesWithoutFileParams{
		Limit:  int64(perPage),
		Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listing missing movies: %w", err)
	}
	movies := make([]Movie, 0, len(rows))
	for _, r := range rows {
		m, err := rowToMovie(r)
		if err != nil {
			s.logger.Warn("skipping movie with bad data", "id", r.ID, "err", err)
			continue
		}
		movies = append(movies, m)
	}
	return movies, total, nil
}

// ListCutoffUnmet returns all monitored movies whose best file quality does not
// meet the quality profile cutoff. Filtering is done in Go to avoid complex SQL
// over JSON-encoded quality columns.
func (s *Service) ListCutoffUnmet(ctx context.Context) ([]Movie, error) {
	rows, err := s.q.ListMonitoredMoviesWithFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing movies with files: %w", err)
	}

	type entry struct {
		row    dbsqlite.ListMonitoredMoviesWithFilesRow
		best   plugin.Quality
		cutoff plugin.Quality
	}
	seen := map[string]*entry{}

	for _, r := range rows {
		var fileQ plugin.Quality
		_ = json.Unmarshal([]byte(r.QualityJson), &fileQ)

		if e, ok := seen[r.ID]; !ok {
			var cutoffQ plugin.Quality
			_ = json.Unmarshal([]byte(r.CutoffJson), &cutoffQ)
			seen[r.ID] = &entry{row: r, best: fileQ, cutoff: cutoffQ}
		} else if fileQ.Score() > e.best.Score() {
			e.best = fileQ
		}
	}

	var result []Movie
	for _, e := range seen {
		if !e.best.AtLeast(e.cutoff) {
			m, err := rowToMovieFromWithFilesRow(e.row)
			if err != nil {
				s.logger.Warn("skipping movie with bad data", "id", e.row.ID, "err", err)
				continue
			}
			result = append(result, m)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Title < result[j].Title })
	return result, nil
}

// CountCutoffUnmet returns the number of monitored movies whose best file
// quality does not meet the quality profile cutoff. Unlike ListCutoffUnmet it
// skips the Movie conversion and sorting since only the count is needed.
func (s *Service) CountCutoffUnmet(ctx context.Context) (int64, error) {
	rows, err := s.q.ListMonitoredMoviesWithFiles(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing movies with files: %w", err)
	}

	type entry struct {
		best   plugin.Quality
		cutoff plugin.Quality
	}
	seen := map[string]*entry{}

	for _, r := range rows {
		var fileQ plugin.Quality
		_ = json.Unmarshal([]byte(r.QualityJson), &fileQ)

		if e, ok := seen[r.ID]; !ok {
			var cutoffQ plugin.Quality
			_ = json.Unmarshal([]byte(r.CutoffJson), &cutoffQ)
			seen[r.ID] = &entry{best: fileQ, cutoff: cutoffQ}
		} else if fileQ.Score() > e.best.Score() {
			e.best = fileQ
		}
	}

	var count int64
	for _, e := range seen {
		if !e.best.AtLeast(e.cutoff) {
			count++
		}
	}
	return count, nil
}

// rowToMovieFromWithFilesRow converts a ListMonitoredMoviesWithFilesRow (which
// includes extra quality columns) into the domain Movie type.
func rowToMovieFromWithFilesRow(r dbsqlite.ListMonitoredMoviesWithFilesRow) (Movie, error) {
	// Delegate to rowToMovie by re-mapping shared fields.
	return rowToMovie(dbsqlite.Movie{
		ID:                  r.ID,
		TmdbID:              r.TmdbID,
		ImdbID:              r.ImdbID,
		Title:               r.Title,
		OriginalTitle:       r.OriginalTitle,
		Year:                r.Year,
		Overview:            r.Overview,
		RuntimeMinutes:      r.RuntimeMinutes,
		GenresJson:          r.GenresJson,
		PosterUrl:           r.PosterUrl,
		FanartUrl:           r.FanartUrl,
		Status:              r.Status,
		Monitored:           r.Monitored,
		LibraryID:           r.LibraryID,
		QualityProfileID:    r.QualityProfileID,
		Path:                r.Path,
		AddedAt:             r.AddedAt,
		UpdatedAt:           r.UpdatedAt,
		MetadataRefreshedAt: r.MetadataRefreshedAt,
		MinimumAvailability: r.MinimumAvailability,
	})
}
