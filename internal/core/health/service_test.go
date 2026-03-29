package health

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/ratelimit"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/testutil"
	"github.com/luminarr/luminarr/pkg/plugin"
)

func TestCheckPathAccessible_Valid(t *testing.T) {
	dir := t.TempDir()
	// Create a file so the directory is non-empty.
	if err := os.WriteFile(dir+"/test.txt", []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkPathAccessible(dir); err != nil {
		t.Errorf("checkPathAccessible(%q) = %v, want nil", dir, err)
	}
}

func TestCheckPathAccessible_Empty(t *testing.T) {
	dir := t.TempDir()
	err := checkPathAccessible(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestCheckPathAccessible_NonExistent(t *testing.T) {
	err := checkPathAccessible("/this/path/does/not/exist-xyz")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestJoinIssues(t *testing.T) {
	tests := []struct {
		name   string
		issues []string
		want   string
	}{
		{"empty", nil, ""},
		{"single", []string{"disk full"}, "disk full"},
		{"multiple", []string{"disk full", "client down"}, "disk full; client down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinIssues(tt.issues)
			if got != tt.want {
				t.Errorf("joinIssues(%v) = %q, want %q", tt.issues, got, tt.want)
			}
		})
	}
}

func TestOverallStatus_Aggregation(t *testing.T) {
	tests := []struct {
		name   string
		checks []CheckResult
		want   Status
	}{
		{
			"all healthy",
			[]CheckResult{
				{Status: StatusHealthy},
				{Status: StatusHealthy},
			},
			StatusHealthy,
		},
		{
			"one degraded",
			[]CheckResult{
				{Status: StatusHealthy},
				{Status: StatusDegraded},
			},
			StatusDegraded,
		},
		{
			"one unhealthy trumps degraded",
			[]CheckResult{
				{Status: StatusDegraded},
				{Status: StatusUnhealthy},
				{Status: StatusHealthy},
			},
			StatusUnhealthy,
		},
		{
			"empty checks",
			[]CheckResult{},
			StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the aggregation logic from Check() to test it in isolation.
			overall := StatusHealthy
			for _, c := range tt.checks {
				if c.Status == StatusUnhealthy {
					overall = StatusUnhealthy
					break
				}
				if c.Status == StatusDegraded && overall != StatusUnhealthy {
					overall = StatusDegraded
				}
			}
			if overall != tt.want {
				t.Errorf("overall = %q, want %q", overall, tt.want)
			}
		})
	}
}


func TestCheck_NoLibrariesNoClientsNoIndexers(t *testing.T) {
	q := testutil.NewTestDB(t)
	logger := slog.Default()
	bus := events.New(logger)
	reg := registry.New()

	libSvc := library.NewService(q, bus, nil)
	dlSvc := downloader.NewService(q, reg, bus)
	idxSvc := indexer.NewService(q, reg, bus, ratelimit.New())

	svc := NewService(libSvc, dlSvc, idxSvc, logger)
	report := svc.Check(context.Background())

	if report.Status != StatusHealthy {
		t.Errorf("status = %q, want %q (empty system should be healthy)", report.Status, StatusHealthy)
	}
	if len(report.Checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(report.Checks))
	}
	// All checks should be healthy when nothing is configured.
	for _, c := range report.Checks {
		if c.Status != StatusHealthy {
			t.Errorf("check %q = %q, want %q", c.Name, c.Status, StatusHealthy)
		}
	}
}

func TestCheck_WithLibrary_LibraryPaths(t *testing.T) {
	q := testutil.NewTestDB(t)
	logger := slog.Default()
	bus := events.New(logger)
	reg := registry.New()

	qualSvc := quality.NewService(q, bus)
	// Create a quality profile so we can create a library.
	cutoff := plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceWEBDL, Codec: plugin.CodecX264, HDR: plugin.HDRNone}
	profile, err := qualSvc.Create(context.Background(), quality.CreateRequest{
		Name:      "Test",
		Cutoff:    cutoff,
		Qualities: []plugin.Quality{cutoff},
	})
	if err != nil {
		t.Fatal(err)
	}

	libSvc := library.NewService(q, bus, nil)
	// Create a library pointing at a temp dir with a file so path check works.
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/movie.mkv", []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = libSvc.Create(context.Background(), library.CreateRequest{
		Name:                    "Test Lib",
		RootPath:                dir,
		DefaultQualityProfileID: profile.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	dlSvc := downloader.NewService(q, reg, bus)
	idxSvc := indexer.NewService(q, reg, bus, ratelimit.New())

	svc := NewService(libSvc, dlSvc, idxSvc, logger)
	report := svc.Check(context.Background())

	// Find the library_paths check.
	var pathCheck *CheckResult
	for i := range report.Checks {
		if report.Checks[i].Name == "library_paths" {
			pathCheck = &report.Checks[i]
			break
		}
	}
	if pathCheck == nil {
		t.Fatal("library_paths check not found in report")
	}
	if pathCheck.Status != StatusHealthy {
		t.Errorf("library_paths = %q (%s), want healthy", pathCheck.Status, pathCheck.Message)
	}
}
