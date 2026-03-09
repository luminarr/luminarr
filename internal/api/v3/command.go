package v3

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/scheduler"
)

func registerCommandRoutes(api huma.API, db *sql.DB, movieSvc *movie.Service, sched *scheduler.Scheduler) {
	type commandInput struct {
		Body RadarrCommand
	}

	huma.Register(api, huma.Operation{
		OperationID: "radarr-command",
		Method:      http.MethodPost,
		Path:        "/api/v3/command",
		Summary:     "Execute command (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, input *commandInput) (*struct{ Body RadarrCommandResponse }, error) {
		resp := RadarrCommandResponse{
			ID:      1,
			Name:    input.Body.Name,
			Status:  "started",
			Started: time.Now().UTC().Format(time.RFC3339),
		}

		switch input.Body.Name {
		case "MoviesSearch":
			// Trigger search for specified movies. In Luminarr this is handled
			// by the RSS sync / automatic search — just acknowledge.
			// Future: actually trigger indexer search for specific movies.
		case "RefreshMovie":
			// Refresh metadata for specific movies.
			if movieSvc != nil {
				for _, rowid := range input.Body.MovieIDs {
					uuid, err := getUUIDByRowID(ctx, db, "movies", rowid)
					if err != nil || uuid == "" {
						continue
					}
					_, _ = movieSvc.RefreshMetadata(ctx, uuid)
				}
			}
		case "RssSync":
			// Trigger RSS sync. Run the scheduler task if available.
			if sched != nil {
				_ = sched.RunNow(ctx, "rss-sync")
			}
		default:
			// Acknowledge unknown commands gracefully — external tools
			// may send commands we don't support yet.
		}

		return &struct{ Body RadarrCommandResponse }{Body: resp}, nil
	})
}
