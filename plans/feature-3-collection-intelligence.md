# Feature 3: Collection Intelligence — "Complete the Set"

## The Problem

Radarr has "lists" — import-from-external-source feeds (IMDB, Trakt, SteelBooks, etc.). But lists are dumb pipes: they push movies in and Radarr blindly adds them. There's no *understanding* of what's in your library in relation to structured groups of films.

Movie collectors think in terms of:
- **Franchises:** "I have 2 of 3 Dark Knight films"
- **Director filmographies:** "I have 7 of 10 Kubrick films"
- **Award categories:** "I'm missing 3 Best Picture winners from the 2010s"
- **Curated sets:** "Complete the Studio Ghibli catalog"

Luminarr already has a `collections` table and TMDB franchise/person endpoints. But the current implementation is basic — it stores collections and shows items. There's no gap analysis, no smart suggestions, no "complete the set" workflow, and no automatic collection detection.

## What This Feature Delivers

1. **Auto-detect collections** — when you add a movie, Luminarr automatically identifies what franchise/collection it belongs to via TMDB
2. **Gap analysis** — "You have 2 of 3 in this trilogy — here's what's missing"
3. **Collection completeness tracking** — visual progress bars for each collection
4. **One-click "complete the set"** — add all missing movies from a collection with default settings
5. **Smart collection suggestions** — "Based on your library, you might want to complete these collections"
6. **Director/actor filmography tracking** (already partially implemented — enhance it)
7. **TMDB collection metadata** — auto-link movies to their TMDB collection (e.g. "The Dark Knight Collection")

---

## Scope & Non-Goals

### In scope
- Auto-detect TMDB collection membership when adding/importing movies
- Store TMDB collection_id on movies (new field)
- Franchise collection pages with gap analysis
- Director/actor collection pages (already exist — enhance with completeness tracking)
- "Suggested collections" based on what's in your library
- Completeness progress bars on collection cards
- "Add missing" bulk action per collection
- Collection stats (most complete, least complete, total gaps)
- Sort collections by completeness percentage

### Out of scope (future)
- Custom user-defined collections (smart playlists with rules like "Oscar winners > 7.5 rating")
- External list sync (Trakt, IMDB, Letterboxd)
- Award-based collections (no reliable structured data source)
- Cross-collection analysis ("you tend to collect Sci-Fi directors")
- Collection sharing/export

---

## Phase 1: TMDB Collection Auto-Detection

### 1.1 TMDB API: `belongs_to_collection`

The TMDB `/movie/{id}` endpoint already returns a `belongs_to_collection` field that our client currently ignores:

```json
{
  "id": 155,
  "title": "The Dark Knight",
  "belongs_to_collection": {
    "id": 263,
    "name": "The Dark Knight Collection",
    "poster_path": "/bqS2lMgGkPqIjgzHLhNiWnBVUx.jpg",
    "backdrop_path": "/bqS2lMgGkPqIjgzHLhNiWnBVUx.jpg"
  },
  ...
}
```

**Change to TMDB client:**

Update `GetMovie()` to also parse `belongs_to_collection`:

```go
type MovieDetail struct {
    // ... existing fields ...
    CollectionID   int    // TMDB collection ID (0 if not part of a collection)
    CollectionName string // "The Dark Knight Collection"
}
```

Update the raw struct in `GetMovie()`:
```go
var raw struct {
    // ... existing ...
    BelongsToCollection *struct {
        ID         int    `json:"id"`
        Name       string `json:"name"`
        PosterPath string `json:"poster_path"`
    } `json:"belongs_to_collection"`
}
```

### 1.2 Store collection link on movies

New migration field:
```sql
ALTER TABLE movies ADD COLUMN tmdb_collection_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE movies ADD COLUMN tmdb_collection_name TEXT NOT NULL DEFAULT '';

CREATE INDEX movies_tmdb_collection_id ON movies(tmdb_collection_id) WHERE tmdb_collection_id != 0;
```

When adding a movie via `movie.Service.Add()`, if TMDB returns a collection, store the collection ID and name on the movie record.

When refreshing metadata via `movie.Service.RefreshMetadata()`, update the collection link.

### 1.3 Backfill existing movies

A one-time migration task or scheduler job:
- For each movie with `tmdb_collection_id = 0` and `tmdb_id != 0`
- Call `GetMovie()` to check for `belongs_to_collection`
- Update if found

This can be a manual "Refresh All Metadata" action or part of the regular metadata refresh job.

---

## Phase 2: Collection Data Model Overhaul

### 2.1 Current state

The existing `collections` table is person-based (director/actor):
```sql
CREATE TABLE collections (
    id TEXT PRIMARY KEY,
    name TEXT,
    person_id INTEGER,
    person_type TEXT,  -- "director", "actor"
    created_at DATETIME
);
```

This is fine for person collections but doesn't cover TMDB franchise collections.

### 2.2 New unified collection model

**Option A: Extend the existing table with a `kind` column:**
```sql
ALTER TABLE collections ADD COLUMN kind TEXT NOT NULL DEFAULT 'person';
-- kind: "person", "franchise"
ALTER TABLE collections ADD COLUMN tmdb_collection_id INTEGER;
ALTER TABLE collections ADD COLUMN poster_url TEXT;
ALTER TABLE collections ADD COLUMN backdrop_url TEXT;
```

**Option B: Separate tables for franchise vs person collections.**

Recommendation: **Option A** — a single `collections` table with `kind` discriminator. Both types share the same UI patterns (list items, gap analysis, completeness).

### 2.3 Updated collections table

New migration: `00031_collection_intelligence.sql`

```sql
-- +goose Up

-- Add new columns to collections
ALTER TABLE collections ADD COLUMN kind TEXT NOT NULL DEFAULT 'person';
ALTER TABLE collections ADD COLUMN tmdb_collection_id INTEGER;
ALTER TABLE collections ADD COLUMN poster_url TEXT;
ALTER TABLE collections ADD COLUMN backdrop_url TEXT;

-- Unique index for franchise collections
CREATE UNIQUE INDEX collections_tmdb_collection_id
    ON collections(tmdb_collection_id)
    WHERE tmdb_collection_id IS NOT NULL;

-- Collection items table (denormalized snapshot of what belongs in each collection)
-- This caches the TMDB response so we don't re-fetch every page load.
CREATE TABLE collection_items (
    id                  TEXT NOT NULL PRIMARY KEY,
    collection_id       TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    tmdb_id             INTEGER NOT NULL,
    title               TEXT NOT NULL,
    year                INTEGER NOT NULL DEFAULT 0,
    poster_url          TEXT,
    sort_order          INTEGER NOT NULL DEFAULT 0,
    -- Denormalized fields (updated on sync):
    in_library          INTEGER NOT NULL DEFAULT 0,  -- 1 if movie exists in Luminarr
    movie_id            TEXT,                         -- FK to movies.id if in library (NULL otherwise)
    has_file            INTEGER NOT NULL DEFAULT 0,   -- 1 if movie has a file on disk
    UNIQUE(collection_id, tmdb_id)
);

CREATE INDEX collection_items_collection_id ON collection_items(collection_id);
CREATE INDEX collection_items_tmdb_id ON collection_items(tmdb_id);

-- +goose Down
DROP INDEX IF EXISTS collection_items_tmdb_id;
DROP INDEX IF EXISTS collection_items_collection_id;
DROP TABLE IF EXISTS collection_items;
DROP INDEX IF EXISTS collections_tmdb_collection_id;
```

### 2.4 Why `collection_items`?

Currently, collection item data is fetched live from TMDB on every page load. This is:
- Slow (API call per collection view)
- Rate-limited (TMDB has rate limits)
- Fragile (no data if TMDB is down)
- Can't cross-reference with library status without N+1 queries

By caching items in `collection_items`, we can:
- Show collection pages instantly from local data
- Run gap analysis purely against the database
- Update `in_library` / `has_file` flags via triggers or sync jobs
- Sort and filter collections efficiently

---

## Phase 3: Auto-Create Franchise Collections

### 3.1 Automatic collection creation

When a movie is added or metadata is refreshed and it has a `tmdb_collection_id`:

1. Check if a collection with that `tmdb_collection_id` exists
2. If not, create one:
   - Fetch `GetFranchise(ctx, collectionID)` from TMDB
   - Create collection record with `kind = "franchise"`
   - Populate `collection_items` with all parts
3. If it exists, update `collection_items.in_library` and `movie_id` for the new movie

### 3.2 Collection sync service

New service: `internal/core/collection/sync.go`

```go
// SyncFranchise refreshes collection_items for a franchise collection from TMDB.
func (s *Service) SyncFranchise(ctx context.Context, collectionID string) error

// SyncPerson refreshes collection_items for a person collection from TMDB.
func (s *Service) SyncPerson(ctx context.Context, collectionID string) error

// SyncAll refreshes all collections.
func (s *Service) SyncAll(ctx context.Context) (*SyncResult, error)

// UpdateLibraryStatus cross-references collection_items against the movies table
// and updates in_library/has_file/movie_id fields.
func (s *Service) UpdateLibraryStatus(ctx context.Context) error
```

### 3.3 Event-driven updates

Subscribe to events:
- `movie_added` → check if movie's TMDB collection exists; create if not; update `in_library`
- `movie_deleted` → update `in_library = 0` for that movie's collection items
- `import_complete` → update `has_file = 1` for the movie's collection item

### 3.4 Scheduler job

Register `collection_sync` job:
- Default interval: 24 hours
- Refreshes all collection items from TMDB (titles, years, new parts)
- Updates `in_library` / `has_file` statuses

---

## Phase 4: Gap Analysis Engine

### 4.1 Collection completeness

```go
type CollectionStats struct {
    ID              string
    Name            string
    Kind            string  // "franchise", "person"
    TotalItems      int
    InLibrary       int
    HasFile         int
    Missing         int     // TotalItems - InLibrary
    Completeness    float64 // InLibrary / TotalItems (0.0 to 1.0)
}
```

Computed from `collection_items` aggregate queries:
```sql
-- name: GetCollectionStats :one
SELECT
    c.id, c.name, c.kind,
    COUNT(ci.id) as total_items,
    SUM(ci.in_library) as in_library,
    SUM(ci.has_file) as has_file,
    COUNT(ci.id) - SUM(ci.in_library) as missing
FROM collections c
JOIN collection_items ci ON ci.collection_id = c.id
WHERE c.id = ?
GROUP BY c.id;

-- name: ListCollectionsByCompleteness :many
SELECT
    c.id, c.name, c.kind, c.poster_url,
    COUNT(ci.id) as total_items,
    SUM(ci.in_library) as in_library,
    COUNT(ci.id) - SUM(ci.in_library) as missing,
    CAST(SUM(ci.in_library) AS REAL) / COUNT(ci.id) as completeness
FROM collections c
JOIN collection_items ci ON ci.collection_id = c.id
GROUP BY c.id
ORDER BY completeness DESC, c.name ASC;
```

### 4.2 Suggested collections

"Based on your library, you might want to complete these collections":

Algorithm:
1. Query all franchise collections where `0 < in_library < total_items` (partially complete)
2. Sort by completeness descending (nearly complete ones first)
3. Optionally weight by: number of items in library, total franchise popularity

```sql
-- name: SuggestedCollections :many
-- Collections that are partially complete (have some but not all)
SELECT
    c.id, c.name, c.kind, c.poster_url,
    COUNT(ci.id) as total_items,
    SUM(ci.in_library) as in_library,
    COUNT(ci.id) - SUM(ci.in_library) as missing,
    CAST(SUM(ci.in_library) AS REAL) / COUNT(ci.id) as completeness
FROM collections c
JOIN collection_items ci ON ci.collection_id = c.id
GROUP BY c.id
HAVING in_library > 0 AND missing > 0
ORDER BY completeness DESC, in_library DESC
LIMIT ?;
```

### 4.3 Missing items per collection

```sql
-- name: ListMissingItems :many
SELECT ci.tmdb_id, ci.title, ci.year, ci.poster_url, ci.sort_order
FROM collection_items ci
WHERE ci.collection_id = ?
  AND ci.in_library = 0
ORDER BY ci.sort_order ASC;
```

---

## Phase 5: API Endpoints

### 5.1 Collection CRUD (enhanced)

Existing endpoints stay the same, with enhanced responses:

`GET /api/v1/collections` — now includes completeness data:
```json
[
  {
    "id": "...",
    "name": "The Dark Knight Collection",
    "kind": "franchise",
    "poster_url": "...",
    "total_items": 3,
    "in_library": 2,
    "has_file": 2,
    "missing": 1,
    "completeness": 0.667
  }
]
```

Supports query params:
- `kind=franchise|person` — filter by type
- `sort=completeness|name|missing` — sort order
- `incomplete_only=true` — only collections with missing items

`GET /api/v1/collections/{id}` — includes items:
```json
{
  "id": "...",
  "name": "The Dark Knight Collection",
  "kind": "franchise",
  "total_items": 3,
  "in_library": 2,
  "missing": 1,
  "completeness": 0.667,
  "items": [
    {
      "tmdb_id": 272,
      "title": "Batman Begins",
      "year": 2005,
      "poster_url": "...",
      "in_library": true,
      "has_file": true,
      "movie_id": "abc-123"
    },
    {
      "tmdb_id": 155,
      "title": "The Dark Knight",
      "year": 2008,
      "in_library": true,
      "has_file": true,
      "movie_id": "def-456"
    },
    {
      "tmdb_id": 49026,
      "title": "The Dark Knight Rises",
      "year": 2012,
      "in_library": false,
      "has_file": false,
      "movie_id": null
    }
  ]
}
```

### 5.2 New endpoints

```
GET /api/v1/collections/suggestions
```
Returns partially-complete collections sorted by how close they are to done:
```json
[
  {
    "id": "...",
    "name": "The Lord of the Rings Collection",
    "missing": 1,
    "total_items": 3,
    "completeness": 0.667,
    "missing_items": [
      {"tmdb_id": 122, "title": "The Lord of the Rings: The Return of the King", "year": 2003}
    ]
  }
]
```

```
POST /api/v1/collections/{id}/add-missing
```
Adds all missing movies from a collection to the library:
```json
{
  "library_id": "...",
  "quality_profile_id": "...",
  "monitored": true,
  "minimum_availability": "released"
}
```

Returns:
```json
{
  "added": 1,
  "skipped": 0,
  "errors": []
}
```

```
POST /api/v1/collections/{id}/sync
```
Refreshes collection items from TMDB.

```
GET /api/v1/collections/stats
```
Returns aggregate collection statistics:
```json
{
  "total_collections": 45,
  "franchise_collections": 30,
  "person_collections": 15,
  "fully_complete": 12,
  "total_missing_items": 87,
  "most_complete": [...],
  "nearest_to_complete": [...]  // 1 item away from complete
}
```

### 5.3 Movie endpoint enrichment

`GET /api/v1/movies/{id}` adds:
```json
{
  "collections": [
    {
      "id": "...",
      "name": "The Dark Knight Collection",
      "kind": "franchise",
      "completeness": 0.667,
      "missing": 1
    }
  ]
}
```

This shows which collections a movie belongs to, directly on the movie detail.

---

## Phase 6: Frontend Changes

### 6.1 Collections page (enhanced)

Current page shows a flat list. Enhanced version:

- **Tabs:** All | Franchises | Directors | Actors
- **Each card shows:**
  - Collection poster/backdrop
  - Name + type badge
  - Progress bar: `2 of 3 | 67%` with visual fill
  - "X missing" label
  - Grid of item posters (small, 4-5 across) with checkmark overlay for owned items
- **Sort options:** Completeness, Name, Missing count, Recently updated
- **Filter:** Incomplete only, Complete only

### 6.2 Collection detail page (enhanced)

Current page shows items in a grid. Enhanced version:

- **Header:** Collection poster, name, completeness bar, "2 of 3" stats
- **Items grid:**
  - Owned items: full color poster + quality badge + file size
  - Missing items: dimmed poster + "Add" button overlay
  - Sort by: release order, year, title
- **Actions:**
  - "Add All Missing" button (opens modal with library/profile selection)
  - "Search Missing" button (auto-search all missing monitored movies)
  - "Sync" button (refresh from TMDB)
- **Missing items callout:**
  - Prominent section at top: "Missing 1 movie to complete this collection"
  - List of missing items with "Add" button per item

### 6.3 Movie detail page

- **Collection membership:** Show collection badge(s) in header
  - e.g. pill: "The Dark Knight Collection (2 of 3)"
  - Clicking navigates to collection detail page

### 6.4 Dashboard integration

- **"Nearly Complete" widget:**
  - Top 5 collections that are 1-2 items from complete
  - "Add missing" quick action per collection
- **Collection stats card:**
  - Total collections, complete count, total missing

### 6.5 Suggested collections section

On Collections page or Dashboard:
- "Complete These Collections" section
- Shows collections sorted by how few items are missing
- One-click "Add Missing" per collection

### 6.6 Types

```typescript
interface Collection {
  id: string;
  name: string;
  kind: 'franchise' | 'person';
  person_id?: number;
  person_type?: string;
  tmdb_collection_id?: number;
  poster_url?: string;
  backdrop_url?: string;
  total_items: number;
  in_library: number;
  has_file: number;
  missing: number;
  completeness: number;  // 0.0 to 1.0
  items?: CollectionItem[];
}

interface CollectionItem {
  tmdb_id: number;
  title: string;
  year: number;
  poster_url?: string;
  sort_order: number;
  in_library: boolean;
  has_file: boolean;
  movie_id?: string;
}

interface CollectionSuggestion {
  id: string;
  name: string;
  missing: number;
  total_items: number;
  completeness: number;
  missing_items: CollectionItem[];
}

interface CollectionStats {
  total_collections: number;
  franchise_collections: number;
  person_collections: number;
  fully_complete: number;
  total_missing_items: number;
}
```

---

## Implementation Order

1. **TMDB client** — parse `belongs_to_collection` from `GetMovie()`
2. **Database migration** — `tmdb_collection_id` on movies, enhanced collections table, `collection_items` table
3. **sqlc queries** — collection CRUD, gap analysis, suggestions
4. **Movie service** — store collection link on add/refresh
5. **Collection sync service** — sync franchise items from TMDB, update library status
6. **Auto-create collections** — on movie add, check for franchise collection
7. **Event handlers** — update collection status on movie add/delete/import
8. **Scheduler job** — daily collection sync
9. **API endpoints** — enhanced collection list, detail, suggestions, add-missing, stats
10. **Backfill** — refresh metadata for existing movies to detect collections
11. **Frontend** — enhanced collections page, detail page, movie detail badges, dashboard widget

Estimated: ~8-10 implementation sessions.

---

## Key Decisions to Make

1. **Auto-create franchise collections on movie add?**
   - Yes (current plan): every movie with a TMDB collection auto-generates a collection
   - No: only manual collection creation, but show "this movie is part of X" as a suggestion
   - Hybrid: auto-detect but don't create until user opts in (show a "Create Collection?" prompt)
   - Recommendation: auto-create silently — most users want this, and it's easy to delete

2. **Should person collections (director/actor) also get collection_items caching?**
   - Yes: consistency, same gap analysis UI
   - No: person filmographies are large (Spielberg has 30+ films) and change often
   - Recommendation: yes, but with a staleness threshold — re-sync if items are older than 7 days

3. **How to handle franchises with unreleased parts?**
   - TMDB collections include announced/upcoming movies
   - Should we show these as "missing" or filter them out?
   - Recommendation: show them with a "Coming Soon" badge, don't count them in completeness %

4. **Collection auto-deletion?**
   - If all movies from a franchise are removed, should the collection be auto-deleted?
   - Recommendation: no — keep the collection (it's lightweight) so the user can re-add later

5. **Interaction with existing collection creation?**
   - Users can already create director/actor collections manually
   - Auto-created franchise collections should coexist without conflict
   - The "Add Collection" modal should clearly distinguish between franchise (TMDB collection) and person (director/actor) types

6. **TMDB rate limiting?**
   - Auto-creating collections triggers `GetFranchise()` calls
   - If user bulk-adds 50 movies from 20 different franchises, that's 20 TMDB API calls
   - TMDB rate limit is ~40 requests/10 seconds
   - Recommendation: use a rate limiter (existing `ratelimit.Registry`) for TMDB calls; queue franchise syncs with a small delay
