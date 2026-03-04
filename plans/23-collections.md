# Plan 23 — Director/Actor Collections

## Problem

Adding movies one at a time works for casual use but not for anyone who collects
systematically. "I want every Christopher Nolan film" currently means opening TMDB,
finding each film manually, adding them one by one. There's no concept of
"give me this filmmaker's complete body of work."

Radarr has Lists, but they're clunky: you import a Trakt/IMDB list and it adds
everything, monitored, no preview, no control. Luminarr should do this better.

---

## Goal

Let users create **collections** based on a director, actor, or other TMDB person.
The collection page shows the complete filmography with a clear view of what you
have and what you're missing, and lets you add missing films in bulk.

Scope: TMDB person filmographies only. No Trakt, no IMDB lists, no custom lists.
One thing, done well.

---

## TMDB API

Endpoint: `GET /person/{person_id}/movie_credits`

Returns `cast[]` and `crew[]` arrays. For directors, filter `crew` where
`job == "Director"`. For actors, use `cast`.

Each entry has: `id` (TMDB movie ID), `title`, `release_date`, `poster_path`.

We fetch this live — not stored — since filmographies change as new movies are
announced. The collection record stores only the person ID and type; the movie
list is always fetched fresh.

---

## Data Model

### Migration (`internal/db/migrations/00021_collections.sql`)

```sql
-- +goose Up
CREATE TABLE collections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    person_id   INTEGER NOT NULL,
    person_type TEXT NOT NULL DEFAULT 'director',  -- 'director' | 'actor'
    created_at  DATETIME NOT NULL
);
CREATE UNIQUE INDEX idx_collections_person ON collections(person_id, person_type);

-- +goose Down
DROP TABLE collections;
```

No junction table needed — filmography is always fetched live from TMDB.

### sqlc queries (`internal/db/queries/sqlite/collections.sql`)

```sql
-- name: CreateCollection :one
INSERT INTO collections (id, name, person_id, person_type, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListCollections :many
SELECT * FROM collections ORDER BY name ASC;

-- name: GetCollection :one
SELECT * FROM collections WHERE id = ?;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = ?;
```

Run `sqlc generate` after adding.

---

## Service (`internal/core/collection/service.go`)

```go
package collection

// CollectionItem is one film in a person's filmography.
type CollectionItem struct {
    TMDBID      int
    Title       string
    Year        int
    PosterPath  string
    InLibrary   bool    // true if movies table has this tmdb_id
    MovieID     string  // set when InLibrary=true
    Monitored   bool    // set when InLibrary=true
}

// Collection is the summary view of a collection.
type Collection struct {
    ID         string
    Name       string
    PersonID   int
    PersonType string
    CreatedAt  time.Time
    // Populated by GetWithItems — nil on List
    Items      []CollectionItem
    Total      int
    InLibrary  int
    Missing    int
}

type Service struct {
    q        dbsqlite.Querier
    metadata metadata.Provider  // for person/filmography lookups
    movieSvc *movie.Service
}

func (s *Service) Create(ctx context.Context, personID int, personType string) (*Collection, error)
func (s *Service) List(ctx context.Context) ([]Collection, error)
func (s *Service) Get(ctx context.Context, id string) (*Collection, error)  // fetches items live
func (s *Service) Delete(ctx context.Context, id string) error
```

`Create` logic:
1. Call TMDB `/person/{personID}` to get their name → use as default collection name
2. Insert into `collections`
3. Return collection (items not yet populated — loaded on Get)

`Get` logic:
1. Fetch collection from DB
2. Call TMDB `/person/{personID}/movie_credits`
3. For each film: query `movies` table by `tmdb_id` to check if in library
4. Sort: in-library first, then by release date desc (most recent missing films at top)
5. Return with Items, Total, InLibrary, Missing counts

---

## TMDB Provider Extension

Add two methods to the `metadata.Provider` interface:

```go
// GetPerson returns name and profile_path for a TMDB person.
GetPerson(ctx context.Context, personID int) (*Person, error)

// GetPersonFilmography returns directed films (type="director") or acted films (type="actor").
GetPersonFilmography(ctx context.Context, personID int, personType string) ([]FilmographyItem, error)
```

Implement in `internal/metadata/tmdb/tmdb.go`.

---

## API (`internal/api/v1/collections.go`)

```
GET    /api/v1/collections           — list all collections (no items)
POST   /api/v1/collections           — create { person_id, person_type }
GET    /api/v1/collections/{id}      — get with full item list (live TMDB fetch)
DELETE /api/v1/collections/{id}      — delete collection record

POST   /api/v1/collections/{id}/add-missing
    Body: { quality_profile_id, library_id, minimum_availability }
    Adds all non-library items as monitored movies in one batch.
    Returns: { added: int, skipped_duplicates: int }
```

Register in `router.go` under `RegisterCollectionRoutes`.

---

## Frontend

### Collections page (`src/pages/collections/CollectionsPage.tsx`)

Sidebar nav: **Collections** between Calendar and Statistics. Icon: `Users` (lucide).

**List view:**
Cards for each collection showing: person name, role (Director / Actor),
movie count, `X / Y in library` progress bar, Delete button.
Plus an **Add Collection** button.

**Add Collection flow:**

1. User clicks **Add Collection**
2. Search field — calls TMDB person search (`GET /api/v1/tmdb/person/search?q=`)
3. Results show: name, photo, known_for_department (Director/Actor/...)
4. User picks Director or Actor role, clicks **Add**
5. Collection created; navigate to collection detail

(Reuse the existing TMDB search API or add a person search endpoint.)

**Collection detail view (`src/pages/collections/CollectionDetail.tsx`):**

Header: person name + profile photo from TMDB, role badge, `X / Y in library`.

Below: grid of movies (same poster card style as movie gallery).

Each card:
- **Have it** → green border, click opens movie detail. No action button.
- **Missing** → standard card + **Add** button (small, below poster)

**Add** button opens a minimal modal:
```
Add "Interstellar" to library

Quality profile: [HD-1080p x265 ▼]
Library:         [/movies ▼]
Min. availability: [Released ▼]

[ Cancel ]  [ Add ]
```

**Add All Missing** button in header: opens same modal but applies settings to
all missing films at once. Shows count: "Add 7 missing films".

---

## TMDB Person Search Endpoint

If not already present, add:

```
GET /api/v1/tmdb/people/search?q=<name>
```

Returns `[{ person_id, name, profile_path, known_for_department }]`.

Calls TMDB `/search/person`. Add to `internal/api/v1/tmdb.go` (or new file).

---

## Tests

**Unit** (`internal/core/collection/service_test.go`):
- `TestCreate` — creates record with correct person_id, name from TMDB
- `TestGet_crossReferencesLibrary` — filmography item with matching tmdb_id shows InLibrary=true
- `TestGet_missing` — filmography item not in movies table shows InLibrary=false
- `TestDelete` — removes record
- `TestAddMissing` — calls movie service to add each missing film

**Integration** (`internal/api/v1/collections_test.go`):
- `TestListCollections` — GET returns correct shape
- `TestCreateCollection` — POST creates record
- `TestDeleteCollection` — DELETE returns 204

TMDB calls in tests: use the existing `metadata.MockProvider` pattern.

---

## Open Questions

1. **Person search UI**: Should search happen inline in the modal or redirect to a
   dedicated person search page? **Recommendation: inline modal** — simpler, consistent
   with how indexer/download client "Add" flows work.

2. **Filmography filtering**: TMDB returns ALL credits including minor roles and
   uncredited appearances for actors. Should we filter below a popularity threshold?
   **Recommendation:** for `actor` type, only show films where they are in the top 5
   billed cast (TMDB's `order` field). Director always shows full filmography.

3. **Refresh**: Should collections auto-refresh? A director adding a new film won't
   show in the collection until the user opens it (live fetch). That's acceptable
   for v1. If users want notifications for new films in a collection, that's a
   future feature.

4. **Duplicate handling**: If user creates "Christopher Nolan - Director" then
   separately tries to add the same person again, `UNIQUE INDEX` on `(person_id, person_type)`
   returns a 409. API and UI should handle this gracefully.
