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
