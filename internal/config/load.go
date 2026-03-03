package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	DefaultHost          = "0.0.0.0"
	DefaultPort          = 8282
	DefaultDBDriver      = "sqlite"
	DefaultLogLevel      = "info"
	DefaultLogFormat     = "json"
	DefaultAIMatchModel  = "claude-sonnet-4-6"
	DefaultAIScoreModel  = "claude-haiku-4-5-20251001"
	DefaultAIFilterModel = "claude-haiku-4-5-20251001"
)

// Load reads configuration from a YAML file and environment variables.
// If cfgFile is empty, the following paths are searched in order:
//
//	/config/config.yaml              (Docker volume mount)
//	$HOME/.config/luminarr/config.yaml
//	/etc/luminarr/config.yaml
//	./config.yaml
//
// Missing config file is not an error — defaults and environment variables
// are always applied.
func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.host", DefaultHost)
	v.SetDefault("server.port", DefaultPort)
	v.SetDefault("database.driver", DefaultDBDriver)
	v.SetDefault("log.level", DefaultLogLevel)
	v.SetDefault("log.format", DefaultLogFormat)
	v.SetDefault("ai.match_model", DefaultAIMatchModel)
	v.SetDefault("ai.score_model", DefaultAIScoreModel)
	v.SetDefault("ai.filter_model", DefaultAIFilterModel)

	// Config file location
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("/config") // Docker volume mount point
		if home != "" {
			v.AddConfigPath(filepath.Join(home, ".config", "luminarr"))
		}
		v.AddConfigPath("/etc/luminarr")
		v.AddConfigPath(".")
	}

	// Environment variable overrides.
	// AutomaticEnv handles simple keys; BindEnv covers keys with underscores
	// in the name (e.g. api_key) where viper's replacer can misfire.
	v.SetEnvPrefix("LUMINARR")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicit bindings for keys that contain underscores — these are
	// not reliably resolved by AutomaticEnv alone.
	_ = v.BindEnv("auth.api_key", "LUMINARR_AUTH_API_KEY")
	_ = v.BindEnv("tmdb.api_key", "LUMINARR_TMDB_API_KEY")
	_ = v.BindEnv("ai.api_key", "LUMINARR_AI_API_KEY")
	_ = v.BindEnv("database.path", "LUMINARR_DATABASE_PATH")
	_ = v.BindEnv("database.dsn", "LUMINARR_DATABASE_DSN")
	_ = v.BindEnv("ai.match_model", "LUMINARR_AI_MATCH_MODEL")
	_ = v.BindEnv("ai.score_model", "LUMINARR_AI_SCORE_MODEL")
	_ = v.BindEnv("ai.filter_model", "LUMINARR_AI_FILTER_MODEL")

	if err := v.ReadInConfig(); err != nil {
		// Missing config file is not an error — we use defaults.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}
	configFileUsed := v.ConfigFileUsed()

	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			secretDecodeHook,
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	)); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Default SQLite path: prefer ~/.config/luminarr/luminarr.db; fall back to
	// /config/luminarr.db (Docker volume) when $HOME is unavailable.
	if cfg.Database.Driver == "sqlite" && cfg.Database.Path == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			cfg.Database.Path = filepath.Join(home, ".config", "luminarr", "luminarr.db")
		} else {
			cfg.Database.Path = "/config/luminarr.db"
		}
	}

	cfg.ConfigFile = configFileUsed

	return &cfg, nil
}

// EnsureAPIKey generates a random API key if none is configured.
// The generated key is returned and must be shown to the user — it is
// only ever logged once.
func EnsureAPIKey(cfg *Config) (generated bool, err error) {
	if !cfg.Auth.APIKey.IsEmpty() {
		return false, nil
	}

	key, err := generateAPIKey()
	if err != nil {
		return false, fmt.Errorf("generating API key: %w", err)
	}

	cfg.Auth.APIKey = Secret(key)
	return true, nil
}

// secretDecodeHook allows mapstructure to convert plain strings into the
// Secret type. Without this, env var values (always strings) cannot be
// decoded into Secret fields.
func secretDecodeHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	if to == reflect.TypeOf(Secret("")) && from.Kind() == reflect.String {
		return Secret(data.(string)), nil
	}
	return data, nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
