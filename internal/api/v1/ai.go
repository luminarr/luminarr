package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/aicommand"
)

type aiCommandInput struct {
	Body struct {
		Text string `json:"text" minLength:"1" maxLength:"500" doc:"Natural language command text"`
	}
}

type aiCommandOutput struct {
	Body *aicommand.CommandResponse
}

type aiConfirmInput struct {
	Body struct {
		ActionID string `json:"action_id" minLength:"1" doc:"Pending action ID to confirm"`
	}
}

type aiConfirmOutput struct {
	Body *aicommand.CommandResponse
}

// RegisterAIRoutes registers the /api/v1/ai/* endpoints.
func RegisterAIRoutes(api huma.API, svc *aicommand.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "ai-command",
		Method:      http.MethodPost,
		Path:        "/api/v1/ai/command",
		Summary:     "Process an AI command",
		Description: "Interprets natural language text and returns a structured action.",
		Tags:        []string{"AI"},
	}, func(ctx context.Context, input *aiCommandInput) (*aiCommandOutput, error) {
		resp, err := svc.ProcessCommand(ctx, input.Body.Text)
		if err != nil {
			if errors.Is(err, aicommand.ErrRateLimited) {
				return nil, huma.NewError(http.StatusTooManyRequests, err.Error())
			}
			if errors.Is(err, aicommand.ErrNotConfigured) {
				return nil, huma.NewError(http.StatusServiceUnavailable, err.Error())
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to process command", err)
		}
		return &aiCommandOutput{Body: resp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "ai-confirm",
		Method:      http.MethodPost,
		Path:        "/api/v1/ai/command/confirm",
		Summary:     "Confirm a pending AI action",
		Description: "Executes a state-modifying action that was previously returned with requires_confirmation.",
		Tags:        []string{"AI"},
	}, func(ctx context.Context, input *aiConfirmInput) (*aiConfirmOutput, error) {
		resp, err := svc.ConfirmAction(ctx, input.Body.ActionID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, err.Error())
		}
		return &aiConfirmOutput{Body: resp}, nil
	})
}
