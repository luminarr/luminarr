# Phase B — Import Lists + Discovery

**Goal:** Auto-add movies from external sources (Trakt, TMDb, Plex Watchlist, IMDb). This is the #2 most requested feature and the primary way users discover and populate their library.

**Depends on:** Phase A (tags — lists assign tags to imported movies).

---

## B1. Import List Plugin System

Follow the existing plugin pattern (indexer/downloader/notification). Each list type is a plugin that returns a list of movies to consider adding.

### Plugin Interface

```go
// pkg/plugin/importlist.go

type ImportListItem struct {
    TMDbID    int    `json:"tmdb_id"`
    IMDbID    string `json:"imdb_id,omitempty"`
    Title     string `json:"title"`
    Year      int    `json:"year"`
}

type ImportList interface {
    // Fetch returns all movies from the list source.
    Fetch(ctx context.Context) ([]ImportListItem, error)
    // Test validates the configuration (credentials, connectivity).
    Test(ctx context.Context) error
    // Info returns plugin metadata.
    Info() ImportListInfo
}

type ImportListInfo struct {
    Kind        string   `json:"kind"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Fields      []Field  `json:"fields"`
}
```

### Registry Extension

Add to `internal/registry/registry.go`:
```go
func (r *Registry) RegisterImportList(kind string, factory func(json.RawMessage) (plugin.ImportList, error))
func (r *Registry) RegisterImportListSanitizer(kind string, fn SanitizerFunc)
```

### Database

**Migration 00028_import_lists.sql:**
```sql
CREATE TABLE import_list_configs (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    kind                TEXT NOT NULL,           -- "trakt_list", "tmdb_popular", etc.
    enabled             BOOLEAN NOT NULL DEFAULT true,
    settings            TEXT NOT NULL DEFAULT '{}',  -- plugin-specific JSON
    quality_profile_id  TEXT REFERENCES quality_profiles(id),
    root_folder         TEXT NOT NULL DEFAULT '',
    monitor             TEXT NOT NULL DEFAULT 'movie_only',  -- movie_only, movie_and_collection, none
    minimum_availability TEXT NOT NULL DEFAULT 'released',
    search_on_add       BOOLEAN NOT NULL DEFAULT true,
    tags                TEXT NOT NULL DEFAULT '[]',           -- JSON array of tag IDs
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE import_exclusions (
    id       TEXT PRIMARY KEY,
    tmdb_id  INTEGER NOT NULL UNIQUE,
    title    TEXT NOT NULL DEFAULT '',
    year     INTEGER NOT NULL DEFAULT 0,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/import-lists` | List all import lists |
| POST | `/api/v1/import-lists` | Create import list |
| GET | `/api/v1/import-lists/{id}` | Get import list details |
| PUT | `/api/v1/import-lists/{id}` | Update import list |
| DELETE | `/api/v1/import-lists/{id}` | Delete import list |
| POST | `/api/v1/import-lists/{id}/test` | Test connection |
| POST | `/api/v1/import-lists/sync` | Trigger manual sync (202) |
| GET | `/api/v1/import-exclusions` | List exclusions (paginated) |
| POST | `/api/v1/import-exclusions` | Add exclusion (TMDb ID + title/year) |
| DELETE | `/api/v1/import-exclusions/{id}` | Remove exclusion |
| POST | `/api/v1/import-exclusions/bulk` | Bulk add exclusions |

### Service: `internal/core/importlist/service.go`

```go
type Service struct {
    q        dbsqlite.Querier
    registry *registry.Registry
    movies   *movie.Service
    bus      *events.Bus
    logger   *slog.Logger
}

func (s *Service) List(ctx) ([]ImportListConfig, error)
func (s *Service) Create(ctx, req) (ImportListConfig, error)
func (s *Service) Update(ctx, id, req) (ImportListConfig, error)
func (s *Service) Delete(ctx, id) error
func (s *Service) Test(ctx, id) error
func (s *Service) Sync(ctx) error           // run all enabled lists
func (s *Service) SyncOne(ctx, id) error     // run single list

// Exclusions
func (s *Service) ListExclusions(ctx, limit, offset) ([]ImportExclusion, error)
func (s *Service) AddExclusion(ctx, tmdbID, title, year) error
func (s *Service) RemoveExclusion(ctx, id) error
```

### Sync Logic

```
For each enabled import list:
  1. Fetch items from plugin
  2. For each item:
     a. Skip if TMDb ID in import_exclusions
     b. Skip if movie already exists in library
     c. Lookup TMDb metadata (title, year, poster, etc.)
     d. Add movie with list's default settings (profile, root folder, monitor, availability, tags)
     e. If search_on_add → trigger auto-search
  3. Log results (added, skipped, excluded, failed)
```

### Scheduler Job

Add `import_list_sync` job:
- Default interval: 24 hours (configurable via future global setting)
- Calls `importlist.Service.Sync(ctx)`

---

## B2. List Plugins (Priority Order)

### B2a. Trakt Lists

**Kind:** `trakt_list`, `trakt_popular`, `trakt_user`

**Settings:**
```json
{
  "access_token": "...",
  "list_type": "watchlist|custom|trending|popular|anticipated|boxoffice",
  "username": "...",          // for custom lists
  "list_name": "...",         // for custom lists
  "limit": 100               // max items
}
```

**Auth:** Trakt uses OAuth2. Options:
- Simple: user provides a personal API key (client ID) — read-only public lists work without OAuth
- Full: OAuth2 device code flow for private lists/watchlists

**Start simple:** Support public lists + user-provided access token. OAuth flow is a later enhancement.

**API calls:**
- Trending: `GET https://api.trakt.tv/movies/trending`
- Popular: `GET https://api.trakt.tv/movies/popular`
- Watchlist: `GET https://api.trakt.tv/users/{user}/watchlist/movies`
- Custom list: `GET https://api.trakt.tv/users/{user}/lists/{list}/items/movies`

Each returns items with `ids.tmdb` and `ids.imdb`.

### B2b. TMDb Lists

**Kinds:** `tmdb_popular`, `tmdb_list`, `tmdb_person`, `tmdb_company`, `tmdb_keyword`, `tmdb_collection`

**Settings vary by kind:**

`tmdb_popular`:
```json
{
  "list_type": "popular|top_rated|upcoming|now_playing",
  "min_vote_average": 0,
  "min_vote_count": 0,
  "certification_country": "US",
  "certification": "",
  "genres": [],
  "limit": 100
}
```

`tmdb_list`:
```json
{
  "list_id": "12345"
}
```

`tmdb_person`:
```json
{
  "person_id": 12345,          // TMDb person ID
  "cast_only": true,           // exclude crew
  "min_vote_average": 0
}
```

`tmdb_company`:
```json
{
  "company_id": 12345          // e.g., A24 = 41077
}
```

`tmdb_keyword`:
```json
{
  "keyword_id": 12345
}
```

`tmdb_collection`:
```json
{
  "collection_id": 12345       // e.g., MCU = 529892
}
```

**Note:** Luminarr already has a TMDb client. Reuse `internal/tmdb/` for all API calls.

### B2c. Plex Watchlist

**Kind:** `plex_watchlist`

**Settings:**
```json
{
  "auth_token": "...",
  "plex_url": "https://plex.tv"
}
```

**API:** `GET https://metadata.provider.plex.tv/library/sections/watchlist/all` with `X-Plex-Token` header.

Plex returns items with GUIDs like `tmdb://12345` or `imdb://tt1234567`. Parse these to get TMDb/IMDb IDs.

### B2d. IMDb Lists

**Kind:** `imdb_list`

**Settings:**
```json
{
  "list_id": "ls012345678"     // IMDb list ID (from URL)
}
```

**Challenge:** IMDb doesn't have a public API. Two approaches:
1. Parse the public list export CSV (`https://www.imdb.com/list/{id}/export`)
2. Scrape the list page HTML

IMDb list export CSV is the cleanest. Each row has `Const` (IMDb ID like `tt1234567`). Map to TMDb via TMDb's "find by external ID" endpoint.

### B2e. RSS List

**Kind:** `rss_list`

**Settings:**
```json
{
  "url": "https://example.com/movies.rss"
}
```

Parse RSS/Atom feed. Extract movie title + year from feed items. Try to match to TMDb by search.

---

## B3. TMDb Collections (Franchise Tracking)

Extend the existing collections system to support TMDb franchise collections (Lord of the Rings, MCU, etc.) with auto-add.

### Database Changes

Add fields to existing collections table (or create new table if needed):
```sql
ALTER TABLE collections ADD COLUMN tmdb_collection_id INTEGER;
ALTER TABLE collections ADD COLUMN monitored BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE collections ADD COLUMN quality_profile_id TEXT REFERENCES quality_profiles(id);
ALTER TABLE collections ADD COLUMN root_folder TEXT NOT NULL DEFAULT '';
ALTER TABLE collections ADD COLUMN minimum_availability TEXT NOT NULL DEFAULT 'released';
ALTER TABLE collections ADD COLUMN search_on_add BOOLEAN NOT NULL DEFAULT true;
```

### Logic

During metadata refresh, if a movie belongs to a TMDb collection:
1. Create/update collection record
2. If collection is monitored:
   - Fetch all movies in collection from TMDb
   - Auto-add missing ones with collection's default settings
   - Skip movies in import exclusions

### API Additions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/collections` | List collections (existing — add monitored/settings fields) |
| PUT | `/api/v1/collections/{id}` | Update collection monitoring/settings |

---

## B4. Import Exclusions

Movies the user never wants auto-added. Prevents import lists and collection sync from re-adding unwanted movies.

UI: Accessible from Settings and from movie delete dialog ("Add to exclusion list" checkbox).

When deleting a movie that was auto-added by a list, offer to add it to exclusions.

---

## Frontend

### New Pages

| Page | Path | Description |
|------|------|-------------|
| Import Lists | Settings > Import Lists | CRUD for list configs, per-kind settings forms |
| Import Exclusions | Settings > Import Lists > Exclusions tab | Paginated list, search by title, bulk delete |
| Collections (enhanced) | Collections page | Add monitoring toggle, settings per collection |

### Import List Form Pattern

Same as indexer/download client forms: select kind → show kind-specific settings sub-form.

Kinds dropdown: Trakt (Watchlist, Custom List, Trending, Popular), TMDb (Popular, List, Person, Company, Keyword, Collection), Plex Watchlist, IMDb List, RSS.

Common fields (all kinds): name, enabled, quality profile, root folder, monitor type, minimum availability, search on add, tags.

---

## Files to Create/Modify

| Action | File |
|--------|------|
| Create | `pkg/plugin/importlist.go` — interface |
| Create | `internal/db/migrations/00028_import_lists.sql` |
| Create | `internal/db/queries/sqlite/import_lists.sql` |
| Create | `internal/core/importlist/service.go` |
| Create | `internal/api/v1/importlists.go` |
| Create | `plugins/importlists/trakt/trakt.go` |
| Create | `plugins/importlists/tmdb/popular.go` |
| Create | `plugins/importlists/tmdb/list.go` |
| Create | `plugins/importlists/tmdb/person.go` |
| Create | `plugins/importlists/plex/watchlist.go` |
| Create | `plugins/importlists/imdb/list.go` |
| Create | `plugins/importlists/rss/rss.go` |
| Create | `internal/scheduler/jobs/import_list_sync.go` |
| Modify | `internal/registry/registry.go` — add import list registration |
| Modify | `internal/api/router.go` — register routes |
| Modify | `cmd/luminarr/main.go` — blank imports + wire service |
| Modify | `internal/api/v1/collections.go` — monitoring/settings |
| Run | `sqlc generate` |

---

## Build Order

1. **B1** — Plugin system + service + API + scheduler job
2. **B2a** — Trakt plugin (most requested)
3. **B2b** — TMDb plugins (already have TMDb client)
4. **B2c** — Plex Watchlist
5. **B2d** — IMDb Lists
6. **B2e** — RSS Lists
7. **B3** — TMDb collection franchise tracking
8. **B4** — Import exclusions (ties into delete flow)

Frontend built incrementally as each plugin lands.

---

## Test Strategy

| Component | Test Type | Key Cases |
|-----------|-----------|-----------|
| Import list service | Unit | Sync logic: skip existing, skip excluded, add new, search_on_add |
| Trakt plugin | Unit | Parse API response, handle pagination, auth errors |
| TMDb plugin | Unit | Parse each list type, handle empty results |
| Plex plugin | Unit | Parse GUID formats (tmdb://, imdb://), auth errors |
| IMDb plugin | Unit | CSV parsing, TMDb ID lookup |
| Exclusions | Unit | CRUD, bulk operations, collision with existing movies |
| Integration | API | Full flow: create list → sync → verify movies added with correct settings |

---

## Estimated Scope

| Sub-phase | New Files | Modified Files | Migrations | Effort |
|-----------|-----------|---------------|------------|--------|
| B1 Plugin system | 5 | 4 | 1 | Medium |
| B2 List plugins (6) | 7 | 1 | 0 | Medium |
| B3 TMDb collections | 0 | 3 | 0 | Small |
| B4 Import exclusions | 0 | 2 | 0 | Small |
| Frontend | 3+ pages | — | — | Medium |
| **Total** | **~15** | **~10** | **1** | **Large** |
