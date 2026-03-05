package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/collection"
)

// ── Request / response shapes ────────────────────────────────────────────────

type collectionItemBody struct {
	TMDBID     int    `json:"tmdb_id"`
	Title      string `json:"title"`
	Year       int    `json:"year"`
	PosterPath string `json:"poster_path"`
	InLibrary  bool   `json:"in_library"`
	HasFile    bool   `json:"has_file,omitempty"`
	MovieID    string `json:"movie_id,omitempty"`
	Monitored  bool   `json:"monitored,omitempty"`
}

type collectionBody struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	PersonID   int                  `json:"person_id"`
	PersonType string               `json:"person_type"`
	CreatedAt  string               `json:"created_at"`
	Items      []collectionItemBody `json:"items,omitempty"`
	Total      int                  `json:"total"`
	InLibrary  int                  `json:"in_library"`
	Missing    int                  `json:"missing"`
}

type collectionListOutput struct {
	Body []collectionBody
}

type collectionOutput struct {
	Body collectionBody
}

type createCollectionInput struct {
	Body struct {
		PersonID   int    `json:"person_id"`
		PersonType string `json:"person_type"`
	}
}

type addMissingInput struct {
	ID   string `path:"id"`
	Body struct {
		LibraryID           string `json:"library_id"`
		QualityProfileID    string `json:"quality_profile_id"`
		MinimumAvailability string `json:"minimum_availability"`
	}
}

type addSelectedInput struct {
	ID   string `path:"id"`
	Body struct {
		TMDBIDs             []int  `json:"tmdb_ids"`
		LibraryID           string `json:"library_id"`
		QualityProfileID    string `json:"quality_profile_id"`
		MinimumAvailability string `json:"minimum_availability"`
	}
}

type addMissingOutput struct {
	Body struct {
		Added             int `json:"added"`
		SkippedDuplicates int `json:"skipped_duplicates"`
	}
}

type personSearchResult struct {
	PersonID           int    `json:"person_id"`
	Name               string `json:"name"`
	ProfilePath        string `json:"profile_path"`
	KnownForDepartment string `json:"known_for_department"`
}

type personSearchInput struct {
	Q string `query:"q"`
}

type personSearchOutput struct {
	Body []personSearchResult
}

type entitySearchResult struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ImagePath  string `json:"image_path"`
	Subtitle   string `json:"subtitle"`
	ResultType string `json:"result_type"`
}

type entitySearchOutput struct {
	Body []entitySearchResult
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func toCollectionBody(c collection.Collection) collectionBody {
	b := collectionBody{
		ID:         c.ID,
		Name:       c.Name,
		PersonID:   c.PersonID,
		PersonType: c.PersonType,
		CreatedAt:  c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Total:      c.Total,
		InLibrary:  c.InLibrary,
		Missing:    c.Missing,
	}
	if c.Items != nil {
		b.Items = make([]collectionItemBody, 0, len(c.Items))
		for _, item := range c.Items {
			b.Items = append(b.Items, collectionItemBody{
				TMDBID:     item.TMDBID,
				Title:      item.Title,
				Year:       item.Year,
				PosterPath: item.PosterPath,
				InLibrary:  item.InLibrary,
				HasFile:    item.HasFile,
				MovieID:    item.MovieID,
				Monitored:  item.Monitored,
			})
		}
	}
	return b
}

// ── Route registration ────────────────────────────────────────────────────────

// RegisterCollectionRoutes registers collection CRUD and person search endpoints.
// svc may be nil (TMDB not configured); endpoints return 503 in that case.
func RegisterCollectionRoutes(api huma.API, svc *collection.Service) {
	// GET /api/v1/collections
	huma.Register(api, huma.Operation{
		OperationID: "list-collections",
		Method:      http.MethodGet,
		Path:        "/api/v1/collections",
		Summary:     "List all collections",
		Tags:        []string{"Collections"},
	}, func(ctx context.Context, _ *struct{}) (*collectionListOutput, error) {
		if svc == nil {
			return &collectionListOutput{Body: []collectionBody{}}, nil
		}
		colls, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		out := &collectionListOutput{Body: make([]collectionBody, 0, len(colls))}
		for _, c := range colls {
			out.Body = append(out.Body, toCollectionBody(c))
		}
		return out, nil
	})

	// POST /api/v1/collections
	huma.Register(api, huma.Operation{
		OperationID:   "create-collection",
		Method:        http.MethodPost,
		Path:          "/api/v1/collections",
		Summary:       "Create a director or actor collection",
		Tags:          []string{"Collections"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *createCollectionInput) (*collectionOutput, error) {
		if svc == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB not configured")
		}
		c, err := svc.Create(ctx, input.Body.PersonID, input.Body.PersonType)
		if err != nil {
			if errors.Is(err, collection.ErrAlreadyExists) {
				return nil, huma.NewError(http.StatusConflict, "collection already exists for this person")
			}
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		return &collectionOutput{Body: toCollectionBody(*c)}, nil
	})

	// GET /api/v1/collections/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-collection",
		Method:      http.MethodGet,
		Path:        "/api/v1/collections/{id}",
		Summary:     "Get a collection with its full item list",
		Tags:        []string{"Collections"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*collectionOutput, error) {
		if svc == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB not configured")
		}
		c, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, collection.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "collection not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		return &collectionOutput{Body: toCollectionBody(*c)}, nil
	})

	// DELETE /api/v1/collections/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-collection",
		Method:        http.MethodDelete,
		Path:          "/api/v1/collections/{id}",
		Summary:       "Delete a collection record",
		Tags:          []string{"Collections"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		if svc == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB not configured")
		}
		if err := svc.Delete(ctx, input.ID); err != nil {
			if errors.Is(err, collection.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "collection not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		return nil, nil
	})

	// POST /api/v1/collections/{id}/add-missing
	huma.Register(api, huma.Operation{
		OperationID: "add-missing-to-collection",
		Method:      http.MethodPost,
		Path:        "/api/v1/collections/{id}/add-missing",
		Summary:     "Add all missing films in a collection to the library",
		Tags:        []string{"Collections"},
	}, func(ctx context.Context, input *addMissingInput) (*addMissingOutput, error) {
		if svc == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB not configured")
		}
		res, err := svc.AddMissing(ctx, input.ID, collection.AddMissingRequest{
			LibraryID:           input.Body.LibraryID,
			QualityProfileID:    input.Body.QualityProfileID,
			MinimumAvailability: input.Body.MinimumAvailability,
		})
		if err != nil {
			if errors.Is(err, collection.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "collection not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		out := &addMissingOutput{}
		out.Body.Added = res.Added
		out.Body.SkippedDuplicates = res.SkippedDuplicates
		return out, nil
	})

	// POST /api/v1/collections/{id}/add-selected
	huma.Register(api, huma.Operation{
		OperationID: "add-selected-to-collection",
		Method:      http.MethodPost,
		Path:        "/api/v1/collections/{id}/add-selected",
		Summary:     "Add a specific set of films (by TMDB ID) to the library",
		Tags:        []string{"Collections"},
	}, func(ctx context.Context, input *addSelectedInput) (*addMissingOutput, error) {
		if svc == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB not configured")
		}
		res, err := svc.AddSelected(ctx, input.ID, collection.AddSelectedRequest{
			TMDBIDs:             input.Body.TMDBIDs,
			LibraryID:           input.Body.LibraryID,
			QualityProfileID:    input.Body.QualityProfileID,
			MinimumAvailability: input.Body.MinimumAvailability,
		})
		if err != nil {
			if errors.Is(err, collection.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "collection not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		out := &addMissingOutput{}
		out.Body.Added = res.Added
		out.Body.SkippedDuplicates = res.SkippedDuplicates
		return out, nil
	})

	// GET /api/v1/tmdb/people/search?q=<name>
	huma.Register(api, huma.Operation{
		OperationID: "search-tmdb-people",
		Method:      http.MethodGet,
		Path:        "/api/v1/tmdb/people/search",
		Summary:     "Search TMDB for people by name",
		Tags:        []string{"Collections"},
	}, func(ctx context.Context, input *personSearchInput) (*personSearchOutput, error) {
		if svc == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB not configured")
		}
		results, err := svc.SearchPeople(ctx, input.Q)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		out := &personSearchOutput{Body: make([]personSearchResult, 0, len(results))}
		for _, r := range results {
			out.Body = append(out.Body, personSearchResult{
				PersonID:           r.ID,
				Name:               r.Name,
				ProfilePath:        r.ProfilePath,
				KnownForDepartment: r.KnownForDepartment,
			})
		}
		return out, nil
	})

	// GET /api/v1/tmdb/search?q=<query>
	// Searches both people and movie franchises, returning a unified list.
	huma.Register(api, huma.Operation{
		OperationID: "search-tmdb-unified",
		Method:      http.MethodGet,
		Path:        "/api/v1/tmdb/search",
		Summary:     "Search TMDB for people and movie franchises",
		Tags:        []string{"Collections"},
	}, func(ctx context.Context, input *personSearchInput) (*entitySearchOutput, error) {
		if svc == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB not configured")
		}
		results, err := svc.SearchAll(ctx, input.Q)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		out := &entitySearchOutput{Body: make([]entitySearchResult, 0, len(results))}
		for _, r := range results {
			out.Body = append(out.Body, entitySearchResult{
				ID:         r.ID,
				Name:       r.Name,
				ImagePath:  r.ImagePath,
				Subtitle:   r.Subtitle,
				ResultType: r.ResultType,
			})
		}
		return out, nil
	})
}
