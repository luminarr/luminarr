# Plan 21 — Library Statistics

## Problem

Users have no insight into their collection beyond a raw count. Questions like
"how much of my library is x265?", "how much disk space is Remux vs WebDL?",
or "which indexer is actually useful?" have no answer in the current UI.

The media-manager project (gitlab.com/davidfic/media-manager) already implements
this pattern in Python/SQLAlchemy. We adapt the same conceptual model to Go/SQLite.

---

## Scope

A new **Statistics** page (sidebar nav, between Calendar and Settings) with four
sections:

1. **Collection overview** — counts and monitored/missing breakdown
2. **Quality distribution** — resolution, source, codec, HDR breakdowns
3. **Storage** — total usage, breakdown by quality tier, trend over time
4. **Grab performance** — success rate, top indexers, recent activity

---

## Storage Trend: Snapshot Table

All other stats are computed on-demand from existing tables. Storage trend
requires periodic point-in-time snapshots (same approach as media-manager).

### Migration (`internal/db/migrations/00019_storage_snapshots.sql`)

```sql
-- +goose Up
CREATE TABLE storage_snapshots (
    id           TEXT PRIMARY KEY,
    captured_at  DATETIME NOT NULL,
    total_bytes  INTEGER NOT NULL DEFAULT 0,
    file_count   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_storage_snapshots_captured_at ON storage_snapshots(captured_at);

-- +goose Down
DROP TABLE storage_snapshots;
```

### sqlc queries (`internal/db/queries/sqlite/stats.sql`)

```sql
-- name: InsertStorageSnapshot :exec
INSERT INTO storage_snapshots (id, captured_at, total_bytes, file_count)
VALUES (?, ?, ?, ?);

-- name: ListStorageSnapshots :many
SELECT * FROM storage_snapshots
ORDER BY captured_at DESC
LIMIT ?;

-- name: PruneOldStorageSnapshots :exec
DELETE FROM storage_snapshots
WHERE captured_at < ?;

-- name: GetCollectionStats :one
SELECT
    COUNT(*)                                          AS total_movies,
    SUM(CASE WHEN monitored = 1 THEN 1 ELSE 0 END)   AS monitored,
    SUM(CASE WHEN path != '' THEN 1 ELSE 0 END)       AS with_file,
    SUM(CASE WHEN monitored = 1 AND path = '' THEN 1 ELSE 0 END) AS missing
FROM movies;

-- name: GetQualityDistribution :many
SELECT
    quality_json,
    COUNT(*) AS count
FROM movie_files
GROUP BY quality_json;

-- name: GetStorageTotals :one
SELECT
    COALESCE(SUM(size_bytes), 0) AS total_bytes,
    COUNT(*)                      AS file_count
FROM movie_files;

-- name: GetGrabStats :one
SELECT
    COUNT(*)                                                          AS total_grabs,
    SUM(CASE WHEN status = 'imported' THEN 1 ELSE 0 END)             AS successful,
    SUM(CASE WHEN status IN ('failed','error') THEN 1 ELSE 0 END)    AS failed
FROM grab_history;

-- name: GetTopIndexers :many
SELECT
    indexer_id,
    COUNT(*) AS grab_count,
    SUM(CASE WHEN status = 'imported' THEN 1 ELSE 0 END) AS success_count
FROM grab_history
WHERE indexer_id != ''
GROUP BY indexer_id
ORDER BY grab_count DESC
LIMIT 10;
```

Run `sqlc generate` after adding.

---

## Statistics Service (`internal/core/stats/service.go`)

```go
package stats

type CollectionStats struct {
    TotalMovies int64
    Monitored   int64
    WithFile    int64
    Missing     int64
}

type QualityBucket struct {
    Resolution string
    Source     string
    Codec      string
    HDR        string
    Count      int64
}

type StorageStat struct {
    TotalBytes int64
    FileCount  int64
}

type StoragePoint struct {
    CapturedAt time.Time
    TotalBytes int64
    FileCount  int64
}

type IndexerStat struct {
    IndexerID    string
    IndexerName  string  // looked up from indexer_configs
    GrabCount    int64
    SuccessCount int64
    SuccessRate  float64
}

type GrabStats struct {
    TotalGrabs  int64
    Successful  int64
    Failed      int64
    SuccessRate float64
}

type Service struct {
    q          dbsqlite.Querier
    indexerSvc *indexer.Service  // for name lookup
}

func (s *Service) Collection(ctx context.Context) (CollectionStats, error)
func (s *Service) QualityDistribution(ctx context.Context) ([]QualityBucket, error)
func (s *Service) Storage(ctx context.Context) (StorageStat, error)
func (s *Service) StorageTrend(ctx context.Context, days int) ([]StoragePoint, error)
func (s *Service) GrabPerformance(ctx context.Context) (GrabStats, []IndexerStat, error)
func (s *Service) TakeSnapshot(ctx context.Context) error  // called by scheduler
```

`QualityDistribution` unmarshals `quality_json` from each `movie_files` row and
aggregates by (Resolution, Source, Codec, HDR) — same as we do elsewhere at the
service boundary. SQLite does the grouping; Go does the JSON unmarshalling.

---

## Scheduler Job

`internal/scheduler/jobs/stats_snapshot.go`:

- Runs every 24 hours
- Calls `statsSvc.TakeSnapshot(ctx)`
- Prunes snapshots older than 90 days

---

## API (`internal/api/v1/stats.go`)

```
GET /api/v1/stats   — returns full stats payload in one call
```

Single endpoint to avoid round-trip chattiness. Response:

```json
{
  "collection": {
    "total_movies": 412,
    "monitored": 398,
    "with_file": 387,
    "missing": 11
  },
  "quality_distribution": [
    { "resolution": "1080p", "source": "Bluray", "codec": "x265", "hdr": "none", "count": 142 },
    ...
  ],
  "storage": {
    "total_bytes": 4398046511104,
    "file_count": 387
  },
  "storage_trend": [
    { "captured_at": "2026-03-01T00:00:00Z", "total_bytes": 4200000000000, "file_count": 380 },
    ...
  ],
  "grab_performance": {
    "total_grabs": 512,
    "successful": 487,
    "failed": 25,
    "success_rate": 0.951
  },
  "top_indexers": [
    { "indexer_id": "abc", "indexer_name": "Prowlarr-1337x", "grab_count": 201, "success_count": 196, "success_rate": 0.975 },
    ...
  ]
}
```

Register in `router.go` under `RegisterStatsRoutes(humaAPI, cfg.StatsService)`.

---

## Frontend (`src/pages/stats/StatsPage.tsx`)

Layout: four cards stacked vertically (full-width on mobile, 2-column grid on desktop).

### Card 1 — Collection Overview

Four stat blocks: Total / Monitored / Have File / Missing.
Simple `<dl>` layout, big number + label.

### Card 2 — Quality Distribution

**Resolution bar chart** (horizontal bars, % of library):
```
2160p  ████████████░░░░░░░░  30%  (123)
1080p  ████████████████████  52%  (214)
 720p  █████░░░░░░░░░░░░░░░  13%  (53)
  SD   ██░░░░░░░░░░░░░░░░░░   5%  (17)
```

**Codec breakdown** — simple table: x265 / x264 / AV1 / other with counts and %.
**HDR breakdown** — HDR10 / Dolby Vision / SDR with counts and %.
**Source breakdown** — Remux / Bluray / WebDL / WEBRip / HDTV with counts and %.

No third-party chart library. Inline CSS bar widths computed from percentages.

### Card 3 — Storage

Large total at top. Break down by resolution tier (compute from quality_distribution
data already fetched). Spark line for 30-day trend using SVG path — no library needed.

### Card 4 — Grab Performance

Success rate (big %) + total/successful/failed counts.
Top indexers table: name, grabs, success rate.

---

## Sidebar Nav

Add "Statistics" between Calendar and Settings. Icon: `BarChart2` (lucide).

---

## Tests

**Unit** (`internal/core/stats/service_test.go`):
- `TestCollectionStats` — seed movies in various states, verify counts
- `TestQualityDistribution` — seed movie_files with different quality JSON, verify buckets
- `TestGrabStats` — seed grab_history rows, verify success rate math
- `TestStorageSnapshot` — snapshot inserts correct totals; prune removes old rows

**Integration** (`internal/api/v1/stats_test.go`):
- `TestGetStats` — GET returns 200 with correct JSON shape
- `TestGetStats_empty` — works with empty DB (zeros, not errors)

---

## Open Questions

1. Should we bucket quality distribution on the frontend (from raw `quality_json`)
   or in SQL? **Recommendation: SQL `GROUP BY` on the serialised JSON string, then
   unmarshal in Go.** Avoids re-fetching all files.

2. Storage trend granularity: daily is fine for 90-day window. No need for hourly.

3. Do we want a "largest files" section? Potentially useful but deferred — add later
   if users request it. (The SQL is trivial: `SELECT * FROM movie_files ORDER BY size_bytes DESC LIMIT 20`.)
