package api

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/davidfic/luminarr/internal/api/middleware"
	v1 "github.com/davidfic/luminarr/internal/api/v1"
	"github.com/davidfic/luminarr/internal/config"
	"github.com/davidfic/luminarr/internal/core/downloader"
	"github.com/davidfic/luminarr/internal/core/health"
	"github.com/davidfic/luminarr/internal/core/indexer"
	"github.com/davidfic/luminarr/internal/core/library"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/core/notification"
	"github.com/davidfic/luminarr/internal/core/quality"
	"github.com/davidfic/luminarr/internal/core/queue"
	"github.com/davidfic/luminarr/internal/radarrimport"
	"github.com/davidfic/luminarr/internal/scheduler"
	"github.com/davidfic/luminarr/internal/version"
	"github.com/davidfic/luminarr/web"
)

// RouterConfig holds everything the router needs to function.
type RouterConfig struct {
	Auth                config.Secret
	Logger              *slog.Logger
	StartTime           time.Time
	DBType              string
	DBPath              string
	ConfigFile          string
	AIEnabled           bool
	QualityService      *quality.Service
	LibraryService      *library.Service
	MovieService        *movie.Service
	IndexerService      *indexer.Service
	DownloaderService   *downloader.Service
	QueueService        *queue.Service
	Scheduler           *scheduler.Scheduler
	NotificationService *notification.Service
	HealthService       *health.Service
	RadarrImportService *radarrimport.Service
}

// NewRouter builds and returns the application HTTP handler.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware — applied to every request including /health.
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.MaxRequestBodySize(1 << 20)) // 1 MiB max request body
	r.Use(middleware.RequestLogger(cfg.Logger))
	r.Use(middleware.Recovery(cfg.Logger))

	// Unauthenticated health check for load balancers / container probes.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Docs and OpenAPI spec are disabled — they expose the full API surface
	// without authentication. Use the Go source or run locally with a client
	// that supports OpenAPI to explore the API.
	humaConfig := huma.DefaultConfig("Luminarr API", version.Version)
	humaConfig.DocsPath = ""
	humaConfig.Info.Description = "Luminarr movie collection manager API. " +
		"All endpoints require the X-Api-Key header."

	humaAPI := humachi.New(r, humaConfig)

	// Register X-Api-Key security scheme so the docs UI shows an Authorize
	// button and Try-it-out requests include the header automatically.
	oapi := humaAPI.OpenAPI()
	if oapi.Components == nil {
		oapi.Components = &huma.Components{}
	}
	if oapi.Components.SecuritySchemes == nil {
		oapi.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	oapi.Components.SecuritySchemes["ApiKeyAuth"] = &huma.SecurityScheme{
		Type: "apiKey",
		In:   "header",
		Name: "X-Api-Key",
	}
	oapi.Security = []map[string][]string{{"ApiKeyAuth": {}}}

	// Auth is enforced via huma middleware, which runs only for registered
	// operations — huma's own docs/spec routes are served directly on the chi
	// router and are therefore unaffected.
	apiKeyBytes := []byte(cfg.Auth.Value())
	humaAPI.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		if subtle.ConstantTimeCompare([]byte(ctx.Header("X-Api-Key")), apiKeyBytes) != 1 {
			_ = huma.WriteErr(humaAPI, ctx, http.StatusUnauthorized, "A valid X-Api-Key header is required.")
			return
		}
		next(ctx)
	})

	v1.RegisterSystemRoutes(humaAPI, cfg.StartTime, cfg.DBType, cfg.DBPath, cfg.ConfigFile, cfg.AIEnabled, cfg.MovieService, cfg.Logger)

	if cfg.QualityService != nil {
		v1.RegisterQualityProfileRoutes(humaAPI, cfg.QualityService)
	}

	if cfg.LibraryService != nil {
		v1.RegisterLibraryRoutes(humaAPI, cfg.LibraryService)
	}

	if cfg.MovieService != nil {
		v1.RegisterMovieRoutes(humaAPI, cfg.MovieService)
	}

	if cfg.IndexerService != nil {
		v1.RegisterIndexerRoutes(humaAPI, cfg.IndexerService)
		v1.RegisterReleaseRoutes(humaAPI, cfg.IndexerService, cfg.MovieService, cfg.DownloaderService, cfg.Logger)
	}

	if cfg.DownloaderService != nil {
		v1.RegisterDownloadClientRoutes(humaAPI, cfg.DownloaderService)
	}

	if cfg.QueueService != nil {
		v1.RegisterQueueRoutes(humaAPI, cfg.QueueService)
	}

	if cfg.Scheduler != nil {
		v1.RegisterTaskRoutes(humaAPI, cfg.Scheduler)
	}

	if cfg.NotificationService != nil {
		v1.RegisterNotificationRoutes(humaAPI, cfg.NotificationService)
	}

	if cfg.HealthService != nil {
		v1.RegisterHealthRoutes(humaAPI, cfg.HealthService)
	}

	if cfg.RadarrImportService != nil {
		v1.RegisterImportRoutes(humaAPI, cfg.RadarrImportService)
	}

	// Serve the embedded React SPA. This handler serves static files when they
	// exist (assets, favicon, etc.) and falls back to index.html for all other
	// paths so React Router can handle client-side navigation. Must come after
	// all API routes so /api/* and /health take precedence.
	r.Handle("/*", web.ServeStatic(cfg.Auth.Value()))

	return r
}
