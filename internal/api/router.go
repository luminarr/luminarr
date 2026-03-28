package api

import (
	"crypto/subtle"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/luminarr/luminarr/internal/api/middleware"
	v1 "github.com/luminarr/luminarr/internal/api/v1"
	v3 "github.com/luminarr/luminarr/internal/api/v3"
	"github.com/luminarr/luminarr/internal/api/ws"
	"github.com/luminarr/luminarr/internal/config"
	"github.com/luminarr/luminarr/internal/core/activity"
	"github.com/luminarr/luminarr/internal/core/aicommand"
	"github.com/luminarr/luminarr/internal/core/autosearch"
	"github.com/luminarr/luminarr/internal/core/blocklist"
	"github.com/luminarr/luminarr/internal/core/collection"
	"github.com/luminarr/luminarr/internal/core/customformat"
	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/downloadhandling"
	"github.com/luminarr/luminarr/internal/core/health"
	"github.com/luminarr/luminarr/internal/core/importlist"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/mediainfo"
	"github.com/luminarr/luminarr/internal/core/mediamanagement"
	"github.com/luminarr/luminarr/internal/core/mediaserver"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/notification"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/core/queue"
	"github.com/luminarr/luminarr/internal/core/stats"
	"github.com/luminarr/luminarr/internal/core/tag"
	"github.com/luminarr/luminarr/internal/core/watchsync"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/logging"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
	"github.com/luminarr/luminarr/internal/plexsync"
	"github.com/luminarr/luminarr/internal/radarrimport"
	"github.com/luminarr/luminarr/internal/scheduler"
	"github.com/luminarr/luminarr/internal/version"
	"github.com/luminarr/luminarr/web"
)

// RouterConfig holds everything the router needs to function.
type RouterConfig struct {
	Auth                     config.Secret
	Logger                   *slog.Logger
	StartTime                time.Time
	DB                       *sql.DB
	DBType                   string
	DBPath                   string
	ConfigFile               string
	TMDBKeyIsDefault         bool
	QualityService           *quality.Service
	QualityDefinitionService *quality.DefinitionService
	LibraryService           *library.Service
	MovieService             *movie.Service
	TMDBClient               *tmdb.Client
	IndexerService           *indexer.Service
	DownloaderService        *downloader.Service
	BlocklistService         *blocklist.Service
	QueueService             *queue.Service
	Scheduler                *scheduler.Scheduler
	NotificationService      *notification.Service
	HealthService            *health.Service
	MediaManagementService   *mediamanagement.Service
	DownloadHandlingService  *downloadhandling.Service
	RadarrImportService      *radarrimport.Service
	StatsService             *stats.Service
	MediaInfoService         *mediainfo.Service
	CollectionService        *collection.Service
	MediaServerService       *mediaserver.Service
	PlexSyncService          *plexsync.Service
	TagService               *tag.Service
	CustomFormatService      *customformat.Service
	AutoSearchService        *autosearch.Service
	AICommandService         *aicommand.Service
	ActivityService          *activity.Service
	WatchSyncService         *watchsync.Service
	ImportListService        *importlist.Service
	LogBuffer                *logging.RingBuffer
	WSHub                    *ws.Hub
	Bus                      *events.Bus
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

	// WebSocket event stream — auth is handled inside the hub (Sec-Fetch-Site
	// for browsers, X-Api-Key for external clients). Must be registered on
	// the raw chi router before huma takes over so the huma auth middleware
	// does not intercept the upgrade request.
	if cfg.WSHub != nil {
		r.Get("/api/v1/ws", cfg.WSHub.ServeHTTP)
	}

	// Backup / restore — registered directly on chi (binary body/response, not JSON).
	// Auth accepts same-origin browser requests (Sec-Fetch-Site) or external API key.
	if cfg.DB != nil && cfg.DBPath != "" {
		authKey := []byte(cfg.Auth.Value())
		withAuth := func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Sec-Fetch-Site") == "same-origin" {
					next(w, r)
					return
				}
				if len(authKey) > 0 && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Api-Key")), authKey) == 1 {
					next(w, r)
					return
				}
				http.Error(w, `{"status":401,"title":"Unauthorized"}`, http.StatusUnauthorized)
			}
		}
		r.Get("/api/v1/system/backup", withAuth(v1.BackupHandler(cfg.DB, cfg.DBPath, cfg.Logger)))
		r.Post("/api/v1/system/restore", withAuth(v1.RestoreHandler(cfg.DBPath, cfg.Logger)))
	}

	// Unauthenticated health check for load balancers / container probes.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	humaConfig := huma.DefaultConfig("Luminarr API", version.Version)
	humaConfig.DocsPath = "/api/docs"
	humaConfig.OpenAPIPath = "/api/openapi"
	humaConfig.SchemasPath = "/api/schemas"
	humaConfig.Info.Description = "Luminarr movie collection manager API. " +
		"Browser requests are authenticated via Sec-Fetch-Site; external clients must provide X-Api-Key."

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

	// Auth middleware: same-origin browser requests (Sec-Fetch-Site header) are
	// trusted without a key (Radarr model). External consumers must provide a
	// valid X-Api-Key header.
	apiKeyBytes := []byte(cfg.Auth.Value())
	humaAPI.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		if ctx.Header("Sec-Fetch-Site") == "same-origin" {
			next(ctx)
			return
		}
		if len(apiKeyBytes) > 0 && subtle.ConstantTimeCompare([]byte(ctx.Header("X-Api-Key")), apiKeyBytes) == 1 {
			next(ctx)
			return
		}
		_ = huma.WriteErr(humaAPI, ctx, http.StatusUnauthorized, "A valid X-Api-Key header is required.")
	})

	v1.RegisterSystemRoutes(humaAPI, cfg.StartTime, cfg.DBType, cfg.DBPath, cfg.ConfigFile, cfg.AICommandService, cfg.TMDBKeyIsDefault, cfg.Auth.Value(), cfg.MovieService, cfg.Logger)

	if cfg.LogBuffer != nil {
		v1.RegisterLogRoutes(humaAPI, cfg.LogBuffer)
	}

	if cfg.QualityService != nil {
		v1.RegisterQualityProfileRoutes(humaAPI, cfg.QualityService)
	}

	if cfg.QualityDefinitionService != nil {
		v1.RegisterQualityDefinitionRoutes(humaAPI, cfg.QualityDefinitionService)
	}

	if cfg.LibraryService != nil {
		v1.RegisterLibraryRoutes(humaAPI, cfg.LibraryService, cfg.MovieService)
	}

	if cfg.MovieService != nil {
		v1.RegisterMovieRoutes(humaAPI, cfg.MovieService, cfg.TagService, cfg.WatchSyncService)
		v1.RegisterMovieFileRoutes(humaAPI, cfg.MovieService, cfg.MediaManagementService, cfg.MediaInfoService)
		v1.RegisterMovieCreditsRoutes(humaAPI, cfg.MovieService, cfg.TMDBClient)
		v1.RegisterDiscoverRoutes(humaAPI, cfg.MovieService, cfg.TMDBClient)
		v1.RegisterWantedRoutes(humaAPI, cfg.MovieService)
	}

	v1.RegisterMediainfoRoutes(humaAPI, cfg.MediaInfoService)

	if cfg.IndexerService != nil {
		v1.RegisterIndexerRoutes(humaAPI, cfg.IndexerService, cfg.TagService)

		autoSvc := cfg.AutoSearchService
		if autoSvc == nil && cfg.MovieService != nil && cfg.DownloaderService != nil && cfg.QualityService != nil {
			autoSvc = autosearch.NewService(
				cfg.IndexerService, cfg.MovieService, cfg.DownloaderService,
				cfg.BlocklistService, cfg.QualityService, cfg.CustomFormatService, cfg.TagService, cfg.Bus, cfg.Logger,
			)
		}
		v1.RegisterReleaseRoutes(humaAPI, cfg.IndexerService, cfg.MovieService, cfg.DownloaderService, cfg.BlocklistService, cfg.QualityService, autoSvc, cfg.Logger)

		v1.RegisterHistoryRoutes(humaAPI, cfg.IndexerService)
	}

	if cfg.BlocklistService != nil {
		v1.RegisterBlocklistRoutes(humaAPI, cfg.BlocklistService)
	}

	if cfg.DownloaderService != nil {
		v1.RegisterDownloadClientRoutes(humaAPI, cfg.DownloaderService, cfg.TagService)
	}

	if cfg.QueueService != nil {
		v1.RegisterQueueRoutes(humaAPI, cfg.QueueService, cfg.BlocklistService)
	}

	if cfg.Scheduler != nil {
		v1.RegisterTaskRoutes(humaAPI, cfg.Scheduler)
	}

	if cfg.NotificationService != nil {
		v1.RegisterNotificationRoutes(humaAPI, cfg.NotificationService, cfg.TagService)
	}

	if cfg.MediaServerService != nil {
		v1.RegisterMediaServerRoutes(humaAPI, cfg.MediaServerService)
	}

	if cfg.PlexSyncService != nil {
		v1.RegisterPlexSyncRoutes(humaAPI, cfg.PlexSyncService)
	}

	if cfg.HealthService != nil {
		v1.RegisterHealthRoutes(humaAPI, cfg.HealthService)
	}

	if cfg.MediaManagementService != nil {
		v1.RegisterMediaManagementRoutes(humaAPI, cfg.MediaManagementService)
	}

	if cfg.DownloadHandlingService != nil {
		v1.RegisterDownloadHandlingRoutes(humaAPI, cfg.DownloadHandlingService)
	}

	if cfg.RadarrImportService != nil {
		v1.RegisterImportRoutes(humaAPI, cfg.RadarrImportService)
	}

	if cfg.StatsService != nil {
		v1.RegisterStatsRoutes(humaAPI, cfg.StatsService)
	}

	v1.RegisterCollectionRoutes(humaAPI, cfg.CollectionService)

	if cfg.TagService != nil {
		v1.RegisterTagRoutes(humaAPI, cfg.TagService)
	}

	if cfg.CustomFormatService != nil {
		v1.RegisterCustomFormatRoutes(humaAPI, cfg.CustomFormatService)
	}

	if cfg.ImportListService != nil {
		v1.RegisterImportListRoutes(humaAPI, cfg.ImportListService, cfg.TagService)
	}

	v1.RegisterPlexAuthRoutes(humaAPI)

	if cfg.LibraryService != nil && cfg.MovieService != nil && cfg.Bus != nil && cfg.Scheduler != nil {
		v1.RegisterHookRoutes(humaAPI, cfg.LibraryService, cfg.MovieService, cfg.Bus, cfg.Scheduler)
	}

	v1.RegisterFilesystemRoutes(humaAPI)
	v1.RegisterParseRoutes(humaAPI)

	if cfg.AICommandService != nil {
		v1.RegisterAIRoutes(humaAPI, cfg.AICommandService)
	}

	if cfg.ActivityService != nil {
		v1.RegisterActivityRoutes(humaAPI, cfg.ActivityService)
	}

	if cfg.WatchSyncService != nil {
		v1.RegisterWatchSyncRoutes(humaAPI, cfg.WatchSyncService)
	}

	// ── Radarr v3 API compatibility layer ────────────────────────────────
	// External tools (Overseerr, Homepage, etc.) can point their "Radarr"
	// integration at Luminarr and it will just work.
	if cfg.DB != nil {
		v3Config := huma.DefaultConfig("Luminarr Radarr-Compatible API", version.Version)
		v3Config.DocsPath = ""
		v3Config.OpenAPIPath = ""
		v3Config.SchemasPath = ""
		v3API := humachi.New(r, v3Config)
		v3API.UseMiddleware(v3.Auth(v3API, apiKeyBytes))
		v3.RegisterRoutes(v3API, v3.Config{
			DB:             cfg.DB,
			MovieService:   cfg.MovieService,
			QualityService: cfg.QualityService,
			LibraryService: cfg.LibraryService,
			QueueService:   cfg.QueueService,
			TagService:     cfg.TagService,
			Scheduler:      cfg.Scheduler,
		})
	}

	// Serve the embedded React SPA. This handler serves static files when they
	// exist (assets, favicon, etc.) and falls back to index.html for all other
	// paths so React Router can handle client-side navigation. Must come after
	// all API routes so /api/* and /health take precedence.
	r.Handle("/*", web.ServeStatic())

	return r
}
