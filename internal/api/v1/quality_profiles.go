package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/quality"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// ── Request / response shapes ────────────────────────────────────────────────

type qualityProfileBody struct {
	ID             string           `json:"id"              doc:"Profile UUID"`
	Name           string           `json:"name"            doc:"Human-readable profile name"`
	Cutoff         plugin.Quality   `json:"cutoff"          doc:"Minimum acceptable quality"`
	Qualities      []plugin.Quality `json:"qualities"       doc:"Ordered list of accepted qualities"`
	UpgradeAllowed bool             `json:"upgrade_allowed" doc:"Whether upgrades are permitted"`
	UpgradeUntil   *plugin.Quality  `json:"upgrade_until,omitempty" doc:"Quality ceiling for upgrades"`
}

type qualityProfileInput struct {
	Name           string           `json:"name"            doc:"Human-readable profile name"`
	Cutoff         plugin.Quality   `json:"cutoff"          doc:"Minimum acceptable quality"`
	Qualities      []plugin.Quality `json:"qualities"       doc:"Ordered list of accepted qualities"`
	UpgradeAllowed bool             `json:"upgrade_allowed" doc:"Whether upgrades are permitted"`
	UpgradeUntil   *plugin.Quality  `json:"upgrade_until,omitempty" doc:"Quality ceiling for upgrades"`
}

// Single-item output.
type qualityProfileOutput struct {
	Body *qualityProfileBody
}

// List output.
type qualityProfileListOutput struct {
	Body []*qualityProfileBody
}

// Create / update request wrapper (huma extracts Body automatically).
type qualityProfileCreateInput struct {
	Body qualityProfileInput
}

type qualityProfileUpdateInput struct {
	ID   string `path:"id"`
	Body qualityProfileInput
}

type qualityProfileGetInput struct {
	ID string `path:"id"`
}

type qualityProfileDeleteInput struct {
	ID string `path:"id"`
}

// 204 No Content — no Body field.
type qualityProfileDeleteOutput struct{}

// ── Helpers ──────────────────────────────────────────────────────────────────

func profileToBody(p quality.Profile) *qualityProfileBody {
	return &qualityProfileBody{
		ID:             p.ID,
		Name:           p.Name,
		Cutoff:         p.Cutoff,
		Qualities:      p.Qualities,
		UpgradeAllowed: p.UpgradeAllowed,
		UpgradeUntil:   p.UpgradeUntil,
	}
}

func inputToCreateRequest(in qualityProfileInput) quality.CreateRequest {
	return quality.CreateRequest{
		Name:           in.Name,
		Cutoff:         in.Cutoff,
		Qualities:      in.Qualities,
		UpgradeAllowed: in.UpgradeAllowed,
		UpgradeUntil:   in.UpgradeUntil,
	}
}

// ── Route registration ───────────────────────────────────────────────────────

// RegisterQualityProfileRoutes registers all /api/v1/quality-profiles endpoints.
func RegisterQualityProfileRoutes(api huma.API, svc *quality.Service) {
	// GET /api/v1/quality-profiles
	huma.Register(api, huma.Operation{
		OperationID: "list-quality-profiles",
		Method:      http.MethodGet,
		Path:        "/api/v1/quality-profiles",
		Summary:     "List quality profiles",
		Tags:        []string{"Quality Profiles"},
	}, func(ctx context.Context, _ *struct{}) (*qualityProfileListOutput, error) {
		profiles, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list quality profiles", err)
		}

		bodies := make([]*qualityProfileBody, len(profiles))
		for i, p := range profiles {
			bodies[i] = profileToBody(p)
		}
		return &qualityProfileListOutput{Body: bodies}, nil
	})

	// POST /api/v1/quality-profiles
	huma.Register(api, huma.Operation{
		OperationID:   "create-quality-profile",
		Method:        http.MethodPost,
		Path:          "/api/v1/quality-profiles",
		Summary:       "Create a quality profile",
		Tags:          []string{"Quality Profiles"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *qualityProfileCreateInput) (*qualityProfileOutput, error) {
		p, err := svc.Create(ctx, inputToCreateRequest(input.Body))
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to create quality profile", err)
		}
		return &qualityProfileOutput{Body: profileToBody(p)}, nil
	})

	// GET /api/v1/quality-profiles/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-quality-profile",
		Method:      http.MethodGet,
		Path:        "/api/v1/quality-profiles/{id}",
		Summary:     "Get a quality profile",
		Tags:        []string{"Quality Profiles"},
	}, func(ctx context.Context, input *qualityProfileGetInput) (*qualityProfileOutput, error) {
		p, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, quality.ErrNotFound) {
				return nil, huma.Error404NotFound("quality profile not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get quality profile", err)
		}
		return &qualityProfileOutput{Body: profileToBody(p)}, nil
	})

	// PUT /api/v1/quality-profiles/{id}
	huma.Register(api, huma.Operation{
		OperationID: "update-quality-profile",
		Method:      http.MethodPut,
		Path:        "/api/v1/quality-profiles/{id}",
		Summary:     "Update a quality profile",
		Tags:        []string{"Quality Profiles"},
	}, func(ctx context.Context, input *qualityProfileUpdateInput) (*qualityProfileOutput, error) {
		p, err := svc.Update(ctx, input.ID, inputToCreateRequest(input.Body))
		if err != nil {
			if errors.Is(err, quality.ErrNotFound) {
				return nil, huma.Error404NotFound("quality profile not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update quality profile", err)
		}
		return &qualityProfileOutput{Body: profileToBody(p)}, nil
	})

	// DELETE /api/v1/quality-profiles/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-quality-profile",
		Method:        http.MethodDelete,
		Path:          "/api/v1/quality-profiles/{id}",
		Summary:       "Delete a quality profile",
		Tags:          []string{"Quality Profiles"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *qualityProfileDeleteInput) (*qualityProfileDeleteOutput, error) {
		err := svc.Delete(ctx, input.ID)
		if err != nil {
			if errors.Is(err, quality.ErrNotFound) {
				return nil, huma.Error404NotFound("quality profile not found")
			}
			if errors.Is(err, quality.ErrInUse) {
				return nil, huma.Error409Conflict("quality profile is in use by one or more movies or libraries")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete quality profile", err)
		}
		return &qualityProfileDeleteOutput{}, nil
	})
}
