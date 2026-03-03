// Package library manages Luminarr library records and their disk-level stats.
package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/events"
)

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
	q   dbsqlite.Querier
	bus *events.Bus
}

// NewService creates a new Service backed by the given querier and event bus.
func NewService(q dbsqlite.Querier, bus *events.Bus) *Service {
	return &Service{q: q, bus: bus}
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

	return scanDisk(lib.RootPath, knownPaths)
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
		} else if os.IsNotExist(statErr) {
			// File gone — mark owning movie as missing.
			_, _ = s.q.UpdateMovieStatus(ctx, dbsqlite.UpdateMovieStatusParams{
				Status:    "missing",
				UpdatedAt: now,
				ID:        f.MovieID,
			})
		}
	}
	return nil
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
