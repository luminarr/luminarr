package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/tag"
	"github.com/luminarr/luminarr/internal/registry"
)

// ── Request / response shapes ────────────────────────────────────────────────

type downloadClientBody struct {
	ID        string          `json:"id"        doc:"Download client UUID"`
	Name      string          `json:"name"      doc:"Display name"`
	Kind      string          `json:"kind"      doc:"Plugin kind: qbittorrent, transmission, etc."`
	Enabled   bool            `json:"enabled"   doc:"Whether this client is active"`
	Priority  int             `json:"priority"  doc:"Priority; lower = used first when multiple clients match"`
	Settings  json.RawMessage `json:"settings"  doc:"Plugin-specific settings (URL, credentials, etc.)"`
	TagIDs    []string        `json:"tag_ids"   doc:"Assigned tag UUIDs"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type downloadClientInput struct {
	Name     string          `json:"name"               doc:"Display name"`
	Kind     string          `json:"kind"               doc:"Plugin kind: qbittorrent, transmission, etc."`
	Enabled  *bool           `json:"enabled,omitempty"  doc:"Whether this client is active (default: true)"`
	Priority *int            `json:"priority,omitempty" doc:"Priority; lower = used first (default: 1)"`
	Settings json.RawMessage `json:"settings"           doc:"Plugin-specific connection settings"`
	TagIDs   []string        `json:"tag_ids,omitempty"  doc:"Tag UUIDs to assign"`
}

type downloadClientOutput struct {
	Body *downloadClientBody
}

type downloadClientListOutput struct {
	Body []*downloadClientBody
}

type downloadClientCreateInput struct {
	Body downloadClientInput
}

type downloadClientGetInput struct {
	ID string `path:"id"`
}

type downloadClientUpdateInput struct {
	ID   string `path:"id"`
	Body downloadClientInput
}

type downloadClientDeleteInput struct {
	ID string `path:"id"`
}

type downloadClientDeleteOutput struct{}

type downloadClientTestInput struct {
	ID string `path:"id"`
}

type downloadClientTestOutput struct {
	Body struct {
		OK      bool   `json:"ok"`
		Message string `json:"message,omitempty"`
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func downloadClientToBody(c downloader.Config) *downloadClientBody {
	return &downloadClientBody{
		ID:        c.ID,
		Name:      c.Name,
		Kind:      c.Kind,
		Enabled:   c.Enabled,
		Priority:  c.Priority,
		Settings:  registry.Default.SanitizeDownloaderSettings(c.Kind, c.Settings),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func downloadClientInputToRequest(in downloadClientInput) downloader.CreateRequest {
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	priority := 1
	if in.Priority != nil {
		priority = *in.Priority
	}
	return downloader.CreateRequest{
		Name:     in.Name,
		Kind:     in.Kind,
		Enabled:  enabled,
		Priority: priority,
		Settings: in.Settings,
	}
}

// ── Route registration ───────────────────────────────────────────────────────

// RegisterDownloadClientRoutes registers all /api/v1/download-clients endpoints.
func RegisterDownloadClientRoutes(api huma.API, svc *downloader.Service, tagSvc *tag.Service) {
	// GET /api/v1/download-clients
	huma.Register(api, huma.Operation{
		OperationID: "list-download-clients",
		Method:      http.MethodGet,
		Path:        "/api/v1/download-clients",
		Summary:     "List download clients",
		Tags:        []string{"Download Clients"},
	}, func(ctx context.Context, _ *struct{}) (*downloadClientListOutput, error) {
		configs, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list download clients", err)
		}
		bodies := make([]*downloadClientBody, len(configs))
		for i, c := range configs {
			b := downloadClientToBody(c)
			if tagSvc != nil {
				b.TagIDs, _ = tagSvc.DownloadClientTagIDs(ctx, c.ID)
			}
			bodies[i] = b
		}
		return &downloadClientListOutput{Body: bodies}, nil
	})

	// POST /api/v1/download-clients
	huma.Register(api, huma.Operation{
		OperationID:   "create-download-client",
		Method:        http.MethodPost,
		Path:          "/api/v1/download-clients",
		Summary:       "Create a download client",
		Tags:          []string{"Download Clients"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *downloadClientCreateInput) (*downloadClientOutput, error) {
		cfg, err := svc.Create(ctx, downloadClientInputToRequest(input.Body))
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "failed to create download client", err)
		}
		b := downloadClientToBody(cfg)
		if tagSvc != nil && len(input.Body.TagIDs) > 0 {
			_ = tagSvc.SetDownloadClientTags(ctx, cfg.ID, input.Body.TagIDs)
			b.TagIDs = input.Body.TagIDs
		}
		if b.TagIDs == nil {
			b.TagIDs = []string{}
		}
		return &downloadClientOutput{Body: b}, nil
	})

	// GET /api/v1/download-clients/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-download-client",
		Method:      http.MethodGet,
		Path:        "/api/v1/download-clients/{id}",
		Summary:     "Get a download client",
		Tags:        []string{"Download Clients"},
	}, func(ctx context.Context, input *downloadClientGetInput) (*downloadClientOutput, error) {
		cfg, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, downloader.ErrNotFound) {
				return nil, huma.Error404NotFound("download client not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get download client", err)
		}
		b := downloadClientToBody(cfg)
		if tagSvc != nil {
			b.TagIDs, _ = tagSvc.DownloadClientTagIDs(ctx, cfg.ID)
		}
		return &downloadClientOutput{Body: b}, nil
	})

	// PUT /api/v1/download-clients/{id}
	huma.Register(api, huma.Operation{
		OperationID: "update-download-client",
		Method:      http.MethodPut,
		Path:        "/api/v1/download-clients/{id}",
		Summary:     "Update a download client",
		Tags:        []string{"Download Clients"},
	}, func(ctx context.Context, input *downloadClientUpdateInput) (*downloadClientOutput, error) {
		cfg, err := svc.Update(ctx, input.ID, downloadClientInputToRequest(input.Body))
		if err != nil {
			if errors.Is(err, downloader.ErrNotFound) {
				return nil, huma.Error404NotFound("download client not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update download client", err)
		}
		b := downloadClientToBody(cfg)
		if tagSvc != nil {
			if input.Body.TagIDs != nil {
				_ = tagSvc.SetDownloadClientTags(ctx, cfg.ID, input.Body.TagIDs)
				b.TagIDs = input.Body.TagIDs
			} else {
				b.TagIDs, _ = tagSvc.DownloadClientTagIDs(ctx, cfg.ID)
			}
		}
		return &downloadClientOutput{Body: b}, nil
	})

	// DELETE /api/v1/download-clients/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-download-client",
		Method:        http.MethodDelete,
		Path:          "/api/v1/download-clients/{id}",
		Summary:       "Delete a download client",
		Tags:          []string{"Download Clients"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *downloadClientDeleteInput) (*downloadClientDeleteOutput, error) {
		err := svc.Delete(ctx, input.ID)
		if err != nil {
			if errors.Is(err, downloader.ErrNotFound) {
				return nil, huma.Error404NotFound("download client not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete download client", err)
		}
		return &downloadClientDeleteOutput{}, nil
	})

	// POST /api/v1/download-clients/{id}/test
	huma.Register(api, huma.Operation{
		OperationID: "test-download-client",
		Method:      http.MethodPost,
		Path:        "/api/v1/download-clients/{id}/test",
		Summary:     "Test download client connectivity",
		Tags:        []string{"Download Clients"},
	}, func(ctx context.Context, input *downloadClientTestInput) (*downloadClientTestOutput, error) {
		err := svc.Test(ctx, input.ID)
		if err != nil {
			if errors.Is(err, downloader.ErrNotFound) {
				return nil, huma.Error404NotFound("download client not found")
			}
			out := &downloadClientTestOutput{}
			out.Body.OK = false
			out.Body.Message = err.Error()
			return out, nil
		}
		out := &downloadClientTestOutput{}
		out.Body.OK = true
		return out, nil
	})
}
