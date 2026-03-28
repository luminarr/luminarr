package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/edition"
	"github.com/luminarr/luminarr/internal/core/mediainfo"
	"github.com/luminarr/luminarr/internal/core/mediamanagement"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/renamer"
	"github.com/luminarr/luminarr/internal/core/tag"
	"github.com/luminarr/luminarr/internal/core/watchsync"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
)

// ── Movie file shapes ─────────────────────────────────────────────────────────

type renamePreviewItemBody struct {
	FileID  string `json:"file_id"`
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

type renameMovieInput struct {
	ID     string `path:"id"`
	DryRun bool   `query:"dry_run"`
}

type renameMovieOutput struct {
	Body struct {
		DryRun  bool                    `json:"dry_run"`
		Renamed []renamePreviewItemBody `json:"renamed"`
	}
}

type movieFileBody struct {
	ID            string          `json:"id"`
	MovieID       string          `json:"movie_id"`
	Path          string          `json:"path"`
	SizeBytes     int64           `json:"size_bytes"`
	Quality       any             `json:"quality"`
	Edition       string          `json:"edition,omitempty"`
	ImportedAt    time.Time       `json:"imported_at"`
	MediainfoJSON json.RawMessage `json:"mediainfo,omitempty"`
}

type movieFilesListOutput struct {
	Body []*movieFileBody
}

type movieFileDeleteInput struct {
	ID     string `path:"id"`
	FileID string `path:"fileId"`
	// DeleteFromDisk instructs the server to also remove the file on disk.
	DeleteFromDisk bool `query:"delete_from_disk"`
}

// ── Request / response shapes ────────────────────────────────────────────────

type movieBody struct {
	ID                  string           `json:"id"                             doc:"Movie UUID"`
	TMDBID              int              `json:"tmdb_id"                        doc:"TMDB movie ID"`
	IMDBID              string           `json:"imdb_id,omitempty"              doc:"IMDB movie ID"`
	Title               string           `json:"title"                          doc:"Movie title"`
	OriginalTitle       string           `json:"original_title"                 doc:"Original-language title"`
	Year                int              `json:"year"                           doc:"Release year"`
	Overview            string           `json:"overview"                       doc:"Plot summary"`
	RuntimeMinutes      int              `json:"runtime_minutes"                doc:"Runtime in minutes"`
	Genres              []string         `json:"genres"                         doc:"Genre list"`
	PosterURL           string           `json:"poster_url,omitempty"           doc:"TMDB poster path"`
	FanartURL           string           `json:"fanart_url,omitempty"           doc:"TMDB backdrop/fanart path"`
	Status              string           `json:"status"                         doc:"Release status: released or announced"`
	Monitored           bool             `json:"monitored"                      doc:"Whether the movie is monitored for downloads"`
	LibraryID           string           `json:"library_id"                     doc:"Parent library UUID"`
	QualityProfileID    string           `json:"quality_profile_id"             doc:"Quality profile UUID"`
	MinimumAvailability string           `json:"minimum_availability"           doc:"Minimum availability before grabbing: tba, announced, in_cinemas, released"`
	PreferredEdition    string           `json:"preferred_edition,omitempty"    doc:"Preferred edition (e.g. Director's Cut, Extended); empty = no preference"`
	ReleaseDate         string           `json:"release_date,omitempty"         doc:"TMDB release date (YYYY-MM-DD)"`
	Path                string           `json:"path,omitempty"                 doc:"Absolute path on disk"`
	TagIDs              []string         `json:"tag_ids"                        doc:"Assigned tag UUIDs"`
	WatchStatus         *watchStatusBody `json:"watch_status,omitempty"     doc:"Watch history status"`
	AddedAt             time.Time        `json:"added_at"                       doc:"When the movie was added"`
	UpdatedAt           time.Time        `json:"updated_at"                     doc:"When the record was last changed"`
	MetadataRefreshedAt *time.Time       `json:"metadata_refreshed_at,omitempty" doc:"When metadata was last refreshed"`
}

type watchStatusBody struct {
	Watched       bool    `json:"watched"                    doc:"At least one watch event exists"`
	PlayCount     int64   `json:"play_count"                 doc:"Total times watched"`
	LastWatchedAt *string `json:"last_watched_at,omitempty"  doc:"Most recent watch (ISO 8601)"`
}

type searchResultBody struct {
	TMDBID        int     `json:"tmdb_id"        doc:"TMDB movie ID"`
	Title         string  `json:"title"          doc:"Movie title"`
	OriginalTitle string  `json:"original_title" doc:"Original-language title"`
	Overview      string  `json:"overview"       doc:"Plot summary"`
	ReleaseDate   string  `json:"release_date"   doc:"Release date (YYYY-MM-DD)"`
	Year          int     `json:"year"           doc:"Release year"`
	PosterPath    string  `json:"poster_path"    doc:"TMDB poster path"`
	BackdropPath  string  `json:"backdrop_path"  doc:"TMDB backdrop path"`
	Popularity    float64 `json:"popularity"     doc:"TMDB popularity score"`
}

// Single-movie output.
type movieOutput struct {
	Body *movieBody
}

// Movie list output.
type movieListOutput struct {
	Body *movieListBody
}

type movieListBody struct {
	Movies  []*movieBody `json:"movies"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
}

// Lookup output — returns a list of search results.
type movieLookupOutput struct {
	Body []*searchResultBody
}

// Suggestions output — parsed filename + ranked TMDB candidates.
type movieSuggestionsOutput struct {
	Body *movieSuggestionsBody
}

type movieSuggestionsBody struct {
	ParsedTitle string              `json:"parsed_title" doc:"Title extracted from the filename"`
	ParsedYear  int                 `json:"parsed_year"  doc:"Year extracted from the filename; 0 if not found"`
	Results     []*searchResultBody `json:"results"      doc:"TMDB search results, ranked by relevance"`
}

// 204 No Content.
type movieDeleteOutput struct{}

// 202 Accepted — used for async operations.
type movieRefreshOutput struct {
	Body *movieRefreshBody
}

type movieRefreshBody struct {
	Message string `json:"message"`
}

// ── Input wrappers ────────────────────────────────────────────────────────────

type listMoviesInput struct {
	LibraryID string `query:"library_id"`
	Page      int    `query:"page"     default:"1"`
	PerPage   int    `query:"per_page" default:"50"`
}

type addMovieInput struct {
	Body struct {
		TMDBID              int      `json:"tmdb_id"                        doc:"TMDB movie ID to add"`
		LibraryID           string   `json:"library_id"                     doc:"Library UUID to place the movie in"`
		QualityProfileID    string   `json:"quality_profile_id"             doc:"Quality profile UUID"`
		Monitored           *bool    `json:"monitored,omitempty"            doc:"Whether to monitor the movie for downloads (default: true)"`
		MinimumAvailability string   `json:"minimum_availability,omitempty" doc:"Minimum availability before grabbing: tba, announced, in_cinemas, released (default: released)"`
		PreferredEdition    string   `json:"preferred_edition,omitempty"    doc:"Preferred edition (e.g. Director's Cut, Extended); empty = no preference"`
		TagIDs              []string `json:"tag_ids,omitempty"              doc:"Tag UUIDs to assign"`
	}
}

type lookupMovieInput struct {
	Body struct {
		Query  string `json:"query,omitempty"   doc:"Search string (used when tmdb_id is 0)"`
		TMDBID int    `json:"tmdb_id,omitempty" doc:"Fetch a specific movie by TMDB ID"`
		Year   int    `json:"year,omitempty"    doc:"Optional year filter for query search"`
	}
}

type getMovieInput struct {
	ID string `path:"id"`
}

type updateMovieInput struct {
	ID   string `path:"id"`
	Body struct {
		Title               string   `json:"title"                          doc:"Updated title"`
		Monitored           bool     `json:"monitored"                      doc:"Whether to monitor the movie for downloads"`
		LibraryID           string   `json:"library_id"                     doc:"Library UUID"`
		QualityProfileID    string   `json:"quality_profile_id"             doc:"Quality profile UUID"`
		MinimumAvailability string   `json:"minimum_availability,omitempty" doc:"Minimum availability before grabbing: tba, announced, in_cinemas, released"`
		PreferredEdition    *string  `json:"preferred_edition,omitempty"    doc:"Preferred edition (e.g. Director's Cut, Extended); null = don't change, empty string = clear"`
		TagIDs              []string `json:"tag_ids,omitempty"              doc:"Tag UUIDs to assign"`
	}
}

type deleteMovieInput struct {
	ID string `path:"id"`
}

type refreshMovieInput struct {
	ID string `path:"id"`
}

type matchMovieInput struct {
	ID   string `path:"id"`
	Body struct {
		TMDBID int `json:"tmdb_id" doc:"TMDB movie ID to associate with this record"`
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func movieToBody(m movie.Movie) *movieBody {
	return &movieBody{
		ID:                  m.ID,
		TMDBID:              m.TMDBID,
		IMDBID:              m.IMDBID,
		Title:               m.Title,
		OriginalTitle:       m.OriginalTitle,
		Year:                m.Year,
		Overview:            m.Overview,
		RuntimeMinutes:      m.RuntimeMinutes,
		Genres:              m.Genres,
		PosterURL:           m.PosterURL,
		FanartURL:           m.FanartURL,
		Status:              m.Status,
		Monitored:           m.Monitored,
		LibraryID:           m.LibraryID,
		QualityProfileID:    m.QualityProfileID,
		MinimumAvailability: m.MinimumAvailability,
		PreferredEdition:    m.PreferredEdition,
		ReleaseDate:         m.ReleaseDate,
		Path:                m.Path,
		AddedAt:             m.AddedAt,
		UpdatedAt:           m.UpdatedAt,
		MetadataRefreshedAt: m.MetadataRefreshedAt,
	}
}

func searchResultToBody(r tmdb.SearchResult) *searchResultBody {
	return &searchResultBody{
		TMDBID:        r.ID,
		Title:         r.Title,
		OriginalTitle: r.OriginalTitle,
		Overview:      r.Overview,
		ReleaseDate:   r.ReleaseDate,
		Year:          r.Year,
		PosterPath:    r.PosterPath,
		BackdropPath:  r.BackdropPath,
		Popularity:    r.Popularity,
	}
}

// ── Route registration ────────────────────────────────────────────────────────

// RegisterMovieRoutes registers all /api/v1/movies endpoints.
func RegisterMovieRoutes(api huma.API, svc *movie.Service, tagSvc *tag.Service, watchSvc ...*watchsync.Service) {
	var wSvc *watchsync.Service
	if len(watchSvc) > 0 {
		wSvc = watchSvc[0]
	}
	// GET /api/v1/movies
	huma.Register(api, huma.Operation{
		OperationID: "list-movies",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies",
		Summary:     "List movies",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *listMoviesInput) (*movieListOutput, error) {
		result, err := svc.List(ctx, movie.ListRequest{
			LibraryID: input.LibraryID,
			Page:      input.Page,
			PerPage:   input.PerPage,
		})
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list movies", err)
		}

		bodies := make([]*movieBody, len(result.Movies))
		for i, m := range result.Movies {
			b := movieToBody(m)
			if tagSvc != nil {
				b.TagIDs, _ = tagSvc.MovieTagIDs(ctx, m.ID)
			}
			bodies[i] = b
		}

		return &movieListOutput{Body: &movieListBody{
			Movies:  bodies,
			Total:   result.Total,
			Page:    result.Page,
			PerPage: result.PerPage,
		}}, nil
	})

	// POST /api/v1/movies
	huma.Register(api, huma.Operation{
		OperationID:   "add-movie",
		Method:        http.MethodPost,
		Path:          "/api/v1/movies",
		Summary:       "Add a movie to the library",
		Tags:          []string{"Movies"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *addMovieInput) (*movieOutput, error) {
		monitored := true
		if input.Body.Monitored != nil {
			monitored = *input.Body.Monitored
		}
		m, err := svc.Add(ctx, movie.AddRequest{
			TMDBID:              input.Body.TMDBID,
			LibraryID:           input.Body.LibraryID,
			QualityProfileID:    input.Body.QualityProfileID,
			Monitored:           monitored,
			MinimumAvailability: input.Body.MinimumAvailability,
			PreferredEdition:    input.Body.PreferredEdition,
		})
		if err != nil {
			if errors.Is(err, movie.ErrAlreadyExists) {
				return nil, huma.Error409Conflict("movie is already in the library")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to add movie", err)
		}
		b := movieToBody(m)
		if tagSvc != nil && len(input.Body.TagIDs) > 0 {
			_ = tagSvc.SetMovieTags(ctx, m.ID, input.Body.TagIDs)
			b.TagIDs = input.Body.TagIDs
		}
		if b.TagIDs == nil {
			b.TagIDs = []string{}
		}
		return &movieOutput{Body: b}, nil
	})

	// POST /api/v1/movies/lookup
	huma.Register(api, huma.Operation{
		OperationID: "lookup-movie",
		Method:      http.MethodPost,
		Path:        "/api/v1/movies/lookup",
		Summary:     "Search TMDB without adding to the library",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *lookupMovieInput) (*movieLookupOutput, error) {
		results, err := svc.Lookup(ctx, movie.LookupRequest{
			Query:  input.Body.Query,
			TMDBID: input.Body.TMDBID,
			Year:   input.Body.Year,
		})
		if err != nil {
			if errors.Is(err, movie.ErrTMDBNotConfigured) {
				return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB API key not configured")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to search TMDB", err)
		}

		bodies := make([]*searchResultBody, len(results))
		for i, r := range results {
			bodies[i] = searchResultToBody(r)
		}
		return &movieLookupOutput{Body: bodies}, nil
	})

	// GET /api/v1/movies/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-movie",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}",
		Summary:     "Get a movie by ID",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *getMovieInput) (*movieOutput, error) {
		m, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get movie", err)
		}
		b := movieToBody(m)
		if tagSvc != nil {
			b.TagIDs, _ = tagSvc.MovieTagIDs(ctx, m.ID)
		}
		if wSvc != nil {
			if ws, err := wSvc.WatchStatusForMovie(ctx, m.ID); err == nil {
				b.WatchStatus = &watchStatusBody{
					Watched:       ws.Watched,
					PlayCount:     ws.PlayCount,
					LastWatchedAt: ws.LastWatchedAt,
				}
			}
		}
		return &movieOutput{Body: b}, nil
	})

	// PUT /api/v1/movies/{id}
	huma.Register(api, huma.Operation{
		OperationID: "update-movie",
		Method:      http.MethodPut,
		Path:        "/api/v1/movies/{id}",
		Summary:     "Update a movie",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *updateMovieInput) (*movieOutput, error) {
		m, err := svc.Update(ctx, input.ID, movie.UpdateRequest{
			Title:               input.Body.Title,
			Monitored:           input.Body.Monitored,
			LibraryID:           input.Body.LibraryID,
			QualityProfileID:    input.Body.QualityProfileID,
			MinimumAvailability: input.Body.MinimumAvailability,
			PreferredEdition:    input.Body.PreferredEdition,
		})
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update movie", err)
		}
		b := movieToBody(m)
		if tagSvc != nil {
			if input.Body.TagIDs != nil {
				_ = tagSvc.SetMovieTags(ctx, m.ID, input.Body.TagIDs)
				b.TagIDs = input.Body.TagIDs
			} else {
				b.TagIDs, _ = tagSvc.MovieTagIDs(ctx, m.ID)
			}
		}
		return &movieOutput{Body: b}, nil
	})

	// DELETE /api/v1/movies/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-movie",
		Method:        http.MethodDelete,
		Path:          "/api/v1/movies/{id}",
		Summary:       "Delete a movie",
		Tags:          []string{"Movies"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *deleteMovieInput) (*movieDeleteOutput, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete movie", err)
		}
		return &movieDeleteOutput{}, nil
	})

	// POST /api/v1/movies/{id}/refresh
	huma.Register(api, huma.Operation{
		OperationID:   "refresh-movie-metadata",
		Method:        http.MethodPost,
		Path:          "/api/v1/movies/{id}/refresh",
		Summary:       "Refresh movie metadata from TMDB",
		Tags:          []string{"Movies"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, input *refreshMovieInput) (*movieRefreshOutput, error) {
		if _, err := svc.RefreshMetadata(ctx, input.ID); err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			if errors.Is(err, movie.ErrTMDBNotConfigured) {
				return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB API key not configured")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to refresh metadata", err)
		}
		return &movieRefreshOutput{Body: &movieRefreshBody{Message: "metadata refresh queued"}}, nil
	})

	// GET /api/v1/movies/{id}/suggestions
	huma.Register(api, huma.Operation{
		OperationID: "suggest-movie-matches",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/suggestions",
		Summary:     "Auto-suggest TMDB matches for an unmatched movie",
		Description: "Parses the stored filename, searches TMDB, and returns ranked candidates.",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *getMovieInput) (*movieSuggestionsOutput, error) {
		results, parsed, err := svc.SuggestMatches(ctx, input.ID)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			if errors.Is(err, movie.ErrTMDBNotConfigured) {
				return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB API key not configured")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to suggest matches", err)
		}
		bodies := make([]*searchResultBody, len(results))
		for i, r := range results {
			bodies[i] = searchResultToBody(r)
		}
		return &movieSuggestionsOutput{Body: &movieSuggestionsBody{
			ParsedTitle: parsed.Title,
			ParsedYear:  parsed.Year,
			Results:     bodies,
		}}, nil
	})

	// POST /api/v1/movies/{id}/match
	huma.Register(api, huma.Operation{
		OperationID: "match-movie-tmdb",
		Method:      http.MethodPost,
		Path:        "/api/v1/movies/{id}/match",
		Summary:     "Associate an unmatched movie with a TMDB entry",
		Description: "Sets the movie's TMDB ID and immediately fetches full metadata from TMDB.",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *matchMovieInput) (*movieOutput, error) {
		m, err := svc.MatchToTMDB(ctx, input.ID, input.Body.TMDBID)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			if errors.Is(err, movie.ErrAlreadyExists) {
				return nil, huma.Error409Conflict("another movie already has that TMDB ID")
			}
			if errors.Is(err, movie.ErrTMDBNotConfigured) {
				return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB API key not configured")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to match movie", err)
		}
		return &movieOutput{Body: movieToBody(m)}, nil
	})

	// GET /api/v1/editions — list canonical edition names for UI dropdowns.
	type editionsOutput struct {
		Body []string
	}
	huma.Register(api, huma.Operation{
		OperationID: "list-editions",
		Method:      http.MethodGet,
		Path:        "/api/v1/editions",
		Summary:     "List canonical edition names",
		Tags:        []string{"Movies"},
	}, func(_ context.Context, _ *struct{}) (*editionsOutput, error) {
		return &editionsOutput{Body: edition.Canonical()}, nil
	})
}

// RegisterMovieFileRoutes registers file management endpoints for a movie.
// mediaSvc may be nil when ffprobe is not configured; in that case scanning
// endpoints return 503 and the mediainfo field is omitted from file responses.
func RegisterMovieFileRoutes(api huma.API, svc *movie.Service, mmSvc *mediamanagement.Service, mediaSvc *mediainfo.Service) {
	// GET /api/v1/movies/{id}/files
	huma.Register(api, huma.Operation{
		OperationID: "list-movie-files",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/files",
		Summary:     "List files associated with a movie",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *getMovieInput) (*movieFilesListOutput, error) {
		if _, err := svc.Get(ctx, input.ID); err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get movie", err)
		}
		files, err := svc.ListFiles(ctx, input.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list files", err)
		}
		bodies := make([]*movieFileBody, len(files))
		for i, f := range files {
			body := &movieFileBody{
				ID:         f.ID,
				MovieID:    f.MovieID,
				Path:       f.Path,
				SizeBytes:  f.SizeBytes,
				Quality:    f.Quality,
				Edition:    f.Edition,
				ImportedAt: f.ImportedAt,
			}
			if f.MediainfoJSON != "" {
				body.MediainfoJSON = json.RawMessage(f.MediainfoJSON)
			}
			bodies[i] = body
		}
		return &movieFilesListOutput{Body: bodies}, nil
	})

	// POST /api/v1/movies/{id}/files/{fileId}/scan — trigger on-demand re-scan
	type fileScanInput struct {
		ID     string `path:"id"`
		FileID string `path:"fileId"`
	}
	huma.Register(api, huma.Operation{
		OperationID:   "scan-movie-file",
		Method:        http.MethodPost,
		Path:          "/api/v1/movies/{id}/files/{fileId}/scan",
		Summary:       "Trigger a mediainfo re-scan for a specific file",
		Tags:          []string{"Movies"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, input *fileScanInput) (*struct{}, error) {
		if mediaSvc == nil || !mediaSvc.Available() {
			return nil, huma.NewError(http.StatusServiceUnavailable, "mediainfo scanning not available — install ffprobe")
		}
		f, err := svc.GetFile(ctx, input.FileID)
		if err != nil {
			if errors.Is(err, movie.ErrFileNotFound) {
				return nil, huma.Error404NotFound("file not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get file", err)
		}
		go func() {
			if scanErr := mediaSvc.ScanFile(context.Background(), f.ID, f.Path); scanErr != nil {
				// Non-fatal; caller can retry.
				_ = scanErr
			}
		}()
		return nil, nil
	})

	// DELETE /api/v1/movies/{id}/files/{fileId}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-movie-file",
		Method:        http.MethodDelete,
		Path:          "/api/v1/movies/{id}/files/{fileId}",
		Summary:       "Delete a movie file record, optionally removing it from disk",
		Tags:          []string{"Movies"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *movieFileDeleteInput) (*struct{}, error) {
		if err := svc.DeleteFile(ctx, input.FileID, input.DeleteFromDisk); err != nil {
			if errors.Is(err, movie.ErrFileNotFound) {
				return nil, huma.Error404NotFound("movie file not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete movie file", err)
		}
		return nil, nil
	})

	// POST /api/v1/movies/{id}/rename
	huma.Register(api, huma.Operation{
		OperationID: "rename-movie-files",
		Method:      http.MethodPost,
		Path:        "/api/v1/movies/{id}/rename",
		Summary:     "Rename movie files to the configured standard format",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *renameMovieInput) (*renameMovieOutput, error) {
		mm, err := mmSvc.Get(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to load media management settings", err)
		}

		settings := movie.RenameSettings{
			Format:           mm.StandardMovieFormat,
			ColonReplacement: renamer.ColonReplacement(mm.ColonReplacement),
		}

		items, err := svc.RenameFiles(ctx, input.ID, settings, input.DryRun)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusUnprocessableEntity, "rename failed", err)
		}

		out := &renameMovieOutput{}
		out.Body.DryRun = input.DryRun
		out.Body.Renamed = make([]renamePreviewItemBody, len(items))
		for i, item := range items {
			out.Body.Renamed[i] = renamePreviewItemBody{
				FileID:  item.FileID,
				OldPath: item.OldPath,
				NewPath: item.NewPath,
			}
		}
		return out, nil
	})
}
