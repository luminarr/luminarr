# Plan: Quality Gap Stats Enhancement

**Status**: Draft
**Scope**: Enhance existing Stats page with combined quality tier view, clickable charts, and upgrade potential indicators
**Depends on**: Existing stats page with quality distribution, upgrade recommendations (upgrade-recommendations plan)

---

## Summary

The Stats page already shows quality distribution by individual dimension (resolution, source, codec, HDR) but does not show combined quality tiers or allow drilling down into specific groups of movies. This plan adds a combined tier view (e.g., "2160p Remux", "1080p BluRay"), makes chart bars clickable for navigation to a filtered dashboard, and adds upgrade potential indicators that link to the Wanted page's Upgrades tab.

---

## Current State

- Stats page QualityCard shows bar charts for each quality dimension independently
- No combined tier view (resolution + source together)
- Charts are display-only -- not interactive
- No connection between stats and navigation (can't click a bar to see those movies)
- No indication of how many movies in each quality tier could be upgraded
- Dashboard does not accept quality filter URL params

---

## Step 1: Combined Quality Tier View

### What

Add a "By Tier" toggle to QualityCard alongside the existing "By Dimension" view. The tier view groups movies by resolution+source combination (e.g., "2160p Remux", "1080p BluRay", "1080p WEB-DL", "720p WEB-DL").

### Tier Label Generation

Combine resolution and source into a single label: `"{resolution} {source}"`. Normalize source names for display (e.g., "BluRay" not "bluray", "WEB-DL" not "webdl").

### Files

| File | Change |
|------|--------|
| `internal/api/v1/stats.go` | Add tier aggregation logic to existing quality stats handler (group by resolution+source) |
| `web/ui/src/pages/stats/QualityCard.tsx` | Add toggle between "By Dimension" and "By Tier" views; render tier bar chart |

---

## Step 2: Clickable Chart Bars

### What

Make chart bars clickable. Clicking a bar navigates to the Dashboard filtered to show only movies matching that quality tier or dimension value.

### Navigation

Clicking "1080p BluRay" bar navigates to: `/dashboard?quality_resolution=1080p&quality_source=bluray`

Clicking a dimension bar (e.g., "1080p" in resolution view) navigates to: `/dashboard?quality_resolution=1080p`

### Files

| File | Change |
|------|--------|
| `web/ui/src/pages/stats/QualityCard.tsx` | Add onClick handlers to chart bars that navigate with URL params |
| `web/ui/src/pages/dashboard/DashboardPage.tsx` | Read `quality_resolution`, `quality_source` from URL params; apply as filters to movie list |

---

## Step 3: Backend Filter API

### What

New API endpoint `GET /api/v1/stats/quality/movies` that returns movie IDs matching a quality filter. The Dashboard uses this to filter its movie list when quality URL params are present.

Query params: `?resolution=X&source=Y&codec=Z&hdr=W` -- all optional, combined with AND.

### Files

| File | Change |
|------|--------|
| `internal/api/v1/stats.go` | Add `handleQualityMovies` handler |
| `internal/api/router.go` | Register `GET /api/v1/stats/quality/movies` |
| `web/ui/src/api/stats.ts` | Add `getQualityMovies(filters)` API call |

---

## Step 4: Upgrade Potential Indicators

### What

For each quality tier in the bar chart, show a small indicator of how many movies in that tier could be upgraded (based on their quality profile settings). This reuses data from the upgrade recommendations system.

Display: next to each bar, a small badge like "+12 upgradable" in a muted color. Clicking the badge navigates to the Wanted page's Upgrades tab filtered to that tier.

### Implementation

When rendering the tier view, make an additional call to the `GET /api/v1/wanted/upgrades` endpoint and cross-reference tier labels with upgrade tier labels to get counts.

### Files

| File | Change |
|------|--------|
| `web/ui/src/pages/stats/QualityCard.tsx` | Fetch upgrade data; render upgrade badges per tier; link to Wanted upgrades tab |
| `web/ui/src/pages/wanted/WantedPage.tsx` | Accept URL param `?tier=X` to pre-filter Upgrades tab |

---

## Implementation Order

```
Step 1: Combined quality tier view      [independent]
Step 2: Clickable chart bars            [depends on 1 for tier view, independent for dimension view]
Step 3: Backend filter API              [depends on 2 for navigation target]
Step 4: Upgrade potential indicators    [depends on upgrade-recommendations plan]
```

**PR Strategy**:
- PR 1: Steps 1-3 (tier view, clickable bars, filter API)
- PR 2: Step 4 (upgrade indicators -- after upgrade-recommendations lands)

---

## Key Design Decisions

- **Tier = resolution + source only**: Including codec or HDR in tier labels would create too many fine-grained tiers. Resolution + source captures the primary quality differences users care about.
- **Dashboard accepts URL params, not a new page**: Reusing the existing Dashboard with filter params avoids creating a separate "filtered movies" page. Filters are shown as removable chips at the top.
- **Upgrade indicators are additive**: If the upgrade recommendations feature isn't available yet, the tier view still works -- just without the upgrade badges.
- **No new database queries for tier view**: Tier grouping is done in the API handler by iterating existing movie file data and grouping in Go. The data set (one row per movie) is small enough for in-memory grouping.

---

## Risks Summary

| Risk | Impact | Mitigation |
|------|--------|------------|
| Too many tiers clutters the chart | Hard to read with 15+ bars | Cap display at top 10 tiers, group remainder into "Other" |
| Dashboard filter state conflicts with existing filters | Confusing UX | Quality URL params are independent; show clear "filtered by quality" chip that can be dismissed |
| Upgrade count computation is expensive | Slow stats page load | Fetch upgrade data lazily (only when tier view is active); cache client-side |
| Tier labels don't match upgrade tier labels | Cross-reference fails | Use same normalization logic in both; share tier label generation code |
