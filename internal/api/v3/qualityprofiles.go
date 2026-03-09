package v3

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/quality"
)

func registerQualityProfileRoutes(api huma.API, db *sql.DB, svc *quality.Service) {
	if svc == nil {
		return
	}

	huma.Register(api, huma.Operation{
		OperationID: "radarr-list-quality-profiles",
		Method:      http.MethodGet,
		Path:        "/api/v3/qualityProfile",
		Summary:     "List quality profiles (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []RadarrQualityProfile }, error) {
		profiles, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list quality profiles", err)
		}

		qpMap, err := buildRowIDMap(ctx, db, "quality_profiles")
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "rowid lookup failed", err)
		}

		result := make([]RadarrQualityProfile, len(profiles))
		for i, p := range profiles {
			result[i] = qualityProfileToRadarr(p, qpMap.uuidToRow[p.ID])
		}
		return &struct{ Body []RadarrQualityProfile }{Body: result}, nil
	})
}
