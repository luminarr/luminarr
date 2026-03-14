# Phase A — Custom Formats + Tags Foundation

**Goal:** Implement the full tag system and custom formats scoring engine. This is the single biggest feature gap and the reason power users won't switch from Radarr.

---

## A1. Tags (Full System)

Tags link movies to specific indexers, download clients, notifications, delay profiles, and (later) import lists. A movie uses resources that share at least one tag — plus all untagged resources.

### Database

**Migration 00025_tags.sql:**
```sql
CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

-- Junction tables for entity-tag relationships
CREATE TABLE movie_tags (
    movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    tag_id   TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (movie_id, tag_id)
);

CREATE TABLE indexer_tags (
    indexer_id TEXT NOT NULL REFERENCES indexer_configs(id) ON DELETE CASCADE,
    tag_id     TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (indexer_id, tag_id)
);

CREATE TABLE download_client_tags (
    download_client_id TEXT NOT NULL REFERENCES download_client_configs(id) ON DELETE CASCADE,
    tag_id             TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (download_client_id, tag_id)
);

CREATE TABLE notification_tags (
    notification_id TEXT NOT NULL REFERENCES notification_configs(id) ON DELETE CASCADE,
    tag_id          TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (notification_id, tag_id)
);
```

Drop the unused `tags_json` column from libraries (or leave it — it's harmless).

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/tags` | List all tags (with usage counts) |
| POST | `/api/v1/tags` | Create tag |
| PUT | `/api/v1/tags/{id}` | Rename tag |
| DELETE | `/api/v1/tags/{id}` | Delete tag (cascades from junctions) |

Tag assignment happens via existing entity endpoints — add `tag_ids []string` to movie, indexer, download client, and notification create/update requests.

### Service: `internal/core/tag/service.go`

```go
type Service struct { q dbsqlite.Querier }

func (s *Service) List(ctx) ([]Tag, error)           // includes usage counts
func (s *Service) Create(ctx, name) (Tag, error)
func (s *Service) Update(ctx, id, name) (Tag, error)
func (s *Service) Delete(ctx, id) error
func (s *Service) TagsForMovie(ctx, movieID) ([]string, error)
func (s *Service) TagsForIndexer(ctx, indexerID) ([]string, error)
// etc.
```

### Tag Matching Logic

When searching for a movie:
1. Load movie's tags
2. Load all enabled indexers
3. Filter: include indexer if (indexer has no tags) OR (movie tags ∩ indexer tags ≠ ∅)
4. Same logic for selecting download client

This filtering lives in `autosearch.Service` and `releases.go` handler.

### Radarr v3 Compat

Update `internal/api/v3/tags.go` to read from the real `tags` table instead of in-memory store.

### Frontend

- Tag management page: Settings > Tags
- Tag pills on movie detail, indexer, download client, notification forms
- Tag filter in movie list
- Bulk tag assignment in movie editor

### Files to Create/Modify

| Action | File |
|--------|------|
| Create | `internal/db/migrations/00025_tags.sql` |
| Create | `internal/db/queries/sqlite/tags.sql` |
| Create | `internal/core/tag/service.go` |
| Create | `internal/api/v1/tags.go` |
| Modify | `internal/api/v1/movies.go` — add tag_ids to create/update |
| Modify | `internal/api/v1/indexers.go` — add tag_ids |
| Modify | `internal/api/v1/downloadclients.go` — add tag_ids |
| Modify | `internal/api/v1/notifications.go` — add tag_ids |
| Modify | `internal/core/autosearch/service.go` — tag-aware indexer/client selection |
| Modify | `internal/api/v1/releases.go` — tag-aware grab routing |
| Modify | `internal/api/router.go` — register tag routes |
| Modify | `internal/api/v3/tags.go` — read from real DB |
| Modify | `cmd/luminarr/main.go` — wire tag service |
| Run | `sqlc generate` |

---

## A2. Custom Formats

Custom formats are user-defined rules that score releases. Each format has conditions (regex on title, match source, match resolution, etc.). Each quality profile assigns a score to each format. The total CF score is added to release ranking.

### Database

**Migration 00026_custom_formats.sql:**
```sql
CREATE TABLE custom_formats (
    id                    TEXT PRIMARY KEY,
    name                  TEXT NOT NULL UNIQUE,
    include_when_renaming BOOLEAN NOT NULL DEFAULT false,
    specifications        TEXT NOT NULL DEFAULT '[]'  -- JSON array of conditions
);

-- Per-quality-profile scoring for each custom format
CREATE TABLE custom_format_scores (
    quality_profile_id TEXT NOT NULL REFERENCES quality_profiles(id) ON DELETE CASCADE,
    custom_format_id   TEXT NOT NULL REFERENCES custom_formats(id) ON DELETE CASCADE,
    score              INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (quality_profile_id, custom_format_id)
);

-- Add CF score thresholds to quality profiles
ALTER TABLE quality_profiles ADD COLUMN min_custom_format_score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE quality_profiles ADD COLUMN upgrade_until_cf_score  INTEGER NOT NULL DEFAULT 0;
```

### Custom Format Specification Types

Each specification is stored in the `specifications` JSON array:

```json
{
  "name": "TrueHD ATMOS",
  "implementation": "release_title",   // condition type
  "negate": false,
  "required": false,
  "fields": {
    "value": "(?i)\\bTrueHD\\.?\\s?Atmos\\b"
  }
}
```

**10 Condition Types (matching Radarr):**

| Type | Implementation | Fields | What it matches |
|------|---------------|--------|----------------|
| Release Title | `release_title` | `value` (regex) | Regex against release title string |
| Edition | `edition` | `value` (regex) | Regex against edition tags |
| Language | `language` | `value` (language ID) | Audio language of release |
| Indexer Flag | `indexer_flag` | `value` (flag ID) | Indexer flags (freeleech etc.) |
| Source | `source` | `value` (source enum) | Source type (TV, Web-DL, Blu-ray, etc.) |
| Resolution | `resolution` | `value` (resolution enum) | Resolution (480p, 720p, 1080p, 2160p) |
| Quality Modifier | `quality_modifier` | `value` (modifier enum) | Modifiers like Remux |
| Size | `size` | `min`, `max` (GB) | File size range |
| Release Group | `release_group` | `value` (regex) | Release group name |
| Year | `year` | `min`, `max` | Release year range |

**Modifiers per condition:**
- `negate` — invert the match (true = "must NOT match")
- `required` — when multiple conditions exist, required ones must ALL match; non-required use OR logic within same type

### Matching Algorithm

```
For each custom format:
  group conditions by implementation type
  for each group:
    required_conditions = [c for c in group if c.required]
    optional_conditions = [c for c in group if !c.required]

    if any required fails → format does NOT match → break
    if no optional matches (and optionals exist) → format does NOT match → break

  if all groups pass → format MATCHES
```

### Scoring Integration

Current scoring is 0–100 across 4 dimensions (resolution, source, codec, HDR). Custom format score is a **separate dimension** that sits alongside quality score.

**Updated release ranking:**
1. Quality rank (from profile order — higher position = preferred)
2. Custom format score (sum of matched CF scores for this profile)
3. Existing 0–100 quality breakdown (tiebreaker)

**Quality profile decision with CF:**
- Reject if CF score < `min_custom_format_score`
- Stop upgrading if CF score >= `upgrade_until_cf_score` (and quality cutoff met)

### Service: `internal/core/customformat/service.go`

```go
type Service struct { q dbsqlite.Querier }

func (s *Service) List(ctx) ([]CustomFormat, error)
func (s *Service) Get(ctx, id) (CustomFormat, error)
func (s *Service) Create(ctx, req) (CustomFormat, error)
func (s *Service) Update(ctx, id, req) (CustomFormat, error)
func (s *Service) Delete(ctx, id) error
func (s *Service) Import(ctx, jsonData) ([]CustomFormat, error)  // TRaSH JSON import
func (s *Service) Export(ctx, ids) ([]byte, error)               // TRaSH JSON export
```

### Matcher: `internal/core/customformat/matcher.go`

```go
// MatchRelease evaluates all custom formats against a release.
// Returns list of matched format IDs.
func MatchRelease(formats []CustomFormat, release ReleaseInfo) []string

// ScoreRelease computes the total CF score for a release given a quality profile.
func ScoreRelease(matched []string, profileScores map[string]int) int

// ReleaseInfo is the input to the matcher — extracted from release title + indexer metadata.
type ReleaseInfo struct {
    Title        string
    Edition      string
    Languages    []string
    IndexerFlags []string
    Source       string
    Resolution   string
    Modifier     string   // "remux", etc.
    SizeBytes    int64
    ReleaseGroup string
    Year         int
}
```

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/custom-formats` | List all custom formats |
| POST | `/api/v1/custom-formats` | Create custom format |
| GET | `/api/v1/custom-formats/{id}` | Get custom format details |
| PUT | `/api/v1/custom-formats/{id}` | Update custom format |
| DELETE | `/api/v1/custom-formats/{id}` | Delete custom format |
| POST | `/api/v1/custom-formats/import` | Import from JSON (TRaSH format) |
| GET | `/api/v1/custom-formats/export` | Export selected formats as JSON |
| GET | `/api/v1/custom-formats/schema` | Return available condition types + fields |

Quality profile endpoints need to accept `custom_format_scores` map and the two new threshold fields.

### Integration Points

| Component | Change |
|-----------|--------|
| `autosearch/service.go` | After quality scoring, run CF matcher. Add CF score to ScoreBreakdown. Reject below min_cf_score. |
| `quality/profile.go` | Add `MinCFScore`, `UpgradeUntilCFScore` fields. Update `WantRelease()` to consider CF thresholds. |
| `plugin/score.go` | Add CF dimension to `ScoreBreakdown` |
| `api/v1/releases.go` | Include matched CFs + CF score in manual search response |
| `renamer/renamer.go` | Add `{Custom Formats}` and `{Custom Format Score}` tokens |
| `api/v1/qualityprofiles.go` | Accept/return CF scores per profile |

### JSON Import/Export Format (TRaSH Compatible)

TRaSH Guides distribute custom formats as JSON. Format:

```json
{
  "trash_id": "unique-identifier",
  "trash_scores": {
    "default": 1750
  },
  "name": "TrueHD ATMOS",
  "includeCustomFormatWhenRenaming": false,
  "specifications": [
    {
      "name": "TrueHD ATMOS",
      "implementation": "ReleaseTitleSpecification",
      "negate": false,
      "required": true,
      "fields": {
        "value": "(?i)\\bTrueHD\\.?\\s?Atmos\\b"
      }
    }
  ]
}
```

Map TRaSH `implementation` names to ours:
| TRaSH | Luminarr |
|-------|----------|
| ReleaseTitleSpecification | release_title |
| EditionSpecification | edition |
| LanguageSpecification | language |
| IndexerFlagSpecification | indexer_flag |
| SourceSpecification | source |
| ResolutionSpecification | resolution |
| QualityModifierSpecification | quality_modifier |
| SizeSpecification | size |
| ReleaseGroupSpecification | release_group |
| YearSpecification | year |

### Files to Create/Modify

| Action | File |
|--------|------|
| Create | `internal/db/migrations/00026_custom_formats.sql` |
| Create | `internal/db/queries/sqlite/custom_formats.sql` |
| Create | `internal/core/customformat/service.go` |
| Create | `internal/core/customformat/matcher.go` |
| Create | `internal/core/customformat/matcher_test.go` |
| Create | `internal/core/customformat/trash.go` (TRaSH import/export) |
| Create | `internal/api/v1/customformats.go` |
| Modify | `internal/core/autosearch/service.go` — CF scoring integration |
| Modify | `internal/core/quality/profile.go` — CF thresholds in WantRelease |
| Modify | `pkg/plugin/score.go` — add CF dimension |
| Modify | `internal/api/v1/qualityprofiles.go` — CF scores in profile CRUD |
| Modify | `internal/core/renamer/renamer.go` — CF naming tokens |
| Modify | `internal/api/v1/releases.go` — show CF matches in search results |
| Modify | `internal/api/router.go` — register CF routes |
| Modify | `cmd/luminarr/main.go` — wire CF service |
| Run | `sqlc generate` |

---

## A3. Indexer Flags

Indexer flags (freeleech, halfleech, internal, scene, etc.) feed into custom format conditions.

### Changes

- Add `indexer_flags` field to `plugin.Release` struct
- Torznab/Newznab parsers: extract flags from indexer response attributes
- Store flags in grab history for reference
- Custom format `indexer_flag` condition uses these flags

### Flag Enum

```go
const (
    FlagFreeleech    = "freeleech"
    FlagHalfleech    = "halfleech"
    FlagDoubleUpload = "double_upload"
    FlagInternal     = "internal"
    FlagScene        = "scene"
    FlagFreeleech75  = "freeleech_75"
    FlagFreeleech25  = "freeleech_25"
    FlagNuked        = "nuked"
)
```

### Files to Modify

| Action | File |
|--------|------|
| Modify | `pkg/plugin/types.go` — add IndexerFlags to Release |
| Modify | `plugins/indexers/torznab/torznab.go` — parse flags from XML attrs |
| Modify | `plugins/indexers/newznab/newznab.go` — parse flags |

---

## A4. Quality Tiers Expansion (14 → 31)

### New Migration 00027_quality_tiers_expansion.sql

Add the 17 missing tiers:

**Pre-release:**
- `workprint` (sort_order 1)
- `cam` (sort_order 2)
- `telesync` (sort_order 3)
- `telecine` (sort_order 4)
- `dvdscr` (sort_order 5)
- `regional` (sort_order 6)

**480p/576p variants:**
- `480p-webdl-x264-none`
- `480p-webrip-x264-none`
- `480p-bluray-x264-none`
- `576p-bluray-x264-none`

**Other missing:**
- `dvdr` (full DVD image)
- `2160p-hdtv-x265-hdr10`
- `2160p-webrip-x265-hdr10`
- `br-disk` (raw Blu-ray disc)
- `raw-hd`

**Quality grouping (DEFERRED):** Radarr lets you group qualities (e.g., treat WEBDL-1080p and WEBRip-1080p as equivalent). Deferred — add a TODO to revisit with a more modern/cleaner UX approach than Radarr's nested drag-and-drop.

### Release Parser Updates

Update the title parser to detect:
- CAM, HDCAM, CAMRip
- TS, TELESYNC, HDTS, PDVD
- TC, TELECINE, HDTC
- DVDSCR, SCREENER, SCR
- WORKPRINT, WP
- R5, REGIONAL
- DVDR, DVD-R, DVD9, DVD5, BDMV, BD25, BD50

### Files to Create/Modify

| Action | File |
|--------|------|
| Create | `internal/db/migrations/00027_quality_tiers_expansion.sql` |
| Modify | Release title parser (add pre-release detection) |

---

## Build Order

1. **A1 Tags** — standalone, no dependencies
2. **A3 Indexer Flags** — small, standalone
3. **A4 Quality Tiers** — standalone
4. **A2 Custom Formats** — depends on A1 (tags for filtering), A3 (flags as CF condition), A4 (more tiers to match against)

Steps 1–3 can be parallelized. Step 4 builds on all three.

---

## Test Strategy

| Component | Test Type | Key Cases |
|-----------|-----------|-----------|
| Tag service | Unit | CRUD, cascade delete, usage counts |
| Tag filtering | Unit | Tag overlap logic, untagged resources included |
| CF matcher | Unit | Each condition type, negate, required, multi-condition, edge cases |
| CF scoring | Unit | Score summation, min threshold reject, upgrade-until stop |
| CF import | Unit | TRaSH JSON parsing, field mapping, error handling |
| Quality tiers | Unit | Parser detects all 31 tiers correctly |
| Integration | API | Full flow: create CF → create profile with scores → search → verify scoring |

CF matcher tests are critical — this is pure logic with many edge cases. Spawn a test subagent for `matcher_test.go` once the interface is locked.

---

## Estimated Scope

| Sub-phase | New Files | Modified Files | Migrations | Effort |
|-----------|-----------|---------------|------------|--------|
| A1 Tags | 4 | 8 | 1 | Medium |
| A2 Custom Formats | 6 | 7 | 1 | Large |
| A3 Indexer Flags | 0 | 3 | 0 | Small |
| A4 Quality Tiers | 0 | 2 | 1 | Small-Medium |
| **Total** | **10** | **~15** | **3** | **Large** |
