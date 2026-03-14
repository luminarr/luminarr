# Feature 4: Library Health Dashboard

## The Problem

Movie collectors accumulate hundreds or thousands of files over years. Over time, libraries drift:
- Old 720p files sit alongside 4K Remuxes with no visibility into what's outdated
- Duplicate movies waste storage silently
- H.264 files that could be re-grabbed as smaller H.265 encodes
- No HDR even though a 4K HDR release is available
- No actionable view of "what should I upgrade?" beyond a raw "cutoff unmet" count

Radarr has a basic "cutoff unmet" list and quality stats. But it doesn't answer questions like:
- "What's the quality breakdown of my 500 movies?"
- "How much space could I save by re-encoding these old x264 files?"
- "Which movies have the best upgrade opportunities right now?"
- "Are there duplicates eating my storage?"

Luminarr already has Phase 21 stats (collection, quality, storage, grabs) and Phase 20 MediaInfo scanning. This feature builds on both to create an actionable health dashboard.

---

## What This Feature Delivers

1. **Library quality breakdown** — visual distribution of resolution, codec, HDR, source
2. **Upgrade opportunities** — ranked list of movies that have better releases available (prioritized by watch count if Feature 2 is implemented)
3. **Duplicate detection** — find movies with multiple files or multiple TMDB IDs pointing to the same film
4. **Codec analysis** — H.264 vs H.265 vs AV1 distribution with size comparisons
5. **Storage projections** — "upgrading all 720p to 1080p would use ~X additional GB"
6. **HDR coverage** — "X of your 4K movies have HDR; Y do not"
7. **Health score** — single number representing overall library health
8. **Actionable cards** — each insight has a "fix it" action (search, bulk upgrade, etc.)

---

## Scope & Non-Goals

### In scope
- Quality distribution charts (resolution, source, codec, HDR) — extends Phase 21
- Library health score (0-100) based on weighted factors
- Upgrade opportunity list (movies with cutoff unmet, ranked by value)
- Duplicate detection (same TMDB ID, same title+year, multiple files)
- Codec migration analysis (H.264 → H.265 savings estimate)
- HDR gap analysis (4K movies without HDR)
- Size anomaly detection (files significantly larger/smaller than expected for their quality)
- Actionable "fix" buttons on each insight
- Export/summary for the whole library

### Out of scope (future)
- Automatic remediation (auto-replace 720p with 1080p) — too risky without user confirmation
- File integrity checking (corrupt files, incomplete downloads)
- Disk health monitoring (S.M.A.R.T. data, filesystem errors)
- Cross-library comparison ("your library vs average")
- Historical quality trend (how quality improved over time) — could add later with snapshots

---

## Phase 1: Extended Quality Analytics

### 1.1 Enhance existing stats queries

Current `GET /api/v1/stats/quality` returns `[]QualityBucket` with resolution/source/codec/hdr counts. Extend with size information:

```go
type QualityBucket struct {
    Resolution string
    Source     string
    Codec      string
    HDR        string
    Count      int64
    TotalBytes int64   // NEW: sum of file sizes in this bucket
    AvgBytes   int64   // NEW: average file size
}
```

New SQL:
```sql
-- name: QualityDistributionWithSize :many
SELECT
    json_extract(mf.quality_json, '$.resolution') as resolution,
    json_extract(mf.quality_json, '$.source') as source,
    json_extract(mf.quality_json, '$.codec') as codec,
    json_extract(mf.quality_json, '$.hdr') as hdr,
    COUNT(*) as count,
    SUM(mf.size_bytes) as total_bytes,
    AVG(mf.size_bytes) as avg_bytes
FROM movie_files mf
GROUP BY resolution, source, codec, hdr
ORDER BY count DESC;
```

### 1.2 New dimension breakdowns

Individual dimension distributions for pie/bar charts:

```sql
-- name: ResolutionDistribution :many
SELECT
    json_extract(mf.quality_json, '$.resolution') as resolution,
    COUNT(*) as count,
    SUM(mf.size_bytes) as total_bytes
FROM movie_files mf
GROUP BY resolution
ORDER BY count DESC;

-- name: CodecDistribution :many
SELECT
    json_extract(mf.quality_json, '$.codec') as codec,
    COUNT(*) as count,
    SUM(mf.size_bytes) as total_bytes
FROM movie_files mf
GROUP BY codec
ORDER BY count DESC;

-- name: HDRDistribution :many
SELECT
    json_extract(mf.quality_json, '$.hdr') as hdr,
    COUNT(*) as count
FROM movie_files mf
WHERE json_extract(mf.quality_json, '$.resolution') = '2160p'
GROUP BY hdr
ORDER BY count DESC;

-- name: SourceDistribution :many
SELECT
    json_extract(mf.quality_json, '$.source') as source,
    COUNT(*) as count,
    SUM(mf.size_bytes) as total_bytes
FROM movie_files mf
GROUP BY source
ORDER BY count DESC;
```

---

## Phase 2: Library Health Score

### 2.1 Scoring model

A single 0-100 score representing overall library health. Composed of weighted factors:

| Factor | Weight | How it's calculated |
|---|---|---|
| Quality coverage | 30 | % of movies meeting their quality profile cutoff |
| File coverage | 25 | % of monitored movies that have a file on disk |
| Codec modernity | 15 | % of files using H.265 or AV1 (vs H.264/XVID) |
| HDR coverage | 10 | % of 4K files that have HDR |
| Edition match | 10 | % of movies with edition preference that have the right edition (Feature 1) |
| No duplicates | 10 | Penalty for duplicate movies/files |

```go
type HealthScore struct {
    Total              int      // 0-100
    QualityCoverage    Factor   // 30 pts max
    FileCoverage       Factor   // 25 pts max
    CodecModernity     Factor   // 15 pts max
    HDRCoverage        Factor   // 10 pts max
    EditionMatch       Factor   // 10 pts max (0 if Feature 1 not implemented)
    NoDuplicates       Factor   // 10 pts max
}

type Factor struct {
    Name       string
    Score      int     // Points earned
    Max        int     // Maximum possible points
    Percentage float64 // Raw ratio (0.0 to 1.0) before weighting
    Detail     string  // Human-readable detail: "280 of 340 movies meet cutoff"
}
```

### 2.2 Grade labels

| Score | Grade | Label |
|---|---|---|
| 90-100 | A | Excellent |
| 80-89 | B | Good |
| 70-79 | C | Fair |
| 60-69 | D | Needs Work |
| 0-59 | F | Poor |

### 2.3 Calculation

All factors are computed from database queries — no external calls needed:

```go
func (s *Service) CalculateHealthScore(ctx context.Context) (*HealthScore, error) {
    // 1. Quality coverage: movies with cutoff met / total monitored movies
    // 2. File coverage: movies with files / total monitored movies
    // 3. Codec modernity: files with x265|av1 / total files
    // 4. HDR coverage: 4K files with HDR / total 4K files (skip if no 4K files)
    // 5. Edition match: movies with preferred_edition matching file edition / total with preference
    // 6. Duplicates: penalty of 1 pt per duplicate pair, capped at 10 pts deduction
}
```

---

## Phase 3: Upgrade Opportunities

### 3.1 What makes a good upgrade opportunity?

An "upgrade opportunity" is a movie where:
1. The current file quality is below the quality profile cutoff
2. Better releases are known to exist (or likely exist based on the movie's age/popularity)

### 3.2 Upgrade ranking

```go
type UpgradeOpportunity struct {
    MovieID         string
    Title           string
    Year            int
    CurrentQuality  plugin.Quality
    CutoffQuality   plugin.Quality
    QualityGap      int    // Score difference between current and cutoff
    FileSizeBytes   int64
    EstUpgradeSize  int64  // Estimated size of upgraded file
    WatchCount      int    // From Feature 2 (0 if not available)
    Priority        int    // Composite score
}
```

Priority calculation:
```
Priority = QualityGap * 10
         + WatchBonus        (from Feature 2: +50 if watched 3+, +25 if watched 1-2, 0 otherwise)
         + RecencyBonus      (from Feature 2: +25 if watched in last 90 days)
         - SizeEstimate / 1GB (small penalty for very large upgrades)
```

### 3.3 SQL query

```sql
-- name: ListUpgradeOpportunities :many
SELECT
    m.id, m.title, m.year,
    mf.quality_json as current_quality,
    mf.size_bytes as current_size,
    qp.cutoff_json as cutoff_quality
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
JOIN quality_profiles qp ON qp.id = m.quality_profile_id
WHERE m.monitored = 1
  AND qp.upgrade_allowed = 1
  -- File quality score < cutoff score (computed in Go, not SQL)
ORDER BY m.title;
```

Filtering and sorting is done in Go after fetching, since quality score comparison requires parsing JSON.

### 3.4 Size estimation

Estimate the size of an upgraded file using quality definitions:

```go
func EstimateUpgradeSize(currentDef, targetDef QualityDefinition, runtimeMinutes int) int64 {
    // targetDef.PreferredSize (MB/min) * runtimeMinutes * 1024 * 1024
    return int64(targetDef.PreferredSize * float64(runtimeMinutes) * 1024 * 1024)
}
```

This gives "upgrading from 720p to 1080p would use approximately X GB."

---

## Phase 4: Duplicate Detection

### 4.1 Types of duplicates

1. **Same TMDB ID, multiple files** — intended multi-edition support, or accidental
2. **Same title + year, different TMDB IDs** — possible data mismatch
3. **Orphaned files** — files in library folders not linked to any movie record

### 4.2 Detection queries

```sql
-- name: ListDuplicateMovieFiles :many
-- Movies with more than one file
SELECT
    m.id as movie_id, m.title, m.year,
    COUNT(mf.id) as file_count,
    SUM(mf.size_bytes) as total_bytes,
    GROUP_CONCAT(mf.path, '|') as file_paths
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
GROUP BY m.id
HAVING file_count > 1
ORDER BY total_bytes DESC;

-- name: ListDuplicateTitles :many
-- Different movie records with the same title and year
SELECT
    m.title, m.year,
    COUNT(*) as record_count,
    GROUP_CONCAT(m.id, '|') as movie_ids
FROM movies m
GROUP BY m.title, m.year
HAVING record_count > 1
ORDER BY m.title;
```

### 4.3 Duplicate resolution actions

Each duplicate gets a suggested action:
- **Multi-file same movie:** "Keep highest quality, delete others" or "These are different editions — keep both"
- **Duplicate records:** "Merge these records" or "These are different movies (remakes)"

The UI shows the duplicates with their files and lets the user choose the action. No automatic deletion.

---

## Phase 5: Codec & HDR Analysis

### 5.1 Codec migration analysis

```go
type CodecAnalysis struct {
    CurrentCodec    string
    FileCount       int
    TotalBytes      int64
    AvgBytesPerMin  float64
    TargetCodec     string    // Suggested replacement
    EstSavingsBytes int64     // Estimated savings if re-grabbed as target codec
    EstSavingsPct   float64   // Percentage savings
}
```

For example:
- "You have 150 x264 movies using 1.2TB. Re-grabbing as x265 would save ~400GB (33%)."
- Size estimate based on quality definitions (x264 vs x265 preferred sizes for same resolution).

### 5.2 HDR gap analysis

```go
type HDRGap struct {
    MovieID    string
    Title      string
    Year       int
    Resolution string
    CurrentHDR string // "none"
    SizeBytes  int64
}
```

Query: 4K files without HDR — these are strong upgrade candidates since 4K+HDR releases are common.

```sql
-- name: List4KWithoutHDR :many
SELECT m.id, m.title, m.year, mf.size_bytes, mf.quality_json
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
WHERE json_extract(mf.quality_json, '$.resolution') = '2160p'
  AND json_extract(mf.quality_json, '$.hdr') IN ('none', 'unknown')
ORDER BY m.title;
```

### 5.3 Size anomaly detection

Detect files that are unusually large or small for their quality:

```go
type SizeAnomaly struct {
    MovieID       string
    Title         string
    FileSizeBytes int64
    ExpectedMin   int64  // Based on quality definition min_size * runtime
    ExpectedMax   int64  // Based on quality definition max_size * runtime
    Anomaly       string // "oversized" or "undersized"
}
```

Undersized files may indicate poor encodes or truncated downloads. Oversized files may indicate bloated encodes that could be re-grabbed more efficiently.

---

## Phase 6: API Endpoints

### 6.1 Health score

```
GET /api/v1/health/library
```

Returns:
```json
{
  "score": 78,
  "grade": "C",
  "label": "Fair",
  "factors": [
    {
      "name": "Quality Coverage",
      "score": 24,
      "max": 30,
      "percentage": 0.82,
      "detail": "280 of 340 movies meet their quality profile cutoff"
    },
    {
      "name": "File Coverage",
      "score": 22,
      "max": 25,
      "percentage": 0.88,
      "detail": "300 of 340 monitored movies have files on disk"
    },
    {
      "name": "Codec Modernity",
      "score": 10,
      "max": 15,
      "percentage": 0.67,
      "detail": "200 of 300 files use modern codecs (H.265 or AV1)"
    },
    {
      "name": "HDR Coverage",
      "score": 8,
      "max": 10,
      "percentage": 0.80,
      "detail": "80 of 100 4K files have HDR"
    },
    {
      "name": "Edition Match",
      "score": 9,
      "max": 10,
      "percentage": 0.90,
      "detail": "9 of 10 preferred editions matched"
    },
    {
      "name": "No Duplicates",
      "score": 5,
      "max": 10,
      "percentage": 0.50,
      "detail": "5 duplicate movies detected"
    }
  ]
}
```

### 6.2 Quality distribution (enhanced)

```
GET /api/v1/health/quality
```

Returns grouped distributions:
```json
{
  "by_resolution": [
    {"value": "2160p", "count": 100, "total_bytes": 800000000000},
    {"value": "1080p", "count": 150, "total_bytes": 300000000000},
    {"value": "720p", "count": 50, "total_bytes": 50000000000}
  ],
  "by_codec": [
    {"value": "x265", "count": 200, "total_bytes": 800000000000},
    {"value": "x264", "count": 80, "total_bytes": 300000000000},
    {"value": "av1", "count": 20, "total_bytes": 50000000000}
  ],
  "by_hdr": [
    {"value": "hdr10", "count": 60, "total_bytes": 500000000000},
    {"value": "dolby_vision", "count": 30, "total_bytes": 250000000000},
    {"value": "none", "count": 210, "total_bytes": 400000000000}
  ],
  "by_source": [
    {"value": "bluray", "count": 150, "total_bytes": 600000000000},
    {"value": "webdl", "count": 100, "total_bytes": 300000000000},
    {"value": "remux", "count": 50, "total_bytes": 250000000000}
  ]
}
```

### 6.3 Upgrade opportunities

```
GET /api/v1/health/upgrades?limit=50&sort=priority
```

Returns:
```json
{
  "total_upgradeable": 60,
  "estimated_total_download": 180000000000,
  "opportunities": [
    {
      "movie_id": "...",
      "title": "The Matrix",
      "year": 1999,
      "current_quality": {"resolution": "720p", "source": "bluray", ...},
      "cutoff_quality": {"resolution": "1080p", "source": "bluray", ...},
      "quality_gap": 130,
      "current_size": 4294967296,
      "estimated_upgrade_size": 8589934592,
      "watch_count": 5,
      "priority": 180
    }
  ]
}
```

### 6.4 Duplicates

```
GET /api/v1/health/duplicates
```

Returns:
```json
{
  "duplicate_files": [
    {
      "movie_id": "...",
      "title": "Movie",
      "year": 2020,
      "files": [
        {"id": "...", "path": "/movies/Movie (2020)/Movie.2020.1080p.mkv", "size_bytes": 4294967296, "quality": {...}},
        {"id": "...", "path": "/movies/Movie (2020)/Movie.2020.720p.mkv", "size_bytes": 2147483648, "quality": {...}}
      ],
      "reclaimable_bytes": 2147483648
    }
  ],
  "duplicate_records": [
    {
      "title": "Movie",
      "year": 2020,
      "records": [
        {"id": "...", "tmdb_id": 12345, "library_id": "..."},
        {"id": "...", "tmdb_id": 12345, "library_id": "..."}
      ]
    }
  ],
  "total_reclaimable_bytes": 10737418240
}
```

### 6.5 Codec analysis

```
GET /api/v1/health/codec
```

Returns:
```json
{
  "analysis": [
    {
      "codec": "x264",
      "file_count": 150,
      "total_bytes": 300000000000,
      "avg_bytes_per_min": 52428800,
      "target_codec": "x265",
      "estimated_savings_bytes": 100000000000,
      "estimated_savings_pct": 0.33
    }
  ],
  "hdr_gaps": [
    {
      "movie_id": "...",
      "title": "Movie",
      "year": 2020,
      "resolution": "2160p",
      "current_hdr": "none",
      "size_bytes": 8589934592
    }
  ],
  "total_hdr_gap_count": 20
}
```

### 6.6 Size anomalies

```
GET /api/v1/health/anomalies
```

Returns:
```json
{
  "oversized": [
    {
      "movie_id": "...",
      "title": "Movie",
      "file_size": 25769803776,
      "expected_max": 12884901888,
      "quality": {"resolution": "1080p", ...},
      "runtime_minutes": 120
    }
  ],
  "undersized": [
    {
      "movie_id": "...",
      "title": "Movie",
      "file_size": 524288000,
      "expected_min": 2147483648,
      "quality": {"resolution": "1080p", ...},
      "runtime_minutes": 120
    }
  ]
}
```

---

## Phase 7: Frontend — Library Health Dashboard

### 7.1 New page: `/health` (or `/library-health`)

A single-page dashboard with multiple sections. Not a settings page — this is a top-level nav item.

### 7.2 Health Score Hero

Top of page:
- Large circular gauge showing the score (0-100)
- Grade letter (A-F) inside the gauge
- Label ("Good", "Fair", etc.)
- Below: factor breakdown as horizontal progress bars
  - Each bar shows: factor name, score/max, percentage
  - Color-coded: green (>80%), yellow (50-80%), red (<50%)

### 7.3 Quality Distribution Section

- **Resolution pie chart** — 2160p vs 1080p vs 720p vs SD (with sizes)
- **Codec bar chart** — x264 vs x265 vs AV1 (with sizes)
- **HDR breakdown** — for 4K content only: HDR10 vs DV vs HLG vs None
- **Source breakdown** — Remux vs BluRay vs WebDL vs HDTV

Each chart is interactive — clicking a segment filters the movie list below it.

### 7.4 Upgrade Opportunities Section

- Table: movie poster, title, current quality → target quality, priority badge, estimated size, action button
- Sort by priority (default), quality gap, size
- "Search for Upgrade" button per movie
- "Search All" bulk button (triggers auto-search for top N)
- Total stats callout: "60 movies upgradeable | ~180GB estimated download"

### 7.5 Duplicates Section

- Card per duplicate group
- Shows all files with sizes, qualities
- "Keep Best, Delete Others" button (with confirmation)
- "Keep All" button (acknowledges multi-edition)
- Total reclaimable space callout

### 7.6 Codec Migration Section

- Per-codec card showing: count, total size, estimated savings if migrated
- "Search for x265 Replacements" bulk action
- Savings callout: "Re-grabbing 150 x264 files as x265 could save ~100GB"

### 7.7 HDR Gaps Section

- List of 4K movies without HDR
- "Search for HDR Version" per movie
- Count callout: "20 of your 100 4K movies don't have HDR"

### 7.8 Size Anomalies Section

- Two lists: oversized and undersized
- Per item: movie, file size, expected range, quality
- "Re-grab" button for undersized (likely bad encode)
- Informational for oversized (might be high-bitrate encode — not necessarily wrong)

### 7.9 Types

```typescript
interface LibraryHealth {
  score: number;
  grade: string;
  label: string;
  factors: HealthFactor[];
}

interface HealthFactor {
  name: string;
  score: number;
  max: number;
  percentage: number;
  detail: string;
}

interface QualityDistribution {
  by_resolution: DistBucket[];
  by_codec: DistBucket[];
  by_hdr: DistBucket[];
  by_source: DistBucket[];
}

interface DistBucket {
  value: string;
  count: number;
  total_bytes: number;
}

interface UpgradeOpportunity {
  movie_id: string;
  title: string;
  year: number;
  current_quality: Quality;
  cutoff_quality: Quality;
  quality_gap: number;
  current_size: number;
  estimated_upgrade_size: number;
  watch_count?: number;
  priority: number;
}

interface DuplicateGroup {
  movie_id: string;
  title: string;
  year: number;
  files: MovieFile[];
  reclaimable_bytes: number;
}

interface CodecAnalysis {
  codec: string;
  file_count: number;
  total_bytes: number;
  target_codec: string;
  estimated_savings_bytes: number;
  estimated_savings_pct: number;
}

interface SizeAnomaly {
  movie_id: string;
  title: string;
  file_size: number;
  expected_min?: number;
  expected_max?: number;
  quality: Quality;
  runtime_minutes: number;
  anomaly: 'oversized' | 'undersized';
}
```

---

## Implementation Order

1. **Enhanced stats queries** — add size info to quality distribution
2. **New distribution queries** — per-dimension breakdowns
3. **Health score service** — `internal/core/health/score.go`
4. **Upgrade opportunities query + ranking**
5. **Duplicate detection queries**
6. **Codec analysis logic** (compare quality definition sizes across codecs)
7. **HDR gap query**
8. **Size anomaly detection** (compare file size to quality definition * runtime)
9. **API endpoints** — health score, quality, upgrades, duplicates, codec, anomalies
10. **Frontend** — health dashboard page with all sections

Estimated: ~6-8 implementation sessions.

---

## Key Decisions to Make

1. **Separate page or section of existing stats page?**
   - Separate page (current plan): dedicated `/health` route, top-level nav item
   - Section in stats: less prominent but avoids another nav item
   - Recommendation: separate page — this is a flagship differentiating feature, give it prominence

2. **Health score factors — are the weights right?**
   - Quality coverage (30) is the biggest factor — is that correct?
   - Should "watch awareness" be a factor? (depends on Feature 2)
   - Should collection completeness be a factor? (depends on Feature 3)
   - Recommendation: start with the proposed weights, make them configurable later if needed

3. **Should the health score be calculated on every page load or cached?**
   - On-demand: always fresh, but requires multiple DB queries
   - Cached: faster, but stale
   - Recommendation: calculate on demand (SQLite is fast for these queries), cache in memory for 5 minutes

4. **Duplicate resolution actions — how destructive?**
   - "Delete Others" removes files from disk — irreversible
   - "Merge Records" changes movie data — mostly safe but could lose metadata
   - Recommendation: always require explicit confirmation with file paths shown, never auto-delete

5. **Size anomaly thresholds — what's "anomalous"?**
   - Current plan: use quality definition min_size/max_size * runtime
   - These are already defined for all 14 quality presets
   - A file is anomalous if < min_size * runtime * 0.5 or > max_size * runtime * 1.5
   - Recommendation: use 0.5x/1.5x thresholds — generous enough to avoid false positives

6. **Dependency on Features 1, 2, 3?**
   - Feature 4 is mostly independent
   - "Edition Match" factor in health score depends on Feature 1 (0 points if not implemented)
   - "Watch Count" in upgrade ranking depends on Feature 2 (ignored if not implemented)
   - Collection completeness could be a health factor if Feature 3 exists
   - Recommendation: build Feature 4 with optional integration points for 1/2/3; works standalone
