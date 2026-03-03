package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/config"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/metadata/tmdb"
	"github.com/davidfic/luminarr/internal/version"
)

// systemStatus holds the shape of the status response body.
type systemStatus struct {
	AppName       string `json:"app_name"               doc:"Application name"`
	Version       string `json:"version"                doc:"Build version"`
	BuildTime     string `json:"build_time"             doc:"UTC build timestamp"`
	GoVersion     string `json:"go_version"             doc:"Go runtime version"`
	DBType        string `json:"db_type"                doc:"Active database driver"`
	DBPath        string `json:"db_path,omitempty"      doc:"SQLite database file path (sqlite only)"`
	UptimeSeconds int64  `json:"uptime_seconds"         doc:"Seconds since startup"`
	StartTime     string `json:"start_time"             doc:"UTC server start time"`
	AIEnabled     bool   `json:"ai_enabled"             doc:"Whether Claude AI features are active"`
	TMDBEnabled   bool   `json:"tmdb_enabled"           doc:"Whether TMDB metadata fetching is active"`
}

type systemStatusOutput struct {
	Body *systemStatus
}

// systemConfigBody is the response body for GET /api/v1/system/config.
type systemConfigBody struct {
	TMDBKeyConfigured bool   `json:"tmdb_key_configured" doc:"Whether a TMDB API key is set"`
	ConfigFile        string `json:"config_file,omitempty" doc:"Path of the loaded config file, if any"`
}

type systemConfigOutput struct {
	Body *systemConfigBody
}

// updateConfigInput is the request body for PUT /api/v1/system/config.
type updateConfigInput struct {
	Body struct {
		TMDBAPIKey string `json:"tmdb_api_key,omitempty" doc:"TMDB API key to persist"`
	}
}

// updateConfigResult is the response body for PUT /api/v1/system/config.
type updateConfigResult struct {
	Saved      bool   `json:"saved"`
	ConfigFile string `json:"config_file"`
}

type updateConfigOutput struct {
	Body *updateConfigResult
}

// RegisterSystemRoutes registers the /api/v1/system/* endpoints.
// movieSvc may be nil in test environments; aiEnabled reflects the static AI key state.
func RegisterSystemRoutes(api huma.API, startTime time.Time, dbType, dbPath, configFile string, aiEnabled bool, movieSvc *movie.Service, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-system-status",
		Method:      "GET",
		Path:        "/api/v1/system/status",
		Summary:     "Get system status",
		Description: "Returns runtime information about the Luminarr server.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, _ *struct{}) (*systemStatusOutput, error) {
		tmdbEnabled := movieSvc != nil && movieSvc.HasMetadataProvider()
		return &systemStatusOutput{
			Body: &systemStatus{
				AppName:       "Luminarr",
				Version:       version.Version,
				BuildTime:     version.BuildTime,
				GoVersion:     version.GoVersion(),
				DBType:        dbType,
				DBPath:        dbPath,
				UptimeSeconds: int64(time.Since(startTime).Seconds()),
				StartTime:     startTime.UTC().Format(time.RFC3339),
				AIEnabled:     aiEnabled,
				TMDBEnabled:   tmdbEnabled,
			},
		}, nil
	})

	// GET /api/v1/system/config — surface what is and isn't configured.
	huma.Register(api, huma.Operation{
		OperationID: "get-system-config",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/config",
		Summary:     "Get system configuration status",
		Tags:        []string{"System"},
	}, func(ctx context.Context, _ *struct{}) (*systemConfigOutput, error) {
		return &systemConfigOutput{Body: &systemConfigBody{
			TMDBKeyConfigured: movieSvc != nil && movieSvc.HasMetadataProvider(),
			ConfigFile:        configFile,
		}}, nil
	})

	// PUT /api/v1/system/config — update config values and activate them immediately.
	huma.Register(api, huma.Operation{
		OperationID: "update-system-config",
		Method:      http.MethodPut,
		Path:        "/api/v1/system/config",
		Summary:     "Update system configuration",
		Description: "Writes the supplied values to the config file and activates them immediately without a restart.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *updateConfigInput) (*updateConfigOutput, error) {
		if input.Body.TMDBAPIKey == "" {
			return nil, huma.NewError(http.StatusBadRequest, "no config values provided")
		}

		writePath, err := config.WriteConfigKey(configFile, "tmdb.api_key", input.Body.TMDBAPIKey)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to write config file", err)
		}

		// Wire up the new TMDB client immediately — no restart needed.
		if movieSvc != nil {
			movieSvc.SetMetadataProvider(tmdb.New(input.Body.TMDBAPIKey, logger))
		}

		return &updateConfigOutput{Body: &updateConfigResult{
			Saved:      true,
			ConfigFile: writePath,
		}}, nil
	})
}
