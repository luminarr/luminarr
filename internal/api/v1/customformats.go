package v1

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/customformat"
)

// --- request / response types ------------------------------------------------

type cfBody struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	IncludeWhenRenaming bool                         `json:"include_when_renaming"`
	Specifications      []customformat.Specification `json:"specifications"`
	CreatedAt           string                       `json:"created_at"`
	UpdatedAt           string                       `json:"updated_at"`
}

type cfCreateInput struct {
	Body struct {
		Name                string                       `json:"name" required:"true" minLength:"1" maxLength:"200"`
		IncludeWhenRenaming bool                         `json:"include_when_renaming"`
		Specifications      []customformat.Specification `json:"specifications" required:"true"`
	}
}

type cfUpdateInput struct {
	ID   string `path:"id"`
	Body struct {
		Name                string                       `json:"name" required:"true" minLength:"1" maxLength:"200"`
		IncludeWhenRenaming bool                         `json:"include_when_renaming"`
		Specifications      []customformat.Specification `json:"specifications" required:"true"`
	}
}

type cfDeleteInput struct {
	ID string `path:"id"`
}

type cfGetInput struct {
	ID string `path:"id"`
}

type cfExportInput struct {
	IDs string `query:"ids" doc:"Comma-separated list of custom format IDs. If empty, exports all."`
}

type cfImportInput struct {
	RawBody []byte
}

type conditionType struct {
	Implementation string   `json:"implementation"`
	Label          string   `json:"label"`
	Fields         []string `json:"fields"`
}

// --- helpers -----------------------------------------------------------------

func cfToBody(cf customformat.CustomFormat) cfBody {
	return cfBody{
		ID:                  cf.ID,
		Name:                cf.Name,
		IncludeWhenRenaming: cf.IncludeWhenRenaming,
		Specifications:      cf.Specifications,
		CreatedAt:           cf.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:           cf.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// --- route registration ------------------------------------------------------

// RegisterCustomFormatRoutes registers the custom format management endpoints.
func RegisterCustomFormatRoutes(api huma.API, svc *customformat.Service) {
	// GET /api/v1/custom-formats
	huma.Register(api, huma.Operation{
		OperationID: "list-custom-formats",
		Method:      http.MethodGet,
		Path:        "/api/v1/custom-formats",
		Summary:     "List all custom formats",
		Tags:        []string{"Custom Formats"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []cfBody }, error) {
		formats, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list custom formats", err)
		}
		items := make([]cfBody, len(formats))
		for i, cf := range formats {
			items[i] = cfToBody(cf)
		}
		return &struct{ Body []cfBody }{Body: items}, nil
	})

	// POST /api/v1/custom-formats
	huma.Register(api, huma.Operation{
		OperationID:   "create-custom-format",
		Method:        http.MethodPost,
		Path:          "/api/v1/custom-formats",
		Summary:       "Create a custom format",
		Tags:          []string{"Custom Formats"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *cfCreateInput) (*struct{ Body cfBody }, error) {
		cf, err := svc.Create(ctx, customformat.CreateRequest{
			Name:                input.Body.Name,
			IncludeWhenRenaming: input.Body.IncludeWhenRenaming,
			Specifications:      input.Body.Specifications,
		})
		if err != nil {
			return nil, huma.NewError(http.StatusConflict, err.Error())
		}
		return &struct{ Body cfBody }{Body: cfToBody(cf)}, nil
	})

	// GET /api/v1/custom-formats/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-custom-format",
		Method:      http.MethodGet,
		Path:        "/api/v1/custom-formats/{id}",
		Summary:     "Get a custom format",
		Tags:        []string{"Custom Formats"},
	}, func(ctx context.Context, input *cfGetInput) (*struct{ Body cfBody }, error) {
		cf, err := svc.Get(ctx, input.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusNotFound, "custom format not found", err)
		}
		return &struct{ Body cfBody }{Body: cfToBody(cf)}, nil
	})

	// PUT /api/v1/custom-formats/{id}
	huma.Register(api, huma.Operation{
		OperationID: "update-custom-format",
		Method:      http.MethodPut,
		Path:        "/api/v1/custom-formats/{id}",
		Summary:     "Update a custom format",
		Tags:        []string{"Custom Formats"},
	}, func(ctx context.Context, input *cfUpdateInput) (*struct{ Body cfBody }, error) {
		cf, err := svc.Update(ctx, input.ID, customformat.UpdateRequest{
			Name:                input.Body.Name,
			IncludeWhenRenaming: input.Body.IncludeWhenRenaming,
			Specifications:      input.Body.Specifications,
		})
		if err != nil {
			return nil, huma.NewError(http.StatusConflict, err.Error())
		}
		return &struct{ Body cfBody }{Body: cfToBody(cf)}, nil
	})

	// DELETE /api/v1/custom-formats/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-custom-format",
		Method:        http.MethodDelete,
		Path:          "/api/v1/custom-formats/{id}",
		Summary:       "Delete a custom format",
		Tags:          []string{"Custom Formats"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *cfDeleteInput) (*struct{}, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete custom format", err)
		}
		return nil, nil
	})

	// POST /api/v1/custom-formats/import
	huma.Register(api, huma.Operation{
		OperationID:   "import-custom-formats",
		Method:        http.MethodPost,
		Path:          "/api/v1/custom-formats/import",
		Summary:       "Import custom formats from TRaSH JSON",
		Tags:          []string{"Custom Formats"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *cfImportInput) (*struct{ Body []cfBody }, error) {
		created, err := svc.Import(ctx, input.RawBody)
		if err != nil {
			items := make([]cfBody, len(created))
			for i, cf := range created {
				items[i] = cfToBody(cf)
			}
			return &struct{ Body []cfBody }{Body: items}, huma.NewError(http.StatusBadRequest, err.Error())
		}
		items := make([]cfBody, len(created))
		for i, cf := range created {
			items[i] = cfToBody(cf)
		}
		return &struct{ Body []cfBody }{Body: items}, nil
	})

	// GET /api/v1/custom-formats/export
	huma.Register(api, huma.Operation{
		OperationID: "export-custom-formats",
		Method:      http.MethodGet,
		Path:        "/api/v1/custom-formats/export",
		Summary:     "Export custom formats as TRaSH-compatible JSON",
		Tags:        []string{"Custom Formats"},
	}, func(ctx context.Context, input *cfExportInput) (*struct{ Body []byte }, error) {
		var ids []string
		if input.IDs != "" {
			ids = strings.Split(input.IDs, ",")
		}
		data, err := svc.Export(ctx, ids)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "export failed", err)
		}
		return &struct{ Body []byte }{Body: data}, nil
	})

	// GET /api/v1/custom-formats/schema
	huma.Register(api, huma.Operation{
		OperationID: "custom-format-schema",
		Method:      http.MethodGet,
		Path:        "/api/v1/custom-formats/schema",
		Summary:     "Return available condition types and their fields",
		Tags:        []string{"Custom Formats"},
	}, func(_ context.Context, _ *struct{}) (*struct{ Body []conditionType }, error) {
		schema := []conditionType{
			{Implementation: "release_title", Label: "Release Title", Fields: []string{"value"}},
			{Implementation: "edition", Label: "Edition", Fields: []string{"value"}},
			{Implementation: "language", Label: "Language", Fields: []string{"value"}},
			{Implementation: "indexer_flag", Label: "Indexer Flag", Fields: []string{"value"}},
			{Implementation: "source", Label: "Source", Fields: []string{"value"}},
			{Implementation: "resolution", Label: "Resolution", Fields: []string{"value"}},
			{Implementation: "quality_modifier", Label: "Quality Modifier", Fields: []string{"value"}},
			{Implementation: "size", Label: "Size", Fields: []string{"min", "max"}},
			{Implementation: "release_group", Label: "Release Group", Fields: []string{"value"}},
			{Implementation: "year", Label: "Year", Fields: []string{"min", "max"}},
		}
		return &struct{ Body []conditionType }{Body: schema}, nil
	})
}
