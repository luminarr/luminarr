package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davidfic/luminarr/internal/api"
	"github.com/davidfic/luminarr/internal/api/ws"
	"github.com/davidfic/luminarr/internal/config"
	"github.com/davidfic/luminarr/internal/core/downloader"
	"github.com/davidfic/luminarr/internal/core/health"
	"github.com/davidfic/luminarr/internal/core/importer"
	"github.com/davidfic/luminarr/internal/core/indexer"
	"github.com/davidfic/luminarr/internal/core/library"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/core/notification"
	"github.com/davidfic/luminarr/internal/core/quality"
	"github.com/davidfic/luminarr/internal/core/queue"
	"github.com/davidfic/luminarr/internal/db"
	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/logging"
	"github.com/davidfic/luminarr/internal/metadata/tmdb"
	"github.com/davidfic/luminarr/internal/notifications"
	"github.com/davidfic/luminarr/internal/radarrimport"
	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/internal/scheduler"
	"github.com/davidfic/luminarr/internal/scheduler/jobs"
	"github.com/davidfic/luminarr/internal/version"

	// Import pgx stdlib for database/sql compatibility (Postgres support).
	_ "github.com/jackc/pgx/v5/stdlib"

	// Blank-import built-in plugins so their init() functions register
	// them with the default registry before any service is constructed.
	_ "github.com/davidfic/luminarr/plugins/downloaders/deluge"
	_ "github.com/davidfic/luminarr/plugins/downloaders/qbittorrent"
	_ "github.com/davidfic/luminarr/plugins/indexers/newznab"
	_ "github.com/davidfic/luminarr/plugins/indexers/torznab"
	_ "github.com/davidfic/luminarr/plugins/notifications/discord"
	_ "github.com/davidfic/luminarr/plugins/notifications/email"
	_ "github.com/davidfic/luminarr/plugins/notifications/webhook"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "", "path to config file (default: ~/.config/luminarr/config.yaml)")
	flag.Parse()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// ── Logger ────────────────────────────────────────────────────────────────
	logger := logging.New(cfg.Log.Level, cfg.Log.Format)

	// Set the global slog default so packages using the top-level slog
	// functions (slog.Info, slog.Error, etc.) pick up the configured handler.
	slog.SetDefault(logger)

	// Advisory config file permission check — use the resolved path so the
	// warning fires whether the file was specified explicitly or found at the
	// default location (~/.config/luminarr/config.yaml).
	checkPath := cfg.ConfigFile
	if checkPath == "" {
		checkPath = cfgFile // may also be "" if no file was found at all
	}
	if checkPath != "" {
		if info, statErr := os.Stat(checkPath); statErr == nil {
			if info.Mode()&0o004 != 0 {
				logger.Warn("config file is world-readable — recommend chmod 600",
					"path", checkPath,
				)
			}
		}
	}

	// ── API Key ───────────────────────────────────────────────────────────────
	generated, err := config.EnsureAPIKey(cfg)
	if err != nil {
		return fmt.Errorf("ensuring API key: %w", err)
	}
	if generated {
		// Try to persist the key so it survives restarts. This works when
		// the config directory is writable (local installs, Docker with a
		// volume at /config). If it fails we log a warning and continue —
		// the key still works for this session but will change on restart.
		if _, persistErr := config.WriteConfigKey(cfg.ConfigFile, "auth.api_key", cfg.Auth.APIKey.Value()); persistErr != nil {
			// Print the key to stderr directly — it must be visible to the operator
			// so they can configure clients, but we do NOT put it in structured logs
			// (which are often shipped to log aggregators and retained long-term).
			fmt.Fprintf(os.Stderr, "\n  !! API key generated but could not be saved to disk.\n"+
				"  !! It will change on next restart. Set it now in your client:\n"+
				"  !!\n"+
				"  !!   API key: %s\n"+
				"  !!\n"+
				"  !! Hint: mount a writable volume at /config (Docker) or ensure\n"+
				"  !!        ~/.config/luminarr/ is writable.\n\n",
				cfg.Auth.APIKey.Value())
			logger.Warn("API key generated but could not be persisted — it will change on next restart",
				"hint", "mount a writable volume at /config (Docker) or ensure ~/.config/luminarr/ is writable",
				"error", persistErr,
			)
		} else {
			logger.Info("API key generated and saved to config — stable across restarts")
		}
	} else {
		key := cfg.Auth.APIKey.Value()
		masked := key
		if len(key) > 4 {
			masked = key[:4] + "****"
		}
		logger.Info("API key loaded", "key_prefix", masked, "source", "config/env")
	}

	// ── Startup banner ────────────────────────────────────────────────────────
	configFile := cfg.ConfigFile
	if configFile == "" {
		configFile = "(none — using defaults/env)"
	}
	logger.Info("Luminarr starting",
		"version", version.Version,
		"build_time", version.BuildTime,
		"go", version.GoVersion(),
		"db", cfg.Database.Driver,
		"config_file", configFile,
	)

	// Log registered plugins so operators can confirm which plugins are active.
	for _, kind := range registry.Default.IndexerKinds() {
		logger.Info("registered indexer plugin", "plugin", kind)
	}
	for _, kind := range registry.Default.DownloaderKinds() {
		logger.Info("registered downloader plugin", "plugin", kind)
	}
	for _, kind := range registry.Default.NotifierKinds() {
		logger.Info("registered notifier plugin", "plugin", kind)
	}

	// ── Feature warnings ──────────────────────────────────────────────────────
	if cfg.TMDB.APIKey.IsEmpty() {
		logger.Warn("TMDB API key not configured — movie metadata and search features are disabled",
			"hint", "set tmdb.api_key in config.yaml or LUMINARR_TMDB_API_KEY env var",
		)
	}
	if cfg.AI.APIKey.IsEmpty() {
		logger.Info("Claude API key not configured — AI features disabled, using rule-based fallbacks")
	}

	// ── Database ──────────────────────────────────────────────────────────────
	database, err := db.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	if database.Driver == "sqlite" {
		logger.Info("database connected", "driver", database.Driver, "path", cfg.Database.Path)
	} else {
		logger.Info("database connected", "driver", database.Driver)
	}

	if err := db.Migrate(database.SQL, database.Driver); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	logger.Info("database migrations up to date")

	// ── Event bus ─────────────────────────────────────────────────────────────
	bus := events.New(logger)

	// ── WebSocket hub ─────────────────────────────────────────────────────────
	wsHub := ws.NewHub(cfg.Auth.APIKey.Value(), logger)
	bus.Subscribe(wsHub.HandleEvent)

	// ── Services ──────────────────────────────────────────────────────────────
	queries := dbsqlite.New(database.SQL)

	qualitySvc := quality.NewService(queries, bus)
	librarySvc := library.NewService(queries, bus)

	var tmdbClient movie.MetadataProvider
	if !cfg.TMDB.APIKey.IsEmpty() {
		tmdbClient = tmdb.New(cfg.TMDB.APIKey.Value(), logger)
	}

	movieSvc := movie.NewService(queries, tmdbClient, bus, logger)
	indexerSvc := indexer.NewService(queries, registry.Default, bus)
	downloaderSvc := downloader.NewService(queries, registry.Default, bus)
	queueSvc := queue.NewService(queries, downloaderSvc, bus, logger)

	importerSvc := importer.NewService(queries, bus, logger)
	importerSvc.Subscribe()

	notifSvc := notification.NewService(queries, registry.Default)
	notifDispatcher := notifications.NewDispatcher(queries, registry.Default, bus, logger)
	notifDispatcher.Subscribe()

	healthSvc := health.NewService(librarySvc, downloaderSvc, indexerSvc, logger)

	radarrImportSvc := radarrimport.NewService(movieSvc, qualitySvc, librarySvc, indexerSvc, downloaderSvc)

	// ── Scheduler ─────────────────────────────────────────────────────────────
	sched := scheduler.New(logger)
	sched.Add(jobs.QueuePoll(queueSvc, logger))
	sched.Add(jobs.LibraryScan(librarySvc, logger))
	sched.Add(jobs.RSSSync(indexerSvc, downloaderSvc, qualitySvc, queries, logger))
	sched.Add(jobs.RefreshMetadata(movieSvc, queries, logger))

	// ── HTTP router ───────────────────────────────────────────────────────────
	startTime := time.Now()
	router := api.NewRouter(api.RouterConfig{
		Auth:                cfg.Auth.APIKey,
		Logger:              logger,
		StartTime:           startTime,
		DBType:              database.Driver,
		DBPath:              cfg.Database.Path,
		ConfigFile:          cfg.ConfigFile,
		AIEnabled:           !cfg.AI.APIKey.IsEmpty(),
		QualityService:      qualitySvc,
		LibraryService:      librarySvc,
		MovieService:        movieSvc,
		IndexerService:      indexerSvc,
		DownloaderService:   downloaderSvc,
		QueueService:        queueSvc,
		Scheduler:           sched,
		NotificationService: notifSvc,
		HealthService:       healthSvc,
		RadarrImportService: radarrImportSvc,
		WSHub:               wsHub,
	})

	// ── HTTP server ───────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ── Start background services ─────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx)

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig)
	}

	// Cancel scheduler and in-flight background jobs.
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped cleanly")
	return nil
}
