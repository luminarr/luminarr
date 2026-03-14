package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/notification"
	"github.com/luminarr/luminarr/internal/core/tag"
	"github.com/luminarr/luminarr/internal/registry"
)

// ── Request / response shapes ─────────────────────────────────────────────────

type notificationBody struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Kind      string          `json:"kind"          doc:"Plugin kind: webhook, discord, email"`
	Enabled   bool            `json:"enabled"`
	Settings  json.RawMessage `json:"settings"      doc:"Plugin-specific settings as JSON"`
	OnEvents  []string        `json:"on_events"     doc:"Event types that trigger this notification"`
	TagIDs    []string        `json:"tag_ids"       doc:"Assigned tag UUIDs"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type notificationListOutput struct {
	Body []*notificationBody
}

type notificationGetOutput struct {
	Body *notificationBody
}

type notificationInput struct {
	ID string `path:"id"`
}

type notificationCreateBody struct {
	Name     string          `json:"name"       minLength:"1"`
	Kind     string          `json:"kind"       minLength:"1"`
	Enabled  bool            `json:"enabled"`
	Settings json.RawMessage `json:"settings,omitempty"`
	OnEvents []string        `json:"on_events,omitempty"`
	TagIDs   []string        `json:"tag_ids,omitempty"  doc:"Tag UUIDs to assign"`
}

type notificationCreateInput struct {
	Body notificationCreateBody
}

type notificationUpdateInput struct {
	ID   string `path:"id"`
	Body notificationCreateBody
}

type notificationDeleteInput struct {
	ID string `path:"id"`
}

type notificationDeleteOutput struct{}

type notificationTestInput struct {
	ID string `path:"id"`
}

type notificationTestOutput struct{}

// ── Helpers ───────────────────────────────────────────────────────────────────

func notifToBody(cfg notification.Config) *notificationBody {
	return &notificationBody{
		ID:        cfg.ID,
		Name:      cfg.Name,
		Kind:      cfg.Kind,
		Enabled:   cfg.Enabled,
		Settings:  registry.Default.SanitizeNotifierSettings(cfg.Kind, cfg.Settings),
		OnEvents:  cfg.OnEvents,
		CreatedAt: cfg.CreatedAt,
		UpdatedAt: cfg.UpdatedAt,
	}
}

// ── Route registration ────────────────────────────────────────────────────────

// RegisterNotificationRoutes registers the /api/v1/notifications endpoints.
func RegisterNotificationRoutes(api huma.API, svc *notification.Service, tagSvc *tag.Service) {
	// GET /api/v1/notifications
	huma.Register(api, huma.Operation{
		OperationID: "list-notifications",
		Method:      http.MethodGet,
		Path:        "/api/v1/notifications",
		Summary:     "List all notification configurations",
		Tags:        []string{"Notifications"},
	}, func(ctx context.Context, _ *struct{}) (*notificationListOutput, error) {
		cfgs, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list notifications", err)
		}
		bodies := make([]*notificationBody, len(cfgs))
		for i, c := range cfgs {
			b := notifToBody(c)
			if tagSvc != nil {
				b.TagIDs, _ = tagSvc.NotificationTagIDs(ctx, c.ID)
			}
			bodies[i] = b
		}
		return &notificationListOutput{Body: bodies}, nil
	})

	// POST /api/v1/notifications
	huma.Register(api, huma.Operation{
		OperationID:   "create-notification",
		Method:        http.MethodPost,
		Path:          "/api/v1/notifications",
		Summary:       "Create a notification configuration",
		Tags:          []string{"Notifications"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *notificationCreateInput) (*notificationGetOutput, error) {
		cfg, err := svc.Create(ctx, notification.CreateRequest{
			Name:     input.Body.Name,
			Kind:     input.Body.Kind,
			Enabled:  input.Body.Enabled,
			Settings: input.Body.Settings,
			OnEvents: input.Body.OnEvents,
		})
		if err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, "failed to create notification", err)
		}
		b := notifToBody(cfg)
		if tagSvc != nil && len(input.Body.TagIDs) > 0 {
			_ = tagSvc.SetNotificationTags(ctx, cfg.ID, input.Body.TagIDs)
			b.TagIDs = input.Body.TagIDs
		}
		if b.TagIDs == nil {
			b.TagIDs = []string{}
		}
		return &notificationGetOutput{Body: b}, nil
	})

	// GET /api/v1/notifications/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-notification",
		Method:      http.MethodGet,
		Path:        "/api/v1/notifications/{id}",
		Summary:     "Get a notification configuration",
		Tags:        []string{"Notifications"},
	}, func(ctx context.Context, input *notificationInput) (*notificationGetOutput, error) {
		cfg, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, notification.ErrNotFound) {
				return nil, huma.Error404NotFound("notification not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get notification", err)
		}
		b := notifToBody(cfg)
		if tagSvc != nil {
			b.TagIDs, _ = tagSvc.NotificationTagIDs(ctx, cfg.ID)
		}
		return &notificationGetOutput{Body: b}, nil
	})

	// PUT /api/v1/notifications/{id}
	huma.Register(api, huma.Operation{
		OperationID: "update-notification",
		Method:      http.MethodPut,
		Path:        "/api/v1/notifications/{id}",
		Summary:     "Update a notification configuration",
		Tags:        []string{"Notifications"},
	}, func(ctx context.Context, input *notificationUpdateInput) (*notificationGetOutput, error) {
		cfg, err := svc.Update(ctx, input.ID, notification.UpdateRequest{
			Name:     input.Body.Name,
			Kind:     input.Body.Kind,
			Enabled:  input.Body.Enabled,
			Settings: input.Body.Settings,
			OnEvents: input.Body.OnEvents,
		})
		if err != nil {
			if errors.Is(err, notification.ErrNotFound) {
				return nil, huma.Error404NotFound("notification not found")
			}
			return nil, huma.NewError(http.StatusUnprocessableEntity, "failed to update notification", err)
		}
		b := notifToBody(cfg)
		if tagSvc != nil {
			if input.Body.TagIDs != nil {
				_ = tagSvc.SetNotificationTags(ctx, cfg.ID, input.Body.TagIDs)
				b.TagIDs = input.Body.TagIDs
			} else {
				b.TagIDs, _ = tagSvc.NotificationTagIDs(ctx, cfg.ID)
			}
		}
		return &notificationGetOutput{Body: b}, nil
	})

	// DELETE /api/v1/notifications/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-notification",
		Method:        http.MethodDelete,
		Path:          "/api/v1/notifications/{id}",
		Summary:       "Delete a notification configuration",
		Tags:          []string{"Notifications"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *notificationDeleteInput) (*notificationDeleteOutput, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			if errors.Is(err, notification.ErrNotFound) {
				return nil, huma.Error404NotFound("notification not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete notification", err)
		}
		return &notificationDeleteOutput{}, nil
	})

	// POST /api/v1/notifications/{id}/test
	huma.Register(api, huma.Operation{
		OperationID:   "test-notification",
		Method:        http.MethodPost,
		Path:          "/api/v1/notifications/{id}/test",
		Summary:       "Send a test notification",
		Tags:          []string{"Notifications"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *notificationTestInput) (*notificationTestOutput, error) {
		if err := svc.Test(ctx, input.ID); err != nil {
			if errors.Is(err, notification.ErrNotFound) {
				return nil, huma.Error404NotFound("notification not found")
			}
			return nil, huma.NewError(http.StatusBadGateway, "test notification failed", err)
		}
		return &notificationTestOutput{}, nil
	})
}
