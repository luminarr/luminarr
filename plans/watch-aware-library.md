# Plan: Watch-Aware Library Management

**Status**: Draft
**Scope**: Ingest watch data from media servers, surface it throughout the UI, enable watch-informed decisions
**Depends on**: Media server plugins, movie service, stats service

---

## Summary

Luminarr currently manages files but has no idea what you've actually watched. It can tell you "you have Alien in 1080p Bluray" but not "you watched Alien 3 times, last time 2 weeks ago." This plan adds watch data ingestion from Plex, Jellyfin, and Emby, stores it locally, and uses it to power features that don't exist anywhere in the *arr ecosystem: watched badges, smart upgrade prioritization, and storage cleanup suggestions.

---

## Why

No tool in the *arr ecosystem closes the loop between "I downloaded this" and "I actually watched this." Tautulli tracks Plex watch history but can't act on it. Radarr/Sonarr don't know what's been watched. The result: users accumulate terabytes of media they'll never watch again, and upgrade recommendations don't consider whether you actually care about a movie.

Watch awareness enables three things nothing else does:
1. **Visual:** "Watched" badge on movies you've seen
2. **Upgrade prioritization:** "You've rewatched Interstellar 4 times at 1080p — upgrade to 4K?" is more valuable than "Interstellar is below your cutoff"
3. **Storage cleanup:** "You watched these 40 movies once, 2+ years ago. Reclaim 290GB?" is actionable in a way that raw file lists aren't

---

## Architecture

### Plugin Interface Extension

Add an optional `WatchProvider` interface that media server plugins can implement:

```go
// WatchProvider is an optional interface for media servers that can report
// watch history. Plugins that don't support it simply don't implement it.
type WatchProvider interface {
    // WatchHistory returns watch events since the given timestamp.
    // Each event represents one completed playback (>= 90% watched).
    WatchHistory(ctx context.Context, since time.Time) ([]WatchEvent, error)
}

type WatchEvent struct {
    TMDBID      int       // movie identifier
    Title       string    // for display/logging
    WatchedAt   time.Time // when playback completed
    UserName    string    // media server user (for multi-user setups)
    DurationSec int       // how long they watched
    Percent     float64   // percentage of movie watched (0.0-1.0)
}
```

**Capability check at runtime:**
```go
if wp, ok := mediaServer.(plugin.WatchProvider); ok {
    events, err := wp.WatchHistory(ctx, since)
    // ...
}
```

Plugins that don't implement `WatchProvider` are silently skipped. No forced interface changes.

### Plugin Implementations

| Plugin | API Endpoint | Notes |
|---|---|---|
| **Plex** | `GET /status/sessions/history/all?sort=viewedAt:desc&filter=viewedAt>={since}` | Returns `viewedAt`, `duration`, `viewOffset`, `ratingKey`. Match to TMDB via Plex GUID. |
| **Jellyfin** | `GET /Users/{userId}/Items?IsPlayed=true&SortBy=DatePlayed&SortOrder=Descending` | Returns `UserData.LastPlayedDate`, `UserData.PlayCount`. Match to TMDB via provider IDs. |
| **Emby** | `GET /Users/{userId}/Items?IsPlayed=true&SortBy=DatePlayed&SortOrder=Descending` | Same API shape as Jellyfin (forked codebase). |

### Data Model

**New table:**

```sql
CREATE TABLE watch_history (
    id          TEXT PRIMARY KEY,
    movie_id    TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    tmdb_id     INTEGER NOT NULL,
    watched_at  TEXT NOT NULL,           -- ISO 8601
    user_name   TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL,           -- "plex", "jellyfin", "emby"
    UNIQUE(movie_id, watched_at, user_name)  -- deduplicate
);
CREATE INDEX idx_watch_history_movie ON watch_history(movie_id);
CREATE INDEX idx_watch_history_watched ON watch_history(watched_at DESC);
```

**Derived fields on movie (computed, not stored):**

```go
type WatchStatus struct {
    Watched       bool      // at least one watch event
    PlayCount     int       // total times watched
    LastWatchedAt time.Time // most recent watch
    FirstWatchedAt time.Time // earliest watch
}
```

These are computed via SQL aggregation (`COUNT`, `MAX`, `MIN` on watch_history grouped by movie_id) and included in the movie API response.

### Sync Service

**New file:** `internal/core/watchsync/service.go`

- Scheduled job: runs every 6 hours (configurable), polls each media server that implements `WatchProvider`
- On each run: calls `WatchHistory(ctx, lastSyncTime)` to get only new events
- Matches TMDB IDs to Luminarr movie IDs, inserts new watch records
- Stores `lastSyncTime` per media server to avoid re-fetching old data
- Also supports on-demand sync via API: `POST /api/v1/watch-sync/run`

### Webhook Alternative

For real-time watch tracking without polling, accept webhooks:

```
POST /api/v1/hooks/media-server
```

Plex, Jellyfin, and Emby all support outbound webhooks. Each plugin implements a `ParseWebhook(r *http.Request) (*WatchEvent, error)` method. The endpoint detects the source from the payload format and delegates to the right plugin.

This is optional — polling works fine for most users. Webhooks are for those who want instant updates.

---

## Frontend

### Movie List (Dashboard)

Add a small "watched" indicator on poster cards:

- **Watched:** subtle checkmark badge in the corner (similar to the existing green/yellow status dots)
- **Unwatched:** no badge (don't add visual noise for the default state)

### Movie Detail

On the Overview tab, add a watch info row below the file status card:

```
┌─────────────────────────────────────────┐
│  ▶ Watched 3 times · Last: 2 weeks ago │
└─────────────────────────────────────────┘
```

Or if unwatched:
```
┌─────────────────────────────────────────┐
│  ○ Not watched                          │
└─────────────────────────────────────────┘
```

### Stats Page

New stat in the Collection row:
- **"Watched"** — count and percentage of library that's been watched
- **"Unwatched"** — count of movies never played

### Wanted Page — New "Smart Upgrades" Tab

Extend the existing Wanted page with a third tab (alongside Missing and Cutoff Unmet):

**"Worth Upgrading"** — movies below cutoff, sorted by play count descending. The logic: if you've watched it 5 times at 720p, upgrading to 4K is worth the disk space. If you've never watched it, maybe don't bother.

Columns: Title, Current Quality, Play Count, Last Watched, Upgrade Target

### Storage Cleanup Page (or section in Stats)

A "Reclaim Space" card that shows:

```
┌─────────────────────────────────────────────────┐
│  STORAGE CLEANUP SUGGESTIONS                    │
│                                                 │
│  Watched once, 2+ years ago:                    │
│  40 movies · 290 GB                             │
│  [View List]                                    │
│                                                 │
│  Watched, below cutoff (would re-download):     │
│  12 movies · 85 GB                              │
│  [View List]                                    │
│                                                 │
│  Never watched, added 1+ year ago:              │
│  28 movies · 195 GB                             │
│  [View List]                                    │
└─────────────────────────────────────────────────┘
```

Clicking "View List" shows the movies with select + delete controls. This is informational — Luminarr suggests, the user decides. No auto-deletion ever.

---

## API

| Endpoint | Method | Description |
|---|---|---|
| `GET /api/v1/movies/{id}` | GET | Extended response includes `watch_status` object |
| `GET /api/v1/movies` | GET | List response includes `watch_status` per movie |
| `GET /api/v1/watch-sync/run` | POST | Trigger manual watch sync |
| `GET /api/v1/stats/watch` | GET | Watch stats: total watched, unwatched, percentage |
| `GET /api/v1/stats/cleanup-suggestions` | GET | Storage cleanup candidates by category |

---

## Files

### New files

| File | Purpose |
|---|---|
| `pkg/plugin/watch.go` | `WatchProvider` and `WatchEvent` types |
| `internal/core/watchsync/service.go` | Watch sync service (poll + webhook + DB write) |
| `internal/api/v1/watchsync.go` | Watch sync API endpoints |
| `internal/db/migrations/000XX_watch_history.sql` | watch_history table |
| `internal/db/queries/watch.sql` | sqlc queries for watch data |
| `web/ui/src/api/watch.ts` | React Query hooks for watch data |

### Modified files

| File | Change |
|---|---|
| `plugins/mediaservers/plex/plugin.go` | Implement `WatchProvider` |
| `plugins/mediaservers/jellyfin/plugin.go` | Implement `WatchProvider` |
| `plugins/mediaservers/emby/plugin.go` | Implement `WatchProvider` |
| `internal/api/v1/movies.go` | Include `watch_status` in movie responses |
| `internal/core/stats/service.go` | Add watch stats + cleanup suggestion queries |
| `internal/scheduler/jobs/` | Add watch sync scheduled job |
| `web/ui/src/pages/dashboard/Dashboard.tsx` | Watched badge on poster cards |
| `web/ui/src/pages/movies/MovieDetail.tsx` | Watch info on Overview tab |
| `web/ui/src/pages/stats/StatsPage.tsx` | Watch stats card |
| `web/ui/src/pages/wanted/WantedPage.tsx` | "Worth Upgrading" tab |
| `web/ui/src/types/index.ts` | Add WatchStatus type |

---

## Implementation Order

1. Plugin interface (`WatchProvider`, `WatchEvent`)
2. Database migration (watch_history table)
3. sqlc queries
4. Watch sync service (polling mode)
5. Plex plugin: implement `WatchProvider`
6. Jellyfin plugin: implement `WatchProvider`
7. Emby plugin: implement `WatchProvider`
8. API: include watch_status in movie responses
9. Frontend: watched badge on dashboard
10. Frontend: watch info on movie detail
11. Frontend: watch stats on Stats page
12. Frontend: "Worth Upgrading" tab on Wanted page
13. Frontend: storage cleanup suggestions
14. Webhook receiver (optional, can ship later)
15. Scheduled job registration

---

## Testing

### Backend — Unit Tests

**Watch sync service — `internal/core/watchsync/service_test.go`:**
- Polls each configured media server that implements `WatchProvider`
- Skips media servers that don't implement `WatchProvider` (no error, no log noise)
- Inserts new watch events into watch_history table
- Deduplicates: same movie_id + watched_at + user_name → no duplicate row
- Matches TMDB IDs to Luminarr movie IDs correctly
- Watch events for TMDB IDs not in library → silently ignored (no orphan records)
- `lastSyncTime` advances after each successful sync
- `lastSyncTime` does not advance on sync failure (retries from same point)
- Concurrent sync of multiple media servers doesn't produce data races
- Empty watch history response → no inserts, no error

**Watch data queries — `internal/db/queries/watch_test.go` (via sqlc):**
- `WatchStatusForMovie(movie_id)` returns correct play_count, last_watched_at, first_watched_at
- `WatchStatusForMovie` for unwatched movie → returns zero play_count, null dates
- `WatchStatusForMovies(movie_ids)` batch query returns correct status per movie
- `CleanupSuggestions` query: watched once 2+ years ago → included in results
- `CleanupSuggestions` query: watched 3 times recently → excluded from results
- `CleanupSuggestions` query: never watched, added 1+ year ago → included in "never watched" category

**Plugin implementations — `plugins/mediaservers/plex/plugin_test.go`:**
- `WatchHistory(ctx, since)` parses Plex session history XML/JSON correctly
- Extracts TMDB ID from both new-agent GUIDs (`tmdb://12345`) and legacy agent GUIDs (`com.plexapp.agents.themoviedb://12345`)
- Filters to events after `since` timestamp
- Skips entries with < 90% watched (partial plays)
- Handles empty history response (no items)
- Handles Plex API errors (401, 500) with descriptive error

**Plugin implementations — `plugins/mediaservers/jellyfin/plugin_test.go`:**
- `WatchHistory(ctx, since)` parses Jellyfin API response correctly
- Extracts TMDB ID from provider IDs
- Handles users with no watch history
- Handles missing provider IDs gracefully (skips item, no error)

**Plugin implementations — `plugins/mediaservers/emby/plugin_test.go`:**
- Same test cases as Jellyfin (forked API shape)

**Stats service — `internal/core/stats/service_test.go`:**
- `WatchStats(ctx)` returns correct total watched, unwatched, percentage
- `CleanupSuggestions(ctx)` returns categorized suggestions with correct counts and byte totals
- Empty library → zero stats, no error
- Library with no watch data → all movies show as unwatched

### Backend — Integration Tests

**Full flow — `internal/api/integration_test.go`:**
- Add movie → sync watch data including that movie → `GET /api/v1/movies/{id}` returns `watch_status` with play_count=1
- Add movie with no watch data → `GET /api/v1/movies/{id}` returns `watch_status` with play_count=0
- `GET /api/v1/movies` list response includes `watch_status` for each movie
- `POST /api/v1/watch-sync/run` triggers sync and returns success
- `GET /api/v1/stats/watch` returns correct aggregate watch statistics
- `GET /api/v1/stats/cleanup-suggestions` returns categorized results
- Delete a movie → watch_history records cascade-deleted (foreign key constraint)

**Movie API contract — `internal/api/v1/movies_test.go`:**
- `watch_status` field present in single-movie response
- `watch_status` field present in list-movie response
- `watch_status.watched` is boolean
- `watch_status.play_count` is integer >= 0
- `watch_status.last_watched_at` is ISO 8601 string or null

### Frontend — Unit Tests

**Dashboard — `web/ui/src/pages/dashboard/Dashboard.test.tsx`:**
- Movie card with `watch_status.watched = true` renders watched badge
- Movie card with `watch_status.watched = false` does not render watched badge
- Movie card with no `watch_status` field does not render badge (backwards compat)

**MovieDetail — `web/ui/src/pages/movies/MovieDetail.test.tsx`:**
- Watch info row renders play count and last watched date when watched
- Watch info row renders "Not watched" when play_count is 0
- Watch info row hidden when watch_status is absent

**WantedPage — `web/ui/src/pages/wanted/WantedPage.test.tsx`:**
- "Worth Upgrading" tab renders movies sorted by play_count descending
- "Worth Upgrading" tab shows current quality, play count, last watched
- "Worth Upgrading" tab empty state when no movies qualify
- Tab is accessible and keyboard-navigable

**StatsPage — `web/ui/src/pages/stats/StatsPage.test.tsx`:**
- Watch stats card renders total watched count and percentage
- Watch stats card renders "0 watched" when no watch data exists
- Cleanup suggestions card renders categories with counts and sizes
- Cleanup suggestions "View List" button navigates or expands correctly

**MSW handlers:**
- Movie list/detail handlers extended with `watch_status` field in fixtures
- Add `GET /api/v1/stats/watch` mock handler
- Add `GET /api/v1/stats/cleanup-suggestions` mock handler
- Add `POST /api/v1/watch-sync/run` mock handler

---

## Risks

| Risk | Mitigation |
|---|---|
| Multi-user Plex: whose watch history counts? | Default: any user's watch counts as watched. Future: configurable user filter. |
| TMDB ID matching failures (Plex GUID format variations) | Reuse the existing `plexsync` GUID parsing logic — it already handles both old and new agent formats |
| Watch sync polling load on media server | 6-hour default interval, only fetches events since last sync. Minimal API load. |
| Privacy concerns (tracking what users watch) | All data stays local. Clear documentation. Manual sync trigger only if preferred. Option to disable entirely. |
| Cleanup suggestions feel dangerous | Purely informational. No auto-delete. Every action requires explicit user confirmation. |
