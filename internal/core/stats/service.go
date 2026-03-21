// Package stats provides library statistics and analytics.
package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// CollectionStats is a summary of the movie library.
type CollectionStats struct {
	TotalMovies       int64 `json:"total_movies"`
	Monitored         int64 `json:"monitored"`
	WithFile          int64 `json:"with_file"`
	Missing           int64 `json:"missing"`
	NeedsUpgrade      int64 `json:"needs_upgrade"`
	EditionMismatches int64 `json:"edition_mismatches"`
	RecentlyAdded     int64 `json:"recently_added"`
}

// QualityBucket is one slice of the quality distribution.
type QualityBucket struct {
	Resolution string `json:"resolution"`
	Source     string `json:"source"`
	Codec      string `json:"codec"`
	HDR        string `json:"hdr"`
	Count      int64  `json:"count"`
}

// QualityTier is a resolution+source group with a properly deduplicated movie count.
type QualityTier struct {
	Resolution string `json:"resolution"`
	Source     string `json:"source"`
	Count      int64  `json:"count"`
}

// StorageStat is the current total storage used by movie files.
type StorageStat struct {
	TotalBytes int64 `json:"total_bytes"`
	FileCount  int64 `json:"file_count"`
}

// StoragePoint is one point in the storage trend history.
type StoragePoint struct {
	CapturedAt time.Time `json:"captured_at"`
	TotalBytes int64     `json:"total_bytes"`
	FileCount  int64     `json:"file_count"`
}

// GrabStats is aggregate information about the grab history.
type GrabStats struct {
	TotalGrabs  int64   `json:"total_grabs"`
	Successful  int64   `json:"successful"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// IndexerStat is per-indexer grab performance.
type IndexerStat struct {
	IndexerID   string  `json:"indexer_id"`
	IndexerName string  `json:"indexer_name"`
	GrabCount   int64   `json:"grab_count"`
	SuccessRate float64 `json:"success_rate"`
}

// Service provides library statistics.
type Service struct {
	q             dbsqlite.Querier
	cutoffCounter cutoffUnmetLister
}

// cutoffUnmetLister is implemented by *movie.Service. Using a local interface
// avoids an import cycle between stats and movie packages.
type cutoffUnmetLister interface {
	CountCutoffUnmet(ctx context.Context) (int64, error)
}

// NewService creates a new statistics Service.
func NewService(q dbsqlite.Querier, cutoff cutoffUnmetLister) *Service {
	return &Service{q: q, cutoffCounter: cutoff}
}

// Collection returns aggregate counts for the movie library.
func (s *Service) Collection(ctx context.Context) (CollectionStats, error) {
	row, err := s.q.GetCollectionStats(ctx)
	if err != nil {
		return CollectionStats{}, fmt.Errorf("getting collection stats: %w", err)
	}

	var needsUpgrade int64
	if s.cutoffCounter != nil {
		needsUpgrade, err = s.cutoffCounter.CountCutoffUnmet(ctx)
		if err != nil {
			// Non-fatal — degrade gracefully.
			needsUpgrade = 0
		}
	}

	var editionMismatches int64
	if cnt, edErr := s.q.CountEditionMismatches(ctx); edErr == nil {
		editionMismatches = cnt
	}

	return CollectionStats{
		TotalMovies:       row.TotalMovies,
		Monitored:         derefFloat(row.Monitored),
		WithFile:          derefFloat(row.WithFile),
		Missing:           derefFloat(row.Missing),
		NeedsUpgrade:      needsUpgrade,
		EditionMismatches: editionMismatches,
		RecentlyAdded:     derefFloat(row.RecentlyAdded),
	}, nil
}

// QualityDistribution returns movie file counts grouped by quality dimensions.
// Quality JSON is decoded in Go to avoid SQLite JSON function limitations.
func (s *Service) QualityDistribution(ctx context.Context) ([]QualityBucket, error) {
	rows, err := s.q.ListMovieFileQualitiesWithIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing file qualities: %w", err)
	}

	type key struct {
		Resolution string
		Source     string
		Codec      string
		HDR        string
	}
	// Count unique movies per quality bucket, not files.
	// A movie with multiple files at the same quality is counted once.
	movieSets := make(map[key]map[string]bool)

	for _, row := range rows {
		var q plugin.Quality
		if err := json.Unmarshal([]byte(row.QualityJson), &q); err != nil {
			continue
		}
		res := string(q.Resolution)
		if res == "" {
			res = "unknown"
		}
		src := string(q.Source)
		if src == "" {
			src = "unknown"
		}
		codec := string(q.Codec)
		if codec == "" {
			codec = "unknown"
		}
		hdr := string(q.HDR)
		if hdr == "" {
			hdr = "none"
		}
		k := key{res, src, codec, hdr}
		if movieSets[k] == nil {
			movieSets[k] = make(map[string]bool)
		}
		movieSets[k][row.MovieID] = true
	}

	buckets := make([]QualityBucket, 0, len(movieSets))
	for k, movies := range movieSets {
		buckets = append(buckets, QualityBucket{
			Resolution: k.Resolution,
			Source:     k.Source,
			Codec:      k.Codec,
			HDR:        k.HDR,
			Count:      int64(len(movies)),
		})
	}
	return buckets, nil
}

// MovieIDsByQualityTier returns movie IDs that have ANY file matching the
// given resolution and/or source. This is consistent with QualityDistribution
// which counts every file, not just the best file per movie.
// Empty filter values match any value.
func (s *Service) MovieIDsByQualityTier(ctx context.Context, resolution, source string) ([]string, error) {
	rows, err := s.q.ListMovieFileQualitiesWithIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing file qualities with IDs: %w", err)
	}

	// Collect unique movie IDs that have any file matching the filter.
	matched := make(map[string]bool)

	for _, row := range rows {
		var q plugin.Quality
		if err := json.Unmarshal([]byte(row.QualityJson), &q); err != nil {
			continue
		}
		// Apply the same normalization as QualityDistribution.
		res := string(q.Resolution)
		if res == "" {
			res = "unknown"
		}
		src := string(q.Source)
		if src == "" {
			src = "unknown"
		}
		if resolution != "" && res != resolution {
			continue
		}
		if source != "" && src != source {
			continue
		}
		matched[row.MovieID] = true
	}

	ids := make([]string, 0, len(matched))
	for id := range matched {
		ids = append(ids, id)
	}
	return ids, nil
}

// QualityTiers returns unique movie counts grouped by resolution+source.
// Each movie is counted once per tier even if it has multiple files at that tier.
func (s *Service) QualityTiers(ctx context.Context) ([]QualityTier, error) {
	rows, err := s.q.ListMovieFileQualitiesWithIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing file qualities: %w", err)
	}

	type tierKey struct{ resolution, source string }
	tierMovies := make(map[tierKey]map[string]bool)

	for _, row := range rows {
		var q plugin.Quality
		if err := json.Unmarshal([]byte(row.QualityJson), &q); err != nil {
			continue
		}
		res := string(q.Resolution)
		if res == "" {
			res = "unknown"
		}
		src := string(q.Source)
		if src == "" {
			src = "unknown"
		}
		k := tierKey{res, src}
		if tierMovies[k] == nil {
			tierMovies[k] = make(map[string]bool)
		}
		tierMovies[k][row.MovieID] = true
	}

	tiers := make([]QualityTier, 0, len(tierMovies))
	for k, movies := range tierMovies {
		tiers = append(tiers, QualityTier{
			Resolution: k.resolution,
			Source:     k.source,
			Count:      int64(len(movies)),
		})
	}
	return tiers, nil
}

// Storage returns the current total bytes and file count.
func (s *Service) Storage(ctx context.Context) (StorageStat, error) {
	row, err := s.q.GetStorageTotals(ctx)
	if err != nil {
		return StorageStat{}, fmt.Errorf("getting storage totals: %w", err)
	}
	return StorageStat{
		TotalBytes: toInt64(row.TotalBytes),
		FileCount:  row.FileCount,
	}, nil
}

// StorageTrend returns the most recent n storage snapshots, oldest first.
func (s *Service) StorageTrend(ctx context.Context, limit int) ([]StoragePoint, error) {
	rows, err := s.q.ListStorageSnapshots(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("listing storage snapshots: %w", err)
	}
	// Rows come back newest-first; reverse for chronological order.
	points := make([]StoragePoint, len(rows))
	for i, r := range rows {
		points[len(rows)-1-i] = StoragePoint{
			CapturedAt: r.CapturedAt,
			TotalBytes: r.TotalBytes,
			FileCount:  r.FileCount,
		}
	}
	return points, nil
}

// GrabPerformance returns aggregate grab stats and per-indexer breakdown.
func (s *Service) GrabPerformance(ctx context.Context) (GrabStats, []IndexerStat, error) {
	gr, err := s.q.GetGrabStats(ctx)
	if err != nil {
		return GrabStats{}, nil, fmt.Errorf("getting grab stats: %w", err)
	}
	successful := derefFloat(gr.Successful)
	failed := derefFloat(gr.Failed)

	var rate float64
	if gr.TotalGrabs > 0 {
		rate = float64(successful) / float64(gr.TotalGrabs)
	}

	grabStats := GrabStats{
		TotalGrabs:  gr.TotalGrabs,
		Successful:  successful,
		Failed:      failed,
		SuccessRate: rate,
	}

	indexerRows, err := s.q.GetTopIndexers(ctx)
	if err != nil {
		return grabStats, nil, fmt.Errorf("getting top indexers: %w", err)
	}

	indexers := make([]IndexerStat, len(indexerRows))
	for i, r := range indexerRows {
		idxID := ""
		if r.IndexerID != nil {
			idxID = *r.IndexerID
		}
		successes := derefFloat(r.SuccessCount)
		var idxRate float64
		if r.GrabCount > 0 {
			idxRate = float64(successes) / float64(r.GrabCount)
		}
		indexers[i] = IndexerStat{
			IndexerID:   idxID,
			IndexerName: r.IndexerName,
			GrabCount:   r.GrabCount,
			SuccessRate: idxRate,
		}
	}

	return grabStats, indexers, nil
}

// DecadeBucket is a movie count for one decade.
type DecadeBucket struct {
	Decade string `json:"decade"` // e.g. "1990s"
	Count  int64  `json:"count"`
}

// GrowthPoint is the number of movies added in a calendar month.
type GrowthPoint struct {
	Month      string `json:"month"`      // "YYYY-MM"
	Added      int64  `json:"added"`      // new movies that month
	Cumulative int64  `json:"cumulative"` // running total
}

// GenreBucket is a movie count for one genre.
type GenreBucket struct {
	Genre string `json:"genre"`
	Count int64  `json:"count"`
}

// DecadeDistribution returns movie counts grouped by decade.
func (s *Service) DecadeDistribution(ctx context.Context) ([]DecadeBucket, error) {
	rows, err := s.q.GetMovieYearDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting year distribution: %w", err)
	}
	totals := make(map[int]int64)
	for _, r := range rows {
		decade := (int(r.Year) / 10) * 10
		totals[decade] += r.Count
	}
	buckets := make([]DecadeBucket, 0, len(totals))
	for decade, count := range totals {
		buckets = append(buckets, DecadeBucket{
			Decade: fmt.Sprintf("%ds", decade),
			Count:  count,
		})
	}
	// Sort chronologically.
	for i := 1; i < len(buckets); i++ {
		for j := i; j > 0 && buckets[j].Decade < buckets[j-1].Decade; j-- {
			buckets[j], buckets[j-1] = buckets[j-1], buckets[j]
		}
	}
	return buckets, nil
}

// LibraryGrowth returns movies-added-per-month with a running cumulative total.
func (s *Service) LibraryGrowth(ctx context.Context) ([]GrowthPoint, error) {
	rows, err := s.q.GetMoviesAddedByMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting monthly growth: %w", err)
	}
	points := make([]GrowthPoint, len(rows))
	var cumulative int64
	for i, r := range rows {
		month := ""
		if s, ok := r.Month.(string); ok {
			month = s
		}
		cumulative += r.Count
		points[i] = GrowthPoint{
			Month:      month,
			Added:      r.Count,
			Cumulative: cumulative,
		}
	}
	return points, nil
}

// GenreDistribution returns the top 15 genres by movie count.
func (s *Service) GenreDistribution(ctx context.Context) ([]GenreBucket, error) {
	rows, err := s.q.ListMovieGenresJSON(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing genres: %w", err)
	}
	counts := make(map[string]int64)
	for _, raw := range rows {
		var genres []string
		if err := json.Unmarshal([]byte(raw), &genres); err != nil {
			continue
		}
		for _, g := range genres {
			if g != "" {
				counts[g]++
			}
		}
	}
	buckets := make([]GenreBucket, 0, len(counts))
	for genre, count := range counts {
		buckets = append(buckets, GenreBucket{Genre: genre, Count: count})
	}
	// Sort by count descending.
	for i := 1; i < len(buckets); i++ {
		for j := i; j > 0 && buckets[j].Count > buckets[j-1].Count; j-- {
			buckets[j], buckets[j-1] = buckets[j-1], buckets[j]
		}
	}
	const maxGenres = 15
	if len(buckets) > maxGenres {
		buckets = buckets[:maxGenres]
	}
	return buckets, nil
}

// TakeSnapshot records the current total storage as a point-in-time snapshot.
func (s *Service) TakeSnapshot(ctx context.Context) error {
	totals, err := s.q.GetStorageTotals(ctx)
	if err != nil {
		return fmt.Errorf("getting storage totals for snapshot: %w", err)
	}
	if err := s.q.InsertStorageSnapshot(ctx, dbsqlite.InsertStorageSnapshotParams{
		ID:         uuid.New().String(),
		CapturedAt: time.Now().UTC(),
		TotalBytes: toInt64(totals.TotalBytes),
		FileCount:  totals.FileCount,
	}); err != nil {
		return fmt.Errorf("inserting storage snapshot: %w", err)
	}
	return nil
}

// PruneSnapshots removes snapshots older than the given duration.
func (s *Service) PruneSnapshots(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan)
	if err := s.q.PruneOldStorageSnapshots(ctx, cutoff); err != nil {
		return fmt.Errorf("pruning snapshots: %w", err)
	}
	return nil
}

// derefFloat returns 0 for a nil *float64 and the rounded int64 otherwise.
func derefFloat(p *float64) int64 {
	if p == nil {
		return 0
	}
	return int64(*p)
}

// toInt64 converts the interface{} returned by COALESCE(SUM(...), 0) to int64.
// SQLite may return int64 or float64 depending on the driver; handle both.
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	}
	return 0
}
