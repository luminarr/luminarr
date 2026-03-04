package stats_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/core/stats"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/testutil"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func newService(t *testing.T) (*stats.Service, *movie.Service, context.Context) {
	t.Helper()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)
	return svc, movieSvc, context.Background()
}

func TestCollectionStats_empty(t *testing.T) {
	svc, _, ctx := newService(t)
	c, err := svc.Collection(ctx)
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if c.TotalMovies != 0 {
		t.Errorf("expected 0 total movies, got %d", c.TotalMovies)
	}
}

func TestCollectionStats_counts(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	// Seed two monitored movies, one with a file.
	withFile := testutil.SeedMovie(t, q, testutil.WithTMDBID(1001))
	_ = testutil.SeedMovie(t, q, testutil.WithTMDBID(1002))

	if err := movieSvc.AttachFile(ctx, withFile.ID, "/movies/test.mkv", 1_000_000_000, plugin.Quality{Resolution: "1080p"}); err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	c, err := svc.Collection(ctx)
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if c.TotalMovies != 2 {
		t.Errorf("TotalMovies: got %d, want 2", c.TotalMovies)
	}
	if c.WithFile != 1 {
		t.Errorf("WithFile: got %d, want 1", c.WithFile)
	}
	if c.Missing != 1 {
		t.Errorf("Missing: got %d, want 1", c.Missing)
	}
}

func TestQualityDistribution_empty(t *testing.T) {
	svc, _, ctx := newService(t)
	buckets, err := svc.QualityDistribution(ctx)
	if err != nil {
		t.Fatalf("QualityDistribution: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(buckets))
	}
}

func TestQualityDistribution_buckets(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	m1 := testutil.SeedMovie(t, q, testutil.WithTMDBID(2001))
	m2 := testutil.SeedMovie(t, q, testutil.WithTMDBID(2002))

	if err := movieSvc.AttachFile(ctx, m1.ID, "/movies/a.mkv", 1_000_000_000, plugin.Quality{
		Resolution: "1080p", Source: "Bluray", Codec: "x265", HDR: "none",
	}); err != nil {
		t.Fatalf("AttachFile m1: %v", err)
	}
	if err := movieSvc.AttachFile(ctx, m2.ID, "/movies/b.mkv", 500_000_000, plugin.Quality{
		Resolution: "1080p", Source: "WebDL", Codec: "x264", HDR: "none",
	}); err != nil {
		t.Fatalf("AttachFile m2: %v", err)
	}

	buckets, err := svc.QualityDistribution(ctx)
	if err != nil {
		t.Fatalf("QualityDistribution: %v", err)
	}
	// Two distinct (source, codec) combos → two buckets.
	if len(buckets) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(buckets))
	}
}

func TestStorageSnapshot_roundtrip(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	m := testutil.SeedMovie(t, q, testutil.WithTMDBID(3001))
	if err := movieSvc.AttachFile(ctx, m.ID, "/movies/c.mkv", 2_000_000_000, plugin.Quality{}); err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	if err := svc.TakeSnapshot(ctx); err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	trend, err := svc.StorageTrend(ctx, 10)
	if err != nil {
		t.Fatalf("StorageTrend: %v", err)
	}
	if len(trend) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(trend))
	}
	if trend[0].TotalBytes != 2_000_000_000 {
		t.Errorf("TotalBytes: got %d, want 2000000000", trend[0].TotalBytes)
	}
	if trend[0].FileCount != 1 {
		t.Errorf("FileCount: got %d, want 1", trend[0].FileCount)
	}
}

func TestGrabStats_empty(t *testing.T) {
	svc, _, ctx := newService(t)
	gs, indexers, err := svc.GrabPerformance(ctx)
	if err != nil {
		t.Fatalf("GrabPerformance: %v", err)
	}
	if gs.TotalGrabs != 0 {
		t.Errorf("expected 0 total grabs, got %d", gs.TotalGrabs)
	}
	if len(indexers) != 0 {
		t.Errorf("expected 0 indexers, got %d", len(indexers))
	}
}

func TestStorageStat_totals(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	m1 := testutil.SeedMovie(t, q, testutil.WithTMDBID(4001))
	m2 := testutil.SeedMovie(t, q, testutil.WithTMDBID(4002))

	if err := movieSvc.AttachFile(ctx, m1.ID, "/movies/x.mkv", 3_000_000_000, plugin.Quality{}); err != nil {
		t.Fatalf("AttachFile m1: %v", err)
	}
	if err := movieSvc.AttachFile(ctx, m2.ID, "/movies/y.mkv", 2_000_000_000, plugin.Quality{}); err != nil {
		t.Fatalf("AttachFile m2: %v", err)
	}

	s, err := svc.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage: %v", err)
	}
	if s.TotalBytes != 5_000_000_000 {
		t.Errorf("TotalBytes: got %d, want 5000000000", s.TotalBytes)
	}
	if s.FileCount != 2 {
		t.Errorf("FileCount: got %d, want 2", s.FileCount)
	}
}
