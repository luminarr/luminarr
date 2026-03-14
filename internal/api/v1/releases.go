package v1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/autosearch"
	"github.com/luminarr/luminarr/internal/core/blocklist"
	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ── Request / response shapes ────────────────────────────────────────────────

type releaseBody struct {
	GUID           string                `json:"guid"`
	Title          string                `json:"title"`
	Indexer        string                `json:"indexer"`
	Protocol       string                `json:"protocol"`
	DownloadURL    string                `json:"download_url"`
	InfoURL        string                `json:"info_url,omitempty"`
	Size           int64                 `json:"size"`
	Seeds          int                   `json:"seeds,omitempty"`
	Peers          int                   `json:"peers,omitempty"`
	AgeDays        float64               `json:"age_days,omitempty"`
	Quality        plugin.Quality        `json:"quality"`
	Edition        string                `json:"edition,omitempty"      doc:"Detected edition (e.g. Director's Cut, Extended)"`
	QualityScore   int                   `json:"quality_score"`
	ScoreBreakdown plugin.ScoreBreakdown `json:"score_breakdown"`
}

type releaseListOutput struct {
	Body []*releaseBody
}

type releasesSearchInput struct {
	MovieID string `path:"id"`
}

// grabHistoryBody is a summary of one recorded grab.
type grabHistoryBody struct {
	ID               string          `json:"id"`
	MovieID          string          `json:"movie_id"`
	IndexerID        *string         `json:"indexer_id,omitempty"`
	ReleaseGUID      string          `json:"release_guid"`
	ReleaseTitle     string          `json:"release_title"`
	Protocol         string          `json:"protocol"`
	Size             int64           `json:"size"`
	DownloadClientID *string         `json:"download_client_id,omitempty"`
	ClientItemID     *string         `json:"client_item_id,omitempty"`
	DownloadStatus   string          `json:"download_status"`
	GrabbedAt        time.Time       `json:"grabbed_at"`
	ScoreBreakdown   json.RawMessage `json:"score_breakdown,omitempty"`
}

// grabInput carries the release the client wants to grab.
type grabInput struct {
	MovieID string `path:"id"`
	Body    grabReleaseBody
}

type grabReleaseBody struct {
	GUID        string          `json:"guid"`
	Title       string          `json:"title"`
	IndexerID   string          `json:"indexer_id,omitempty"`
	Protocol    string          `json:"protocol"`
	DownloadURL string          `json:"download_url"`
	Size        int64           `json:"size"`
	Quality     json.RawMessage `json:"quality,omitempty"`
}

type grabOutput struct {
	Body *grabHistoryBody
}

// ── Auto-search shapes ──────────────────────────────────────────────────────

type autoSearchInput struct {
	MovieID string `path:"id"`
}

type autoSearchOutput struct {
	Body *autosearch.Result
}

type bulkSearchInput struct {
	Body struct {
		MovieIDs []string `json:"movie_ids" minItems:"1" maxItems:"100" doc:"Movie UUIDs to search (max 100)"`
	}
}

type bulkSearchAcceptedBody struct {
	Message string `json:"message"`
	Total   int    `json:"total"`
}

type bulkSearchAcceptedOutput struct {
	Body bulkSearchAcceptedBody
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func indexerResultToBody(r indexer.SearchResult) *releaseBody {
	return &releaseBody{
		GUID:           r.GUID,
		Title:          r.Title,
		Indexer:        r.Indexer,
		Protocol:       string(r.Protocol),
		DownloadURL:    r.DownloadURL,
		InfoURL:        r.InfoURL,
		Size:           r.Size,
		Seeds:          r.Seeds,
		Peers:          r.Peers,
		AgeDays:        r.AgeDays,
		Quality:        r.Quality,
		Edition:        r.Edition,
		QualityScore:   r.QualityScore,
		ScoreBreakdown: r.ScoreBreakdown,
	}
}

// ── Route registration ───────────────────────────────────────────────────────

// RegisterReleaseRoutes registers the release search, grab, and auto-search endpoints.
// downloaderSvc may be nil; in that case grabs are recorded to history without
// being sent to a download client (backward-compatible with Phase 2 mode).
// blocklistSvc may be nil; in that case blocklist checking is skipped.
// qualitySvc may be nil; in that case score breakdowns are omitted from responses.
// autoSearchSvc may be nil; in that case auto-search endpoints are not registered.
func RegisterReleaseRoutes(api huma.API, indexerSvc *indexer.Service, movieSvc *movie.Service, downloaderSvc *downloader.Service, blocklistSvc *blocklist.Service, qualitySvc *quality.Service, autoSearchSvc *autosearch.Service, logger *slog.Logger) {
	// GET /api/v1/movies/{id}/releases
	huma.Register(api, huma.Operation{
		OperationID: "search-releases",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/releases",
		Summary:     "Search for releases for a movie across all enabled indexers",
		Tags:        []string{"Releases"},
	}, func(ctx context.Context, input *releasesSearchInput) (*releaseListOutput, error) {
		m, err := movieSvc.Get(ctx, input.MovieID)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get movie", err)
		}

		query := plugin.SearchQuery{
			TMDBID: m.TMDBID,
			IMDBID: m.IMDBID,
			Query:  m.Title,
			Year:   m.Year,
		}

		results, searchErr := indexerSvc.Search(ctx, query, nil)

		// Load quality profile once so we can compute breakdown per release.
		var prof *quality.Profile
		if qualitySvc != nil {
			if p, err := qualitySvc.Get(ctx, m.QualityProfileID); err == nil {
				prof = &p
			}
		}

		bodies := make([]*releaseBody, len(results))
		for i, r := range results {
			if prof != nil {
				r.QualityScore, r.ScoreBreakdown = prof.ScoreWithBreakdown(r.Quality)
			}
			bodies[i] = indexerResultToBody(r)
		}

		if len(bodies) == 0 && searchErr != nil {
			return nil, huma.NewError(http.StatusBadGateway, searchErr.Error())
		}

		return &releaseListOutput{Body: bodies}, nil
	})

	// POST /api/v1/movies/{id}/releases/grab
	huma.Register(api, huma.Operation{
		OperationID:   "grab-release",
		Method:        http.MethodPost,
		Path:          "/api/v1/movies/{id}/releases/grab",
		Summary:       "Grab a release — submits to a download client and records history",
		Tags:          []string{"Releases"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *grabInput) (*grabOutput, error) {
		mov, err := movieSvc.Get(ctx, input.MovieID)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get movie", err)
		}

		var qual plugin.Quality
		if len(input.Body.Quality) > 0 {
			_ = json.Unmarshal(input.Body.Quality, &qual)
		}

		release := plugin.Release{
			GUID:        input.Body.GUID,
			Title:       input.Body.Title,
			Protocol:    plugin.Protocol(input.Body.Protocol),
			DownloadURL: input.Body.DownloadURL,
			Size:        input.Body.Size,
			Quality:     qual,
		}

		// Reject releases that are on the blocklist.
		if blocklistSvc != nil {
			blocked, blErr := blocklistSvc.IsBlocklisted(ctx, input.Body.GUID)
			if blErr != nil {
				logger.Warn("grab: blocklist check failed", "guid", input.Body.GUID, "error", blErr)
			} else if blocked {
				return nil, huma.NewError(http.StatusConflict, "release is blocklisted")
			}
		}

		// Submit to download client when one is configured.
		var dcID, itemID string
		if downloaderSvc != nil {
			id, item, err := downloaderSvc.Add(ctx, release, nil)
			if err != nil {
				if errors.Is(err, downloader.ErrNoCompatibleClient) {
					return nil, huma.NewError(http.StatusServiceUnavailable,
						"no download client configured for this protocol — add one at /api/v1/download-clients", err)
				}
				// Auto-blocklist releases that the download client rejects.
				if blocklistSvc != nil {
					blErr := blocklistSvc.Add(ctx, input.MovieID, input.Body.GUID, input.Body.Title,
						input.Body.IndexerID, input.Body.Protocol, input.Body.Size, "grab failed")
					if blErr != nil && !errors.Is(blErr, blocklist.ErrAlreadyBlocklisted) {
						logger.Warn("grab: failed to auto-blocklist rejected release",
							"guid", input.Body.GUID, "error", blErr)
					}
				}
				return nil, huma.NewError(http.StatusBadGateway, "download client: "+err.Error())
			}
			dcID = id
			itemID = item
		}

		// Compute score breakdown against the movie's quality profile.
		var breakdownJSON string
		if qualitySvc != nil {
			if p, pErr := qualitySvc.Get(ctx, mov.QualityProfileID); pErr == nil {
				_, bd := p.ScoreWithBreakdown(release.Quality)
				if b, bErr := json.Marshal(bd); bErr == nil {
					breakdownJSON = string(b)
				}
			}
		}

		history, err := indexerSvc.Grab(ctx, input.MovieID, input.Body.IndexerID, release, dcID, itemID, breakdownJSON)
		if err != nil {
			logger.Error("grab: failed to record grab history",
				"movie_id", input.MovieID,
				"indexer_id", input.Body.IndexerID,
				"dc_id", dcID,
				"item_id", itemID,
				"error", err,
			)
			return nil, huma.NewError(http.StatusInternalServerError, "failed to record grab: "+err.Error())
		}

		grabbedAt, _ := time.Parse(time.RFC3339, history.GrabbedAt)
		out := &grabHistoryBody{
			ID:               history.ID,
			MovieID:          history.MovieID,
			IndexerID:        history.IndexerID,
			ReleaseGUID:      history.ReleaseGuid,
			ReleaseTitle:     history.ReleaseTitle,
			Protocol:         history.Protocol,
			Size:             history.Size,
			DownloadClientID: history.DownloadClientID,
			ClientItemID:     history.ClientItemID,
			DownloadStatus:   history.DownloadStatus,
			GrabbedAt:        grabbedAt,
		}
		if history.ScoreBreakdown != "" {
			out.ScoreBreakdown = json.RawMessage(history.ScoreBreakdown)
		}
		return &grabOutput{Body: out}, nil
	})

	if autoSearchSvc == nil {
		return
	}

	// POST /api/v1/movies/{id}/search — single-movie automatic search.
	huma.Register(api, huma.Operation{
		OperationID: "auto-search-movie",
		Method:      http.MethodPost,
		Path:        "/api/v1/movies/{id}/search",
		Summary:     "Automatic search — find the best release and grab it",
		Description: "Searches all indexers, picks the best release that satisfies the movie's quality profile, and submits it to a download client. Works on both monitored and unmonitored movies.",
		Tags:        []string{"Releases"},
	}, func(ctx context.Context, input *autoSearchInput) (*autoSearchOutput, error) {
		result, err := autoSearchSvc.SearchMovie(ctx, input.MovieID)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			errMsg := err.Error()
			switch {
			case strings.Contains(errMsg, "no download client configured"):
				return nil, huma.NewError(http.StatusServiceUnavailable, errMsg)
			case strings.Contains(errMsg, "all indexers failed"):
				return nil, huma.NewError(http.StatusBadGateway, errMsg)
			default:
				return nil, huma.NewError(http.StatusInternalServerError, errMsg)
			}
		}
		return &autoSearchOutput{Body: result}, nil
	})

	// POST /api/v1/movies/search — bulk automatic search (async).
	huma.Register(api, huma.Operation{
		OperationID:   "bulk-auto-search",
		Method:        http.MethodPost,
		Path:          "/api/v1/movies/search",
		Summary:       "Bulk automatic search — search and grab for multiple movies",
		Description:   "Accepts up to 100 movie IDs, returns 202 immediately, and processes searches asynchronously. Progress is pushed via WebSocket events (bulk_search_progress, bulk_search_complete).",
		Tags:          []string{"Releases"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, input *bulkSearchInput) (*bulkSearchAcceptedOutput, error) {
		ids := input.Body.MovieIDs
		if len(ids) > autosearch.MaxBulkMovies {
			return nil, huma.NewError(http.StatusBadRequest,
				"too many movie IDs (max 100)")
		}

		// Run async — detach from the HTTP request context.
		go autoSearchSvc.SearchMovies(context.WithoutCancel(ctx), ids)

		return &bulkSearchAcceptedOutput{Body: bulkSearchAcceptedBody{
			Message: "bulk search started",
			Total:   len(ids),
		}}, nil
	})
}
