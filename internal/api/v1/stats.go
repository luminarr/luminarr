package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/stats"
)

// ── Response bodies ───────────────────────────────────────────────────────────

type statsCollectionBody struct {
	TotalMovies   int64 `json:"total_movies"`
	Monitored     int64 `json:"monitored"`
	WithFile      int64 `json:"with_file"`
	Missing       int64 `json:"missing"`
	NeedsUpgrade  int64 `json:"needs_upgrade"`
	RecentlyAdded int64 `json:"recently_added"`
}

type statsQualityBucketBody struct {
	Resolution string `json:"resolution"`
	Source     string `json:"source"`
	Codec      string `json:"codec"`
	HDR        string `json:"hdr"`
	Count      int64  `json:"count"`
}

type statsStoragePointBody struct {
	CapturedAt time.Time `json:"captured_at"`
	TotalBytes int64     `json:"total_bytes"`
	FileCount  int64     `json:"file_count"`
}

type statsStorageBody struct {
	TotalBytes int64                   `json:"total_bytes"`
	FileCount  int64                   `json:"file_count"`
	Trend      []statsStoragePointBody `json:"trend"`
}

type statsIndexerBody struct {
	IndexerID   string  `json:"indexer_id"`
	IndexerName string  `json:"indexer_name"`
	GrabCount   int64   `json:"grab_count"`
	SuccessRate float64 `json:"success_rate"`
}

type statsGrabsBody struct {
	TotalGrabs  int64              `json:"total_grabs"`
	Successful  int64              `json:"successful"`
	Failed      int64              `json:"failed"`
	SuccessRate float64            `json:"success_rate"`
	TopIndexers []statsIndexerBody `json:"top_indexers"`
}

// ── Output wrappers ───────────────────────────────────────────────────────────

type statsCollectionOutput struct {
	Body statsCollectionBody
}

type statsQualityOutput struct {
	Body []statsQualityBucketBody
}

type statsStorageOutput struct {
	Body statsStorageBody
}

type statsGrabsOutput struct {
	Body statsGrabsBody
}

type statsDecadeBucketBody struct {
	Decade string `json:"decade"`
	Count  int64  `json:"count"`
}

type statsDecadesOutput struct {
	Body []statsDecadeBucketBody
}

type statsGrowthPointBody struct {
	Month      string `json:"month"`
	Added      int64  `json:"added"`
	Cumulative int64  `json:"cumulative"`
}

type statsGrowthOutput struct {
	Body []statsGrowthPointBody
}

type statsGenreBucketBody struct {
	Genre string `json:"genre"`
	Count int64  `json:"count"`
}

type statsGenresOutput struct {
	Body []statsGenreBucketBody
}

// ── Route registration ────────────────────────────────────────────────────────

// RegisterStatsRoutes registers the four statistics endpoints.
func RegisterStatsRoutes(api huma.API, svc *stats.Service) {
	// GET /api/v1/stats/collection
	huma.Register(api, huma.Operation{
		OperationID: "get-stats-collection",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/collection",
		Summary:     "Collection overview statistics",
		Tags:        []string{"Statistics"},
	}, func(ctx context.Context, _ *struct{}) (*statsCollectionOutput, error) {
		c, err := svc.Collection(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get collection stats", err)
		}
		out := &statsCollectionOutput{}
		out.Body = statsCollectionBody{
			TotalMovies:   c.TotalMovies,
			Monitored:     c.Monitored,
			WithFile:      c.WithFile,
			Missing:       c.Missing,
			NeedsUpgrade:  c.NeedsUpgrade,
			RecentlyAdded: c.RecentlyAdded,
		}
		return out, nil
	})

	// GET /api/v1/stats/quality
	huma.Register(api, huma.Operation{
		OperationID: "get-stats-quality",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/quality",
		Summary:     "Quality distribution of movie files",
		Tags:        []string{"Statistics"},
	}, func(ctx context.Context, _ *struct{}) (*statsQualityOutput, error) {
		buckets, err := svc.QualityDistribution(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get quality distribution", err)
		}
		body := make([]statsQualityBucketBody, len(buckets))
		for i, b := range buckets {
			body[i] = statsQualityBucketBody{
				Resolution: b.Resolution,
				Source:     b.Source,
				Codec:      b.Codec,
				HDR:        b.HDR,
				Count:      b.Count,
			}
		}
		out := &statsQualityOutput{}
		out.Body = body
		return out, nil
	})

	// GET /api/v1/stats/storage
	huma.Register(api, huma.Operation{
		OperationID: "get-stats-storage",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/storage",
		Summary:     "Storage usage and 90-day trend",
		Tags:        []string{"Statistics"},
	}, func(ctx context.Context, _ *struct{}) (*statsStorageOutput, error) {
		storage, err := svc.Storage(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get storage stats", err)
		}
		trend, err := svc.StorageTrend(ctx, 90)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get storage trend", err)
		}
		trendBody := make([]statsStoragePointBody, len(trend))
		for i, p := range trend {
			trendBody[i] = statsStoragePointBody{
				CapturedAt: p.CapturedAt,
				TotalBytes: p.TotalBytes,
				FileCount:  p.FileCount,
			}
		}
		out := &statsStorageOutput{}
		out.Body = statsStorageBody{
			TotalBytes: storage.TotalBytes,
			FileCount:  storage.FileCount,
			Trend:      trendBody,
		}
		return out, nil
	})

	// GET /api/v1/stats/decades
	huma.Register(api, huma.Operation{
		OperationID: "get-stats-decades",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/decades",
		Summary:     "Movie count by decade",
		Tags:        []string{"Statistics"},
	}, func(ctx context.Context, _ *struct{}) (*statsDecadesOutput, error) {
		buckets, err := svc.DecadeDistribution(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get decade distribution", err)
		}
		body := make([]statsDecadeBucketBody, len(buckets))
		for i, b := range buckets {
			body[i] = statsDecadeBucketBody{Decade: b.Decade, Count: b.Count}
		}
		out := &statsDecadesOutput{}
		out.Body = body
		return out, nil
	})

	// GET /api/v1/stats/growth
	huma.Register(api, huma.Operation{
		OperationID: "get-stats-growth",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/growth",
		Summary:     "Library growth over time (monthly)",
		Tags:        []string{"Statistics"},
	}, func(ctx context.Context, _ *struct{}) (*statsGrowthOutput, error) {
		points, err := svc.LibraryGrowth(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get library growth", err)
		}
		body := make([]statsGrowthPointBody, len(points))
		for i, p := range points {
			body[i] = statsGrowthPointBody{Month: p.Month, Added: p.Added, Cumulative: p.Cumulative}
		}
		out := &statsGrowthOutput{}
		out.Body = body
		return out, nil
	})

	// GET /api/v1/stats/genres
	huma.Register(api, huma.Operation{
		OperationID: "get-stats-genres",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/genres",
		Summary:     "Top genres by movie count",
		Tags:        []string{"Statistics"},
	}, func(ctx context.Context, _ *struct{}) (*statsGenresOutput, error) {
		buckets, err := svc.GenreDistribution(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get genre distribution", err)
		}
		body := make([]statsGenreBucketBody, len(buckets))
		for i, b := range buckets {
			body[i] = statsGenreBucketBody{Genre: b.Genre, Count: b.Count}
		}
		out := &statsGenresOutput{}
		out.Body = body
		return out, nil
	})

	// GET /api/v1/stats/grabs
	huma.Register(api, huma.Operation{
		OperationID: "get-stats-grabs",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats/grabs",
		Summary:     "Grab performance statistics",
		Tags:        []string{"Statistics"},
	}, func(ctx context.Context, _ *struct{}) (*statsGrabsOutput, error) {
		grabStats, indexers, err := svc.GrabPerformance(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get grab stats", err)
		}
		idxBody := make([]statsIndexerBody, len(indexers))
		for i, idx := range indexers {
			idxBody[i] = statsIndexerBody{
				IndexerID:   idx.IndexerID,
				IndexerName: idx.IndexerName,
				GrabCount:   idx.GrabCount,
				SuccessRate: idx.SuccessRate,
			}
		}
		out := &statsGrabsOutput{}
		out.Body = statsGrabsBody{
			TotalGrabs:  grabStats.TotalGrabs,
			Successful:  grabStats.Successful,
			Failed:      grabStats.Failed,
			SuccessRate: grabStats.SuccessRate,
			TopIndexers: idxBody,
		}
		return out, nil
	})
}
