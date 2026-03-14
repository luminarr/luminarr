package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/pathutil"
)

// ── Request / response shapes ────────────────────────────────────────────────

type libraryBody struct {
	ID                      string    `json:"id"                          doc:"Library UUID"`
	Name                    string    `json:"name"                        doc:"Human-readable library name"`
	RootPath                string    `json:"root_path"                   doc:"Absolute path to the library root"`
	DefaultQualityProfileID string    `json:"default_quality_profile_id"  doc:"Default quality profile UUID"`
	NamingFormat            *string   `json:"naming_format,omitempty"     doc:"Optional file naming template"`
	MinFreeSpaceGB          int       `json:"min_free_space_gb"           doc:"Minimum free disk space in gigabytes"`
	Tags                    []string  `json:"tags"                        doc:"User-defined tags"`
	CreatedAt               time.Time `json:"created_at"                  doc:"Creation timestamp (UTC)"`
	UpdatedAt               time.Time `json:"updated_at"                  doc:"Last update timestamp (UTC)"`
}

type libraryInput struct {
	Name                    string   `json:"name"                        doc:"Human-readable library name"`
	RootPath                string   `json:"root_path"                   doc:"Absolute path to the library root"`
	DefaultQualityProfileID string   `json:"default_quality_profile_id"  doc:"Default quality profile UUID"`
	NamingFormat            *string  `json:"naming_format,omitempty"     doc:"Optional file naming template"`
	MinFreeSpaceGB          *int     `json:"min_free_space_gb,omitempty" doc:"Minimum free disk space in gigabytes (default: 0)"`
	Tags                    []string `json:"tags,omitempty"              doc:"User-defined tags (default: [])"`
}

type libraryStatsBody struct {
	MovieCount     int64  `json:"movie_count"      doc:"Number of movies in this library"`
	TotalSizeBytes int64  `json:"total_size_bytes" doc:"Combined size of all movie files in bytes"`
	FreeSpaceBytes int64  `json:"free_space_bytes" doc:"Available disk space at root path in bytes; -1 if unavailable"`
	HealthOK       bool   `json:"health_ok"        doc:"True when disk space meets the minimum requirement"`
	HealthMessage  string `json:"health_message"   doc:"Human-readable health description"`
}

// Huma output wrappers.
type libraryOutput struct {
	Body *libraryBody
}

type libraryListOutput struct {
	Body []*libraryBody
}

type libraryStatsOutput struct {
	Body *libraryStatsBody
}

// 204 No Content.
type libraryDeleteOutput struct{}

// POST /api/v1/libraries/{id}/scan returns 202 Accepted.
type libraryScanInput struct {
	ID string `path:"id"`
}

type libraryScanOutput struct{}

// GET /api/v1/libraries/{id}/disk-scan
type libraryDiskScanInput struct {
	ID string `path:"id"`
}

type tmdbMatchBody struct {
	TMDBID        int    `json:"tmdb_id"         doc:"TMDB movie ID"`
	Title         string `json:"title"           doc:"TMDB movie title"`
	OriginalTitle string `json:"original_title"  doc:"TMDB original title"`
	Year          int    `json:"year"            doc:"Release year"`
}

type diskFileBody struct {
	Path        string         `json:"path"                   doc:"Absolute path to the file"`
	SizeBytes   int64          `json:"size_bytes"             doc:"File size in bytes"`
	ParsedTitle string         `json:"parsed_title"           doc:"Title guessed from filename"`
	ParsedYear  int            `json:"parsed_year"            doc:"Year guessed from filename; 0 if not found"`
	TMDBMatch   *tmdbMatchBody `json:"tmdb_match,omitempty"   doc:"Pre-computed TMDB match, if available"`
}

type libraryDiskScanOutput struct {
	Body []*diskFileBody
}

// POST /api/v1/libraries/{id}/import-file
type libraryImportFileInput struct {
	ID   string `path:"id"`
	Body libraryImportFileBody
}

type libraryImportFileBody struct {
	FilePath string `json:"file_path" doc:"Absolute path of the file to import"`
	TmdbID   int    `json:"tmdb_id"   doc:"TMDB movie ID to associate with this file"`
}

type libraryImportFileOutput struct {
	Body *movieBody
}

// Huma request wrappers.
type libraryCreateInput struct {
	Body libraryInput
}

type libraryGetInput struct {
	ID string `path:"id"`
}

type libraryUpdateInput struct {
	ID   string `path:"id"`
	Body libraryInput
}

type libraryDeleteInput struct {
	ID string `path:"id"`
}

type libraryStatsInput struct {
	ID string `path:"id"`
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func libToBody(lib library.Library) *libraryBody {
	return &libraryBody{
		ID:                      lib.ID,
		Name:                    lib.Name,
		RootPath:                lib.RootPath,
		DefaultQualityProfileID: lib.DefaultQualityProfileID,
		NamingFormat:            lib.NamingFormat,
		MinFreeSpaceGB:          lib.MinFreeSpaceGB,
		Tags:                    lib.Tags,
		CreatedAt:               lib.CreatedAt,
		UpdatedAt:               lib.UpdatedAt,
	}
}

func libInputToCreateRequest(in libraryInput) library.CreateRequest {
	minFree := 0
	if in.MinFreeSpaceGB != nil {
		minFree = *in.MinFreeSpaceGB
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}
	return library.CreateRequest{
		Name:                    in.Name,
		RootPath:                in.RootPath,
		DefaultQualityProfileID: in.DefaultQualityProfileID,
		NamingFormat:            in.NamingFormat,
		MinFreeSpaceGB:          minFree,
		Tags:                    tags,
	}
}

// validateRootPath checks that the given path exists and is a directory.
// Returns a user-friendly error when the path is not accessible — the most
// common cause is a missing Docker volume mount.
func validateRootPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("root path %q is not accessible — if running in Docker, ensure it is mounted as a volume", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path %q is not a directory", path)
	}
	return nil
}

// ── Route registration ───────────────────────────────────────────────────────

// RegisterLibraryRoutes registers all /api/v1/libraries endpoints.
// movieSvc may be nil; the disk-import endpoints are skipped when it is.
func RegisterLibraryRoutes(api huma.API, svc *library.Service, movieSvc *movie.Service) {
	// GET /api/v1/libraries
	huma.Register(api, huma.Operation{
		OperationID: "list-libraries",
		Method:      http.MethodGet,
		Path:        "/api/v1/libraries",
		Summary:     "List libraries",
		Tags:        []string{"Libraries"},
	}, func(ctx context.Context, _ *struct{}) (*libraryListOutput, error) {
		libs, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list libraries", err)
		}

		bodies := make([]*libraryBody, len(libs))
		for i, lib := range libs {
			bodies[i] = libToBody(lib)
		}
		return &libraryListOutput{Body: bodies}, nil
	})

	// POST /api/v1/libraries
	huma.Register(api, huma.Operation{
		OperationID:   "create-library",
		Method:        http.MethodPost,
		Path:          "/api/v1/libraries",
		Summary:       "Create a library",
		Tags:          []string{"Libraries"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *libraryCreateInput) (*libraryOutput, error) {
		if err := validateRootPath(input.Body.RootPath); err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, err.Error())
		}
		lib, err := svc.Create(ctx, libInputToCreateRequest(input.Body))
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to create library", err)
		}
		return &libraryOutput{Body: libToBody(lib)}, nil
	})

	// GET /api/v1/libraries/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-library",
		Method:      http.MethodGet,
		Path:        "/api/v1/libraries/{id}",
		Summary:     "Get a library",
		Tags:        []string{"Libraries"},
	}, func(ctx context.Context, input *libraryGetInput) (*libraryOutput, error) {
		lib, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				return nil, huma.Error404NotFound("library not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get library", err)
		}
		return &libraryOutput{Body: libToBody(lib)}, nil
	})

	// PUT /api/v1/libraries/{id}
	huma.Register(api, huma.Operation{
		OperationID: "update-library",
		Method:      http.MethodPut,
		Path:        "/api/v1/libraries/{id}",
		Summary:     "Update a library",
		Tags:        []string{"Libraries"},
	}, func(ctx context.Context, input *libraryUpdateInput) (*libraryOutput, error) {
		if err := validateRootPath(input.Body.RootPath); err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, err.Error())
		}
		lib, err := svc.Update(ctx, input.ID, libInputToCreateRequest(input.Body))
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				return nil, huma.Error404NotFound("library not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update library", err)
		}
		return &libraryOutput{Body: libToBody(lib)}, nil
	})

	// DELETE /api/v1/libraries/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-library",
		Method:        http.MethodDelete,
		Path:          "/api/v1/libraries/{id}",
		Summary:       "Delete a library",
		Tags:          []string{"Libraries"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *libraryDeleteInput) (*libraryDeleteOutput, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			if errors.Is(err, library.ErrNotFound) {
				return nil, huma.Error404NotFound("library not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete library", err)
		}
		return &libraryDeleteOutput{}, nil
	})

	// POST /api/v1/libraries/{id}/scan
	huma.Register(api, huma.Operation{
		OperationID:   "scan-library",
		Method:        http.MethodPost,
		Path:          "/api/v1/libraries/{id}/scan",
		Summary:       "Trigger a library scan",
		Description:   "Reconciles tracked movie files with disk contents. Runs asynchronously.",
		Tags:          []string{"Libraries"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, input *libraryScanInput) (*libraryScanOutput, error) {
		go func() {
			if err := svc.Scan(context.Background(), input.ID); err != nil {
				// Errors are best-effort; the caller already received 202.
				_ = err
			}
		}()
		return &libraryScanOutput{}, nil
	})

	// GET /api/v1/libraries/{id}/stats
	huma.Register(api, huma.Operation{
		OperationID: "get-library-stats",
		Method:      http.MethodGet,
		Path:        "/api/v1/libraries/{id}/stats",
		Summary:     "Get library stats",
		Description: "Returns movie count, total file size, and disk space for a library.",
		Tags:        []string{"Libraries"},
	}, func(ctx context.Context, input *libraryStatsInput) (*libraryStatsOutput, error) {
		stats, err := svc.Stats(ctx, input.ID)
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				return nil, huma.Error404NotFound("library not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get library stats", err)
		}
		return &libraryStatsOutput{
			Body: &libraryStatsBody{
				MovieCount:     stats.MovieCount,
				TotalSizeBytes: stats.TotalSizeBytes,
				FreeSpaceBytes: stats.FreeSpaceBytes,
				HealthOK:       stats.HealthOK,
				HealthMessage:  stats.HealthMessage,
			},
		}, nil
	})

	// GET /api/v1/libraries/{id}/disk-scan
	// Walks the library root path and returns video files not yet in movie_files.
	huma.Register(api, huma.Operation{
		OperationID: "library-disk-scan",
		Method:      http.MethodGet,
		Path:        "/api/v1/libraries/{id}/disk-scan",
		Summary:     "Scan library disk for untracked video files",
		Description: "Returns video files found on disk that are not already tracked as movie files.",
		Tags:        []string{"Libraries"},
	}, func(ctx context.Context, input *libraryDiskScanInput) (*libraryDiskScanOutput, error) {
		diskFiles, err := svc.ScanDisk(ctx, input.ID)
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				return nil, huma.Error404NotFound("library not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to scan library disk", err)
		}
		bodies := make([]*diskFileBody, len(diskFiles))
		for i, f := range diskFiles {
			b := &diskFileBody{
				Path:        f.Path,
				SizeBytes:   f.SizeBytes,
				ParsedTitle: f.ParsedTitle,
				ParsedYear:  f.ParsedYear,
			}
			if f.TMDBMatch != nil {
				b.TMDBMatch = &tmdbMatchBody{
					TMDBID:        f.TMDBMatch.TMDBID,
					Title:         f.TMDBMatch.Title,
					OriginalTitle: f.TMDBMatch.OriginalTitle,
					Year:          f.TMDBMatch.Year,
				}
			}
			bodies[i] = b
		}
		return &libraryDiskScanOutput{Body: bodies}, nil
	})

	// GET /api/v1/libraries/{id}/candidates
	// Returns candidates stored in the DB without re-walking the disk.
	huma.Register(api, huma.Operation{
		OperationID: "library-candidates",
		Method:      http.MethodGet,
		Path:        "/api/v1/libraries/{id}/candidates",
		Summary:     "List pre-scanned file candidates",
		Description: "Returns video files previously discovered by a disk scan, with their stored TMDB matches. Does not re-walk the disk.",
		Tags:        []string{"Libraries"},
	}, func(ctx context.Context, input *libraryDiskScanInput) (*libraryDiskScanOutput, error) {
		files, err := svc.ListCandidates(ctx, input.ID)
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				return nil, huma.Error404NotFound("library not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list candidates", err)
		}
		bodies := make([]*diskFileBody, len(files))
		for i, f := range files {
			b := &diskFileBody{
				Path:        f.Path,
				SizeBytes:   f.SizeBytes,
				ParsedTitle: f.ParsedTitle,
				ParsedYear:  f.ParsedYear,
			}
			if f.TMDBMatch != nil {
				b.TMDBMatch = &tmdbMatchBody{
					TMDBID:        f.TMDBMatch.TMDBID,
					Title:         f.TMDBMatch.Title,
					OriginalTitle: f.TMDBMatch.OriginalTitle,
					Year:          f.TMDBMatch.Year,
				}
			}
			bodies[i] = b
		}
		return &libraryDiskScanOutput{Body: bodies}, nil
	})

	if movieSvc == nil {
		return
	}

	// POST /api/v1/libraries/{id}/import-file
	// Adds a movie to the library (or finds the existing one) and links the file.
	huma.Register(api, huma.Operation{
		OperationID:   "library-import-file",
		Method:        http.MethodPost,
		Path:          "/api/v1/libraries/{id}/import-file",
		Summary:       "Import a file into a library movie",
		Description:   "Adds the movie (via TMDB ID) if not already present, then links the file on disk.",
		Tags:          []string{"Libraries"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *libraryImportFileInput) (*libraryImportFileOutput, error) {
		lib, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				return nil, huma.Error404NotFound("library not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get library", err)
		}

		// Reject paths in sensitive system directories.
		if err := pathutil.ValidateContentPath(input.Body.FilePath); err != nil {
			return nil, huma.NewError(http.StatusBadRequest, err.Error())
		}

		// Verify the file exists on disk before doing anything else.
		info, err := os.Stat(input.Body.FilePath)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest,
				fmt.Sprintf("file not accessible: %s", input.Body.FilePath), err)
		}
		sizeBytes := info.Size()

		// If this file is already attached to a movie, return it directly
		// instead of creating a duplicate.
		if existing, err := movieSvc.GetByFilePath(ctx, input.Body.FilePath); err == nil {
			_ = svc.DeleteCandidate(ctx, input.ID, input.Body.FilePath)
			return &libraryImportFileOutput{Body: movieToBody(existing)}, nil
		}

		// Use the library's default quality profile; fall back to an empty string
		// which movie.Add will accept (profile validation is relaxed on import).
		qpID := lib.DefaultQualityProfileID

		var m movie.Movie
		var addErr error
		if input.Body.TmdbID == 0 {
			// No TMDB match — create an unmatched placeholder using the raw
			// filename stem as the title. The record is not monitored.
			stem := strings.TrimSuffix(
				filepath.Base(input.Body.FilePath),
				filepath.Ext(input.Body.FilePath),
			)
			m, addErr = movieSvc.AddUnmatched(ctx, movie.AddUnmatchedRequest{
				Title:            stem,
				LibraryID:        lib.ID,
				QualityProfileID: qpID,
			})
			if addErr != nil {
				slog.Error("import-file: AddUnmatched failed", "path", input.Body.FilePath, "err", addErr)
				return nil, huma.NewError(http.StatusInternalServerError, "failed to add unmatched movie", addErr)
			}
		} else {
			// Add the movie if it does not exist yet; tolerate ErrAlreadyExists.
			m, addErr = movieSvc.Add(ctx, movie.AddRequest{
				TMDBID:           input.Body.TmdbID,
				LibraryID:        lib.ID,
				QualityProfileID: qpID,
				Monitored:        true,
			})
			if addErr != nil && !errors.Is(addErr, movie.ErrAlreadyExists) {
				slog.Error("import-file: Add failed", "path", input.Body.FilePath, "tmdb_id", input.Body.TmdbID, "err", addErr)
				return nil, huma.NewError(http.StatusInternalServerError, "failed to add movie", addErr)
			}
			if errors.Is(addErr, movie.ErrAlreadyExists) {
				m, err = movieSvc.GetByTMDBID(ctx, input.Body.TmdbID)
				if err != nil {
					return nil, huma.NewError(http.StatusInternalServerError, "failed to get existing movie", err)
				}
			}
		}

		quality := library.ParseQualityFromPath(input.Body.FilePath)
		if err := movieSvc.AttachFile(ctx, m.ID, input.Body.FilePath, sizeBytes, quality); err != nil {
			slog.Error("import-file: AttachFile failed", "path", input.Body.FilePath, "movie_id", m.ID, "err", err)
			return nil, huma.NewError(http.StatusInternalServerError, "failed to attach file to movie", err)
		}

		// Remove the candidate so it no longer appears in future disk scans.
		_ = svc.DeleteCandidate(ctx, input.ID, input.Body.FilePath)

		// Re-fetch to return the fully updated movie record.
		updated, err := movieSvc.Get(ctx, m.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to read updated movie", err)
		}
		return &libraryImportFileOutput{Body: movieToBody(updated)}, nil
	})
}
