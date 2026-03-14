# Feature 2: Media Server Integration — Watch-Aware Decisions

## The Problem

Radarr is completely blind to whether anyone actually watches the content it manages. It grabs, upgrades, and stores movies with zero awareness of consumption patterns. This leads to:

- Upgrading files nobody has ever watched (wasted bandwidth, time, indexer hits)
- Treating a movie watched 10 times the same as one never opened
- No data to inform storage reclamation decisions
- No way to prioritize grabs for active watchers

Luminarr already has media server plugins (Plex, Jellyfin, Emby) that can `RefreshLibrary()`, `Test()`, `ListSections()`, and `ListMovies()`. But the integration is one-directional — Luminarr pushes; it never pulls watch data.

## What This Feature Delivers

1. **Watch data sync** — pull play counts, last watched dates, and watch status from Plex/Jellyfin/Emby
2. **Smart upgrade prioritization** — upgrade frequently-watched movies first; deprioritize never-watched
3. **Storage reclamation suggestions** — "these 40 movies haven't been watched in 3 years, taking 800GB"
4. **Watch-aware dashboard** — "Recently Watched", "Never Watched", "Most Watched" views
5. **Upgrade ROI** — show estimated bandwidth savings from skipping upgrades on unwatched content

---

## Scope & Non-Goals

### In scope
- Fetch watch data from Plex, Jellyfin, Emby (play count, last played, watch status per user)
- Store aggregated watch data per movie (across all media server users)
- Watch-aware scoring in auto-search (watched movies get priority upgrades)
- "Unwatched" filter on movie list
- Storage suggestions page ("reclaim X GB by downgrading/removing unwatched movies")
- Scheduler job for periodic watch data sync
- Watch stats on movie detail page

### Out of scope (future)
- Per-user watch tracking (we aggregate across all media server users)
- Multi-user request systems ("family member X wants this movie")
- Progress tracking (partial watches / resume points)
- Watch history timeline (we store last_watched + play_count, not every play event)
- Automatic deletion of unwatched content (too dangerous; suggestions only)
- Rating sync (Plex ratings → Luminarr)

---

## Phase 1: Media Server Plugin API Extension

### 1.1 Extend the MediaServer interface

Current interface (`pkg/plugin/mediaserver.go`):
```go
type MediaServer interface {
    Name() string
    RefreshLibrary(ctx context.Context, moviePath string) error
    Test(ctx context.Context) error
}
```

New methods to add:
```go
type MediaServer interface {
    Name() string
    RefreshLibrary(ctx context.Context, moviePath string) error
    Test(ctx context.Context) error

    // GetWatchData returns watch statistics for all movies in the library.
    // Each WatchData entry is keyed by TMDB ID for cross-referencing.
    // Returns nil, nil if the media server doesn't support watch data.
    GetWatchData(ctx context.Context) ([]WatchData, error)
}

// WatchData represents aggregated watch statistics for a single movie
// across all users on a media server.
type WatchData struct {
    TmdbID      int       // For matching to Luminarr movies
    Title       string    // For display/debugging
    PlayCount   int       // Total plays across all users
    LastPlayed  time.Time // Most recent play across all users (zero if never played)
    WatchedBy   int       // Number of distinct users who've watched it
    TotalUsers  int       // Total users who have access to this item
}
```

### 1.2 Plex implementation

Plex API provides per-user watch data via:
- `GET /library/sections/{key}/all` — includes `viewCount` and `lastViewedAt` per video
- `GET /library/sections/{key}/all?X-Plex-Container-Start=0&X-Plex-Container-Size=100` — paginated
- `viewCount` attribute on `<Video>` elements = total views across all home users
- `lastViewedAt` attribute = epoch timestamp

New fields to parse from `plexVideo`:
```go
type plexVideo struct {
    // ... existing fields ...
    ViewCount    int    `xml:"viewCount,attr"`     // 0 if never watched
    LastViewedAt int64  `xml:"lastViewedAt,attr"`  // Unix epoch, 0 if never watched
}
```

Implementation: iterate all movie sections, collect video metadata, return `[]WatchData` with TMDB ID matching via existing `extractTmdbID()`.

### 1.3 Jellyfin implementation

Jellyfin API:
- `GET /Users/{userId}/Items?IncludeItemTypes=Movie&Fields=ProviderIds,UserData` — includes play data
- `UserData.PlayCount`, `UserData.LastPlayedDate`, `UserData.Played`
- Need to iterate all users (or use admin API): `GET /Users`

```go
type jellyfinItem struct {
    ID          string            `json:"Id"`
    Name        string            `json:"Name"`
    ProviderIDs map[string]string `json:"ProviderIds"` // {"Tmdb": "12345", "Imdb": "tt1234567"}
    UserData    struct {
        PlayCount      int    `json:"PlayCount"`
        LastPlayedDate string `json:"LastPlayedDate"` // ISO 8601 or empty
        Played         bool   `json:"Played"`
    } `json:"UserData"`
}
```

For multi-user aggregation: list all users, query items per user, aggregate.

### 1.4 Emby implementation

Emby's API is nearly identical to Jellyfin's (fork heritage). Same approach, same fields.

### 1.5 Graceful degradation

- If a media server plugin doesn't support `GetWatchData()`, return `nil, nil`
- The sync job skips servers that return nil
- UI shows "watch data unavailable" for unconfigured servers

---

## Phase 2: Database Schema

### 2.1 New migration: `00030_watch_data.sql`

```sql
-- +goose Up

CREATE TABLE movie_watch_data (
    movie_id            TEXT NOT NULL PRIMARY KEY REFERENCES movies(id) ON DELETE CASCADE,
    play_count          INTEGER NOT NULL DEFAULT 0,
    last_played         DATETIME,           -- NULL if never played
    watched_by_users    INTEGER NOT NULL DEFAULT 0,
    total_users         INTEGER NOT NULL DEFAULT 0,
    synced_at           DATETIME NOT NULL,  -- Last time this record was refreshed
    source_server_id    TEXT                -- Which media server config provided this data
);

CREATE INDEX movie_watch_data_play_count   ON movie_watch_data(play_count);
CREATE INDEX movie_watch_data_last_played  ON movie_watch_data(last_played);

-- +goose Down
DROP INDEX IF EXISTS movie_watch_data_last_played;
DROP INDEX IF EXISTS movie_watch_data_play_count;
DROP TABLE IF EXISTS movie_watch_data;
```

### 2.2 Why a separate table (not columns on `movies`)?

- Watch data is synced from an external source on a different cadence than movie metadata
- Not every movie will have watch data (no media server configured, or movie not in media server yet)
- Clean separation of concerns
- Can be wiped/resynced without touching movie records
- LEFT JOIN when needed, no NULL columns cluttering the movies table

### 2.3 Multi-server aggregation

If the user has multiple media servers (e.g. Plex + Jellyfin), we aggregate:
- `play_count` = MAX across servers (not SUM — same user likely watches on one server)
- `last_played` = most recent across servers
- `watched_by_users` = MAX across servers
- `source_server_id` = the server that provided the most recent data

Alternative: store per-server rows. This is more accurate but adds complexity. Recommendation: single row per movie (aggregated) for v1. Revisit if users request per-server breakdown.

### 2.4 sqlc queries

```sql
-- name: UpsertWatchData :exec
INSERT INTO movie_watch_data (movie_id, play_count, last_played, watched_by_users, total_users, synced_at, source_server_id)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(movie_id) DO UPDATE SET
    play_count = MAX(movie_watch_data.play_count, excluded.play_count),
    last_played = MAX(movie_watch_data.last_played, excluded.last_played),
    watched_by_users = MAX(movie_watch_data.watched_by_users, excluded.watched_by_users),
    total_users = MAX(movie_watch_data.total_users, excluded.total_users),
    synced_at = excluded.synced_at,
    source_server_id = excluded.source_server_id;

-- name: GetWatchData :one
SELECT * FROM movie_watch_data WHERE movie_id = ?;

-- name: ListUnwatchedMoviesWithFiles :many
SELECT m.id, m.title, m.year, mf.size_bytes, mf.quality_json, m.added_at
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
LEFT JOIN movie_watch_data wd ON wd.movie_id = m.id
WHERE (wd.play_count IS NULL OR wd.play_count = 0)
ORDER BY m.added_at ASC;

-- name: ListMostWatchedMovies :many
SELECT m.id, m.title, m.year, wd.play_count, wd.last_played
FROM movies m
JOIN movie_watch_data wd ON wd.movie_id = m.id
WHERE wd.play_count > 0
ORDER BY wd.play_count DESC
LIMIT ?;

-- name: ListStaleMovies :many
-- Movies with files that haven't been watched in N days
SELECT m.id, m.title, m.year, mf.size_bytes, mf.quality_json, wd.last_played, wd.play_count
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
JOIN movie_watch_data wd ON wd.movie_id = m.id
WHERE wd.last_played < ?
  AND wd.play_count > 0
ORDER BY wd.last_played ASC;

-- name: WatchDataSummary :one
SELECT
    COUNT(*) FILTER (WHERE wd.play_count > 0) as watched_count,
    COUNT(*) FILTER (WHERE wd.play_count = 0 OR wd.movie_id IS NULL) as unwatched_count,
    COALESCE(SUM(CASE WHEN wd.play_count = 0 OR wd.movie_id IS NULL THEN mf.size_bytes ELSE 0 END), 0) as unwatched_bytes
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
LEFT JOIN movie_watch_data wd ON wd.movie_id = m.id;

-- name: DeleteWatchData :exec
DELETE FROM movie_watch_data WHERE movie_id = ?;

-- name: DeleteAllWatchData :exec
DELETE FROM movie_watch_data;
```

---

## Phase 3: Watch Sync Service

### 3.1 New service: `internal/core/watchsync/service.go`

```go
package watchsync

type Service struct {
    db          *db.Queries
    mediaServers *mediaserver.Service  // existing service
    movies      *movie.Service
    logger      *slog.Logger
}

// Sync fetches watch data from all enabled media servers and upserts into the database.
func (s *Service) Sync(ctx context.Context) (*SyncResult, error)

// SyncResult reports what happened during a sync.
type SyncResult struct {
    ServersQueried int
    MoviesMatched  int     // Movies with TMDB ID matches
    MoviesUpdated  int     // Watch records upserted
    Errors         []string
}
```

**Sync algorithm:**
1. List all enabled media server configs
2. For each server, instantiate the plugin and call `GetWatchData(ctx)`
3. For each `WatchData` item with a non-zero `TmdbID`:
   a. Look up the movie in Luminarr by TMDB ID
   b. If found, upsert watch data
4. Return summary

**Match by TMDB ID:** Both Plex and Jellyfin expose TMDB IDs via their GUIDs/ProviderIds. This is the most reliable matching strategy. Fallback to title+year matching if no TMDB ID.

### 3.2 Scheduler job

Register a new scheduler job: `watch_sync`
- Default interval: 6 hours (watch data doesn't change that fast)
- Also runnable on demand via `POST /api/v1/system/tasks/watch_sync/run-now`
- First run on startup (after a 30s delay to let media servers come up)

### 3.3 Event integration

On `import_complete` event, consider triggering a targeted sync for that specific movie (not a full sync). This ensures watch data is fresh for newly imported movies.

---

## Phase 4: Watch-Aware Auto-Search

### 4.1 Upgrade priority scoring

Modify `autosearch.Service` to factor in watch data when ranking movies for bulk upgrades:

**Current behavior:** All monitored movies with cutoff-unmet files are searched equally.

**New behavior:** Movies are scored for upgrade priority:

```go
type UpgradePriority struct {
    MovieID    string
    BaseScore  int    // From quality profile (how far below cutoff)
    WatchScore int    // From watch data
    TotalScore int    // BaseScore + WatchScore
}
```

Watch score calculation:
| Watch Status | Score |
|---|---|
| Watched 5+ times | +100 |
| Watched 2-4 times | +75 |
| Watched once | +50 |
| Watched in last 30 days | +25 (additive) |
| Never watched, added > 90 days ago | -25 |
| Never watched, added > 365 days ago | -50 |

Movies with higher total scores are searched first in bulk operations.

### 4.2 "Skip unwatched upgrades" setting

New global setting in `media_management` or a new config section:
```
skip_upgrades_for_unwatched: false  (default)
skip_upgrades_after_days: 0         (0 = disabled)
```

When enabled:
- Movies that have never been watched AND were added more than N days ago are skipped in auto-search
- They can still be manually searched
- This is a bandwidth/indexer-hit optimization

### 4.3 Individual movie override

Per-movie flag: `upgrade_priority` (enum: `default`, `high`, `low`, `skip`)
- `high`: always prioritize upgrades regardless of watch status
- `low`: deprioritize but still search
- `skip`: never auto-upgrade (manual only)
- `default`: use watch data to determine priority

---

## Phase 5: API Endpoints

### 5.1 Watch data on movie detail

`GET /api/v1/movies/{id}` response adds (via LEFT JOIN):
```json
{
  "watch_data": {
    "play_count": 3,
    "last_played": "2025-12-15T20:30:00Z",
    "watched_by_users": 2,
    "total_users": 4,
    "synced_at": "2026-03-13T06:00:00Z"
  }
}
```

`null` if no watch data available.

### 5.2 Watch data endpoints

```
GET /api/v1/watch/summary
```
Returns aggregated watch stats:
```json
{
  "watched_count": 280,
  "unwatched_count": 60,
  "unwatched_bytes": 128849018880,
  "most_watched": [
    {"movie_id": "...", "title": "The Matrix", "play_count": 12},
    ...
  ],
  "recently_watched": [
    {"movie_id": "...", "title": "Dune", "last_played": "2026-03-12"},
    ...
  ]
}
```

```
GET /api/v1/watch/unwatched
```
Returns movies with files that have never been played:
```json
{
  "movies": [
    {
      "id": "...",
      "title": "Movie Nobody Watched",
      "size_bytes": 4294967296,
      "added_at": "2024-06-15",
      "quality": {"resolution": "1080p", ...}
    }
  ],
  "total_bytes": 128849018880
}
```

```
GET /api/v1/watch/stale?days=365
```
Returns movies not watched in N days:
```json
{
  "movies": [...],
  "total_bytes": 85899345920,
  "suggestion": "Downgrading these 30 movies from 4K to 1080p would save ~45GB"
}
```

```
POST /api/v1/watch/sync
```
Triggers an immediate watch data sync (same as running the scheduler job).

### 5.3 Movie list filtering

`GET /api/v1/movies` adds query params:
- `watched=true|false` — filter by watch status
- `sort=play_count|last_played` — sort options
- `min_play_count=N` — minimum plays

### 5.4 Stats integration

`GET /api/v1/stats/collection` adds:
```json
{
  "watched": 280,
  "unwatched": 60,
  "unwatched_bytes": 128849018880
}
```

---

## Phase 6: Frontend Changes

### 6.1 Movie detail page

- **Watch badge** in header: play count icon + count, last watched date
  - `"Watched 3 times | Last: 2 weeks ago"` with a play icon
  - `"Never watched"` with a dimmed icon for unwatched
- **Watch data section** (new tab or inline section):
  - Play count, last played, watched by X of Y users
  - "Synced 2 hours ago" timestamp

### 6.2 Movie list

- Optional "Watched" column (play count or checkmark)
- Filter bar: "All | Watched | Unwatched"
- Sort by play count / last watched

### 6.3 New page: Storage Suggestions

Under Settings or as a top-level page:
- **Unwatched Movies** — list with total size, individual sizes
- **Stale Movies** — watched but not in >1 year, with sizes
- **Upgrade ROI** — "skipping upgrades for 40 unwatched movies would save ~200GB of downloads"
- Each section has a total bytes callout and optional bulk actions (unmonitor, etc.)

### 6.4 Dashboard integration

- "Watched vs Unwatched" donut chart in stats
- "Recently Watched" mini-list
- "Never Watched" count as a card

### 6.5 Types

```typescript
interface WatchData {
  play_count: number;
  last_played: string | null;  // ISO 8601 or null
  watched_by_users: number;
  total_users: number;
  synced_at: string;
}

interface Movie {
  // ... existing ...
  watch_data?: WatchData | null;
}

interface WatchSummary {
  watched_count: number;
  unwatched_count: number;
  unwatched_bytes: number;
  most_watched: WatchedMovie[];
  recently_watched: WatchedMovie[];
}

interface WatchedMovie {
  movie_id: string;
  title: string;
  play_count: number;
  last_played?: string;
  size_bytes?: number;
}
```

---

## Implementation Order

1. **MediaServer interface extension** (add `GetWatchData()`)
2. **Plex plugin** — implement `GetWatchData()` (Plex first — most common)
3. **Database migration** — `movie_watch_data` table
4. **sqlc queries** — upsert, list, summary
5. **Watch sync service** — sync logic + match by TMDB ID
6. **Scheduler job** — 6-hour watch_sync
7. **API endpoints** — watch summary, unwatched, stale, sync trigger
8. **Movie detail enrichment** — watch data on movie GET
9. **Auto-search changes** — upgrade priority scoring
10. **Jellyfin plugin** — implement `GetWatchData()`
11. **Emby plugin** — implement `GetWatchData()`
12. **Frontend** — movie detail badges, filters, storage suggestions page
13. **Dashboard** — watched/unwatched charts

Estimated: ~10-12 implementation sessions.

---

## Key Decisions to Make

1. **Aggregation strategy for multi-server?**
   - MAX (current plan): simplest, assumes same user base
   - SUM: would double-count if same person watches on both servers
   - Per-server rows: most accurate but more complex
   - Recommendation: MAX for v1

2. **Sync frequency?**
   - 6 hours (current plan): low overhead, watch data is slow-moving
   - 1 hour: more responsive but more API calls to media servers
   - Configurable: let user set it
   - Recommendation: default 6h, configurable via settings

3. **Should "never watched" affect auto-search for new movies?**
   - A movie added yesterday hasn't been watched yet — it shouldn't be deprioritized
   - Grace period: don't penalize movies added in the last N days
   - Recommendation: 90-day grace period before "never watched" penalty applies

4. **Privacy considerations?**
   - Watch data reveals viewing habits
   - All data stays local (no external sync)
   - No per-user breakdown stored (aggregated only)
   - Should we add a "clear watch data" button?
   - Recommendation: yes, add clear button + auto-delete when media server config is removed

5. **What if the user doesn't have a media server configured?**
   - Feature gracefully degrades — watch data section shows "Connect a media server to see watch data"
   - No watch-based scoring in auto-search (all movies treated equally)
   - Storage suggestions page still works for basic stats (size, quality), just no watch-based suggestions

6. **Impact on the MediaServer plugin interface?**
   - Adding `GetWatchData()` is a breaking change to the interface
   - All three plugins (Plex, Jellyfin, Emby) need to implement it
   - Can we add it as an optional interface? `type WatchDataProvider interface { GetWatchData() }`
   - Recommendation: optional interface (type assertion at call site) — cleaner for future plugins that may not support watch data
