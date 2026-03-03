package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/radarrimport"
)

// ── Request / response shapes ────────────────────────────────────────────────

type radarrPreviewInput struct {
	Body struct {
		URL    string `json:"url"     doc:"Radarr base URL, e.g. http://localhost:7878"`
		APIKey string `json:"api_key" doc:"Radarr API key"`
	}
}

type radarrPreviewOutput struct {
	Body *radarrimport.PreviewResult
}

type radarrExecuteInput struct {
	Body struct {
		URL     string                     `json:"url"     doc:"Radarr base URL"`
		APIKey  string                     `json:"api_key" doc:"Radarr API key"`
		Options radarrimport.ImportOptions `json:"options" doc:"Which categories to import"`
	}
}

type radarrExecuteOutput struct {
	Body *radarrimport.ImportResult
}

// ── Route registration ───────────────────────────────────────────────────────

// RegisterImportRoutes registers all /api/v1/import endpoints.
func RegisterImportRoutes(api huma.API, svc *radarrimport.Service) {
	// POST /api/v1/import/radarr/preview
	huma.Register(api, huma.Operation{
		OperationID: "radarr-import-preview",
		Method:      http.MethodPost,
		Path:        "/api/v1/import/radarr/preview",
		Summary:     "Preview Radarr import",
		Description: "Connects to a Radarr instance and returns a summary of what would be imported without making any changes.",
		Tags:        []string{"Import"},
	}, func(ctx context.Context, input *radarrPreviewInput) (*radarrPreviewOutput, error) {
		result, err := svc.Preview(ctx, input.Body.URL, input.Body.APIKey)
		if err != nil {
			return nil, huma.Error502BadGateway("could not reach Radarr: " + err.Error())
		}
		return &radarrPreviewOutput{Body: result}, nil
	})

	// POST /api/v1/import/radarr/execute
	huma.Register(api, huma.Operation{
		OperationID: "radarr-import-execute",
		Method:      http.MethodPost,
		Path:        "/api/v1/import/radarr/execute",
		Summary:     "Execute Radarr import",
		Description: "Imports selected categories from a Radarr instance into Luminarr.",
		Tags:        []string{"Import"},
	}, func(ctx context.Context, input *radarrExecuteInput) (*radarrExecuteOutput, error) {
		result, err := svc.Execute(ctx, input.Body.URL, input.Body.APIKey, input.Body.Options)
		if err != nil {
			return nil, huma.Error502BadGateway("could not reach Radarr: " + err.Error())
		}
		return &radarrExecuteOutput{Body: result}, nil
	})
}
