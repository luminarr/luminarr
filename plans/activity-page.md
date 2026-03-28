# Plan: Activity Page

**Status**: Draft
**Scope**: New Activity page replacing History as the primary "what happened" view
**Depends on**: Existing event bus, grab history, WebSocket hub

---

## Summary

Add an Activity page that shows a unified, real-time timeline of everything Luminarr has done — grabs, imports, task runs, health changes, metadata refreshes, and movie additions. The current History page only shows grab events. The Activity page answers "what has Luminarr been doing while I wasn't looking?" in a single chronological feed.

---

## Why

The History page shows grab events only. To understand what happened, users currently need to check History (grabs), Queue (downloads), System > Logs (everything else), and individual movie histories. There's no single place that shows "RSS sync ran, found 3 matches, grabbed 2, 1 was blocklisted" as a coherent narrative.

---

## Design

### Data Model

New `activity_log` table storing events persistently (the event bus is in-memory and ephemeral):

```sql
CREATE TABLE activity_log (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL,          -- event type (grab_started, import_complete, task_finished, etc.)
    category    TEXT NOT NULL,          -- "grab", "import", "task", "health", "movie", "system"
    movie_id    TEXT,                   -- nullable, for movie-related events
    title       TEXT NOT NULL,          -- human-readable summary
    detail      TEXT,                   -- optional extra context (JSON)
    created_at  TEXT NOT NULL           -- ISO 8601 timestamp
);
CREATE INDEX idx_activity_log_created ON activity_log(created_at DESC);
CREATE INDEX idx_activity_log_category ON activity_log(category);
CREATE INDEX idx_activity_log_movie ON activity_log(movie_id);
```

### Event → Activity Mapping

| Event Bus Type | Category | Title Example |
|---|---|---|
| `grab_started` | grab | "Grabbed Alien.1979.DC.1080p.BluRay from The Pirate Bay" |
| `grab_failed` | grab | "Grab failed for Alien: download client rejected" |
| `download_done` | import | "Download complete: Alien.1979.DC.1080p.BluRay" |
| `import_complete` | import | "Imported Alien (1979) — 1080p Bluray Director's Cut" |
| `import_failed` | import | "Import failed for Alien: file not found" |
| `movie_added` | movie | "Added Alien (1979) to library" |
| `movie_deleted` | movie | "Deleted Alien (1979)" |
| `task_started` | task | "RSS Sync started" |
| `task_finished` | task | "RSS Sync completed — checked 3 indexers" |
| `health_issue` | health | "disk_space: path not accessible" |
| `health_ok` | health | "disk_space: recovered" |
| `bulk_search_complete` | grab | "Bulk search: searched 12 movies, grabbed 3" |

### Backend

**New files:**
| File | Purpose |
|---|---|
| `internal/core/activity/service.go` | Activity log service — subscribe to event bus, persist to DB, query API |
| `internal/api/v1/activity.go` | `GET /api/v1/activity` with category filter, pagination, and since-timestamp |
| `internal/db/migrations/000XX_activity_log.sql` | Migration for activity_log table |
| `internal/db/queries/activity.sql` | sqlc queries: insert, list with filters, prune old entries |

**Pruning:** Auto-prune entries older than 30 days via a scheduled job (configurable). Keep the table bounded.

**API shape:**

```
GET /api/v1/activity?category=grab&limit=50&since=2026-03-01T00:00:00Z
```

Response:
```json
{
  "activities": [
    {
      "id": "...",
      "type": "grab_started",
      "category": "grab",
      "movie_id": "...",
      "title": "Grabbed Alien.1979.DC.1080p.BluRay from The Pirate Bay",
      "detail": {"indexer": "The Pirate Bay", "quality": "1080p bluray"},
      "created_at": "2026-03-28T10:15:00Z"
    }
  ],
  "total": 142
}
```

### Frontend

**New files:**
| File | Purpose |
|---|---|
| `web/ui/src/pages/activity/ActivityPage.tsx` | Activity timeline page |
| `web/ui/src/api/activity.ts` | React Query hook for activity API |

**UI layout:**
- Sidebar link: "Activity" between Dashboard and Calendar (with unread dot for new events since last visit)
- Category filter pills at top: All, Grabs, Imports, Tasks, Health, Movies
- Timeline layout: each entry shows icon (by category), title, relative timestamp, and optional movie link
- Real-time updates via existing WebSocket — new activities push to the top without refresh
- Empty state: "No recent activity"
- Click any movie-related activity to navigate to that movie's detail page

### What this replaces

The History page stays as-is (it has specific grab-level detail like score breakdowns that the Activity page won't duplicate). Activity is a higher-level view. History is for debugging a specific grab decision; Activity is for "what happened today."

---

## Implementation Order

1. Migration + sqlc queries
2. Activity service (event bus subscriber + DB writer)
3. API endpoint
4. Pruning scheduled job
5. Frontend page + API hook
6. WebSocket integration for real-time push
7. Sidebar navigation update

---

## Testing

### Backend

**Unit tests — `internal/core/activity/service_test.go`:**
- Event subscription: publish each event type → verify activity record created with correct category, title, and movie_id
- Title generation: verify human-readable titles are correct for each event type (grab_started includes release title and indexer, import_complete includes quality, etc.)
- Detail JSON: verify the detail field contains the expected structured data for each event type
- Deduplication: publish the same event twice → verify only one activity record (if applicable)
- Movie ID mapping: events with movie_id set → activity has movie_id; events without → activity has null movie_id
- Pruning: insert records older than retention period → run prune → verify they're deleted, recent records survive
- Pruning boundary: records exactly at the retention boundary → verify correct inclusion/exclusion
- Concurrent writes: publish 100 events concurrently → verify all recorded, no data races

**API tests — `internal/api/v1/activity_test.go`:**
- `GET /api/v1/activity` — returns all activities, newest first
- Category filter: `?category=grab` returns only grab activities
- Category filter: invalid category → 400 error
- Pagination: `?limit=10` returns at most 10 results with correct total count
- Since filter: `?since=<timestamp>` returns only activities after that time
- Combined filters: `?category=grab&limit=5&since=<timestamp>` → correct intersection
- Empty result set → 200 with empty array, not 404
- Movie link: activities with movie_id include it in response; activities without → null/omitted

**Integration tests — `internal/api/integration_test.go`:**
- Full flow: add a movie via API → verify `movie_added` activity appears in `GET /api/v1/activity`
- Full flow: trigger grab → verify `grab_started` activity with correct movie_id and release details

**Scheduled job tests:**
- Prune job: seed DB with old + recent activities → run job → verify only old entries removed
- Configurable retention: set retention to 7 days → verify 8-day-old records pruned, 6-day-old survive

### Frontend

**Component tests — `web/ui/src/pages/activity/ActivityPage.test.tsx`:**
- Renders activity list from mock API data
- Renders empty state when no activities
- Category filter pills: clicking a category filters the displayed results
- Each activity row shows icon, title, and relative timestamp
- Movie-linked activities render as clickable links to `/movies/{id}`
- Non-movie activities render as plain text (not clickable)
- Loading state: shows skeletons while fetching
- Error state: shows error message on API failure

**MSW handlers — `web/ui/src/test/handlers.ts`:**
- Add `GET /api/v1/activity` mock handler returning test fixtures covering all categories

**API hook tests — `web/ui/src/api/activity.test.tsx`:**
- `useActivity()` calls correct endpoint with default params
- `useActivity({ category: "grab" })` passes query parameter
- Error responses handled correctly

### Test Data

Test fixtures should cover:
- At least one activity per category (grab, import, task, health, movie, system)
- Activities with and without movie_id
- Activities with and without detail JSON
- Timestamps spanning multiple days (for relative time display testing)

---

## Risks

| Risk | Mitigation |
|---|---|
| High write volume (every event = a DB write) | Batch inserts, 30-day auto-prune |
| Duplicate with History page | Different purpose — Activity is timeline, History is grab detail. Keep both. |
| Table growth on busy instances | Pruning job + configurable retention |
