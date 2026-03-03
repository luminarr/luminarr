package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/indexer"
)

type historyItemBody struct {
	ID                string    `json:"id"`
	MovieID           string    `json:"movie_id"`
	ReleaseTitle      string    `json:"release_title"`
	ReleaseSource     string    `json:"release_source,omitempty"`
	ReleaseResolution string    `json:"release_resolution,omitempty"`
	Protocol          string    `json:"protocol"`
	Size              int64     `json:"size"`
	DownloadStatus    string    `json:"download_status"`
	GrabbedAt         time.Time `json:"grabbed_at"`
}

type historyListInput struct {
	Limit int `query:"limit" default:"100" minimum:"1" maximum:"1000"`
}

type historyListOutput struct {
	Body []*historyItemBody
}

// RegisterHistoryRoutes registers the global grab history endpoint.
func RegisterHistoryRoutes(api huma.API, svc *indexer.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-history",
		Method:      http.MethodGet,
		Path:        "/api/v1/history",
		Summary:     "List grab history",
		Tags:        []string{"History"},
	}, func(ctx context.Context, input *historyListInput) (*historyListOutput, error) {
		limit := input.Limit
		if limit == 0 {
			limit = 100
		}
		rows, err := svc.ListHistory(ctx, limit)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list history", err)
		}
		items := make([]*historyItemBody, len(rows))
		for i, r := range rows {
			grabbedAt, _ := time.Parse(time.RFC3339, r.GrabbedAt)
			items[i] = &historyItemBody{
				ID:                r.ID,
				MovieID:           r.MovieID,
				ReleaseTitle:      r.ReleaseTitle,
				ReleaseSource:     r.ReleaseSource,
				ReleaseResolution: r.ReleaseResolution,
				Protocol:          r.Protocol,
				Size:              r.Size,
				DownloadStatus:    r.DownloadStatus,
				GrabbedAt:         grabbedAt,
			}
		}
		return &historyListOutput{Body: items}, nil
	})
}
