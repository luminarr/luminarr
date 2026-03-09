package v3

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/library"
)

func registerRootFolderRoutes(api huma.API, db *sql.DB, svc *library.Service) {
	if svc == nil {
		return
	}

	huma.Register(api, huma.Operation{
		OperationID: "radarr-list-root-folders",
		Method:      http.MethodGet,
		Path:        "/api/v3/rootfolder",
		Summary:     "List root folders (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []RadarrRootFolder }, error) {
		libs, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list libraries", err)
		}

		libMap, err := buildRowIDMap(ctx, db, "libraries")
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "rowid lookup failed", err)
		}

		result := make([]RadarrRootFolder, len(libs))
		for i, lib := range libs {
			var freeSpace int64
			if stats, err := svc.Stats(ctx, lib.ID); err == nil {
				freeSpace = stats.FreeSpaceBytes
			}
			result[i] = libraryToRadarrRootFolder(lib, libMap.uuidToRow[lib.ID], freeSpace)
		}
		return &struct{ Body []RadarrRootFolder }{Body: result}, nil
	})
}
