// Package movie manages movie records in the Luminarr library.
package movie

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/metadata/tmdb"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// Sentinel errors returned by Service methods.
var (
	ErrNotFound          = errors.New("movie not found")
	ErrAlreadyExists     = errors.New("movie already in library")
	ErrTMDBNotConfigured = errors.New("TMDB API key not configured")
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
	Path                string
	AddedAt             time.Time
	UpdatedAt           time.Time
	MetadataRefreshedAt *time.Time
}

// AddRequest carries the fields needed to add a movie to the library.
type AddRequest struct {
	TMDBID           int
	LibraryID        string
	QualityProfileID string
	Monitored        bool
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
	Title            string
	Monitored        bool
	LibraryID        string
	QualityProfileID string
}

// LookupRequest carries parameters for searching TMDB without adding to the library.
type LookupRequest struct {
	Query  string
	TMDBID int // if set, fetch exact movie; Query is ignored
	Year   int // optional year filter for query search
}

// Service manages movie records.
type Service struct {
	q      dbsqlite.Querier
	meta   MetadataProvider
	mu     sync.RWMutex
	bus    *events.Bus
	logger *slog.Logger
}

// NewService creates a new Service backed by the given querier, metadata
// provider, event bus, and logger. meta may be nil when TMDB is not configured;
// methods that require it return ErrTMDBNotConfigured.
func NewService(q dbsqlite.Querier, meta MetadataProvider, bus *events.Bus, logger *slog.Logger) *Service {
	return &Service{q: q, meta: meta, bus: bus, logger: logger}
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

	row, err := s.q.UpdateMovie(ctx, dbsqlite.UpdateMovieParams{
		ID:               id,
		Title:            title,
		OriginalTitle:    existing.OriginalTitle,
		Year:             existing.Year,
		Overview:         existing.Overview,
		RuntimeMinutes:   existing.RuntimeMinutes,
		GenresJson:       existing.GenresJson,
		PosterUrl:        existing.PosterUrl,
		FanartUrl:        existing.FanartUrl,
		Status:           existing.Status,
		Monitored:        monitored,
		LibraryID:        libraryID,
		QualityProfileID: qualityProfileID,
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
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
		ID:               id,
		Title:            detail.Title,
		OriginalTitle:    detail.OriginalTitle,
		Year:             int64(detail.Year),
		Overview:         detail.Overview,
		RuntimeMinutes:   runtimeMinutes,
		GenresJson:       genresJSON,
		PosterUrl:        posterURL,
		FanartUrl:        fanartURL,
		Status:           detail.Status,
		Monitored:        existing.Monitored,
		LibraryID:        existing.LibraryID,
		QualityProfileID: existing.QualityProfileID,
		UpdatedAt:        now,
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

// AttachFile links a file on disk to an existing movie record. It sets
// movies.path to the file's parent directory, creates a movie_file record,
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
	dir := filepath.Dir(filePath)

	if _, err := s.q.UpdateMoviePath(ctx, dbsqlite.UpdateMoviePathParams{
		Path:      &dir,
		UpdatedAt: now,
		ID:        movieID,
	}); err != nil {
		return fmt.Errorf("updating movie path for %q: %w", movieID, err)
	}

	if _, err := s.q.CreateMovieFile(ctx, dbsqlite.CreateMovieFileParams{
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

	if _, err := s.q.UpdateMovieStatus(ctx, dbsqlite.UpdateMovieStatusParams{
		Status:    "downloaded",
		UpdatedAt: now,
		ID:        movieID,
	}); err != nil {
		return fmt.Errorf("updating movie status for %q: %w", movieID, err)
	}

	return nil
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
		Path:                path,
		AddedAt:             addedAt,
		UpdatedAt:           updatedAt,
		MetadataRefreshedAt: metadataRefreshedAt,
	}, nil
}
