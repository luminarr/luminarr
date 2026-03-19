# Plan: Smart Upgrade Recommendations

**Status**: Draft
**Scope**: New upgrade analysis layer and Wanted page tab
**Depends on**: Quality profiles with UpgradeUntil (existing), edition-aware library management (merged)

---

## Summary

Add a dedicated upgrade recommendations system that groups movies by the quality transition they need (e.g., "720p WEB-DL -> 1080p BluRay") and distinguishes between mandatory upgrades (cutoff unmet) and optional upgrades (above cutoff but below UpgradeUntil). Surfaced as a new "Upgrades" tab on the Wanted page alongside Missing and Cutoff Unmet.

---

## Current State

- Wanted page has "Missing" and "Cutoff Unmet" tabs
- Cutoff Unmet shows movies below the quality cutoff but does not indicate *what* the upgrade path would be
- `UpgradeUntil` quality and `UpgradeUntilCFScore` exist on quality profiles but are only used during grab-time decisions
- No way to see optional upgrade opportunities (movies above cutoff but below UpgradeUntil)
- No grouping by upgrade tier -- users see a flat list with no sense of priority

---

## Step 1: SQL Query for Upgrade Candidates

### What

New sqlc query `ListMonitoredMoviesWithFilesAndUpgradeInfo` that joins movies, movie_files, and quality_profiles to return monitored movies that have files, along with upgrade profile settings.

Returns: movie ID, current resolution, source, codec, HDR, quality profile cutoff, upgrade_allowed flag, upgrade_until quality JSON, upgrade_until CF score.

### Files

| File | Change |
|------|--------|
| `internal/db/queries/sqlite/movies.sql` | Add `ListMonitoredMoviesWithFilesAndUpgradeInfo` query |
| `internal/db/generated/sqlite/` | Regenerate with sqlc |

---

## Step 2: Upgrade Tier Computation

### What

New method `ListUpgradeRecommendations()` on the movie service that:

1. Fetches all upgrade candidates from the new query
2. For each movie, computes the current quality score and compares against cutoff and UpgradeUntil
3. Classifies as `cutoff_unmet` (mandatory) or `optional_upgrade` (above cutoff, below UpgradeUntil)
4. Generates a tier label from the quality dimension differences (e.g., "720p -> 1080p", "WEB-DL -> BluRay", "SDR -> HDR10")
5. Groups movies into `UpgradeTier` structs by their transition label

### New Types

`UpgradeTier` struct:
- `Label` -- human-readable tier label (e.g., "1080p WEB-DL -> 2160p Remux")
- `Priority` -- `cutoff_unmet` or `optional_upgrade`
- `Movies` -- list of movie summaries in this tier
- `Count` -- number of movies

`UpgradeRecommendation` per movie:
- `MovieID`, `Title`, `Year`
- `CurrentQuality` -- what's on disk
- `TargetQuality` -- what the profile wants (cutoff or UpgradeUntil)
- `Priority` -- cutoff_unmet or optional_upgrade

### Tier Label Generation

Compare current quality dimensions against target:
- If resolution differs: "720p -> 1080p"
- If source differs: "WEB-DL -> BluRay"
- If both differ: "720p WEB-DL -> 1080p BluRay"
- If only HDR differs: "SDR -> HDR10"
- Combine all differing dimensions into one label

### Files

| File | Change |
|------|--------|
| `internal/core/movie/service.go` | Add `ListUpgradeRecommendations()` method |
| `internal/core/movie/upgrade.go` | New file: `UpgradeTier`, `UpgradeRecommendation` types, tier label generation logic |
| `internal/core/movie/upgrade_test.go` | New file: test tier grouping and label generation |

---

## Step 3: API Endpoints

### What

Two new endpoints:

- `GET /api/v1/wanted/upgrades` -- returns upgrade tiers with movie lists, supports `?priority=cutoff_unmet|optional_upgrade` filter
- `POST /api/v1/wanted/upgrades/search` -- bulk search for upgrades, accepts `{ movie_ids: [] }` or `{ tier: "label" }` to search all movies in a tier

### Files

| File | Change |
|------|--------|
| `internal/api/v1/wanted.go` | Add `handleUpgradeRecommendations`, `handleBulkUpgradeSearch` handlers |
| `internal/api/router.go` | Register both endpoints |

---

## Step 4: Frontend

### What

New "Upgrades" tab on the Wanted page:

- Tab bar: Missing | Cutoff Unmet | **Upgrades**
- Default view groups by tier, showing tier label, count, and priority badge
- Expand a tier to see individual movies with current vs target quality
- "Search All" button per tier triggers bulk search
- Toggle between cutoff_unmet only and all upgrades
- Link upgrade count badges to this tab from the stats page

### Files

| File | Change |
|------|--------|
| `web/ui/src/api/wanted.ts` | Add `getUpgradeRecommendations()`, `bulkUpgradeSearch()` API calls |
| `web/ui/src/pages/wanted/UpgradesTab.tsx` | New file: upgrades tab component |
| `web/ui/src/pages/wanted/WantedPage.tsx` | Add Upgrades tab to tab bar |
| `web/ui/src/types/index.ts` | Add `UpgradeTier`, `UpgradeRecommendation` types |

---

## Implementation Order

```
Step 1: SQL query                     [independent]
Step 2: Tier computation              [depends on 1]
Step 3: API endpoints                 [depends on 2]
Step 4: Frontend                      [depends on 3]
```

**PR Strategy**: Single PR for all steps -- the feature is self-contained and has no behavioral side effects on existing functionality.

---

## Key Design Decisions

- **Computed on demand, no background job**: Upgrade recommendations are derived from current profile settings and library state at request time. No caching, no stale data, no scheduler job.
- **Two priority levels**: `cutoff_unmet` (profile says quality is unacceptable) vs `optional_upgrade` (quality is acceptable but could be better). This matches the existing profile semantics without inventing new concepts.
- **Tier grouping by quality transition**: Grouping by "what you have -> what you want" gives users actionable insight instead of a flat list. A user with 50 movies needing "WEB-DL -> BluRay" can search them all at once.
- **Bulk search reuses existing autosearch**: `POST /api/v1/wanted/upgrades/search` calls the same `SearchMovie` logic per movie, just batched.

---

## Risks Summary

| Risk | Impact | Mitigation |
|------|--------|------------|
| Large libraries = slow tier computation | API response takes seconds | Pagination on movie list within tiers; computation is simple struct grouping |
| Bulk search hammers indexers | Rate limiting or bans from indexers | Reuse existing rate limiting in indexer service; sequential search with delays |
| Tier labels too granular | Too many tiers with 1 movie each | Group by primary dimension difference only; collapse minor differences |
| UpgradeUntil not set on most profiles | Optional tier is always empty | Show helpful message when UpgradeUntil is not configured; still useful for cutoff_unmet |
