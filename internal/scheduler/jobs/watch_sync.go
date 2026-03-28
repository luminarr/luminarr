package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/luminarr/luminarr/internal/core/watchsync"
	"github.com/luminarr/luminarr/internal/scheduler"
)

// WatchSync returns a Job that polls media servers for watch history
// every 6 hours.
func WatchSync(svc *watchsync.Service, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:     "watch_sync",
		Interval: 6 * time.Hour,
		Fn: func(ctx context.Context) {
			logger.Info("task started", "task", "watch_sync")
			start := time.Now()

			if err := svc.Sync(ctx); err != nil {
				logger.Warn("task completed with errors",
					"task", "watch_sync",
					"error", err,
					"duration_ms", time.Since(start).Milliseconds(),
				)
				return
			}

			logger.Info("task finished",
				"task", "watch_sync",
				"duration_ms", time.Since(start).Milliseconds(),
			)
		},
	}
}
