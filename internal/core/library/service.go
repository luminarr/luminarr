// Package library manages Luminarr library records and their disk-level stats.
package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/luminarr/luminarr/internal/core/edition"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
)

// tmdbSearcher is the minimal TMDB interface needed for background candidate matching.
type tmdbSearcher interface {
	SearchMovies(ctx context.Context, query string, year int) ([]tmdb.SearchResult, error)
}

// ErrNotFound is returned when a library does not exist.
var ErrNotFound = errors.New("library not found")

// CreateRequest carries the fields needed to create a library.
type CreateRequest struct {
	Name                    string
	RootPath                string
	DefaultQualityProfileID string
	NamingFormat            *string
	MinFreeSpaceGB          int
	Tags                    []string
}

// UpdateRequest carries the fields needed to update a library.
// It is identical in shape to CreateRequest.
type UpdateRequest = CreateRequest

// Library is the domain representation of a library record.
type Library struct {
	ID                      string
	Name                    string
	RootPath                string
	DefaultQualityProfileID string
	NamingFormat            *string
	MinFreeSpaceGB          int
	Tags                    []string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Stats holds runtime metrics about a library's disk usage and health.
type Stats struct {
	MovieCount     int64
	TotalSizeBytes int64
	FreeSpaceBytes int64
	HealthOK       bool
	HealthMessage  string
}

// Service manages library records.
type Service struct {
	q    dbsqlite.Querier
	bus  *events.Bus
	meta tmdbSearcher // nil when TMDB is not configured
}

// NewService creates a new Service backed by the given querier, event bus, and
// optional TMDB searcher (pass nil to disable background candidate matching).
func NewService(q dbsqlite.Querier, bus *events.Bus, meta tmdbSearcher) *Service {
	return &Service{q: q, bus: bus, meta: meta}
}

// Create inserts a new library and returns the persisted domain type.
func (s *Service) Create(ctx context.Context, req CreateRequest) (Library, error) {
	tagsJSON, err := marshalTags(req.Tags)
	if err != nil {
		return Library{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	row, err := s.q.CreateLibrary(ctx, dbsqlite.CreateLibraryParams{
		ID:                      uuid.New().String(),
		Name:                    req.Name,
		RootPath:                req.RootPath,
		DefaultQualityProfileID: req.DefaultQualityProfileID,
		NamingFormat:            req.NamingFormat,
		MinFreeSpaceGb:          int64(req.MinFreeSpaceGB),
		TagsJson:                tagsJSON,
		CreatedAt:               now,
		UpdatedAt:               now,
	})
	if err != nil {
		return Library{}, fmt.Errorf("inserting library: %w", err)
	}

	return rowToLibrary(row)
}

// Get returns a library by ID.
// Returns ErrNotFound if no library with that ID exists.
func (s *Service) Get(ctx context.Context, id string) (Library, error) {
	row, err := s.q.GetLibrary(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Library{}, ErrNotFound
		}
		return Library{}, fmt.Errorf("fetching library %q: %w", id, err)
	}
	return rowToLibrary(row)
}

// List returns all libraries ordered by name.
func (s *Service) List(ctx context.Context) ([]Library, error) {
	rows, err := s.q.ListLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing libraries: %w", err)
	}

	libs := make([]Library, 0, len(rows))
	for _, row := range rows {
		lib, err := rowToLibrary(row)
		if err != nil {
			return nil, err
		}
		libs = append(libs, lib)
	}
	return libs, nil
}

// Update replaces the mutable fields of an existing library.
// Returns ErrNotFound if the library does not exist.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Library, error) {
	// Confirm existence before update so we can surface ErrNotFound clearly.
	if _, err := s.q.GetLibrary(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Library{}, ErrNotFound
		}
		return Library{}, fmt.Errorf("fetching library %q for update: %w", id, err)
	}

	tagsJSON, err := marshalTags(req.Tags)
	if err != nil {
		return Library{}, err
	}

	row, err := s.q.UpdateLibrary(ctx, dbsqlite.UpdateLibraryParams{
		ID:                      id,
		Name:                    req.Name,
		RootPath:                req.RootPath,
		DefaultQualityProfileID: req.DefaultQualityProfileID,
		NamingFormat:            req.NamingFormat,
		MinFreeSpaceGb:          int64(req.MinFreeSpaceGB),
		TagsJson:                tagsJSON,
		UpdatedAt:               time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return Library{}, fmt.Errorf("updating library %q: %w", id, err)
	}

	return rowToLibrary(row)
}

// Delete removes a library by ID.
// Returns ErrNotFound if the library does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.q.GetLibrary(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("fetching library %q for delete: %w", id, err)
	}

	if err := s.q.DeleteLibrary(ctx, id); err != nil {
		return fmt.Errorf("deleting library %q: %w", id, err)
	}
	return nil
}

// Stats computes runtime metrics for the library: movie count, total file size,
// and available disk space at the library root.
func (s *Service) Stats(ctx context.Context, id string) (Stats, error) {
	lib, err := s.Get(ctx, id)
	if err != nil {
		return Stats{}, err
	}

	movieCount, err := s.q.CountMoviesInLibrary(ctx, id)
	if err != nil {
		return Stats{}, fmt.Errorf("counting movies in library %q: %w", id, err)
	}

	rawSize, err := s.q.SumMovieFileSizesByLibrary(ctx, id)
	if err != nil {
		return Stats{}, fmt.Errorf("summing file sizes for library %q: %w", id, err)
	}

	// SumMovieFileSizesByLibrary returns interface{} — it may be nil when the
	// library has no files, or a numeric type depending on the SQLite driver.
	var totalSizeBytes int64
	if rawSize != nil {
		switch v := rawSize.(type) {
		case int64:
			totalSizeBytes = v
		case float64:
			totalSizeBytes = int64(v)
		}
	}

	// Determine available disk space via syscall. If the path is unavailable
	// (e.g. not yet mounted) we report -1 and mark health as degraded.
	freeBytes := diskFree(lib.RootPath)

	healthOK := true
	healthMsg := "ok"

	minBytes := int64(lib.MinFreeSpaceGB) * 1024 * 1024 * 1024
	switch {
	case freeBytes < 0:
		healthOK = false
		healthMsg = "root path is not accessible"
	case lib.MinFreeSpaceGB > 0 && freeBytes < minBytes:
		healthOK = false
		healthMsg = fmt.Sprintf("free space below minimum: %d GB required", lib.MinFreeSpaceGB)
	}

	return Stats{
		MovieCount:     movieCount,
		TotalSizeBytes: totalSizeBytes,
		FreeSpaceBytes: freeBytes,
		HealthOK:       healthOK,
		HealthMessage:  healthMsg,
	}, nil
}

// ScanDisk walks the library's root path and returns video files that are not
// yet tracked in the movie_files table. It is the read-only counterpart to
// Scan — callers may use the result to let users select files for import.
//
// As a side-effect, newly discovered files are upserted into
// library_file_candidates and a background TMDB matching pass is triggered for
// any unmatched candidates.
func (s *Service) ScanDisk(ctx context.Context, libraryID string) ([]DiskFile, error) {
	lib, err := s.Get(ctx, libraryID)
	if err != nil {
		return nil, err
	}

	existingFiles, err := s.q.ListMovieFilesByLibrary(ctx, libraryID)
	if err != nil {
		return nil, fmt.Errorf("listing existing files for library %q: %w", libraryID, err)
	}

	knownPaths := make(map[string]bool, len(existingFiles))
	for _, f := range existingFiles {
		knownPaths[f.Path] = true
	}

	files, err := scanDisk(lib.RootPath, knownPaths)
	if err != nil {
		return nil, err
	}

	// Upsert candidates (preserves existing TMDB matches on conflict).
	now := time.Now().UTC().Format(time.RFC3339)
	for _, f := range files {
		_ = s.q.UpsertLibraryFileCandidate(ctx, dbsqlite.UpsertLibraryFileCandidateParams{
			LibraryID:   libraryID,
			FilePath:    f.Path,
			FileSize:    f.SizeBytes,
			ParsedTitle: f.ParsedTitle,
			ParsedYear:  int64(f.ParsedYear),
			ScannedAt:   now,
		})
	}

	// Remove candidates not seen in this scan (e.g. deleted files, #recycle paths
	// that were scanned before the directory-skip fix was applied).
	_ = s.q.PruneStaleLibraryFileCandidates(ctx, dbsqlite.PruneStaleLibraryFileCandidatesParams{
		LibraryID: libraryID,
		ScannedAt: now,
	})

	// Fetch stored TMDB matches and attach them to the returned files.
	candidates, _ := s.q.ListLibraryFileCandidates(ctx, libraryID)
	matchMap := make(map[string]dbsqlite.LibraryFileCandidate, len(candidates))
	for _, c := range candidates {
		if c.TmdbID > 0 {
			matchMap[c.FilePath] = c
		}
	}
	for i, f := range files {
		if c, ok := matchMap[f.Path]; ok {
			files[i].TMDBMatch = &DiskFileTMDBMatch{
				TMDBID:        int(c.TmdbID),
				Title:         c.TmdbTitle,
				OriginalTitle: c.TmdbOriginalTitle,
				Year:          int(c.TmdbYear),
			}
		}
	}

	// Trigger background TMDB matching for unmatched candidates.
	if s.meta != nil {
		go s.matchCandidates(context.Background(), libraryID)
	}

	return files, nil
}

// Scan walks a library's root path and reconciles tracked movie files against
// what is actually on disk:
//
//   - Files that exist → indexed_at is updated.
//   - Files that are missing → the owning movie is marked "missing".
//
// Scan is idempotent and safe to run concurrently with normal operations.
func (s *Service) Scan(ctx context.Context, libraryID string) error {
	// Verify library exists before doing any work.
	if _, err := s.Get(ctx, libraryID); err != nil {
		return err
	}

	files, err := s.q.ListMovieFilesByLibrary(ctx, libraryID)
	if err != nil {
		return fmt.Errorf("listing movie files for library %q: %w", libraryID, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, f := range files {
		if _, statErr := os.Stat(f.Path); statErr == nil {
			// File present — refresh indexed_at.
			_ = s.q.UpdateMovieFileIndexed(ctx, dbsqlite.UpdateMovieFileIndexedParams{
				IndexedAt: now,
				ID:        f.ID,
			})
			// Backfill edition from filename if not already set.
			if f.Edition == nil {
				if ed := edition.Parse(filepath.Base(f.Path)); ed != nil {
					_ = s.q.UpdateMovieFileEdition(ctx, dbsqlite.UpdateMovieFileEditionParams{
						Edition: &ed.Name,
						ID:      f.ID,
					})
				}
			}
		} else if os.IsNotExist(statErr) {
			// File gone — mark owning movie as missing.
			_, _ = s.q.UpdateMovieStatus(ctx, dbsqlite.UpdateMovieStatusParams{
				Status:    "missing",
				UpdatedAt: now,
				ID:        f.MovieID,
			})
		}
	}

	// Trigger background TMDB matching for any unmatched candidates.
	if s.meta != nil {
		go s.matchCandidates(context.Background(), libraryID)
	}
	return nil
}

// matchCandidates looks up TMDB for every unmatched candidate in a library and
// stores the result. It runs entirely in the background and is nil-safe.
func (s *Service) matchCandidates(ctx context.Context, libraryID string) {
	unmatched, err := s.q.ListUnmatchedLibraryFileCandidates(ctx, libraryID)
	if err != nil || len(unmatched) == 0 {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range unmatched {
		results, err := s.meta.SearchMovies(ctx, c.ParsedTitle, int(c.ParsedYear))
		if err != nil || len(results) == 0 {
			continue
		}
		best := pickBestCandidateMatch(results, c.ParsedTitle, int(c.ParsedYear))
		if best == nil {
			continue
		}
		_ = s.q.SetLibraryFileCandidateMatch(ctx, dbsqlite.SetLibraryFileCandidateMatchParams{
			TmdbID:            int64(best.ID),
			TmdbTitle:         best.Title,
			TmdbYear:          int64(best.Year),
			TmdbOriginalTitle: best.OriginalTitle,
			MatchedAt:         &now,
			LibraryID:         libraryID,
			FilePath:          c.FilePath,
		})
	}
}

// pickBestCandidateMatch returns the first result whose (normalized) title
// matches parsedTitle AND whose year matches parsedYear. Returns nil when no
// confident match is found.
func pickBestCandidateMatch(results []tmdb.SearchResult, parsedTitle string, parsedYear int) *tmdb.SearchResult {
	if parsedYear == 0 || len(results) == 0 {
		return nil
	}
	norm := normalizeCandidateTitle(parsedTitle)
	for i, r := range results {
		if (normalizeCandidateTitle(r.Title) == norm || normalizeCandidateTitle(r.OriginalTitle) == norm) &&
			r.Year == parsedYear {
			return &results[i]
		}
	}
	return nil
}

var candidateTitleNormRe = regexp.MustCompile(`[^a-z0-9\s]`)

func normalizeCandidateTitle(s string) string {
	s = strings.ToLower(s)
	s = candidateTitleNormRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// ListCandidates returns all file candidates for a library from the database
// without walking the disk. This is the fast path used by the import modal;
// callers that need a fresh disk walk should use ScanDisk instead.
func (s *Service) ListCandidates(ctx context.Context, libraryID string) ([]DiskFile, error) {
	rows, err := s.q.ListLibraryFileCandidates(ctx, libraryID)
	if err != nil {
		return nil, fmt.Errorf("listing candidates for library %q: %w", libraryID, err)
	}
	files := make([]DiskFile, 0, len(rows))
	for _, c := range rows {
		f := DiskFile{
			Path:        c.FilePath,
			SizeBytes:   c.FileSize,
			ParsedTitle: c.ParsedTitle,
			ParsedYear:  int(c.ParsedYear),
		}
		if c.TmdbID > 0 {
			f.TMDBMatch = &DiskFileTMDBMatch{
				TMDBID:        int(c.TmdbID),
				Title:         c.TmdbTitle,
				OriginalTitle: c.TmdbOriginalTitle,
				Year:          int(c.TmdbYear),
			}
		}
		files = append(files, f)
	}
	return files, nil
}

// DeleteCandidate removes a file from the candidate table after it has been
// imported. It is a best-effort call — errors are intentionally ignored.
func (s *Service) DeleteCandidate(ctx context.Context, libraryID, filePath string) error {
	return s.q.DeleteLibraryFileCandidate(ctx, dbsqlite.DeleteLibraryFileCandidateParams{
		LibraryID: libraryID,
		FilePath:  filePath,
	})
}

// diskFree returns the number of free bytes at the given path using
// syscall.Statfs. Returns -1 if the call fails.
func diskFree(path string) int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return -1
	}
	return int64(stat.Bfree) * int64(stat.Bsize) //nolint:gosec // G115: disk free bytes fit well within int64 range
}

// marshalTags encodes a tag slice as a JSON array string. A nil or empty slice
// becomes the JSON string "[]".
func marshalTags(tags []string) (string, error) {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", fmt.Errorf("marshaling tags: %w", err)
	}
	return string(b), nil
}

// rowToLibrary converts a DB row into the domain Library type.
func rowToLibrary(row dbsqlite.Library) (Library, error) {
	var tags []string
	if err := json.Unmarshal([]byte(row.TagsJson), &tags); err != nil {
		return Library{}, fmt.Errorf("unmarshaling tags for library %q: %w", row.ID, err)
	}
	if tags == nil {
		tags = []string{}
	}

	createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
	if err != nil {
		return Library{}, fmt.Errorf("parsing created_at for library %q: %w", row.ID, err)
	}

	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		return Library{}, fmt.Errorf("parsing updated_at for library %q: %w", row.ID, err)
	}

	return Library{
		ID:                      row.ID,
		Name:                    row.Name,
		RootPath:                row.RootPath,
		DefaultQualityProfileID: row.DefaultQualityProfileID,
		NamingFormat:            row.NamingFormat,
		MinFreeSpaceGB:          int(row.MinFreeSpaceGb),
		Tags:                    tags,
		CreatedAt:               createdAt,
		UpdatedAt:               updatedAt,
	}, nil
}
