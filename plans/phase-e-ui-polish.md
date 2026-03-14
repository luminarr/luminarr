# Phase E — UI & Settings Completeness

**Goal:** UI settings, logs page, saved filters, iCal feed, and URL base support. The final polish layer that makes Luminarr feel complete and production-ready.

**Depends on:** Core features from Phases A–D. This phase is mostly frontend + small backend endpoints.

---

## E1. UI Settings

User-configurable display preferences.

### Storage

Two approaches:
- **Server-side:** `ui_settings` table, API-backed. Settings persist across browsers/devices.
- **Client-side:** localStorage. Simpler, no API needed, but per-browser.

**Recommendation:** Server-side, single row table (like media_management). Matches Radarr's approach.

### Database

**Migration 00033_ui_settings.sql:**
```sql
CREATE TABLE ui_settings (
    id                  TEXT PRIMARY KEY DEFAULT 'default',
    first_day_of_week   INTEGER NOT NULL DEFAULT 0,      -- 0=Sunday, 1=Monday
    short_date_format   TEXT NOT NULL DEFAULT 'MMM D YYYY',
    long_date_format    TEXT NOT NULL DEFAULT 'dddd, MMMM D YYYY',
    time_format         TEXT NOT NULL DEFAULT 'h:mma',    -- 'h:mma' (12hr) or 'HH:mm' (24hr)
    show_relative_dates BOOLEAN NOT NULL DEFAULT true,
    runtime_format      TEXT NOT NULL DEFAULT 'hours_minutes', -- 'hours_minutes' or 'minutes'
    movie_info_language TEXT NOT NULL DEFAULT 'en',
    theme               TEXT NOT NULL DEFAULT 'auto',      -- 'auto', 'light', 'dark'
    enable_color_impaired_mode BOOLEAN NOT NULL DEFAULT false
);

INSERT INTO ui_settings (id) VALUES ('default');
```

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/ui-settings` | Get UI settings |
| PUT | `/api/v1/ui-settings` | Update UI settings |

### Frontend Implementation

Create a `useUISettings()` hook that:
1. Fetches settings on app load (cached in React Query)
2. Provides format helper functions:
   - `formatDate(date, 'short' | 'long')` — uses configured format
   - `formatTime(date)` — 12hr or 24hr
   - `formatRelativeDate(date)` — "Today", "Yesterday", or absolute
   - `formatRuntime(minutes)` — "2h 15m" or "135 min"

Replace all hardcoded date/time formatting across the app with these helpers.

### Settings Page

New section in Settings: **UI** or **Display**
- First day of week dropdown (Sunday/Monday)
- Date format examples with live preview
- Time format toggle (12hr/24hr)
- Relative dates toggle
- Runtime format toggle
- Movie info language dropdown
- Theme selector (auto/light/dark)
- Color-impaired mode toggle

---

## E2. Logs UI Page

The backend endpoint exists (`GET /api/v1/logs`). Build the frontend page.

### Page: `src/pages/settings/system/LogsPage.tsx`

**Features:**
- Log level filter (debug, info, warn, error)
- Text search/filter
- Auto-scroll to bottom (live tail mode)
- Pause auto-scroll on manual scroll up
- Timestamp + level + message columns
- Color-coded by level (info=default, warn=yellow, error=red)
- Refresh button
- Optional: WebSocket-based live streaming (future enhancement)

### Backend Enhancement

Current `GET /api/v1/logs` returns logs from a ring buffer. Ensure it supports:
- `?level=warn` — minimum level filter
- `?limit=500` — max entries
- `?search=keyword` — text filter

If not already supported, add these query params.

### Log Download

Add endpoint to download log file:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/logs/file` | Download current log file |

This streams the log file (if file-based logging is configured) for support/debugging.

---

## E3. Saved/Custom Filters

Save filter combinations in the movie library for quick access.

### Database

**Migration 00034_custom_filters.sql:**
```sql
CREATE TABLE custom_filters (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'movie',  -- 'movie' for now, extensible
    conditions TEXT NOT NULL DEFAULT '[]',      -- JSON array of filter conditions
    sort_order INTEGER NOT NULL DEFAULT 0
);
```

### Filter Condition Format

```json
[
  { "field": "monitored", "operator": "eq", "value": true },
  { "field": "status", "operator": "in", "value": ["missing", "wanted"] },
  { "field": "quality_profile_id", "operator": "eq", "value": "profile-uuid" },
  { "field": "year", "operator": "gte", "value": 2020 },
  { "field": "tags", "operator": "contains", "value": "tag-uuid" }
]
```

**Supported fields:**
- `monitored` (bool)
- `status` (string enum)
- `quality_profile_id` (string)
- `year` (int)
- `tags` (string array)
- `has_file` (bool)
- `minimum_availability` (string enum)
- `library_id` (string)

**Operators:** `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `in`, `not_in`, `contains`

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/custom-filters` | List saved filters |
| POST | `/api/v1/custom-filters` | Create filter |
| PUT | `/api/v1/custom-filters/{id}` | Update filter |
| DELETE | `/api/v1/custom-filters/{id}` | Delete filter |

### Frontend

- Filter dropdown in movie list header (alongside existing sort/view controls)
- Predefined filters: All, Monitored, Unmonitored, Missing, Wanted, Cutoff Unmet
- Custom filters listed below predefined ones
- "Save current filter" button when custom conditions are active
- Filter builder: add/remove conditions, select field/operator/value

### Server-Side vs Client-Side Filtering

Current movie list API likely returns all movies (or paginated). Two options:
1. **Server-side:** Translate filter conditions to SQL WHERE clauses. More efficient for large libraries.
2. **Client-side:** Filter in the React app. Simpler, but requires loading all movies.

**Recommendation:** Server-side. Add `?filter=<json>` query param to `GET /api/v1/movies` that applies conditions as SQL filters. Client sends the saved filter's conditions.

---

## E4. iCal Calendar Feed

Expose an iCal endpoint that external calendar apps (Google Calendar, Apple Calendar) can subscribe to.

### Endpoint

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/feed/v1/calendar/luminarr.ics` | iCal calendar feed |

**Query params:**
- `pastDays=30` — include movies released in the last N days
- `futureDays=90` — include movies releasing in the next N days
- `tags=tag1,tag2` — filter by tags (optional)
- `unmonitored=true` — include unmonitored movies (default: false)

**Auth:** Include API key as query param `?apikey=xxx` (standard for iCal subscriptions since they can't set headers).

### iCal Format

```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Luminarr//Calendar//EN
X-WR-CALNAME:Luminarr
BEGIN:VEVENT
UID:luminarr-movie-12345-release@luminarr
DTSTART;VALUE=DATE:20240315
DTEND;VALUE=DATE:20240316
SUMMARY:Movie Title (2024) [Released]
DESCRIPTION:Runtime: 120 min\nQuality: 1080p Bluray\nStatus: Downloaded
CATEGORIES:Released
URL:https://www.themoviedb.org/movie/12345
END:VEVENT
END:VCALENDAR
```

### Event Types

Generate separate events for:
- **In Cinemas** date (if known)
- **Physical Release** date (if known, and different from digital)
- **Digital Release** date (if known)

Label each: `[In Cinemas]`, `[Physical]`, `[Digital]`

### Implementation

Single handler, no service layer needed. Query movies with release dates in range, format as iCal, return with `Content-Type: text/calendar`.

### Files to Create

| Action | File |
|--------|------|
| Create | `internal/api/v1/calendar_feed.go` |
| Modify | `internal/api/router.go` — register feed route |

---

## E5. URL Base (Reverse Proxy Support)

Serve Luminarr under a configurable path prefix (e.g., `https://myserver.com/luminarr/`).

### Configuration

Add to config.yaml:
```yaml
server:
  host: "0.0.0.0"
  port: 7878
  url_base: "/luminarr"   # NEW
```

### Backend Changes

1. **Router:** Strip URL base prefix from incoming requests before routing.
   ```go
   if cfg.Server.URLBase != "" {
       handler = http.StripPrefix(cfg.Server.URLBase, handler)
   }
   ```

2. **API base:** All API routes already use `/api/v1/` prefix. With URL base, they become `/luminarr/api/v1/`.

3. **Static assets:** The embedded SPA needs to know its base path. Inject `<base href="/luminarr/">` into `index.html`.

4. **WebSocket:** WS URL must include base path: `ws://host/luminarr/api/v1/ws`.

5. **Redirects:** Any server-side redirects must include the base path.

### Frontend Changes

1. **Vite config:** Set `base` to the URL base path (for asset URLs in built HTML).
   - Problem: base is runtime config, not build-time.
   - Solution: Use relative asset paths (`./`) and inject base path via `<base>` tag.

2. **API client:** `apiFetch` needs to prepend URL base to all requests.
   ```typescript
   const BASE = document.querySelector('base')?.getAttribute('href')?.replace(/\/$/, '') ?? '';
   const url = `${BASE}/api/v1/${endpoint}`;
   ```

3. **Router:** React Router needs `basename` prop:
   ```tsx
   <BrowserRouter basename={BASE}>
   ```

4. **WebSocket:** WS connection URL needs base path.

### Implementation Notes

- URL base must start with `/` and not end with `/`
- If empty string, behavior is unchanged (current default)
- Validate at startup: strip trailing slashes, ensure leading slash

### Files to Modify

| Action | File |
|--------|------|
| Modify | `internal/config/config.go` — add URLBase field |
| Modify | `internal/api/router.go` — StripPrefix middleware |
| Modify | `web/embed.go` — inject `<base>` tag into index.html |
| Modify | `web/ui/src/api/client.ts` — use base path |
| Modify | `web/ui/src/api/websocket.ts` — use base path for WS |
| Modify | `web/ui/src/App.tsx` or router setup — basename prop |

---

## Build Order

1. **E1 UI Settings** — standalone, polishes the whole app
2. **E2 Logs Page** — frontend only (backend exists), quick win
3. **E4 iCal Feed** — single endpoint, standalone
4. **E3 Saved Filters** — medium effort, good power-user feature
5. **E5 URL Base** — touches many files, do last (highest risk of regressions)

---

## Test Strategy

| Component | Test Type | Key Cases |
|-----------|-----------|-----------|
| UI Settings | API | CRUD, default values, invalid format rejection |
| Date formatting | Unit (TS) | Each format, relative dates, timezone handling |
| Logs API | API | Level filter, search, limit, empty results |
| Custom filters | Unit | Each operator, each field, multi-condition AND logic |
| Filter SQL generation | Unit | Correct WHERE clauses, SQL injection prevention |
| iCal generation | Unit | Valid iCal format, date-only events, special chars, timezone |
| iCal auth | API | API key in query param, reject without key |
| URL base | Integration | Routes work with base, static assets load, WS connects |
| URL base edge cases | Unit | Empty base, double slashes, trailing slash handling |

---

## Estimated Scope

| Sub-phase | New Files | Modified Files | Migrations | Effort |
|-----------|-----------|---------------|------------|--------|
| E1 UI Settings | 2 | 3 | 1 | Small-Medium |
| E2 Logs Page | 1 | 1 | 0 | Small |
| E3 Saved Filters | 3 | 3 | 1 | Medium |
| E4 iCal Feed | 1 | 1 | 0 | Small |
| E5 URL Base | 0 | 6 | 0 | Medium |
| Frontend | 3+ pages | — | — | Medium |
| **Total** | **~10** | **~14** | **2** | **Medium** |

---

## Phase E Summary

This is the lightest phase — mostly frontend polish and small backend features. But it's what makes Luminarr feel **finished** rather than "functional but rough." The iCal feed, UI settings, and URL base are features users expect from a mature *arr app, and their absence signals "not ready for daily use."

After Phase E, Luminarr should be at **~90%+ Radarr feature parity** — enough to be a credible drop-in replacement for the vast majority of users.
