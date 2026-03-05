package v1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/library"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/scheduler"
)

// ── Request / Response types ────────────────────────────────────────────────

type hookScanRequest struct {
	Body struct {
		LibraryID string `json:"library_id,omitempty" doc:"Scan a specific library. If omitted, all libraries are scanned."`
	}
}

type hookRefreshRequest struct {
	Body struct {
		MovieID string `json:"movie_id,omitempty" doc:"Refresh a specific movie. If omitted, runs the global refresh job."`
	}
}

type hookNotifyRequest struct {
	Body struct {
		Type    string         `json:"type" doc:"Event type (e.g. 'custom')." minLength:"1"`
		Message string         `json:"message,omitempty" doc:"Human-readable message."`
		Data    map[string]any `json:"data,omitempty" doc:"Arbitrary key-value data forwarded to notifiers."`
	}
}

type hookAccepted struct {
	Status int `header:"Status" json:"-"`
}

// ── Registration ────────────────────────────────────────────────────────────

// RegisterHookRoutes registers inbound webhook trigger endpoints.
func RegisterHookRoutes(
	api huma.API,
	libSvc *library.Service,
	movieSvc *movie.Service,
	bus *events.Bus,
	sched *scheduler.Scheduler,
) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/api/v1/hooks/scan",
		Summary:     "Trigger library scan",
		Description: "Trigger a disk scan for one or all libraries. Returns immediately; scan runs in the background.",
		Tags:        []string{"Hooks"},
	}, func(ctx context.Context, req *hookScanRequest) (*hookAccepted, error) {
		if req.Body.LibraryID != "" {
			if _, err := libSvc.Get(ctx, req.Body.LibraryID); err != nil {
				return nil, huma.Error404NotFound(fmt.Sprintf("library %q not found", req.Body.LibraryID))
			}
			go libSvc.Scan(context.Background(), req.Body.LibraryID) //nolint:errcheck // fire-and-forget
		} else {
			libs, err := libSvc.List(ctx)
			if err != nil {
				return nil, fmt.Errorf("listing libraries: %w", err)
			}
			for _, lib := range libs {
				go libSvc.Scan(context.Background(), lib.ID) //nolint:errcheck // fire-and-forget
			}
		}
		return &hookAccepted{Status: http.StatusAccepted}, nil
	})

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/api/v1/hooks/refresh",
		Summary:     "Trigger metadata refresh",
		Description: "Refresh TMDB metadata for one movie or all movies. Returns immediately.",
		Tags:        []string{"Hooks"},
	}, func(ctx context.Context, req *hookRefreshRequest) (*hookAccepted, error) {
		if req.Body.MovieID != "" {
			if _, err := movieSvc.Get(ctx, req.Body.MovieID); err != nil {
				return nil, huma.Error404NotFound(fmt.Sprintf("movie %q not found", req.Body.MovieID))
			}
			go movieSvc.RefreshMetadata(context.Background(), req.Body.MovieID) //nolint:errcheck // fire-and-forget
		} else {
			if err := sched.RunNow(ctx, "refresh_metadata"); err != nil {
				return nil, fmt.Errorf("triggering refresh job: %w", err)
			}
		}
		return &hookAccepted{Status: http.StatusAccepted}, nil
	})

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/api/v1/hooks/notify",
		Summary:     "Send a custom notification",
		Description: "Publish a custom event through the notification pipeline. All enabled notifiers matching the event type will fire.",
		Tags:        []string{"Hooks"},
	}, func(ctx context.Context, req *hookNotifyRequest) (*hookAccepted, error) {
		data := req.Body.Data
		if data == nil {
			data = make(map[string]any)
		}
		if req.Body.Message != "" {
			data["message"] = req.Body.Message
		}

		bus.Publish(ctx, events.Event{
			Type: events.Type(req.Body.Type),
			Data: data,
		})
		return &hookAccepted{Status: http.StatusAccepted}, nil
	})
}
