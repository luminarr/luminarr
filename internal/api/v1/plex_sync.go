package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/plexsync"
	plexpkg "github.com/davidfic/luminarr/plugins/mediaservers/plex"
)

// ── Request / response shapes ────────────────────────────────────────────────

type plexSectionsInput struct {
	ID string `path:"id"`
}

type plexSectionsOutput struct {
	Body []plexpkg.Section
}

type plexSyncPreviewInput struct {
	ID   string `path:"id"`
	Body struct {
		SectionKey string `json:"section_key" minLength:"1"`
	}
}

type plexSyncPreviewOutput struct {
	Body *plexsync.SyncPreview
}

type plexSyncImportInput struct {
	ID   string `path:"id"`
	Body plexsync.SyncImportOptions
}

type plexSyncImportOutput struct {
	Body *plexsync.SyncImportResult
}

// ── Route registration ───────────────────────────────────────────────────────

// RegisterPlexSyncRoutes registers the Plex library sync endpoints.
func RegisterPlexSyncRoutes(api huma.API, svc *plexsync.Service) {
	// GET /api/v1/media-servers/{id}/sections
	huma.Register(api, huma.Operation{
		OperationID: "list-plex-sections",
		Method:      http.MethodGet,
		Path:        "/api/v1/media-servers/{id}/sections",
		Summary:     "List Plex movie library sections",
		Tags:        []string{"Plex Sync"},
	}, func(ctx context.Context, input *plexSectionsInput) (*plexSectionsOutput, error) {
		sections, err := svc.Sections(ctx, input.ID)
		if err != nil {
			return nil, huma.Error502BadGateway("could not list Plex sections: " + err.Error())
		}
		return &plexSectionsOutput{Body: sections}, nil
	})

	// POST /api/v1/media-servers/{id}/sync/preview
	huma.Register(api, huma.Operation{
		OperationID: "plex-sync-preview",
		Method:      http.MethodPost,
		Path:        "/api/v1/media-servers/{id}/sync/preview",
		Summary:     "Preview Plex library sync diff",
		Tags:        []string{"Plex Sync"},
	}, func(ctx context.Context, input *plexSyncPreviewInput) (*plexSyncPreviewOutput, error) {
		result, err := svc.Preview(ctx, input.ID, input.Body.SectionKey)
		if err != nil {
			return nil, huma.Error502BadGateway("plex sync preview failed: " + err.Error())
		}
		return &plexSyncPreviewOutput{Body: result}, nil
	})

	// POST /api/v1/media-servers/{id}/sync/import
	huma.Register(api, huma.Operation{
		OperationID: "plex-sync-import",
		Method:      http.MethodPost,
		Path:        "/api/v1/media-servers/{id}/sync/import",
		Summary:     "Import selected Plex movies into Luminarr",
		Tags:        []string{"Plex Sync"},
	}, func(ctx context.Context, input *plexSyncImportInput) (*plexSyncImportOutput, error) {
		result, err := svc.Import(ctx, input.Body)
		if err != nil {
			return nil, huma.Error502BadGateway("plex sync import failed: " + err.Error())
		}
		return &plexSyncImportOutput{Body: result}, nil
	})
}
