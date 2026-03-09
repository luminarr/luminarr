package v3

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/movie"
)

func registerMovieRoutes(api huma.API, db *sql.DB, movieSvc *movie.Service, libSvc *library.Service) {
	if movieSvc == nil {
		return
	}

	// GET /api/v3/movie — list all movies
	huma.Register(api, huma.Operation{
		OperationID: "radarr-list-movies",
		Method:      http.MethodGet,
		Path:        "/api/v3/movie",
		Summary:     "List movies (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []RadarrMovie }, error) {
		result, err := movieSvc.List(ctx, movie.ListRequest{Page: 1, PerPage: 10000})
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list movies", err)
		}

		// Build ROWID maps for movies, quality profiles, and libraries.
		movieMap, err := buildRowIDMap(ctx, db, "movies")
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "rowid lookup failed", err)
		}
		qpMap, err := buildRowIDMap(ctx, db, "quality_profiles")
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "rowid lookup failed", err)
		}

		// Build library root path and rowid maps.
		libs, err := libSvc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list libraries", err)
		}
		libRootPaths := make(map[string]string, len(libs))
		for _, lib := range libs {
			libRootPaths[lib.ID] = lib.RootPath
		}

		movies := make([]RadarrMovie, 0, len(result.Movies))
		for _, m := range result.Movies {
			files, _ := movieSvc.ListFiles(ctx, m.ID)
			rowid := movieMap.uuidToRow[m.ID]
			qpRowID := qpMap.uuidToRow[m.QualityProfileID]
			libRootPath := libRootPaths[m.LibraryID]
			movies = append(movies, movieToRadarr(m, rowid, files, libRootPath, qpRowID))
		}
		return &struct{ Body []RadarrMovie }{Body: movies}, nil
	})

	// GET /api/v3/movie/{id} — get single movie by ROWID
	type getMovieInput struct {
		ID int64 `path:"id"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "radarr-get-movie",
		Method:      http.MethodGet,
		Path:        "/api/v3/movie/{id}",
		Summary:     "Get movie by ID (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, input *getMovieInput) (*struct{ Body RadarrMovie }, error) {
		uuid, err := getUUIDByRowID(ctx, db, "movies", input.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "lookup failed", err)
		}
		if uuid == "" {
			return nil, huma.NewError(http.StatusNotFound, "movie not found")
		}
		m, err := movieSvc.Get(ctx, uuid)
		if err != nil {
			return nil, huma.NewError(http.StatusNotFound, "movie not found")
		}
		files, _ := movieSvc.ListFiles(ctx, m.ID)
		qpRowID, _ := getRowIDByUUID(ctx, db, "quality_profiles", m.QualityProfileID)

		var libRootPath string
		if libSvc != nil {
			if lib, err := libSvc.Get(ctx, m.LibraryID); err == nil {
				libRootPath = lib.RootPath
			}
		}
		return &struct{ Body RadarrMovie }{Body: movieToRadarr(m, input.ID, files, libRootPath, qpRowID)}, nil
	})

	// GET /api/v3/movie/lookup — lookup by tmdb: or imdb: prefix
	type lookupInput struct {
		Term string `query:"term" required:"true"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "radarr-lookup-movie",
		Method:      http.MethodGet,
		Path:        "/api/v3/movie/lookup",
		Summary:     "Lookup movie (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, input *lookupInput) (*struct{ Body []RadarrMovie }, error) {
		req := movie.LookupRequest{}
		term := input.Term

		switch {
		case strings.HasPrefix(term, "tmdb:"):
			id, err := strconv.Atoi(strings.TrimPrefix(term, "tmdb:"))
			if err != nil {
				return nil, huma.NewError(http.StatusBadRequest, "invalid tmdb ID")
			}
			req.TMDBID = id
		case strings.HasPrefix(term, "imdb:"):
			// Search by IMDB ID: look up in our DB first, then fall back to TMDB.
			imdbID := strings.TrimPrefix(term, "imdb:")
			// Try to find in local DB by IMDB ID — not directly supported, so search TMDB.
			req.Query = imdbID
		default:
			req.Query = term
		}

		results, err := movieSvc.Lookup(ctx, req)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "lookup failed", err)
		}

		movies := make([]RadarrMovie, len(results))
		for i, r := range results {
			movies[i] = tmdbResultToRadarrMovie(r)

			// If the movie exists in our DB, enrich with local data.
			if existing, err := movieSvc.GetByTMDBID(ctx, r.ID); err == nil {
				rowid, _ := getRowIDByUUID(ctx, db, "movies", existing.ID)
				qpRowID, _ := getRowIDByUUID(ctx, db, "quality_profiles", existing.QualityProfileID)
				files, _ := movieSvc.ListFiles(ctx, existing.ID)
				var libRootPath string
				if libSvc != nil {
					if lib, err := libSvc.Get(ctx, existing.LibraryID); err == nil {
						libRootPath = lib.RootPath
					}
				}
				movies[i] = movieToRadarr(existing, rowid, files, libRootPath, qpRowID)
			}
		}
		return &struct{ Body []RadarrMovie }{Body: movies}, nil
	})

	// POST /api/v3/movie — add a movie
	type addMovieInput struct {
		Body struct {
			TmdbID              int    `json:"tmdbId" required:"true"`
			Title               string `json:"title"`
			QualityProfileID    int64  `json:"qualityProfileId"`
			RootFolderPath      string `json:"rootFolderPath"`
			Monitored           bool   `json:"monitored"`
			MinimumAvailability string `json:"minimumAvailability"`
			Tags                []int  `json:"tags"`
			AddOptions          struct {
				SearchForMovie bool `json:"searchForMovie"`
			} `json:"addOptions"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "radarr-add-movie",
		Method:      http.MethodPost,
		Path:        "/api/v3/movie",
		Summary:     "Add movie (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, input *addMovieInput) (*struct {
		Status int
		Body   RadarrMovie
	}, error) {
		// Resolve quality profile ROWID → UUID.
		qpUUID, err := getUUIDByRowID(ctx, db, "quality_profiles", input.Body.QualityProfileID)
		if err != nil || qpUUID == "" {
			return nil, huma.NewError(http.StatusBadRequest, "invalid qualityProfileId")
		}

		// Resolve root folder path → library UUID.
		var libraryID string
		if libSvc != nil {
			libs, err := libSvc.List(ctx)
			if err == nil {
				for _, lib := range libs {
					if lib.RootPath == input.Body.RootFolderPath {
						libraryID = lib.ID
						break
					}
				}
			}
		}
		if libraryID == "" {
			return nil, huma.NewError(http.StatusBadRequest, "rootFolderPath does not match any library")
		}

		m, err := movieSvc.Add(ctx, movie.AddRequest{
			TMDBID:              input.Body.TmdbID,
			LibraryID:           libraryID,
			QualityProfileID:    qpUUID,
			Monitored:           input.Body.Monitored,
			MinimumAvailability: input.Body.MinimumAvailability,
		})
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "failed to add movie", err)
		}

		rowid, _ := getRowIDByUUID(ctx, db, "movies", m.ID)
		qpRowID, _ := getRowIDByUUID(ctx, db, "quality_profiles", m.QualityProfileID)
		var libRootPath string
		if libSvc != nil {
			if lib, err := libSvc.Get(ctx, m.LibraryID); err == nil {
				libRootPath = lib.RootPath
			}
		}

		return &struct {
			Status int
			Body   RadarrMovie
		}{Status: http.StatusCreated, Body: movieToRadarr(m, rowid, nil, libRootPath, qpRowID)}, nil
	})

	// PUT /api/v3/movie/{id} — update a movie
	type updateMovieInput struct {
		ID   int64 `path:"id"`
		Body struct {
			Monitored           bool   `json:"monitored"`
			QualityProfileID    int64  `json:"qualityProfileId"`
			MinimumAvailability string `json:"minimumAvailability"`
			RootFolderPath      string `json:"rootFolderPath"`
			Title               string `json:"title"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "radarr-update-movie",
		Method:      http.MethodPut,
		Path:        "/api/v3/movie/{id}",
		Summary:     "Update movie (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, input *updateMovieInput) (*struct{ Body RadarrMovie }, error) {
		uuid, err := getUUIDByRowID(ctx, db, "movies", input.ID)
		if err != nil || uuid == "" {
			return nil, huma.NewError(http.StatusNotFound, "movie not found")
		}

		// Get current movie to fill in defaults.
		existing, err := movieSvc.Get(ctx, uuid)
		if err != nil {
			return nil, huma.NewError(http.StatusNotFound, "movie not found")
		}

		req := movie.UpdateRequest{
			Title:               existing.Title,
			Monitored:           input.Body.Monitored,
			LibraryID:           existing.LibraryID,
			QualityProfileID:    existing.QualityProfileID,
			MinimumAvailability: existing.MinimumAvailability,
		}

		if input.Body.Title != "" {
			req.Title = input.Body.Title
		}
		if input.Body.MinimumAvailability != "" {
			req.MinimumAvailability = input.Body.MinimumAvailability
		}
		if input.Body.QualityProfileID > 0 {
			qpUUID, err := getUUIDByRowID(ctx, db, "quality_profiles", input.Body.QualityProfileID)
			if err != nil || qpUUID == "" {
				return nil, huma.NewError(http.StatusBadRequest, "invalid qualityProfileId")
			}
			req.QualityProfileID = qpUUID
		}

		m, err := movieSvc.Update(ctx, uuid, req)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update movie", err)
		}

		files, _ := movieSvc.ListFiles(ctx, m.ID)
		qpRowID, _ := getRowIDByUUID(ctx, db, "quality_profiles", m.QualityProfileID)
		var libRootPath string
		if libSvc != nil {
			if lib, err := libSvc.Get(ctx, m.LibraryID); err == nil {
				libRootPath = lib.RootPath
			}
		}

		return &struct{ Body RadarrMovie }{Body: movieToRadarr(m, input.ID, files, libRootPath, qpRowID)}, nil
	})

	// DELETE /api/v3/movie/{id}
	type deleteMovieInput struct {
		ID int64 `path:"id"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "radarr-delete-movie",
		Method:      http.MethodDelete,
		Path:        "/api/v3/movie/{id}",
		Summary:     "Delete movie (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, input *deleteMovieInput) (*struct{}, error) {
		uuid, err := getUUIDByRowID(ctx, db, "movies", input.ID)
		if err != nil || uuid == "" {
			return nil, huma.NewError(http.StatusNotFound, "movie not found")
		}
		if err := movieSvc.Delete(ctx, uuid); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete movie", err)
		}
		return nil, nil
	})
}
