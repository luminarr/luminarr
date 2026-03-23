package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/anthropic"
	"github.com/luminarr/luminarr/internal/config"
	"github.com/luminarr/luminarr/internal/core/aicommand"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
	"github.com/luminarr/luminarr/internal/safedialer"
	"github.com/luminarr/luminarr/internal/version"
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
	TMDBKeySource     string `json:"tmdb_key_source" doc:"Source of the TMDB key: default, custom, or none"`
	APIKey            string `json:"api_key" doc:"API key for external integrations"`
	ConfigFile        string `json:"config_file,omitempty" doc:"Path of the loaded config file, if any"`
}

type systemConfigOutput struct {
	Body *systemConfigBody
}

// updateConfigInput is the request body for PUT /api/v1/system/config.
type updateConfigInput struct {
	Body struct {
		TMDBAPIKey string `json:"tmdb_api_key,omitempty" doc:"TMDB API key to persist"`
		AIAPIKey   string `json:"ai_api_key,omitempty"   doc:"Anthropic API key to persist"`
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

// updateCheckBody is the response body for GET /api/v1/system/updates.
type updateCheckBody struct {
	UpdateAvailable bool   `json:"update_available"         doc:"Whether a newer version exists"`
	CurrentVersion  string `json:"current_version"          doc:"Currently running version"`
	LatestVersion   string `json:"latest_version"           doc:"Latest release tag from GitHub"`
	ReleaseURL      string `json:"release_url,omitempty"    doc:"URL to the GitHub release page"`
	ReleaseNotes    string `json:"release_notes,omitempty"  doc:"Release notes (markdown)"`
	PublishedAt     string `json:"published_at,omitempty"   doc:"When the release was published (ISO 8601)"`
}

type updateCheckOutput struct {
	Body *updateCheckBody
}

// githubRelease is the subset of the GitHub releases API response we need.
type githubRelease struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

// RegisterSystemRoutes registers the /api/v1/system/* endpoints.
// movieSvc/aiCmdSvc may be nil in test environments.
func RegisterSystemRoutes(api huma.API, startTime time.Time, dbType, dbPath, configFile string, aiCmdSvc *aicommand.Service, tmdbKeyIsDefault bool, apiKey string, movieSvc *movie.Service, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-system-status",
		Method:      "GET",
		Path:        "/api/v1/system/status",
		Summary:     "Get system status",
		Description: "Returns runtime information about the Luminarr server.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, _ *struct{}) (*systemStatusOutput, error) {
		tmdbEnabled := movieSvc != nil && movieSvc.HasMetadataProvider()
		aiEnabled := aiCmdSvc != nil && aiCmdSvc.Enabled()
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
		hasProvider := movieSvc != nil && movieSvc.HasMetadataProvider()
		source := "none"
		if hasProvider {
			if tmdbKeyIsDefault {
				source = "default"
			} else {
				source = "custom"
			}
		}
		maskedKey := apiKey
		if len(apiKey) > 4 {
			maskedKey = apiKey[:4] + "****"
		}
		return &systemConfigOutput{Body: &systemConfigBody{
			TMDBKeyConfigured: hasProvider,
			TMDBKeySource:     source,
			APIKey:            maskedKey,
			ConfigFile:        configFile,
		}}, nil
	})

	// GET /api/v1/system/config/apikey — reveal the full API key on explicit request.
	huma.Register(api, huma.Operation{
		OperationID: "get-api-key",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/config/apikey",
		Summary:     "Reveal the full API key",
		Tags:        []string{"System"},
	}, func(_ context.Context, _ *struct{}) (*struct {
		Body struct {
			APIKey string `json:"api_key"`
		}
	}, error) {
		return &struct {
			Body struct {
				APIKey string `json:"api_key"`
			}
		}{
			Body: struct {
				APIKey string `json:"api_key"`
			}{APIKey: apiKey},
		}, nil
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
		if input.Body.TMDBAPIKey == "" && input.Body.AIAPIKey == "" {
			return nil, huma.NewError(http.StatusBadRequest, "no config values provided")
		}

		var writePath string
		var err error

		if input.Body.TMDBAPIKey != "" {
			writePath, err = config.WriteConfigKey(configFile, "tmdb.api_key", input.Body.TMDBAPIKey)
			if err != nil {
				return nil, huma.NewError(http.StatusInternalServerError, "failed to write TMDB config", err)
			}
			// Wire up the new TMDB client immediately — no restart needed.
			if movieSvc != nil {
				movieSvc.SetMetadataProvider(tmdb.New(input.Body.TMDBAPIKey, logger))
			}
		}

		if input.Body.AIAPIKey != "" {
			writePath, err = config.WriteConfigKey(configFile, "ai.api_key", input.Body.AIAPIKey)
			if err != nil {
				return nil, huma.NewError(http.StatusInternalServerError, "failed to write AI config", err)
			}
			// Wire up the new Anthropic client immediately — no restart needed.
			if aiCmdSvc != nil {
				aiCmdSvc.SetClient(anthropic.New(input.Body.AIAPIKey))
			}
		}

		return &updateConfigOutput{Body: &updateConfigResult{
			Saved:      true,
			ConfigFile: writePath,
		}}, nil
	})

	// GET /api/v1/system/updates — check GitHub for a newer release.
	huma.Register(api, huma.Operation{
		OperationID: "check-for-updates",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/updates",
		Summary:     "Check for updates",
		Description: "Queries the GitHub releases API and compares the latest tag against the running version.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, _ *struct{}) (*updateCheckOutput, error) {
		client := &http.Client{
			Transport: safedialer.Transport(),
			Timeout:   10 * time.Second,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://api.github.com/repos/luminarr/luminarr/releases/latest", nil)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "failed to build GitHub request", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", fmt.Sprintf("Luminarr/%s", version.Version))

		resp, err := client.Do(req)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "failed to reach GitHub", err)
		}
		defer resp.Body.Close()

		// GitHub returns 404 when there are no releases.
		if resp.StatusCode == http.StatusNotFound {
			return &updateCheckOutput{Body: &updateCheckBody{
				UpdateAvailable: false,
				CurrentVersion:  version.Version,
				LatestVersion:   version.Version,
			}}, nil
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return nil, huma.NewError(http.StatusBadGateway,
				fmt.Sprintf("GitHub returned %d: %s", resp.StatusCode, string(body)))
		}

		var release githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "failed to parse GitHub response", err)
		}

		currentBare := strings.TrimPrefix(version.Version, "v")
		latestBare := strings.TrimPrefix(release.TagName, "v")

		result := &updateCheckBody{
			UpdateAvailable: currentBare != latestBare,
			CurrentVersion:  version.Version,
			LatestVersion:   release.TagName,
		}

		if result.UpdateAvailable {
			result.ReleaseURL = release.HTMLURL
			result.ReleaseNotes = release.Body
			result.PublishedAt = release.PublishedAt
		}

		return &updateCheckOutput{Body: result}, nil
	})
}
