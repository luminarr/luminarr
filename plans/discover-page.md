# Plan: Discover Page

**Status**: Draft
**Scope**: New browsable Discover page for finding and adding movies from TMDB
**Depends on**: TMDB client, existing Add Movie dialog, import list infrastructure

---

## Summary

Add a Discover page where users can browse TMDB popular, trending, upcoming, top-rated, and by-genre movies — and add them to their library with one click. Currently the only way to add a movie is the search box in the Add Movie dialog, which requires you to already know what you want. Discover lets users find movies they didn't know they wanted. This is the #1 reason Overseerr exists in the *arr ecosystem — Radarr's "add movie" experience is just a search box.

---

## Why

- Import lists run on a schedule and auto-add movies, but there's no way to *browse* what's available before committing
- The Add Movie flow requires typing a title — there's no visual browsing
- Overseerr/Jellyseerr exist primarily because Radarr doesn't have this. Building it into Luminarr eliminates the need for a separate request management app for single-user setups
- TMDB's API is free and already integrated — the data is there, we just don't surface it

---

## Design

### Page Layout

**Sidebar link:** "Discover" between Dashboard and Calendar (with a compass or telescope icon)

**Page structure:**

```
┌─────────────────────────────────────────────────────────┐
│  Discover                                               │
│  Find movies to add to your library                     │
├─────────────────────────────────────────────────────────┤
│  [Trending] [Popular] [Top Rated] [Upcoming] [By Genre] │ ← tab pills
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐    │
│  │    │ │    │ │    │ │    │ │    │ │    │ │    │    │  ← poster grid
│  │    │ │    │ │    │ │    │ │    │ │    │ │    │    │
│  │    │ │    │ │    │ │    │ │    │ │    │ │    │    │
│  └────┘ └────┘ └────┘ └────┘ └────┘ └────┘ └────┘    │
│  Title   Title   Title   Title   Title   Title   Title  │
│  2024    2024    2025    2024    2025    2024    2024    │
│                                                         │
│  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐    │
│  ...                                                    │
│                                                         │
│              [Load More]                                │
└─────────────────────────────────────────────────────────┘
```

### Tabs / Categories

| Tab | TMDB Endpoint | Description |
|---|---|---|
| Trending | `/trending/movie/week` | Movies trending this week |
| Popular | `/movie/popular` | All-time popular |
| Top Rated | `/movie/top_rated` | Highest rated |
| Upcoming | `/movie/upcoming` | Upcoming releases |
| By Genre | `/discover/movie?with_genres={id}` | Filter by genre with genre dropdown |

All endpoints support pagination (`page=1,2,3...`). TMDB returns 20 results per page.

### Movie Card

Each card in the grid shows:

- Poster (using the new `<Poster>` component from the posterless plan)
- Title + year
- TMDB rating (star + number, e.g. "7.4")
- **Status badge:**
  - **"In Library"** (green) — already tracked in Luminarr
  - **"+ Add"** (accent color button) — click to add
  - **"Excluded"** (muted) — on the import exclusion list

### Add Flow

Clicking "+ Add" on a card opens the existing `ConfigureStep` from the Add Movie dialog (library selector, quality profile, monitor toggle) as a small modal. No search step needed — we already have the TMDB ID.

If the movie is already in the library, clicking "In Library" navigates to the movie detail page.

### Genre Browser

The "By Genre" tab shows a genre dropdown populated from TMDB's genre list (`/genre/movie/list`). Selecting a genre loads `/discover/movie?with_genres={id}`. Could later support multiple genre filters, year range, rating minimum — but start simple with single genre.

---

## Backend

### New TMDB Client Methods

| File | Method | TMDB Endpoint |
|---|---|---|
| `internal/metadata/tmdb/client.go` | `Trending(ctx, page) ([]SearchResult, int, error)` | `GET /trending/movie/week?page=N` |
| | `Popular(ctx, page) ([]SearchResult, int, error)` | `GET /movie/popular?page=N` |
| | `TopRated(ctx, page) ([]SearchResult, int, error)` | `GET /movie/top_rated?page=N` |
| | `Upcoming(ctx, page) ([]SearchResult, int, error)` | `GET /movie/upcoming?page=N` |
| | `DiscoverByGenre(ctx, genreID, page) ([]SearchResult, int, error)` | `GET /discover/movie?with_genres=N&page=N` |
| | `GenreList(ctx) ([]Genre, error)` | `GET /genre/movie/list` |

Return type reuses existing `SearchResult`. The second return value is `total_pages` for pagination.

### New API Endpoints

| File | Endpoint | Description |
|---|---|---|
| `internal/api/v1/discover.go` | `GET /api/v1/discover/trending?page=1` | Trending movies |
| | `GET /api/v1/discover/popular?page=1` | Popular movies |
| | `GET /api/v1/discover/top-rated?page=1` | Top rated |
| | `GET /api/v1/discover/upcoming?page=1` | Upcoming releases |
| | `GET /api/v1/discover/genre/{id}?page=1` | By genre |
| | `GET /api/v1/discover/genres` | Genre list for dropdown |

Each discover endpoint cross-references results against the movie DB and the import exclusion list to set `in_library` and `excluded` flags on each result.

**Response shape:**
```json
{
  "results": [
    {
      "tmdb_id": 12345,
      "title": "Movie Title",
      "year": 2024,
      "overview": "...",
      "poster_path": "/abc.jpg",
      "rating": 7.4,
      "in_library": true,
      "excluded": false,
      "library_movie_id": "uuid-if-in-library"
    }
  ],
  "page": 1,
  "total_pages": 50
}
```

### Frontend

**New files:**
| File | Purpose |
|---|---|
| `web/ui/src/pages/discover/DiscoverPage.tsx` | Main discover page with tabs and grid |
| `web/ui/src/api/discover.ts` | React Query hooks for discover endpoints |

**Reused components:**
- `Poster` component (from posterless plan)
- `ConfigureStep` from Add Movie dialog (extracted into a reusable component)

---

## Implementation Order

1. TMDB client: add trending/popular/top-rated/upcoming/discover/genres methods
2. API endpoints with in_library cross-referencing
3. Frontend: DiscoverPage with Trending tab (simplest)
4. Frontend: remaining tabs (Popular, Top Rated, Upcoming)
5. Frontend: By Genre tab with genre dropdown
6. Frontend: "Add" flow using extracted ConfigureStep
7. Add to sidebar navigation
8. Add to command palette navigation commands

---

## Testing

### Backend — Unit Tests

**TMDB client — `internal/metadata/tmdb/client_test.go`:**
- `Trending(ctx, 1)` calls correct TMDB endpoint with page parameter
- `Popular(ctx, 1)` calls correct endpoint
- `TopRated(ctx, 1)` calls correct endpoint
- `Upcoming(ctx, 1)` calls correct endpoint
- `DiscoverByGenre(ctx, 28, 1)` calls `/discover/movie?with_genres=28&page=1`
- `GenreList(ctx)` returns list of genres with id and name
- All methods parse `total_pages` from response
- Empty results → returns empty slice with total_pages=0, no error
- TMDB API error (401, 429, 500) → returns descriptive error

**Discover API — `internal/api/v1/discover_test.go`:**
- `GET /api/v1/discover/trending` returns results with correct shape
- `GET /api/v1/discover/trending?page=2` passes page to TMDB client
- `GET /api/v1/discover/popular` returns results
- `GET /api/v1/discover/top-rated` returns results
- `GET /api/v1/discover/upcoming` returns results
- `GET /api/v1/discover/genre/28` returns action movies
- `GET /api/v1/discover/genre/invalid` → 400 error
- `GET /api/v1/discover/genres` returns genre list
- `in_library` flag: add a movie to DB → discover results containing that TMDB ID have `in_library: true`
- `in_library` flag: movie not in DB → `in_library: false`
- `excluded` flag: add TMDB ID to import exclusion list → discover results have `excluded: true`
- `library_movie_id` present when `in_library: true`, null otherwise
- TMDB not configured (no API key) → 503 with clear error message
- Pagination: page=0 or negative → defaults to 1

### Backend — Integration Tests

**Full flow — `internal/api/integration_test.go`:**
- Browse trending → find a movie → add it via `POST /api/v1/movies` → browse trending again → verify `in_library` flipped to true
- Add a movie → delete it → browse again → verify `in_library` back to false
- Add TMDB ID to exclusion list → browse → verify `excluded: true` on that result

### Frontend — Unit Tests

**DiscoverPage — `web/ui/src/pages/discover/DiscoverPage.test.tsx`:**
- Default tab (Trending) renders on page load
- Renders grid of movie cards from mock data
- Each card shows poster, title, year, and TMDB rating
- "In Library" badge shown for movies already in library
- "Excluded" badge shown for excluded movies
- "+ Add" button shown for movies not in library and not excluded
- Clicking "+ Add" opens add movie configuration modal
- Clicking "In Library" navigates to `/movies/{id}`
- Tab switching: clicking "Popular" fetches popular endpoint and renders results
- Tab switching: clicking "By Genre" shows genre dropdown
- Genre dropdown: selecting a genre fetches discover endpoint with genre ID
- "Load More" button appears when `page < total_pages`
- "Load More" fetches next page and appends results (not replaces)
- Empty state: no results for a genre → shows "No movies found" message
- Loading state: shows skeleton grid while fetching
- Error state: API failure → shows error message with retry option

**API hooks — `web/ui/src/api/discover.test.tsx`:**
- `useTrending()` calls `/discover/trending`
- `usePopular()` calls `/discover/popular`
- `useDiscoverByGenre(28)` calls `/discover/genre/28`
- `useGenreList()` calls `/discover/genres`

**MSW handlers:**
- Add mock handlers for all 6 discover endpoints with realistic test fixtures
- Add handler for genre list returning at least 5 genres
- Add error handler variant for TMDB unavailable scenario

---

## Risks

| Risk | Mitigation |
|---|---|
| TMDB rate limiting | These are cacheable endpoints — add HTTP-level caching (5-minute TTL) in the TMDB client. Same data for all users. |
| Discover replaces import lists | Different use cases. Discover is interactive browsing. Import lists are scheduled automation. Both have value. |
| Page feels empty without good posters | TMDB trending/popular almost always have posters. Posterless fallback handles the rest. |
| "Add" flow friction (too many clicks) | Consider a "quick add" mode: click "+ Add" → immediately adds with default library + profile, toast confirmation. Power users configure per-movie; casual users get one-click. |
