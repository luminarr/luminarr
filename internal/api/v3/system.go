package v3

import (
	"context"
	"net/http"
	"runtime"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/version"
)

func registerSystemRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "radarr-system-status",
		Method:      http.MethodGet,
		Path:        "/api/v3/system/status",
		Summary:     "System status (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(_ context.Context, _ *struct{}) (*struct{ Body RadarrSystemStatus }, error) {
		return &struct{ Body RadarrSystemStatus }{Body: RadarrSystemStatus{
			AppName:        "Luminarr",
			InstanceName:   "Luminarr",
			Version:        version.Version,
			Branch:         "main",
			Authentication: "apiKey",
			URLBase:        "",
			RuntimeName:    "go",
			RuntimeVersion: runtime.Version(),
			StartupPath:    "",
			AppData:        "",
			IsDocker:       false,
			IsLinux:        runtime.GOOS == "linux",
			IsWindows:      runtime.GOOS == "windows",
			IsOsx:          runtime.GOOS == "darwin",
		}}, nil
	})
}
