package v3

import (
	"crypto/subtle"
	"database/sql"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/core/queue"
	"github.com/luminarr/luminarr/internal/scheduler"
)

// Config holds the dependencies for the Radarr v3 compatibility layer.
type Config struct {
	DB             *sql.DB
	MovieService   *movie.Service
	QualityService *quality.Service
	LibraryService *library.Service
	QueueService   *queue.Service
	Scheduler      *scheduler.Scheduler
}

// Auth returns a huma middleware that authenticates requests using:
//  1. Sec-Fetch-Site: same-origin (browser)
//  2. X-Api-Key header
//  3. ?apikey= query parameter
func Auth(api huma.API, apiKey []byte) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		// Browser same-origin requests.
		if ctx.Header("Sec-Fetch-Site") == "same-origin" {
			next(ctx)
			return
		}
		// X-Api-Key header (standard Radarr auth).
		if subtle.ConstantTimeCompare([]byte(ctx.Header("X-Api-Key")), apiKey) == 1 {
			next(ctx)
			return
		}
		// ?apikey= query param (Homepage, Home Assistant style).
		if subtle.ConstantTimeCompare([]byte(ctx.Query("apikey")), apiKey) == 1 {
			next(ctx)
			return
		}
		_ = huma.WriteErr(api, ctx, 401, "A valid API key is required.")
	}
}

// RegisterRoutes registers all Radarr v3 compatible endpoints on the given
// huma API instance.
func RegisterRoutes(api huma.API, cfg Config) {
	registerSystemRoutes(api)
	registerMovieRoutes(api, cfg.DB, cfg.MovieService, cfg.LibraryService)
	registerQualityProfileRoutes(api, cfg.DB, cfg.QualityService)
	registerRootFolderRoutes(api, cfg.DB, cfg.LibraryService)
	registerTagRoutes(api)
	registerQueueRoutes(api, cfg.QueueService)
	registerCommandRoutes(api, cfg.DB, cfg.MovieService, cfg.Scheduler)
}
