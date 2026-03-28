package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.themoviedb.org/3"
	httpTimeout    = 30 * time.Second
	userAgent      = "Luminarr/0.1.0"
	redactedAPIKey = "***"
)

// Client is a TMDB API v3 HTTP client.
// All outbound requests are logged. The API key is never logged.
// Client is safe for concurrent use.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	logger  *slog.Logger
}

// New creates a new TMDB client.
// apiKey must not be empty. It is stored and used in query parameters.
// logger is used to log outbound requests (the key value is replaced with "***" in logged URLs).
func New(apiKey string, logger *slog.Logger) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: httpTimeout},
		logger:  logger,
	}
}

// SearchMovies searches TMDB for movies matching query.
// If year is non-zero it is sent as the primary_release_year filter.
func (c *Client) SearchMovies(ctx context.Context, query string, year int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if year != 0 {
		params.Set("primary_release_year", strconv.Itoa(year))
	}

	var envelope struct {
		Results []struct {
			ID            int     `json:"id"`
			Title         string  `json:"title"`
			OriginalTitle string  `json:"original_title"`
			Overview      string  `json:"overview"`
			ReleaseDate   string  `json:"release_date"`
			PosterPath    string  `json:"poster_path"`
			BackdropPath  string  `json:"backdrop_path"`
			Popularity    float64 `json:"popularity"`
		} `json:"results"`
	}

	if err := c.get(ctx, "/search/movie", params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb search movies: %w", err)
	}

	results := make([]SearchResult, 0, len(envelope.Results))
	for _, r := range envelope.Results {
		results = append(results, SearchResult{
			ID:            r.ID,
			Title:         r.Title,
			OriginalTitle: r.OriginalTitle,
			Overview:      r.Overview,
			ReleaseDate:   r.ReleaseDate,
			Year:          parseYear(r.ReleaseDate),
			PosterPath:    r.PosterPath,
			BackdropPath:  r.BackdropPath,
			Popularity:    r.Popularity,
		})
	}

	return results, nil
}

// GetMovie fetches full movie details by TMDB ID.
func (c *Client) GetMovie(ctx context.Context, tmdbID int) (*MovieDetail, error) {
	var raw struct {
		ID            int    `json:"id"`
		IMDBId        string `json:"imdb_id"`
		Title         string `json:"title"`
		OriginalTitle string `json:"original_title"`
		Overview      string `json:"overview"`
		ReleaseDate   string `json:"release_date"`
		Runtime       int    `json:"runtime"`
		Genres        []struct {
			Name string `json:"name"`
		} `json:"genres"`
		PosterPath   string `json:"poster_path"`
		BackdropPath string `json:"backdrop_path"`
		Status       string `json:"status"`
	}

	path := fmt.Sprintf("/movie/%d", tmdbID)
	if err := c.get(ctx, path, nil, &raw); err != nil {
		return nil, fmt.Errorf("tmdb get movie %d: %w", tmdbID, err)
	}

	genres := make([]string, 0, len(raw.Genres))
	for _, g := range raw.Genres {
		genres = append(genres, g.Name)
	}

	return &MovieDetail{
		ID:             raw.ID,
		IMDBId:         raw.IMDBId,
		Title:          raw.Title,
		OriginalTitle:  raw.OriginalTitle,
		Overview:       raw.Overview,
		ReleaseDate:    raw.ReleaseDate,
		Year:           parseYear(raw.ReleaseDate),
		RuntimeMinutes: raw.Runtime,
		Genres:         genres,
		PosterPath:     raw.PosterPath,
		BackdropPath:   raw.BackdropPath,
		Status:         mapStatus(raw.Status),
	}, nil
}

// GetMovieExtended fetches movie details with credits and recommendations in
// a single API call using TMDB's append_to_response parameter.
func (c *Client) GetMovieExtended(ctx context.Context, tmdbID int) (*MovieDetail, error) {
	var raw struct {
		ID            int    `json:"id"`
		IMDBId        string `json:"imdb_id"`
		Title         string `json:"title"`
		OriginalTitle string `json:"original_title"`
		Overview      string `json:"overview"`
		ReleaseDate   string `json:"release_date"`
		Runtime       int    `json:"runtime"`
		Genres        []struct {
			Name string `json:"name"`
		} `json:"genres"`
		PosterPath   string `json:"poster_path"`
		BackdropPath string `json:"backdrop_path"`
		Status       string `json:"status"`
		Credits      struct {
			Cast []struct {
				ID          int    `json:"id"`
				Name        string `json:"name"`
				Character   string `json:"character"`
				ProfilePath string `json:"profile_path"`
				Order       int    `json:"order"`
			} `json:"cast"`
			Crew []struct {
				ID          int    `json:"id"`
				Name        string `json:"name"`
				Job         string `json:"job"`
				Department  string `json:"department"`
				ProfilePath string `json:"profile_path"`
			} `json:"crew"`
		} `json:"credits"`
		Recommendations struct {
			Results []struct {
				ID          int    `json:"id"`
				Title       string `json:"title"`
				ReleaseDate string `json:"release_date"`
				PosterPath  string `json:"poster_path"`
			} `json:"results"`
		} `json:"recommendations"`
	}

	params := url.Values{}
	params.Set("append_to_response", "credits,recommendations")

	path := fmt.Sprintf("/movie/%d", tmdbID)
	if err := c.get(ctx, path, params, &raw); err != nil {
		return nil, fmt.Errorf("tmdb get movie extended %d: %w", tmdbID, err)
	}

	genres := make([]string, 0, len(raw.Genres))
	for _, g := range raw.Genres {
		genres = append(genres, g.Name)
	}

	// Top 10 cast by billing order.
	maxCast := 10
	if len(raw.Credits.Cast) < maxCast {
		maxCast = len(raw.Credits.Cast)
	}
	cast := make([]CastMember, 0, maxCast)
	for _, c := range raw.Credits.Cast[:maxCast] {
		cast = append(cast, CastMember{
			ID:          c.ID,
			Name:        c.Name,
			Character:   c.Character,
			ProfilePath: c.ProfilePath,
			Order:       c.Order,
		})
	}

	// Key crew: Director, Screenplay/Writer, Original Music Composer.
	keyJobs := map[string]bool{
		"Director":                true,
		"Screenplay":              true,
		"Writer":                  true,
		"Original Music Composer": true,
	}
	crew := make([]CrewMember, 0)
	for _, c := range raw.Credits.Crew {
		if keyJobs[c.Job] {
			crew = append(crew, CrewMember{
				ID:          c.ID,
				Name:        c.Name,
				Job:         c.Job,
				Department:  c.Department,
				ProfilePath: c.ProfilePath,
			})
		}
	}

	// Up to 10 recommendations.
	maxRecs := 10
	if len(raw.Recommendations.Results) < maxRecs {
		maxRecs = len(raw.Recommendations.Results)
	}
	recs := make([]MovieRecommendation, 0, maxRecs)
	for _, r := range raw.Recommendations.Results[:maxRecs] {
		recs = append(recs, MovieRecommendation{
			TMDBID:     r.ID,
			Title:      r.Title,
			Year:       parseYear(r.ReleaseDate),
			PosterPath: r.PosterPath,
		})
	}

	return &MovieDetail{
		ID:              raw.ID,
		IMDBId:          raw.IMDBId,
		Title:           raw.Title,
		OriginalTitle:   raw.OriginalTitle,
		Overview:        raw.Overview,
		ReleaseDate:     raw.ReleaseDate,
		Year:            parseYear(raw.ReleaseDate),
		RuntimeMinutes:  raw.Runtime,
		Genres:          genres,
		PosterPath:      raw.PosterPath,
		BackdropPath:    raw.BackdropPath,
		Status:          mapStatus(raw.Status),
		Cast:            cast,
		Crew:            crew,
		Recommendations: recs,
	}, nil
}

// GetPerson fetches a TMDB person by ID.
func (c *Client) GetPerson(ctx context.Context, personID int) (*Person, error) {
	var raw struct {
		ID                 int    `json:"id"`
		Name               string `json:"name"`
		ProfilePath        string `json:"profile_path"`
		KnownForDepartment string `json:"known_for_department"`
	}
	path := fmt.Sprintf("/person/%d", personID)
	if err := c.get(ctx, path, nil, &raw); err != nil {
		return nil, fmt.Errorf("tmdb get person %d: %w", personID, err)
	}
	return &Person{
		ID:                 raw.ID,
		Name:               raw.Name,
		ProfilePath:        raw.ProfilePath,
		KnownForDepartment: raw.KnownForDepartment,
	}, nil
}

// GetPersonFilmography returns directed films (personType="director") or
// top-billed acted films (personType="actor") for the given TMDB person.
// For actors, only credits where billing order < 5 are returned.
func (c *Client) GetPersonFilmography(ctx context.Context, personID int, personType string) ([]FilmographyItem, error) {
	var envelope struct {
		Cast []struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			ReleaseDate string `json:"release_date"`
			PosterPath  string `json:"poster_path"`
			Order       int    `json:"order"`
		} `json:"cast"`
		Crew []struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			ReleaseDate string `json:"release_date"`
			PosterPath  string `json:"poster_path"`
			Job         string `json:"job"`
		} `json:"crew"`
	}
	path := fmt.Sprintf("/person/%d/movie_credits", personID)
	if err := c.get(ctx, path, nil, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb get person filmography %d: %w", personID, err)
	}

	var items []FilmographyItem
	if personType == "director" {
		for _, cr := range envelope.Crew {
			if cr.Job != "Director" {
				continue
			}
			items = append(items, FilmographyItem{
				TMDBID:     cr.ID,
				Title:      cr.Title,
				Year:       parseYear(cr.ReleaseDate),
				PosterPath: cr.PosterPath,
			})
		}
	} else {
		// actor: only top-5 billed credits
		for _, cr := range envelope.Cast {
			if cr.Order >= 5 {
				continue
			}
			items = append(items, FilmographyItem{
				TMDBID:     cr.ID,
				Title:      cr.Title,
				Year:       parseYear(cr.ReleaseDate),
				PosterPath: cr.PosterPath,
				Order:      cr.Order,
			})
		}
	}
	return items, nil
}

// SearchPeople searches TMDB for people by name.
func (c *Client) SearchPeople(ctx context.Context, query string) ([]PersonSearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	var envelope struct {
		Results []struct {
			ID                 int    `json:"id"`
			Name               string `json:"name"`
			ProfilePath        string `json:"profile_path"`
			KnownForDepartment string `json:"known_for_department"`
		} `json:"results"`
	}
	if err := c.get(ctx, "/search/person", params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb search people: %w", err)
	}
	results := make([]PersonSearchResult, 0, len(envelope.Results))
	for _, r := range envelope.Results {
		results = append(results, PersonSearchResult{
			ID:                 r.ID,
			Name:               r.Name,
			ProfilePath:        r.ProfilePath,
			KnownForDepartment: r.KnownForDepartment,
		})
	}
	return results, nil
}

// SearchFranchises searches TMDB for movie collections (franchises) by name.
func (c *Client) SearchFranchises(ctx context.Context, query string) ([]FranchiseSearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	var envelope struct {
		Results []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			PosterPath string `json:"poster_path"`
		} `json:"results"`
	}
	if err := c.get(ctx, "/search/collection", params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb search franchises: %w", err)
	}
	results := make([]FranchiseSearchResult, 0, len(envelope.Results))
	for _, r := range envelope.Results {
		results = append(results, FranchiseSearchResult{
			ID:         r.ID,
			Name:       r.Name,
			PosterPath: r.PosterPath,
		})
	}
	return results, nil
}

// GetFranchise fetches the full details of a TMDB movie collection by ID.
func (c *Client) GetFranchise(ctx context.Context, collectionID int) (*FranchiseDetail, error) {
	var raw struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Parts []struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			ReleaseDate string `json:"release_date"`
			PosterPath  string `json:"poster_path"`
		} `json:"parts"`
	}
	path := fmt.Sprintf("/collection/%d", collectionID)
	if err := c.get(ctx, path, nil, &raw); err != nil {
		return nil, fmt.Errorf("tmdb get franchise %d: %w", collectionID, err)
	}
	parts := make([]FilmographyItem, 0, len(raw.Parts))
	for _, p := range raw.Parts {
		parts = append(parts, FilmographyItem{
			TMDBID:     p.ID,
			Title:      p.Title,
			Year:       parseYear(p.ReleaseDate),
			PosterPath: p.PosterPath,
		})
	}
	return &FranchiseDetail{
		ID:    raw.ID,
		Name:  raw.Name,
		Parts: parts,
	}, nil
}

// GetPopularMovies fetches the TMDB popular movies list.
// page starts at 1. Returns up to 20 results per page.
func (c *Client) GetPopularMovies(ctx context.Context, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieList(ctx, "/movie/popular", params)
}

// GetUpcomingMovies fetches upcoming movies from TMDB.
// page starts at 1. Returns up to 20 results per page.
func (c *Client) GetUpcomingMovies(ctx context.Context, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieList(ctx, "/movie/upcoming", params)
}

// GetNowPlayingMovies fetches movies currently in theaters from TMDB.
// page starts at 1. Returns up to 20 results per page.
func (c *Client) GetNowPlayingMovies(ctx context.Context, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieList(ctx, "/movie/now_playing", params)
}

// GetTopRatedMovies fetches the all-time highest rated movies from TMDB.
// page starts at 1. Returns up to 20 results per page.
func (c *Client) GetTopRatedMovies(ctx context.Context, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieList(ctx, "/movie/top_rated", params)
}

// GetTrendingMovies fetches trending movies from TMDB.
// window must be "day" or "week". page starts at 1.
func (c *Client) GetTrendingMovies(ctx context.Context, window string, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	path := "/trending/movie/" + window
	return c.fetchMovieList(ctx, path, params)
}

// GetList fetches a user-created TMDB list by its numeric or string ID.
func (c *Client) GetList(ctx context.Context, listID string, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))

	var envelope struct {
		Items []struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			ReleaseDate string `json:"release_date"`
			PosterPath  string `json:"poster_path"`
		} `json:"items"`
	}

	path := "/list/" + listID
	if err := c.get(ctx, path, params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb get list %s: %w", listID, err)
	}

	results := make([]SearchResult, 0, len(envelope.Items))
	for _, r := range envelope.Items {
		results = append(results, SearchResult{
			ID:         r.ID,
			Title:      r.Title,
			Year:       parseYear(r.ReleaseDate),
			PosterPath: r.PosterPath,
		})
	}
	return results, nil
}

// FindByIMDbID looks up a movie by its IMDb ID using the /find endpoint.
// Returns the TMDB ID and title, or 0/"" if not found.
func (c *Client) FindByIMDbID(ctx context.Context, imdbID string) (int, string, error) {
	params := url.Values{}
	params.Set("external_source", "imdb_id")

	var envelope struct {
		MovieResults []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"movie_results"`
	}

	path := "/find/" + imdbID
	if err := c.get(ctx, path, params, &envelope); err != nil {
		return 0, "", fmt.Errorf("tmdb find by imdb %s: %w", imdbID, err)
	}

	if len(envelope.MovieResults) == 0 {
		return 0, "", nil
	}
	return envelope.MovieResults[0].ID, envelope.MovieResults[0].Title, nil
}

// GenreList fetches the list of official TMDB movie genres.
func (c *Client) GenreList(ctx context.Context) ([]Genre, error) {
	var envelope struct {
		Genres []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
	}
	if err := c.get(ctx, "/genre/movie/list", nil, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb genre list: %w", err)
	}
	genres := make([]Genre, 0, len(envelope.Genres))
	for _, g := range envelope.Genres {
		genres = append(genres, Genre{ID: g.ID, Name: g.Name})
	}
	return genres, nil
}

// DiscoverByGenre fetches movies filtered by genre ID.
func (c *Client) DiscoverByGenre(ctx context.Context, genreID, page int) (*PaginatedResults, error) {
	params := url.Values{}
	params.Set("with_genres", strconv.Itoa(genreID))
	params.Set("sort_by", "popularity.desc")
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieListPaginated(ctx, "/discover/movie", params)
}

// FetchTrending fetches trending movies with pagination metadata.
func (c *Client) FetchTrending(ctx context.Context, page int) (*PaginatedResults, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieListPaginated(ctx, "/trending/movie/week", params)
}

// FetchPopular fetches popular movies with pagination metadata.
func (c *Client) FetchPopular(ctx context.Context, page int) (*PaginatedResults, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieListPaginated(ctx, "/movie/popular", params)
}

// FetchTopRated fetches top-rated movies with pagination metadata.
func (c *Client) FetchTopRated(ctx context.Context, page int) (*PaginatedResults, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieListPaginated(ctx, "/movie/top_rated", params)
}

// FetchUpcoming fetches upcoming movies with pagination metadata.
func (c *Client) FetchUpcoming(ctx context.Context, page int) (*PaginatedResults, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieListPaginated(ctx, "/movie/upcoming", params)
}

// fetchMovieListPaginated is like fetchMovieList but also returns pagination metadata.
func (c *Client) fetchMovieListPaginated(ctx context.Context, path string, params url.Values) (*PaginatedResults, error) {
	var envelope struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
		Results    []struct {
			ID            int     `json:"id"`
			Title         string  `json:"title"`
			OriginalTitle string  `json:"original_title"`
			Overview      string  `json:"overview"`
			ReleaseDate   string  `json:"release_date"`
			PosterPath    string  `json:"poster_path"`
			BackdropPath  string  `json:"backdrop_path"`
			Popularity    float64 `json:"popularity"`
			VoteAverage   float64 `json:"vote_average"`
		} `json:"results"`
	}

	if err := c.get(ctx, path, params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb %s: %w", path, err)
	}

	results := make([]SearchResult, 0, len(envelope.Results))
	for _, r := range envelope.Results {
		results = append(results, SearchResult{
			ID:            r.ID,
			Title:         r.Title,
			OriginalTitle: r.OriginalTitle,
			Overview:      r.Overview,
			ReleaseDate:   r.ReleaseDate,
			Year:          parseYear(r.ReleaseDate),
			PosterPath:    r.PosterPath,
			BackdropPath:  r.BackdropPath,
			Popularity:    r.VoteAverage, // Use vote_average as "popularity" for discover
		})
	}
	return &PaginatedResults{
		Results:    results,
		Page:       envelope.Page,
		TotalPages: envelope.TotalPages,
	}, nil
}

// fetchMovieList is a shared helper for endpoints that return a paginated list
// of movies using the standard {results: [...]} envelope (popular, upcoming, etc.).
func (c *Client) fetchMovieList(ctx context.Context, path string, params url.Values) ([]SearchResult, error) {
	var envelope struct {
		Results []struct {
			ID            int     `json:"id"`
			Title         string  `json:"title"`
			OriginalTitle string  `json:"original_title"`
			Overview      string  `json:"overview"`
			ReleaseDate   string  `json:"release_date"`
			PosterPath    string  `json:"poster_path"`
			BackdropPath  string  `json:"backdrop_path"`
			Popularity    float64 `json:"popularity"`
		} `json:"results"`
	}

	if err := c.get(ctx, path, params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb %s: %w", path, err)
	}

	results := make([]SearchResult, 0, len(envelope.Results))
	for _, r := range envelope.Results {
		results = append(results, SearchResult{
			ID:            r.ID,
			Title:         r.Title,
			OriginalTitle: r.OriginalTitle,
			Overview:      r.Overview,
			ReleaseDate:   r.ReleaseDate,
			Year:          parseYear(r.ReleaseDate),
			PosterPath:    r.PosterPath,
			BackdropPath:  r.BackdropPath,
			Popularity:    r.Popularity,
		})
	}
	return results, nil
}

// get performs a GET against the TMDB API, decodes the JSON body into dst,
// and returns a structured error on non-200 responses.
func (c *Client) get(ctx context.Context, path string, params url.Values, dst any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", c.apiKey)

	rawURL := c.baseURL + path + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	// Log the URL with the API key redacted.
	c.logger.InfoContext(ctx, "tmdb request",
		slog.String("method", http.MethodGet),
		slog.String("url", redactAPIKey(rawURL, c.apiKey)),
	)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to extract the TMDB error message for context.
		var apiErr struct {
			StatusMessage string `json:"status_message"`
			StatusCode    int    `json:"status_code"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.StatusMessage != "" {
			return fmt.Errorf("http %d: %s", resp.StatusCode, apiErr.StatusMessage)
		}
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// parseYear extracts the four-digit year from a "YYYY-MM-DD" date string.
// Returns 0 if the string is empty or malformed.
func parseYear(date string) int {
	if date == "" {
		return 0
	}
	parts := strings.SplitN(date, "-", 2)
	y, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return y
}

// mapStatus converts a TMDB status string to our internal status vocabulary.
func mapStatus(tmdbStatus string) string {
	if tmdbStatus == "Released" {
		return "released"
	}
	return "announced"
}

// redactAPIKey replaces the api_key query parameter value in a URL string with "***".
// This is a best-effort operation; the original string is returned if parsing fails.
func redactAPIKey(rawURL, apiKey string) string {
	if apiKey == "" {
		return rawURL
	}
	return strings.ReplaceAll(rawURL, "api_key="+apiKey, "api_key="+redactedAPIKey)
}
