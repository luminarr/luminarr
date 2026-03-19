package stats_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/stats"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/testutil"
	"github.com/luminarr/luminarr/pkg/plugin"
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

func TestDecadeDistribution_empty(t *testing.T) {
	svc, _, ctx := newService(t)
	buckets, err := svc.DecadeDistribution(ctx)
	if err != nil {
		t.Fatalf("DecadeDistribution: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(buckets))
	}
}

func TestDecadeDistribution_groups(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	// Default SeedMovie year is 2010.
	testutil.SeedMovie(t, q, testutil.WithTMDBID(5001))
	testutil.SeedMovie(t, q, testutil.WithTMDBID(5002))

	buckets, err := svc.DecadeDistribution(ctx)
	if err != nil {
		t.Fatalf("DecadeDistribution: %v", err)
	}
	if len(buckets) != 1 {
		t.Fatalf("expected 1 decade bucket, got %d", len(buckets))
	}
	if buckets[0].Decade != "2010s" {
		t.Errorf("Decade = %q, want 2010s", buckets[0].Decade)
	}
	if buckets[0].Count != 2 {
		t.Errorf("Count = %d, want 2", buckets[0].Count)
	}
}

func TestLibraryGrowth_empty(t *testing.T) {
	svc, _, ctx := newService(t)
	points, err := svc.LibraryGrowth(ctx)
	if err != nil {
		t.Fatalf("LibraryGrowth: %v", err)
	}
	if len(points) != 0 {
		t.Errorf("expected 0 points, got %d", len(points))
	}
}

func TestLibraryGrowth_cumulative(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	// Seed movies — they'll all have the same added_at month.
	testutil.SeedMovie(t, q, testutil.WithTMDBID(6001))
	testutil.SeedMovie(t, q, testutil.WithTMDBID(6002))

	points, err := svc.LibraryGrowth(ctx)
	if err != nil {
		t.Fatalf("LibraryGrowth: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 month point, got %d", len(points))
	}
	if points[0].Added != 2 {
		t.Errorf("Added = %d, want 2", points[0].Added)
	}
	if points[0].Cumulative != 2 {
		t.Errorf("Cumulative = %d, want 2", points[0].Cumulative)
	}
}

func TestGenreDistribution_empty(t *testing.T) {
	svc, _, ctx := newService(t)
	buckets, err := svc.GenreDistribution(ctx)
	if err != nil {
		t.Fatalf("GenreDistribution: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(buckets))
	}
}

func TestGenreDistribution_counts(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	// Default SeedMovie genres: ["Action","Sci-Fi"]
	testutil.SeedMovie(t, q, testutil.WithTMDBID(7001))
	testutil.SeedMovie(t, q, testutil.WithTMDBID(7002))

	buckets, err := svc.GenreDistribution(ctx)
	if err != nil {
		t.Fatalf("GenreDistribution: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 genre buckets, got %d", len(buckets))
	}
	// Both should have count=2 (both movies have Action and Sci-Fi).
	for _, b := range buckets {
		if b.Count != 2 {
			t.Errorf("genre %q: Count = %d, want 2", b.Genre, b.Count)
		}
	}
}

func TestPruneSnapshots(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	// Seed a movie + file so snapshots have data.
	m := testutil.SeedMovie(t, q, testutil.WithTMDBID(8001))
	if err := movieSvc.AttachFile(ctx, m.ID, "/movies/z.mkv", 1_000_000_000, plugin.Quality{}); err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	if err := svc.TakeSnapshot(ctx); err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	// Prune anything older than 0 (should remove everything since snapshot is now-ish).
	// Use a very long duration so nothing gets pruned.
	if err := svc.PruneSnapshots(ctx, 365*24*time.Hour); err != nil {
		t.Fatalf("PruneSnapshots: %v", err)
	}
	trend, _ := svc.StorageTrend(ctx, 10)
	if len(trend) != 1 {
		t.Errorf("expected 1 snapshot after prune with long duration, got %d", len(trend))
	}

	// Now prune with zero duration (everything older than now).
	if err := svc.PruneSnapshots(ctx, 0); err != nil {
		t.Fatalf("PruneSnapshots(0): %v", err)
	}
	trend, _ = svc.StorageTrend(ctx, 10)
	if len(trend) != 0 {
		t.Errorf("expected 0 snapshots after prune with 0 duration, got %d", len(trend))
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

func TestMovieIDsByQualityTier(t *testing.T) {
	q := testutil.NewTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := stats.NewService(q, movieSvc)

	// Seed three movies with different qualities.
	m1080BluRay := testutil.SeedMovie(t, q, testutil.WithTMDBID(9001))
	m1080WebDL := testutil.SeedMovie(t, q, testutil.WithTMDBID(9002))
	m720WebDL := testutil.SeedMovie(t, q, testutil.WithTMDBID(9003))
	// A movie with no file — should not appear in any result.
	_ = testutil.SeedMovie(t, q, testutil.WithTMDBID(9004))

	if err := movieSvc.AttachFile(ctx, m1080BluRay.ID, "/movies/a.mkv", 8_000_000_000, plugin.Quality{
		Resolution: "1080p", Source: "Bluray", Codec: "x265",
	}); err != nil {
		t.Fatalf("AttachFile m1080BluRay: %v", err)
	}
	if err := movieSvc.AttachFile(ctx, m1080WebDL.ID, "/movies/b.mkv", 4_000_000_000, plugin.Quality{
		Resolution: "1080p", Source: "WebDL", Codec: "x264",
	}); err != nil {
		t.Fatalf("AttachFile m1080WebDL: %v", err)
	}
	if err := movieSvc.AttachFile(ctx, m720WebDL.ID, "/movies/c.mkv", 2_000_000_000, plugin.Quality{
		Resolution: "720p", Source: "WebDL", Codec: "x264",
	}); err != nil {
		t.Fatalf("AttachFile m720WebDL: %v", err)
	}

	t.Run("filter by resolution only", func(t *testing.T) {
		ids, err := svc.MovieIDsByQualityTier(ctx, "1080p", "")
		if err != nil {
			t.Fatalf("MovieIDsByQualityTier: %v", err)
		}
		if len(ids) != 2 {
			t.Fatalf("expected 2 movie IDs for 1080p, got %d", len(ids))
		}
		idSet := toSet(ids)
		if !idSet[m1080BluRay.ID] {
			t.Errorf("expected m1080BluRay (%s) in results", m1080BluRay.ID)
		}
		if !idSet[m1080WebDL.ID] {
			t.Errorf("expected m1080WebDL (%s) in results", m1080WebDL.ID)
		}
	})

	t.Run("filter by source only", func(t *testing.T) {
		ids, err := svc.MovieIDsByQualityTier(ctx, "", "WebDL")
		if err != nil {
			t.Fatalf("MovieIDsByQualityTier: %v", err)
		}
		if len(ids) != 2 {
			t.Fatalf("expected 2 movie IDs for WebDL, got %d", len(ids))
		}
		idSet := toSet(ids)
		if !idSet[m1080WebDL.ID] {
			t.Errorf("expected m1080WebDL (%s) in results", m1080WebDL.ID)
		}
		if !idSet[m720WebDL.ID] {
			t.Errorf("expected m720WebDL (%s) in results", m720WebDL.ID)
		}
	})

	t.Run("filter by both resolution and source — intersection", func(t *testing.T) {
		ids, err := svc.MovieIDsByQualityTier(ctx, "1080p", "Bluray")
		if err != nil {
			t.Fatalf("MovieIDsByQualityTier: %v", err)
		}
		if len(ids) != 1 {
			t.Fatalf("expected 1 movie ID for 1080p+Bluray, got %d", len(ids))
		}
		if ids[0] != m1080BluRay.ID {
			t.Errorf("expected m1080BluRay (%s), got %s", m1080BluRay.ID, ids[0])
		}
	})

	t.Run("empty filters — returns all movies with files", func(t *testing.T) {
		ids, err := svc.MovieIDsByQualityTier(ctx, "", "")
		if err != nil {
			t.Fatalf("MovieIDsByQualityTier: %v", err)
		}
		if len(ids) != 3 {
			t.Fatalf("expected 3 movie IDs with empty filters, got %d", len(ids))
		}
	})

	t.Run("no matches — returns empty slice", func(t *testing.T) {
		ids, err := svc.MovieIDsByQualityTier(ctx, "2160p", "")
		if err != nil {
			t.Fatalf("MovieIDsByQualityTier: %v", err)
		}
		if len(ids) != 0 {
			t.Errorf("expected 0 movie IDs for 2160p, got %d", len(ids))
		}
	})
}

// toSet converts a string slice to a set for membership testing.
func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
