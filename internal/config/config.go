package config

// Config holds all application configuration.
// Values are loaded from config.yaml and can be overridden by
// LUMINARR_* environment variables (e.g. LUMINARR_SERVER_PORT=8080).
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Log       LogConfig       `mapstructure:"log"`
	TMDB      TMDBConfig      `mapstructure:"tmdb"`
	AI        AIConfig        `mapstructure:"ai"`
	Auth      AuthConfig      `mapstructure:"auth"`
	MediaInfo MediaInfoConfig `mapstructure:"mediainfo"`

	// ConfigFile is the path of the config file that was loaded, if any.
	// Empty when running on defaults/env vars only.
	ConfigFile string `mapstructure:"-"`

	// TMDBKeyIsDefault is true when the TMDB key came from the build-time
	// default rather than user configuration.
	TMDBKeyIsDefault bool `mapstructure:"-"`
}

// MediaInfoConfig controls optional ffprobe-based media scanning.
type MediaInfoConfig struct {
	// FFprobePath is the path to the ffprobe binary. Empty means search $PATH.
	FFprobePath string `mapstructure:"ffprobe_path"`
	// ScanTimeout is the per-file timeout for ffprobe. Default: "30s".
	ScanTimeout string `mapstructure:"scan_timeout"`
	// ScanOnImport enables automatic scanning after a successful import.
	ScanOnImport bool `mapstructure:"scan_on_import"`
}

// ServerConfig controls the HTTP server.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig selects and configures the database driver.
type DatabaseConfig struct {
	// Driver is "sqlite" (default) or "postgres".
	Driver string `mapstructure:"driver"`
	// Path is the SQLite database file path. Ignored for Postgres.
	Path string `mapstructure:"path"`
	// DSN is the Postgres connection string. Ignored for SQLite.
	DSN Secret `mapstructure:"dsn"`
}

// LogConfig controls log output format and verbosity.
type LogConfig struct {
	// Level is one of: debug, info, warn, error. Default: info.
	Level string `mapstructure:"level"`
	// Format is one of: json, text. Default: json.
	Format string `mapstructure:"format"`
}

// TMDBConfig holds The Movie Database API credentials.
type TMDBConfig struct {
	APIKey Secret `mapstructure:"api_key"`
}

// AIConfig holds Claude API credentials and model selection.
type AIConfig struct {
	APIKey      Secret `mapstructure:"api_key"`
	MatchModel  string `mapstructure:"match_model"`
	ScoreModel  string `mapstructure:"score_model"`
	FilterModel string `mapstructure:"filter_model"`
}

// AuthConfig holds the Luminarr API key used to authenticate requests.
type AuthConfig struct {
	APIKey Secret `mapstructure:"api_key"`
}

// DefaultTMDBKey is set at build time via ldflags. Users can override
// via config file or LUMINARR_TMDB_API_KEY env var.
var DefaultTMDBKey string
