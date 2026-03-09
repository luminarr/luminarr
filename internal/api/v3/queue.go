package v3

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/queue"
)

func registerQueueRoutes(api huma.API, svc *queue.Service) {
	if svc == nil {
		return
	}

	huma.Register(api, huma.Operation{
		OperationID: "radarr-queue-status",
		Method:      http.MethodGet,
		Path:        "/api/v3/queue/status",
		Summary:     "Queue status (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body RadarrQueueStatus }, error) {
		items, err := svc.GetQueue(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get queue", err)
		}
		return &struct{ Body RadarrQueueStatus }{Body: RadarrQueueStatus{
			TotalCount:           len(items),
			Count:                len(items),
			HasEnabledDownloader: true,
		}}, nil
	})
}
