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

	"github.com/luminarr/luminarr/internal/api"
	"github.com/luminarr/luminarr/internal/api/ws"
	"github.com/luminarr/luminarr/internal/config"
	"github.com/luminarr/luminarr/internal/core/blocklist"
	"github.com/luminarr/luminarr/internal/core/collection"
	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/downloadhandling"
	"github.com/luminarr/luminarr/internal/core/health"
	"github.com/luminarr/luminarr/internal/core/importer"
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
	"github.com/luminarr/luminarr/internal/db"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/logging"
	"github.com/luminarr/luminarr/internal/mediaservers"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
	"github.com/luminarr/luminarr/internal/notifications"
	"github.com/luminarr/luminarr/internal/plexsync"
	"github.com/luminarr/luminarr/internal/radarrimport"
	"github.com/luminarr/luminarr/internal/ratelimit"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/scheduler"
	"github.com/luminarr/luminarr/internal/scheduler/jobs"
	"github.com/luminarr/luminarr/internal/version"

	// Import pgx stdlib for database/sql compatibility (Postgres support).
	_ "github.com/jackc/pgx/v5/stdlib"

	// Blank-import built-in plugins so their init() functions register
	// them with the default registry before any service is constructed.
	_ "github.com/luminarr/luminarr/plugins/downloaders/deluge"
	_ "github.com/luminarr/luminarr/plugins/downloaders/nzbget"
	_ "github.com/luminarr/luminarr/plugins/downloaders/qbittorrent"
	_ "github.com/luminarr/luminarr/plugins/downloaders/sabnzbd"
	_ "github.com/luminarr/luminarr/plugins/downloaders/transmission"
	_ "github.com/luminarr/luminarr/plugins/indexers/newznab"
	_ "github.com/luminarr/luminarr/plugins/indexers/torznab"
	_ "github.com/luminarr/luminarr/plugins/mediaservers/emby"
	_ "github.com/luminarr/luminarr/plugins/mediaservers/jellyfin"
	_ "github.com/luminarr/luminarr/plugins/mediaservers/plex"
	_ "github.com/luminarr/luminarr/plugins/notifications/command"
	_ "github.com/luminarr/luminarr/plugins/notifications/discord"
	_ "github.com/luminarr/luminarr/plugins/notifications/email"
	_ "github.com/luminarr/luminarr/plugins/notifications/gotify"
	_ "github.com/luminarr/luminarr/plugins/notifications/ntfy"
	_ "github.com/luminarr/luminarr/plugins/notifications/pushover"
	_ "github.com/luminarr/luminarr/plugins/notifications/slack"
	_ "github.com/luminarr/luminarr/plugins/notifications/telegram"
	_ "github.com/luminarr/luminarr/plugins/notifications/webhook"
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
	for _, kind := range registry.Default.MediaServerKinds() {
		logger.Info("registered media server plugin", "plugin", kind)
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

	// ── Database restore staging check ───────────────────────────────────────
	// If a restore file exists (written by the /api/v1/system/restore endpoint),
	// swap it in before opening the database.
	if cfg.Database.Path != "" {
		stagingPath := cfg.Database.Path + ".restore"
		if _, statErr := os.Stat(stagingPath); statErr == nil {
			if renameErr := os.Rename(stagingPath, cfg.Database.Path); renameErr == nil {
				logger.Info("database restored from backup")
			} else {
				logger.Warn("failed to swap restore file into place", "error", renameErr)
			}
		}
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
	wsHub := ws.NewHub(logger, []byte(cfg.Auth.APIKey.Value()))
	bus.Subscribe(wsHub.HandleEvent)

	// ── Services ──────────────────────────────────────────────────────────────
	queries := dbsqlite.New(database.SQL)

	qualitySvc := quality.NewService(queries, bus)
	qualityDefSvc := quality.NewDefinitionService(queries)

	var rawTMDB *tmdb.Client
	if !cfg.TMDB.APIKey.IsEmpty() {
		rawTMDB = tmdb.New(cfg.TMDB.APIKey.Value(), logger)
	}
	// tmdbClient is the interface used by movie and library services.
	// Declared separately to keep the nil-interface semantics correct.
	var tmdbClient movie.MetadataProvider
	if rawTMDB != nil {
		tmdbClient = rawTMDB
	}

	librarySvc := library.NewService(queries, bus, tmdbClient)
	movieSvc := movie.NewService(queries, tmdbClient, bus, logger)
	blocklistSvc := blocklist.NewService(queries)
	indexerRL := ratelimit.New()
	indexerSvc := indexer.NewService(queries, registry.Default, bus, indexerRL)
	downloaderSvc := downloader.NewService(queries, registry.Default, bus)
	queueSvc := queue.NewService(queries, downloaderSvc, bus, logger)

	mmSvc := mediamanagement.NewService(queries)
	dhSvc := downloadhandling.NewService(queries)

	// ── MediaInfo scanner ──────────────────────────────────────────────────
	// Resolve scan_timeout; fall back to 30s if unparseable.
	scanTimeout := 30 * time.Second
	if cfg.MediaInfo.ScanTimeout != "" {
		if d, err := time.ParseDuration(cfg.MediaInfo.ScanTimeout); err == nil {
			scanTimeout = d
		}
	}
	mediainfoScanner := mediainfo.New(cfg.MediaInfo.FFprobePath, scanTimeout)
	if mediainfoScanner.Available() {
		logger.Info("mediainfo scanner available", "ffprobe", mediainfoScanner.FFprobePath())
	} else {
		logger.Info("mediainfo scanner unavailable — ffprobe not found; set mediainfo.ffprobe_path in config or install ffprobe")
	}
	mediainfoSvc := mediainfo.NewService(mediainfoScanner, queries, logger)

	importerSvc := importer.NewService(queries, bus, logger, mmSvc, dhSvc, mediainfoSvc)
	importerSvc.Subscribe()

	notifSvc := notification.NewService(queries, registry.Default)
	notifDispatcher := notifications.NewDispatcher(queries, registry.Default, bus, logger, movieSvc)
	notifDispatcher.Subscribe()

	mediaServerSvc := mediaserver.NewService(queries, registry.Default)
	msDispatcher := mediaservers.NewDispatcher(queries, registry.Default, bus, logger)
	msDispatcher.Subscribe()

	plexSyncSvc := plexsync.NewService(mediaServerSvc, movieSvc, queries)

	healthSvc := health.NewService(librarySvc, downloaderSvc, indexerSvc, logger)

	radarrImportSvc := radarrimport.NewService(movieSvc, qualitySvc, librarySvc, indexerSvc, downloaderSvc)
	statsSvc := stats.NewService(queries, movieSvc)

	var collectionSvc *collection.Service
	if rawTMDB != nil {
		collectionSvc = collection.NewService(queries, rawTMDB, movieSvc, logger)
	}

	// ── Scheduler ─────────────────────────────────────────────────────────────
	// Load queue poll interval from download handling settings. Default to 60s
	// on error so the scheduler always starts.
	queuePollInterval, err := dhSvc.CheckInterval(context.Background())
	if err != nil {
		logger.Warn("failed to load download handling interval, using 60s default", "error", err)
		queuePollInterval = 60 * time.Second
	}

	sched := scheduler.New(logger)
	sched.Add(jobs.QueuePoll(queueSvc, queuePollInterval, logger))
	sched.Add(jobs.LibraryScan(librarySvc, logger))
	sched.Add(jobs.RSSSync(indexerSvc, downloaderSvc, qualitySvc, queries, logger))
	sched.Add(jobs.RefreshMetadata(movieSvc, queries, logger))
	sched.Add(jobs.StatsSnapshot(statsSvc, logger))

	// ── HTTP router ───────────────────────────────────────────────────────────
	startTime := time.Now()
	router := api.NewRouter(api.RouterConfig{
		Auth:                     cfg.Auth.APIKey,
		Logger:                   logger,
		StartTime:                startTime,
		DB:                       database.SQL,
		DBType:                   database.Driver,
		DBPath:                   cfg.Database.Path,
		ConfigFile:               cfg.ConfigFile,
		AIEnabled:                !cfg.AI.APIKey.IsEmpty(),
		TMDBKeyIsDefault:         cfg.TMDBKeyIsDefault,
		QualityService:           qualitySvc,
		QualityDefinitionService: qualityDefSvc,
		LibraryService:           librarySvc,
		MovieService:             movieSvc,
		IndexerService:           indexerSvc,
		DownloaderService:        downloaderSvc,
		BlocklistService:         blocklistSvc,
		QueueService:             queueSvc,
		Scheduler:                sched,
		NotificationService:      notifSvc,
		HealthService:            healthSvc,
		MediaManagementService:   mmSvc,
		DownloadHandlingService:  dhSvc,
		RadarrImportService:      radarrImportSvc,
		StatsService:             statsSvc,
		MediaInfoService:         mediainfoSvc,
		CollectionService:        collectionSvc,
		MediaServerService:       mediaServerSvc,
		PlexSyncService:          plexSyncSvc,
		WSHub:                    wsHub,
		Bus:                      bus,
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
