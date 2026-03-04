package movie_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/core/renamer"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/testutil"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// defaultRenameSettings uses the default file format and delete-colon strategy.
var defaultRenameSettings = movie.RenameSettings{
	Format:           renamer.DefaultFileFormat,
	ColonReplacement: renamer.ColonDelete,
}

// newRenameTestService seeds a movie with a file at tmpPath. The file is named
// with an arbitrary "old" filename so that a rename to standard format produces
// a different name.
func newRenameTestService(t *testing.T, tmpPath string) (*movie.Service, string) {
	t.Helper()
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	seeded := testutil.SeedMovie(t, q)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(tmpPath, []byte("fake video"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := svc.AttachFile(ctx, seeded.ID, tmpPath, int64(len("fake video")), plugin.Quality{Name: "Bluray-1080p"}); err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	return svc, seeded.ID
}

func TestRenameFiles_DryRun(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	// Use a non-standard filename so rename produces a different name.
	tmpPath := filepath.Join(dir, "inception.2010.bluray.mkv")
	svc, movieID := newRenameTestService(t, tmpPath)

	items, err := svc.RenameFiles(ctx, movieID, defaultRenameSettings, true)
	if err != nil {
		t.Fatalf("RenameFiles dry-run: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.OldPath != tmpPath {
		t.Errorf("OldPath: got %q, want %q", item.OldPath, tmpPath)
	}
	wantFilename := "Inception (2010) Bluray-1080p.mkv"
	wantNew := filepath.Join(dir, wantFilename)
	if item.NewPath != wantNew {
		t.Errorf("NewPath: got %q, want %q", item.NewPath, wantNew)
	}

	// Dry run must not touch the disk.
	if _, statErr := os.Stat(tmpPath); statErr != nil {
		t.Errorf("original file should still exist after dry-run: %v", statErr)
	}
	if _, statErr := os.Stat(wantNew); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("new path should NOT exist after dry-run")
	}
}

func TestRenameFiles_Execute(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "inception.2010.bluray.mkv")
	svc, movieID := newRenameTestService(t, tmpPath)

	items, err := svc.RenameFiles(ctx, movieID, defaultRenameSettings, false)
	if err != nil {
		t.Fatalf("RenameFiles execute: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 renamed item, got %d", len(items))
	}

	wantNew := filepath.Join(dir, "Inception (2010) Bluray-1080p.mkv")

	// New file must exist on disk.
	if _, statErr := os.Stat(wantNew); statErr != nil {
		t.Errorf("new file should exist after rename: %v", statErr)
	}
	// Old file must be gone.
	if _, statErr := os.Stat(tmpPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("old file should be gone after rename")
	}

	// DB path should be updated.
	files, err := svc.ListFiles(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != wantNew {
		t.Errorf("DB file path: got %q, want %q", files[0].Path, wantNew)
	}

	// movies.path should also be synced.
	m, err := svc.Get(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Path != wantNew {
		t.Errorf("movie.Path: got %q, want %q", m.Path, wantNew)
	}
}

func TestRenameFiles_AlreadyCorrect(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	// Already matches the standard format.
	tmpPath := filepath.Join(dir, "Inception (2010) Bluray-1080p.mkv")
	svc, movieID := newRenameTestService(t, tmpPath)

	items, err := svc.RenameFiles(ctx, movieID, defaultRenameSettings, false)
	if err != nil {
		t.Fatalf("RenameFiles: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items when file is already correctly named, got %d", len(items))
	}
}

func TestRenameFiles_NoFiles(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	seeded := testutil.SeedMovie(t, q)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	items, err := svc.RenameFiles(ctx, seeded.ID, defaultRenameSettings, false)
	if err != nil {
		t.Fatalf("expected nil error for movie with no files, got: %v", err)
	}
	if items != nil {
		t.Errorf("expected nil items for movie with no files, got %v", items)
	}
}

func TestRenameFiles_NotFound(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	_, err := svc.RenameFiles(ctx, "nonexistent-id", defaultRenameSettings, false)
	if !errors.Is(err, movie.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRenameFiles_TargetExists(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "inception.2010.mkv")
	svc, movieID := newRenameTestService(t, tmpPath)

	// Pre-create the target path to simulate a conflict.
	wantNew := filepath.Join(dir, "Inception (2010) Bluray-1080p.mkv")
	if err := os.WriteFile(wantNew, []byte("already there"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := svc.RenameFiles(ctx, movieID, defaultRenameSettings, false)
	// Should return an error because the target exists.
	if err == nil {
		t.Error("expected error when target path already exists, got nil")
	}

	// Original file should still be there (rename was skipped).
	if _, statErr := os.Stat(tmpPath); statErr != nil {
		t.Errorf("original file should still exist after skipped rename: %v", statErr)
	}
}

func TestRenameFiles_RenameError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "inception.mkv")

	q := testutil.NewTestDB(t)
	seeded := testutil.SeedMovie(t, q)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	if err := os.WriteFile(tmpPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := svc.AttachFile(ctx, seeded.ID, tmpPath, 4, plugin.Quality{Name: "Bluray-1080p"}); err != nil {
		t.Fatal(err)
	}

	// Override renameFile to simulate a disk error.
	svc.SetRenameFunc(func(_, _ string) error {
		return errors.New("permission denied")
	})

	_, err := svc.RenameFiles(ctx, seeded.ID, defaultRenameSettings, false)
	if err == nil {
		t.Error("expected error from failed rename, got nil")
	}
}
