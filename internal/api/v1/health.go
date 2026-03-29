package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/health"
)

type healthCheckBody struct {
	Name    string `json:"name"`
	Status  string `json:"status"  doc:"healthy, degraded, or unhealthy"`
	Message string `json:"message"`
}

type healthReportOutput struct {
	Body *healthReportBody
}

type healthReportBody struct {
	Status string             `json:"status"  doc:"Overall: healthy, degraded, or unhealthy"`
	Checks []*healthCheckBody `json:"checks"`
}

// RegisterHealthRoutes registers the GET /api/v1/system/health endpoint.
func RegisterHealthRoutes(api huma.API, svc *health.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "get-system-health",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/health",
		Summary:     "Get system health",
		Description: "Runs live checks against library paths, download clients, and indexers.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, _ *struct{}) (*healthReportOutput, error) {
		report := svc.Check(ctx)

		checks := make([]*healthCheckBody, len(report.Checks))
		for i, c := range report.Checks {
			checks[i] = &healthCheckBody{
				Name:    c.Name,
				Status:  string(c.Status),
				Message: c.Message,
			}
		}

		return &healthReportOutput{Body: &healthReportBody{
			Status: string(report.Status),
			Checks: checks,
		}}, nil
	})
}
