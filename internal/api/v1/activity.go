package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/activity"
)

type listActivitiesInput struct {
	Category *string `query:"category" doc:"Filter by category: grab, import, task, health, movie"`
	Since    *string `query:"since"    doc:"Only return activities after this ISO 8601 timestamp"`
	Limit    int64   `query:"limit"    doc:"Max results (default 100, max 500)" default:"100"`
}

type listActivitiesOutput struct {
	Body *activity.ListResult
}

// RegisterActivityRoutes registers the /api/v1/activity endpoints.
func RegisterActivityRoutes(api huma.API, svc *activity.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-activity",
		Method:      http.MethodGet,
		Path:        "/api/v1/activity",
		Summary:     "List activity log",
		Description: "Returns a chronological feed of system events: grabs, imports, task runs, health changes, and movie additions.",
		Tags:        []string{"Activity"},
	}, func(ctx context.Context, input *listActivitiesInput) (*listActivitiesOutput, error) {
		if input.Category != nil && !activity.ValidCategory(*input.Category) {
			return nil, huma.NewError(http.StatusBadRequest, "invalid category: must be one of grab, import, task, health, movie")
		}

		result, err := svc.List(ctx, input.Category, input.Since, input.Limit)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list activities", err)
		}

		return &listActivitiesOutput{Body: result}, nil
	})
}
