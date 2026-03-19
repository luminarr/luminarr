# Plan: Release Group Reputation Tracking

**Status**: Draft
**Scope**: New event tracking table, aggregation service, and stats page component
**Depends on**: Event bus (existing), release group extraction (release-group-audio-presets Step 1)

---

## Summary

Track release group performance over time by recording import and replacement events. When a movie file is imported, record which group produced it. When a file is later replaced by a better one, mark the original as replaced. Aggregate into per-group statistics (imports, still in library, replaced, retention rate) and surface on the Stats page.

All data stays local -- no external API calls, no phone-home. Data accumulates going forward only; no backfill for V1.

---

## Current State

- Release group is parsed from release titles (after release-group-audio-presets Step 1)
- Event bus exists with `TypeImportComplete` events fired when a movie file is imported
- No tracking of which release group produced a given file
- No way to know which groups tend to get replaced quickly
- Stats page exists but has no release group information

---

## Step 1: Database Migration

### What

New table `release_group_events` to record import and replacement events.

### Schema

```sql
CREATE TABLE release_group_events (
    id INTEGER PRIMARY KEY,
    group_name TEXT NOT NULL,
    movie_id INTEGER NOT NULL,
    event_type TEXT NOT NULL,  -- 'imported' or 'replaced'
    imported_at TEXT NOT NULL,
    replaced_at TEXT,
    replaced_by_group TEXT
);
```

Indexes on `group_name` and `movie_id` for efficient aggregation.

### Files

| File | Change |
|------|--------|
| `internal/db/migrations/00031_release_group_events.sql` | New migration file |
| `internal/db/queries/sqlite/release_group_events.sql` | New sqlc query file |
| `internal/db/generated/sqlite/` | Regenerate with sqlc |

### Queries

- `InsertReleaseGroupEvent` -- insert new import event
- `MarkReplacedByMovie` -- update existing event for a movie as replaced, set replaced_at and replaced_by_group
- `AggregateReleaseGroupStats` -- group by group_name, count imports, count still-in-library (replaced_at IS NULL), count replaced, compute avg days to replacement

---

## Step 2: Release Group Service

### What

New service `internal/core/releasegroup/service.go` that subscribes to the event bus and manages release group event recording.

### Behavior

On `TypeImportComplete` event:
1. Parse group name from the grab title using `quality.ParseReleaseGroup()`
2. If group name is empty, skip (no tracking for unknown groups)
3. Mark any existing event for this movie_id as replaced (set `replaced_at = now`, `replaced_by_group = new_group`)
4. Insert new event with `event_type = 'imported'`, `imported_at = now`

### Files

| File | Change |
|------|--------|
| `internal/core/releasegroup/service.go` | New file: service with event bus subscription |
| `internal/core/releasegroup/service_test.go` | New file: test event handling and replacement marking |
| `internal/registry/registry.go` | Wire releasegroup service and subscribe to event bus |

---

## Step 3: Stats Aggregation

### What

Add `GetReleaseGroupStats()` method to the releasegroup service that calls the aggregation query and returns sorted results.

### Response Type

`ReleaseGroupStat` struct:
- `GroupName` -- the release group name
- `TotalImports` -- total number of times this group's releases were imported
- `InLibrary` -- currently in library (not replaced)
- `Replaced` -- number of times replaced by another group's release
- `RetentionRate` -- percentage still in library (`InLibrary / TotalImports * 100`)
- `AvgDaysToReplacement` -- average days before replacement (null if never replaced)

Default sort: by TotalImports descending, with a minimum threshold of 2 imports to appear in results (avoids noise from one-off groups).

### Files

| File | Change |
|------|--------|
| `internal/core/releasegroup/service.go` | Add `GetReleaseGroupStats()` method |

---

## Step 4: API Endpoint

### What

New endpoint `GET /api/v1/stats/release-groups` returning aggregated release group stats.

Supports optional query params:
- `?min_imports=N` -- minimum import count to include (default 2)
- `?sort=retention|imports|replaced` -- sort order

### Files

| File | Change |
|------|--------|
| `internal/api/v1/stats.go` | Add `handleReleaseGroupStats` handler |
| `internal/api/router.go` | Register `GET /api/v1/stats/release-groups` |

---

## Step 5: Frontend

### What

New `ReleaseGroupCard` component on the Stats page showing a table of release group reputation data.

Table columns: Group Name, Imports, In Library, Replaced, Retention %, Avg Days to Replacement.

Color-code retention rate (green > 80%, yellow 50-80%, red < 50%). Sortable columns.

### Files

| File | Change |
|------|--------|
| `web/ui/src/api/stats.ts` | Add `getReleaseGroupStats()` API call |
| `web/ui/src/pages/stats/ReleaseGroupCard.tsx` | New file: release group reputation table component |
| `web/ui/src/pages/stats/StatsPage.tsx` | Add ReleaseGroupCard to page layout |
| `web/ui/src/types/index.ts` | Add `ReleaseGroupStat` type |

---

## Implementation Order

```
Step 1: Database migration + queries     [independent]
Step 2: Release group service            [depends on 1]
Step 3: Stats aggregation                [depends on 1+2]
Step 4: API endpoint                     [depends on 3]
Step 5: Frontend                         [depends on 4]
```

**PR Strategy**:
- PR 1: Steps 1-3 (backend: migration, service, aggregation)
- PR 2: Steps 4-5 (API + frontend)

---

## Key Design Decisions

- **Event-sourced, not snapshot**: Recording individual import/replacement events (not just current state) allows computing trends over time in future versions.
- **Forward-only, no backfill**: V1 does not attempt to reconstruct history from existing movie files. Data starts accumulating from the moment the migration runs. This avoids complex and error-prone backfill logic.
- **Privacy by design**: All data stays in the local SQLite database. No group names or stats are sent anywhere. No external API calls.
- **Minimum import threshold**: Groups with only 1 import are excluded from stats by default to reduce noise. The threshold is configurable via query param.
- **Replacement chain**: When movie A (group X) is replaced by movie A (group Y), and then replaced again by movie A (group Z), only the most recent event is "in library". Both X and Y are marked as replaced.

---

## Risks Summary

| Risk | Impact | Mitigation |
|------|--------|------------|
| Group extraction fails (empty string) | Events not recorded for those imports | Acceptable -- unknown groups are simply not tracked |
| Table grows unbounded | Disk usage over years | Events are tiny (~100 bytes each); 10k movies * 3 replacements = ~3MB. Can add retention policy later |
| Race condition on concurrent imports | Duplicate or missed replacement marking | Use transaction for mark-replaced + insert; SQLite serializes writes |
| Stats query slow on large event tables | Slow stats page load | Index on group_name; aggregation is a simple GROUP BY |
