package v1

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/metadata/tmdb"
)

// ── Request / response shapes ────────────────────────────────────────────────

type movieBody struct {
	ID                  string     `json:"id"                             doc:"Movie UUID"`
	TMDBID              int        `json:"tmdb_id"                        doc:"TMDB movie ID"`
	IMDBID              string     `json:"imdb_id,omitempty"              doc:"IMDB movie ID"`
	Title               string     `json:"title"                          doc:"Movie title"`
	OriginalTitle       string     `json:"original_title"                 doc:"Original-language title"`
	Year                int        `json:"year"                           doc:"Release year"`
	Overview            string     `json:"overview"                       doc:"Plot summary"`
	RuntimeMinutes      int        `json:"runtime_minutes"                doc:"Runtime in minutes"`
	Genres              []string   `json:"genres"                         doc:"Genre list"`
	PosterURL           string     `json:"poster_url,omitempty"           doc:"TMDB poster path"`
	FanartURL           string     `json:"fanart_url,omitempty"           doc:"TMDB backdrop/fanart path"`
	Status              string     `json:"status"                         doc:"Release status: released or announced"`
	Monitored           bool       `json:"monitored"                      doc:"Whether the movie is monitored for downloads"`
	LibraryID           string     `json:"library_id"                     doc:"Parent library UUID"`
	QualityProfileID    string     `json:"quality_profile_id"             doc:"Quality profile UUID"`
	MinimumAvailability string     `json:"minimum_availability"           doc:"Minimum availability before grabbing: tba, announced, in_cinemas, released"`
	Path                string     `json:"path,omitempty"                 doc:"Absolute path on disk"`
	AddedAt             time.Time  `json:"added_at"                       doc:"When the movie was added"`
	UpdatedAt           time.Time  `json:"updated_at"                     doc:"When the record was last changed"`
	MetadataRefreshedAt *time.Time `json:"metadata_refreshed_at,omitempty" doc:"When metadata was last refreshed"`
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
		TMDBID              int    `json:"tmdb_id"                        doc:"TMDB movie ID to add"`
		LibraryID           string `json:"library_id"                     doc:"Library UUID to place the movie in"`
		QualityProfileID    string `json:"quality_profile_id"             doc:"Quality profile UUID"`
		Monitored           *bool  `json:"monitored,omitempty"            doc:"Whether to monitor the movie for downloads (default: true)"`
		MinimumAvailability string `json:"minimum_availability,omitempty" doc:"Minimum availability before grabbing: tba, announced, in_cinemas, released (default: released)"`
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
		Title               string `json:"title"                          doc:"Updated title"`
		Monitored           bool   `json:"monitored"                      doc:"Whether to monitor the movie for downloads"`
		LibraryID           string `json:"library_id"                     doc:"Library UUID"`
		QualityProfileID    string `json:"quality_profile_id"             doc:"Quality profile UUID"`
		MinimumAvailability string `json:"minimum_availability,omitempty" doc:"Minimum availability before grabbing: tba, announced, in_cinemas, released"`
	}
}

type deleteMovieInput struct {
	ID string `path:"id"`
}

type refreshMovieInput struct {
	ID string `path:"id"`
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
func RegisterMovieRoutes(api huma.API, svc *movie.Service) {
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
			bodies[i] = movieToBody(m)
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
		})
		if err != nil {
			if errors.Is(err, movie.ErrAlreadyExists) {
				return nil, huma.Error409Conflict("movie is already in the library")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to add movie", err)
		}
		return &movieOutput{Body: movieToBody(m)}, nil
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
		return &movieOutput{Body: movieToBody(m)}, nil
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
		})
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update movie", err)
		}
		return &movieOutput{Body: movieToBody(m)}, nil
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
}
