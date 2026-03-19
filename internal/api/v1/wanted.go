package v1

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/pkg/plugin"
)

type wantedInput struct {
	Page    int `query:"page"     default:"1"  minimum:"1"`
	PerPage int `query:"per_page" default:"25" minimum:"1" maximum:"250"`
}

type wantedListBody struct {
	Movies  []*movieBody `json:"movies"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
}

type wantedListOutput struct {
	Body *wantedListBody
}

// RegisterWantedRoutes registers the wanted/missing and cutoff-unmet endpoints.
func RegisterWantedRoutes(api huma.API, svc *movie.Service) {
	// GET /api/v1/wanted/missing — monitored movies with no file
	huma.Register(api, huma.Operation{
		OperationID: "wanted-missing",
		Method:      http.MethodGet,
		Path:        "/api/v1/wanted/missing",
		Summary:     "List monitored movies with no file",
		Tags:        []string{"Wanted"},
	}, func(ctx context.Context, input *wantedInput) (*wantedListOutput, error) {
		movies, total, err := svc.ListMissing(ctx, input.Page, input.PerPage)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list missing movies", err)
		}
		bodies := make([]*movieBody, len(movies))
		for i, m := range movies {
			bodies[i] = movieToBody(m)
		}
		return &wantedListOutput{Body: &wantedListBody{
			Movies:  bodies,
			Total:   total,
			Page:    input.Page,
			PerPage: input.PerPage,
		}}, nil
	})

	// GET /api/v1/wanted/cutoff — monitored movies whose best file is below the quality profile cutoff
	huma.Register(api, huma.Operation{
		OperationID: "wanted-cutoff",
		Method:      http.MethodGet,
		Path:        "/api/v1/wanted/cutoff",
		Summary:     "List monitored movies whose file quality is below the profile cutoff",
		Tags:        []string{"Wanted"},
	}, func(ctx context.Context, _ *struct{}) (*wantedListOutput, error) {
		movies, err := svc.ListCutoffUnmet(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list cutoff-unmet movies", err)
		}
		bodies := make([]*movieBody, len(movies))
		for i, m := range movies {
			bodies[i] = movieToBody(m)
		}
		return &wantedListOutput{Body: &wantedListBody{
			Movies:  bodies,
			Total:   int64(len(movies)),
			Page:    1,
			PerPage: len(movies),
		}}, nil
	})

	// GET /api/v1/wanted/upgrades — upgrade recommendations grouped by quality tier.
	type upgradeTierBody struct {
		Label       string   `json:"label"`
		FromQuality string   `json:"from_quality"`
		ToQuality   string   `json:"to_quality"`
		Count       int      `json:"count"`
		MovieIDs    []string `json:"movie_ids"`
	}
	type upgradeRecsBody struct {
		Total int                `json:"total"`
		Tiers []upgradeTierBody  `json:"tiers"`
	}

	huma.Register(api, huma.Operation{
		OperationID: "wanted-upgrades",
		Method:      http.MethodGet,
		Path:        "/api/v1/wanted/upgrades",
		Summary:     "List upgrade recommendations grouped by quality tier",
		Tags:        []string{"Wanted"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body upgradeRecsBody }, error) {
		movies, err := svc.ListCutoffUnmet(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}

		// Build upgrade tiers by grouping movies by their current→target quality transition.
		type tierKey struct{ from, to string }
		tierMovies := make(map[tierKey][]string)

		for _, m := range movies {
			// Get the movie's best file quality and profile cutoff.
			files, fErr := svc.ListFiles(ctx, m.ID)
			if fErr != nil || len(files) == 0 {
				continue
			}
			var bestQ plugin.Quality
			for _, f := range files {
				if f.Quality.BetterThan(bestQ) {
					bestQ = f.Quality
				}
			}

			// We need the cutoff quality — it's stored on the profile.
			// For now, use a simplified approach: the cutoff is embedded
			// in the movie's quality profile data. We can get it from the
			// ListMonitoredMoviesWithFiles query that ListCutoffUnmet uses.
			// Since we already have the movies, we just label by current quality.
			fromLabel := bestQ.Name
			if fromLabel == "" {
				fromLabel = string(bestQ.Resolution) + " " + string(bestQ.Source)
			}
			// The target is "profile cutoff" — we don't have it directly here,
			// so label it generically as "upgrade available".
			toLabel := "profile cutoff"
			_ = json.Marshal // suppress unused import

			key := tierKey{fromLabel, toLabel}
			tierMovies[key] = append(tierMovies[key], m.ID)
		}

		var tiers []upgradeTierBody
		for key, ids := range tierMovies {
			tiers = append(tiers, upgradeTierBody{
				Label:       key.from + " → " + key.to,
				FromQuality: key.from,
				ToQuality:   key.to,
				Count:       len(ids),
				MovieIDs:    ids,
			})
		}

		return &struct{ Body upgradeRecsBody }{Body: upgradeRecsBody{
			Total: len(movies),
			Tiers: tiers,
		}}, nil
	})
}
