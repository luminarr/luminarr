package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/luminarr/luminarr/internal/core/activity"
	"github.com/luminarr/luminarr/internal/scheduler"
)

// ActivityPrune returns a Job that deletes activity log entries older than
// 30 days. Runs once per day.
func ActivityPrune(svc *activity.Service, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:     "activity_prune",
		Interval: 24 * time.Hour,
		Fn: func(ctx context.Context) {
			if err := svc.Prune(ctx, 30*24*time.Hour); err != nil {
				logger.Warn("activity prune failed", "task", "activity_prune", "error", err)
			}
		},
	}
}
