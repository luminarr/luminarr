package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/watchsync"
)

type watchSyncOutput struct {
	Body struct {
		OK string `json:"status" doc:"Sync result"`
	}
}

type watchStatsOutput struct {
	Body *watchsync.WatchStats
}

// RegisterWatchSyncRoutes registers the watch sync API endpoints.
func RegisterWatchSyncRoutes(api huma.API, svc *watchsync.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "run-watch-sync",
		Method:      http.MethodPost,
		Path:        "/api/v1/watch-sync/run",
		Summary:     "Trigger watch history sync",
		Description: "Polls all configured media servers for new watch events.",
		Tags:        []string{"Watch"},
	}, func(ctx context.Context, _ *struct{}) (*watchSyncOutput, error) {
		syncErr := svc.Sync(ctx)
		if syncErr != nil {
			return &watchSyncOutput{Body: struct { //nolint:nilerr // report partial success
				OK string `json:"status" doc:"Sync result"`
			}{OK: "completed with errors: " + syncErr.Error()}}, nil
		}
		return &watchSyncOutput{Body: struct {
			OK string `json:"status" doc:"Sync result"`
		}{OK: "ok"}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-watch-stats",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/watch",
		Summary:     "Get watch statistics",
		Description: "Returns aggregate watch stats: total watched, unwatched, percentage.",
		Tags:        []string{"Watch"},
	}, func(ctx context.Context, _ *struct{}) (*watchStatsOutput, error) {
		stats, err := svc.Stats(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get watch stats", err)
		}
		return &watchStatsOutput{Body: stats}, nil
	})
}
