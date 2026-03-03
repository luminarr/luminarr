package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/library"
	"github.com/davidfic/luminarr/internal/core/movie"
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

type diskFileBody struct {
	Path        string `json:"path"         doc:"Absolute path to the file"`
	SizeBytes   int64  `json:"size_bytes"   doc:"File size in bytes"`
	ParsedTitle string `json:"parsed_title" doc:"Title guessed from filename"`
	ParsedYear  int    `json:"parsed_year"  doc:"Year guessed from filename; 0 if not found"`
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
			bodies[i] = &diskFileBody{
				Path:        f.Path,
				SizeBytes:   f.SizeBytes,
				ParsedTitle: f.ParsedTitle,
				ParsedYear:  f.ParsedYear,
			}
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

		// Verify the file exists on disk before doing anything else.
		info, err := os.Stat(input.Body.FilePath)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest,
				fmt.Sprintf("file not accessible: %s", input.Body.FilePath), err)
		}
		sizeBytes := info.Size()

		// Use the library's default quality profile; fall back to an empty string
		// which movie.Add will accept (profile validation is relaxed on import).
		qpID := lib.DefaultQualityProfileID

		// Add the movie if it does not exist yet; tolerate ErrAlreadyExists.
		m, addErr := movieSvc.Add(ctx, movie.AddRequest{
			TMDBID:           input.Body.TmdbID,
			LibraryID:        lib.ID,
			QualityProfileID: qpID,
			Monitored:        true,
		})
		if addErr != nil && !errors.Is(addErr, movie.ErrAlreadyExists) {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to add movie", addErr)
		}
		if errors.Is(addErr, movie.ErrAlreadyExists) {
			m, err = movieSvc.GetByTMDBID(ctx, input.Body.TmdbID)
			if err != nil {
				return nil, huma.NewError(http.StatusInternalServerError, "failed to get existing movie", err)
			}
		}

		quality := library.ParseQualityFromPath(input.Body.FilePath)
		if err := movieSvc.AttachFile(ctx, m.ID, input.Body.FilePath, sizeBytes, quality); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to attach file to movie", err)
		}

		// Re-fetch to return the fully updated movie record.
		updated, err := movieSvc.Get(ctx, m.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to read updated movie", err)
		}
		return &libraryImportFileOutput{Body: movieToBody(updated)}, nil
	})
}
