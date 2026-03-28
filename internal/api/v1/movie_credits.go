package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
)

// creditsCastBody is a single cast member in the API response.
type creditsCastBody struct {
	ID          int    `json:"id"           doc:"TMDB person ID"`
	Name        string `json:"name"         doc:"Actor name"`
	Character   string `json:"character"    doc:"Character name"`
	ProfilePath string `json:"profile_path" doc:"TMDB profile image path"`
	Order       int    `json:"order"        doc:"Billing order"`
}

// creditsCrewBody is a single crew member in the API response.
type creditsCrewBody struct {
	ID          int    `json:"id"           doc:"TMDB person ID"`
	Name        string `json:"name"         doc:"Crew member name"`
	Job         string `json:"job"          doc:"Role (Director, Screenplay, etc.)"`
	Department  string `json:"department"   doc:"Department"`
	ProfilePath string `json:"profile_path" doc:"TMDB profile image path"`
}

// recommendationBody is a recommended movie in the API response.
type recommendationBody struct {
	TMDBID     int    `json:"tmdb_id"     doc:"TMDB movie ID"`
	Title      string `json:"title"       doc:"Movie title"`
	Year       int    `json:"year"        doc:"Release year"`
	PosterPath string `json:"poster_path" doc:"TMDB poster path"`
	InLibrary  bool   `json:"in_library"  doc:"Whether this movie is in the Luminarr library"`
	MovieID    string `json:"movie_id,omitempty" doc:"Luminarr movie ID if in library"`
}

// movieCreditsBody is the response for GET /api/v1/movies/{id}/credits.
type movieCreditsBody struct {
	Cast            []creditsCastBody    `json:"cast"`
	Crew            []creditsCrewBody    `json:"crew"`
	Recommendations []recommendationBody `json:"recommendations"`
}

type movieCreditsOutput struct {
	Body *movieCreditsBody
}

// RegisterMovieCreditsRoutes registers the /api/v1/movies/{id}/credits endpoint.
// tmdbClient may be nil (TMDB not configured).
func RegisterMovieCreditsRoutes(api huma.API, movieSvc *movie.Service, tmdbClient *tmdb.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-movie-credits",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/credits",
		Summary:     "Get movie credits and recommendations",
		Description: "Returns cast, key crew, and TMDB recommendations for a movie. Requires TMDB to be configured.",
		Tags:        []string{"Movies"},
	}, func(ctx context.Context, input *getMovieInput) (*movieCreditsOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}

		m, err := movieSvc.Get(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("movie not found")
		}
		if m.TMDBID == 0 {
			// Unmatched movie — no credits available.
			return &movieCreditsOutput{Body: &movieCreditsBody{
				Cast:            []creditsCastBody{},
				Crew:            []creditsCrewBody{},
				Recommendations: []recommendationBody{},
			}}, nil
		}

		detail, tmdbErr := tmdbClient.GetMovieExtended(ctx, m.TMDBID)
		if tmdbErr != nil {
			// TMDB unavailable — return empty so the page still loads.
			return &movieCreditsOutput{Body: &movieCreditsBody{ //nolint:nilerr // graceful degradation

				Cast:            []creditsCastBody{},
				Crew:            []creditsCrewBody{},
				Recommendations: []recommendationBody{},
			}}, nil
		}

		cast := make([]creditsCastBody, 0, len(detail.Cast))
		for _, c := range detail.Cast {
			cast = append(cast, creditsCastBody{
				ID:          c.ID,
				Name:        c.Name,
				Character:   c.Character,
				ProfilePath: c.ProfilePath,
				Order:       c.Order,
			})
		}

		crew := make([]creditsCrewBody, 0, len(detail.Crew))
		for _, c := range detail.Crew {
			crew = append(crew, creditsCrewBody{
				ID:          c.ID,
				Name:        c.Name,
				Job:         c.Job,
				Department:  c.Department,
				ProfilePath: c.ProfilePath,
			})
		}

		recs := make([]recommendationBody, 0, len(detail.Recommendations))
		for _, r := range detail.Recommendations {
			rec := recommendationBody{
				TMDBID:     r.TMDBID,
				Title:      r.Title,
				Year:       r.Year,
				PosterPath: r.PosterPath,
			}
			// Cross-reference against library.
			if libraryMovie, lookupErr := movieSvc.GetByTMDBID(ctx, r.TMDBID); lookupErr == nil {
				rec.InLibrary = true
				rec.MovieID = libraryMovie.ID
			}
			recs = append(recs, rec)
		}

		return &movieCreditsOutput{Body: &movieCreditsBody{
			Cast:            cast,
			Crew:            crew,
			Recommendations: recs,
		}}, nil
	})
}
