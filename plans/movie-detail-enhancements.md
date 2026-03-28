# Plan: Movie Detail Page Enhancements

**Status**: Draft
**Scope**: Richer movie detail page with cast, crew, similar movies, and inline file status
**Depends on**: TMDB client, existing MovieDetail.tsx

---

## Summary

The movie detail page currently shows metadata + a file path on the Overview tab, with cast, crew, and similar movies absent. The TMDB API already returns this data (the `MovieDetail` struct has it, the `Person`/`FilmographyItem` types exist). This plan enriches the Overview tab so it feels like a movie page rather than a database record, and surfaces file status inline so users don't need to click into the Files tab to know what they have.

---

## Current State

The Overview tab shows:
- Poster (180px), title, year
- Quick facts row: runtime, status, TMDB ID, IMDB ID
- Minimum Availability selector, Preferred Edition selector
- Genres (tag list)
- Overview text (synopsis)
- File path (if exists)

Missing:
- Cast (actors with character names)
- Crew (director, writer, composer)
- Similar/recommended movies from TMDB
- Inline file quality summary (what quality do I have without clicking Files tab?)
- Backdrop image (fanart_url exists but isn't used on the detail page)

---

## Design

### Backend Changes

**Extend TMDB MovieDetail fetch to include credits and recommendations:**

| File | Change |
|---|---|
| `internal/metadata/tmdb/types.go` | Add `Credits` (cast + crew) and `Recommendations` fields to `MovieDetail` |
| `internal/metadata/tmdb/client.go` | Append `?append_to_response=credits,recommendations` to the `/movie/{id}` API call — single request, no extra API calls |
| `internal/core/movie/service.go` | Expose credits and recommendations in the movie API response |
| `internal/api/v1/movies.go` | Add `credits` and `recommendations` to movie detail response |

**New types:**

```go
type CastMember struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Character   string `json:"character"`
    ProfilePath string `json:"profile_path"` // TMDB image path
    Order       int    `json:"order"`        // billing order
}

type CrewMember struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Job         string `json:"job"`          // "Director", "Screenplay", etc.
    Department  string `json:"department"`
    ProfilePath string `json:"profile_path"`
}

type MovieRecommendation struct {
    TMDBID     int     `json:"tmdb_id"`
    Title      string  `json:"title"`
    Year       int     `json:"year"`
    PosterPath string  `json:"poster_path"`
    InLibrary  bool    `json:"in_library"`   // cross-reference against Luminarr DB
}
```

**Storage decision:** Do NOT persist cast/crew/recommendations to the database. Fetch from TMDB on each detail page load (cached by the TMDB client's HTTP cache). This avoids schema bloat and stale data. The `append_to_response` parameter is free — it doesn't count as additional TMDB API calls.

### Frontend Changes

**Modified file:** `web/ui/src/pages/movies/MovieDetail.tsx`

#### 1. Backdrop header

Use `fanart_url` (already in the Movie type) as a blurred background behind the poster and title area. Gradient fade to the page background. Purely cosmetic but significantly elevates the visual feel.

#### 2. Inline file status card

Below the synopsis, before the file path, add a compact card:

```
┌─────────────────────────────────────────┐
│  ✓ On Disk — 1080p Bluray x265          │
│    Director's Cut · 2.2 GB · Imported 3d ago  │
│    ⚠ Mismatch: actual codec is x264     │  (only if mediainfo flagged it)
└─────────────────────────────────────────┘
```

Or if no file:
```
┌─────────────────────────────────────────┐
│  ○ Missing — Monitored, searching       │
└─────────────────────────────────────────┘
```

This uses data already available from `useMovieFiles()` — no new API call needed. Just render it on the Overview tab instead of hiding it behind the Files tab.

#### 3. Cast & crew section

Below the file status card:

**Director / Writer row:**
```
Directed by Ridley Scott  ·  Written by Dan O'Bannon
```

Clickable names — clicking a person name could navigate to a filtered view or open a popover showing their other movies in the library.

**Cast strip:**
Horizontal scrollable row of cast member cards (headshot circle + name + character). Show top 10 by billing order. Each card is ~80px wide.

```
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│  ○   │ │  ○   │ │  ○   │ │  ○   │  ← horizontal scroll
│Sigour│ │ Tom  │ │ John │ │Veroni│
│Ripley│ │Dallas│ │ Kane │ │Lamber│
└──────┘ └──────┘ └──────┘ └──────┘
```

Headshots from TMDB (`https://image.tmdb.org/t/p/w185{profile_path}`). Fallback to initials circle if no photo.

#### 4. Similar movies section

Below cast, a "You might also like" row:

Horizontal scrollable row of poster thumbnails (similar to dashboard poster grid but smaller, ~120px wide). Each shows:
- Poster
- Title + year
- "In Library" badge if already tracked
- Click: if in library, navigate to that movie; if not, open Add Movie dialog pre-filled

Uses TMDB recommendations data. Show up to 10.

---

## Implementation Order

1. Backend: extend TMDB client with `append_to_response=credits,recommendations`
2. Backend: add types, include in movie detail API response
3. Backend: cross-reference recommendations against DB for `in_library` flag
4. Frontend: backdrop header with fanart
5. Frontend: inline file status card on Overview tab
6. Frontend: director/writer row
7. Frontend: cast strip (horizontal scroll)
8. Frontend: similar movies row
9. Update TypeScript types

---

## Testing

### Backend — Unit Tests

**TMDB client — `internal/metadata/tmdb/client_test.go`:**
- `append_to_response=credits,recommendations` is included in the request URL
- Credits response parsed correctly: cast members have id, name, character, profile_path, order
- Crew response parsed correctly: filters to key roles (Director, Screenplay, Original Music Composer)
- Recommendations response parsed correctly: tmdb_id, title, year, poster_path
- Empty credits (no cast/crew) → returns empty slices, no error
- Empty recommendations → returns empty slice, no error
- Malformed credits JSON → graceful error, does not break movie detail fetch

**Movie API — `internal/api/v1/movies_test.go`:**
- `GET /api/v1/movies/{id}` includes `credits` and `recommendations` in response
- Credits: top 10 cast by billing order, key crew only (Director, Writer, Composer)
- Recommendations: each entry has `in_library` flag correctly set (true when TMDB ID exists in movies table, false otherwise)
- Movie with `tmdb_id = 0` (unmatched) → credits and recommendations are null/omitted, not an error
- Movie with valid TMDB ID but TMDB API unavailable → movie detail still returns, credits/recommendations are null

### Backend — Integration Tests

**Full flow — `internal/api/integration_test.go`:**
- Add a movie via API → fetch movie detail → verify credits and recommendations are present
- Add two movies that TMDB recommends to each other → verify `in_library: true` in recommendations for the one that's tracked
- Delete a movie → fetch the other movie's detail → verify the deleted movie shows `in_library: false` in recommendations

### Frontend — Unit Tests

**MovieDetail component — `web/ui/src/pages/movies/MovieDetail.test.tsx`:**
- Overview tab renders backdrop image when `fanart_url` is present
- Overview tab renders no backdrop when `fanart_url` is absent (no broken image)
- File status card renders quality badge and size when movie has files
- File status card renders "Missing — Monitored, searching" when movie has no files
- File status card renders mismatch warning when mediainfo flags a discrepancy
- Cast strip renders up to 10 cast members with names and character names
- Cast strip renders initials fallback when `profile_path` is null
- Cast strip renders empty state gracefully when credits are empty
- Director/writer row renders correctly with single director, multiple writers
- Director/writer row omitted when no crew data available
- Similar movies row renders poster thumbnails with "In Library" badge where applicable
- Similar movies row: clicking "In Library" movie navigates to `/movies/{id}`
- Similar movies row: clicking non-library movie opens Add Movie dialog
- Similar movies row hidden when recommendations array is empty

**MSW handlers:**
- Add movie detail mock handler that includes credits and recommendations fixtures

---

## Risks

| Risk | Mitigation |
|---|---|
| TMDB rate limiting from credits fetch | `append_to_response` doesn't count as extra calls — it's the same single request |
| Large response size (credits can be 100+ people) | Only send top 10 cast + key crew (director, writer, composer) to frontend |
| Recommendations stale | Fetched live from TMDB on each page load, always fresh |
| Movies without TMDB ID (unmatched) | Skip credits/recommendations entirely — show existing sparse view |
