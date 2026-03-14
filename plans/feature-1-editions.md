# Feature 1: Edition-Aware Library Management

> **Status:** Decisions finalized, ready to build.

## The Problem

Radarr treats every movie as a single file slot. Movie collectors care deeply about *which version* they have — Theatrical, Director's Cut, Extended, Unrated, IMAX, Criterion, etc. Radarr cannot:

- Track which edition a file actually is
- Let users specify which edition they *want*
- Search specifically for a named edition
- Show "You have Theatrical, Director's Cut is available"

Luminarr already has an `edition` column on `movie_files` but it's never parsed from filenames, never shown prominently, never used in search/grab decisions, and there's no concept of a "preferred edition" per movie.

## What This Feature Delivers

1. **Edition parsing** — detect edition from release titles and filenames
2. **Preferred edition per movie** — user says "I want the Director's Cut of Blade Runner"
3. **Edition-aware search** — search results annotated with detected edition
4. **Edition-aware grab logic** — auto-search gives a score bonus to the preferred edition
5. **Edition visibility** — UI shows which edition you have, which you want, which are available

---

## Decisions (Locked)

1. **Soft preference only.** Edition is a scoring bonus, not a hard filter. If the preferred edition isn't available, Luminarr still grabs the best quality release. No per-movie "edition required" toggle — keep it simple for v1.

2. **Untagged releases are neutral.** Releases with no detected edition get no bonus and no penalty. Only releases with a *matching* edition tag get the bonus. This must be clearly documented and visible in the UI so users aren't confused about why an untagged release was grabbed.

3. **Single file per movie (no multi-edition for v1).** New grabs replace the existing file, same as today. Multi-edition (keeping Director's Cut AND Theatrical) is deferred to a future iteration.

4. **Edition match gets a score bonus (+30 pts).** Enough to be a tiebreaker within the same resolution/source tier, but NOT enough to override a resolution jump (1080p preferred edition should not beat a 4K non-preferred).

5. **Freeform edition names.** Dropdown suggestions from the canonical list, but users can type any edition name they want.

6. **No ambiguous abbreviations.** "DC" (Director's Cut vs DC Comics), "SE" (Special Edition vs Swedish), "CC" (Criterion vs closed captions), "B&C" — all dropped. Only full-form patterns matched. Remastered and IMAX are kept as editions.

---

## Scope & Non-Goals

### In scope
- Edition detection from release titles (parser)
- Edition detection from existing filenames (library scan)
- Preferred edition field on `movies` table
- Edition displayed on search results, file list, movie detail
- Auto-search gives score bonus to preferred edition
- Edition shown in rename templates
- Edition stats in library health
- Clear UI and documentation around how edition preference works (soft, untagged = neutral)

### Out of scope (future)
- Multi-edition support (keep multiple files for different editions of one movie)
- Hard edition filter mode ("only grab this edition")
- Edition-specific quality profiles (e.g. "I want Director's Cut in 4K but Theatrical in 1080p")
- Automatic edition discovery ("does a Director's Cut of this movie exist?") — no reliable data source
- Cross-referencing TMDB alternative versions (TMDB has no structured edition data)

---

## Phase 1: Edition Parser

### 1.1 New parser: `internal/core/edition/parser.go`

A pure function that extracts edition info from a release title or filename.

```go
package edition

// Edition represents a detected movie edition.
type Edition struct {
    Name  string // Canonical name: "Director's Cut", "Extended", "Theatrical", etc.
    Raw   string // The raw matched token from the title: "Directors.Cut", etc.
}

// Parse extracts edition information from a release title.
// Returns nil if no edition is detected.
// The absence of an edition tag means "unknown/default", NOT "Theatrical".
func Parse(title string) *Edition
```

**Canonical edition names** (normalized from many variants):
| Canonical Name | Matched Patterns |
|---|---|
| `Theatrical` | `Theatrical`, `Theatrical.Cut`, `Theatrical.Edition`, `Theatrical.Release` |
| `Director's Cut` | `Directors.Cut`, `Director's.Cut`, `Directors.Edition` |
| `Extended` | `Extended`, `Extended.Cut`, `Extended.Edition`, `Extended.Version` |
| `Unrated` | `Unrated`, `Unrated.Cut`, `Unrated.Edition`, `Uncensored` |
| `Ultimate` | `Ultimate.Cut`, `Ultimate.Edition`, `Ultimate.Collector` |
| `Special Edition` | `Special.Edition` |
| `Criterion` | `Criterion`, `Criterion.Collection` |
| `IMAX` | `IMAX`, `IMAX.Edition` |
| `Remastered` | `Remastered`, `4K.Remastered`, `Digitally.Remastered` |
| `Anniversary` | `Anniversary`, `Anniversary.Edition`, `Xth.Anniversary` |
| `Final Cut` | `Final.Cut` |
| `Redux` | `Redux` |
| `Rogue Cut` | `Rogue.Cut` |
| `Black and Chrome` | `Black.and.Chrome` |
| `Open Matte` | `Open.Matte` |

**Parser behavior:**
- Runs *after* quality parsing (editions appear in the same region of a release title as quality tokens)
- Case-insensitive, dot/space/underscore normalized
- Returns the *first* match (release titles rarely have multiple edition tags)
- If no edition token is found, returns `nil` — "unknown/default", not "Theatrical"

**Position in title:** Edition tokens typically appear after the year and before or intermixed with quality tokens:
```
Movie.Title.2021.Directors.Cut.2160p.BluRay.REMUX.HEVC.DoVi-GRP
                 ^^^^^^^^^^^^^
Movie.Title.2021.2160p.Extended.BluRay.x265-GRP
                       ^^^^^^^^
```

### 1.2 Integration with quality parser

**Separate parser, called independently.**
- `quality.Parse(title)` continues to return `Quality`
- `edition.Parse(title)` is a new, independent call
- Callers that need both call both
- Clean separation, no coupling between orthogonal concerns

### 1.3 Tests

Comprehensive test suite with real-world release titles:
```
"Blade.Runner.1982.The.Final.Cut.2160p.UHD.BluRay.x265-GROUP"                  → Final Cut
"Apocalypse.Now.1979.Redux.1080p.BluRay.x264-GROUP"                             → Redux
"Justice.League.2021.IMAX.2160p.WEB-DL.DDP5.1.HDR.HEVC-GROUP"                  → IMAX
"The.Lord.of.the.Rings.2001.Extended.2160p.UHD.BluRay.REMUX.HDR.HEVC-GROUP"    → Extended
"Movie.2020.1080p.BluRay.x264-GROUP"                                            → nil (no edition)
"Aliens.1986.Special.Edition.1080p.BluRay.x265-GROUP"                           → Special Edition
"Jaws.1975.Remastered.1080p.BluRay.x265-GROUP"                                 → Remastered
"Batman.Begins.2005.IMAX.Edition.2160p.WEB-DL.DDP5.1.HEVC-GROUP"               → IMAX
"Blade.Runner.1982.Directors.Cut.1080p.BluRay.x264-GROUP"                       → Director's Cut
```

**Negative tests (must NOT match):**
```
"DC.League.of.Super-Pets.2022.1080p.BluRay.x264-GROUP"                         → nil (DC is part of title)
"Movie.2020.DC.1080p.BluRay.x264-GROUP"                                        → nil (DC abbreviation not matched)
"Movie.2020.SE.1080p.BluRay.x264-GROUP"                                        → nil (SE abbreviation not matched)
```

---

## Phase 2: Database & Model Changes

### 2.1 New migration: `00029_editions.sql`

```sql
-- +goose Up

-- Add preferred edition to movies (NULL = no preference = accept any)
ALTER TABLE movies ADD COLUMN preferred_edition TEXT;

-- Add index for edition queries on existing column
CREATE INDEX movie_files_edition ON movie_files(edition);

-- Store detected edition on grab history
ALTER TABLE grab_history ADD COLUMN release_edition TEXT;

-- +goose Down
DROP INDEX IF EXISTS movie_files_edition;
-- SQLite doesn't support DROP COLUMN; forward-only in practice
```

### 2.2 Movie model changes

```go
// In internal/core/movie/service.go — Movie struct
type Movie struct {
    // ... existing fields ...
    PreferredEdition string // empty = no preference
}
```

### 2.3 AddRequest / UpdateRequest changes

```go
type AddRequest struct {
    // ... existing fields ...
    PreferredEdition string // optional
}

type UpdateRequest struct {
    // ... existing fields ...
    PreferredEdition *string // pointer = explicit set; nil = don't change
}
```

### 2.4 sqlc query changes

New/modified queries in `movies.sql`:
```sql
-- name: UpdateMoviePreferredEdition :exec
UPDATE movies SET preferred_edition = ?, updated_at = ? WHERE id = ?;

-- name: ListMoviesWithEditionMismatch :many
-- Movies that have a preferred edition set but their file's edition doesn't match.
-- Used by the Wanted > Edition Mismatch tab.
SELECT m.id, m.title, m.year, m.preferred_edition, mf.edition as file_edition
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.preferred_edition IS NOT NULL
  AND m.preferred_edition != ''
  AND (mf.edition IS NULL OR mf.edition != m.preferred_edition);

-- name: CountEditionMismatches :one
SELECT COUNT(*) FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.preferred_edition IS NOT NULL
  AND m.preferred_edition != ''
  AND (mf.edition IS NULL OR mf.edition != m.preferred_edition);
```

---

## Phase 3: Edition in the Grab Pipeline

### 3.1 Release struct changes

```go
// In pkg/plugin/types.go — Release struct
type Release struct {
    // ... existing fields ...
    Edition string // Detected edition (empty = no edition detected)
}
```

### 3.2 Search result annotation

When `indexer.Service.Search()` processes results, also parse editions:

```go
// In indexer search loop, after quality parsing:
if ed := edition.Parse(release.Title); ed != nil {
    release.Edition = ed.Name
}
```

### 3.3 Auto-search edition scoring

In `autosearch.Service.SearchMovie()`, when ranking results:

**Current behavior:** Pick the release with the highest quality score (0-100).

**New behavior:**
1. If movie has `PreferredEdition` set:
   - Releases whose detected edition **matches** the preferred edition → +30 bonus
   - Releases with a **different** detected edition → no bonus
   - Releases with **no** detected edition → no bonus, no penalty (neutral)
2. If movie has no preferred edition → no change to current behavior

**Why +30:**
Quality score range: 0-100 (40 resolution + 30 source + 20 codec + 10 HDR).
- Same quality, edition match wins: 87 + 30 = 117 vs 87 → match wins
- 1080p edition match vs 4K no edition: ~87 + 30 = 117 vs ~130 → 4K still wins
- 1080p BluRay edition match vs 1080p WebDL no edition: ~87 + 30 = 117 vs ~77 → edition match wins

### 3.4 Score breakdown changes

Add an `Edition` dimension to `ScoreBreakdown`:
```go
// Edition dimension added to the existing dimensions array:
// Name: "edition", Max: 30, Score: 30 (match) or 0 (no match/no tag),
// Matched: true/false, Got: detected edition, Want: preferred edition
```

When no preferred edition is set on the movie: the edition dimension is omitted from the breakdown entirely (not shown as "0 of 30" — just absent).

### 3.5 Grab history

The migration (Phase 2) adds `release_edition TEXT` to `grab_history`. Populate it when recording a grab.

---

## Phase 4: Edition in Import & File Management

### 4.1 File import

When a file is imported (importer processes a completed download), parse the edition from the filename and store it in `movie_files.edition`:

```go
ed := edition.Parse(filename)
if ed != nil {
    file.Edition = ed.Name
}
```

The `edition` column already exists on `movie_files` — we just need to populate it.

### 4.2 Library scan backfill

When scanning an existing library (`library.Service.Scan()`), parse editions from filenames on disk and update `movie_files.edition` for existing records that have `edition = NULL`.

This is a one-time backfill for existing libraries, but also runs on every scan for newly added files.

### 4.3 Upgrade logic changes

**Current:** `WantRelease()` checks if the new release is better quality than the existing file.

**New:** `WantRelease()` also considers edition:
- If preferred edition is set and current file's edition does NOT match preferred → want releases that are at least the same quality (even if not a strict quality upgrade), because we're seeking the right edition
- If preferred edition is set and current file's edition matches preferred → normal quality upgrade logic (only grab if strict quality improvement)
- If no preferred edition → normal quality upgrade logic (no change)

This means: if you have a 1080p Theatrical and want the Director's Cut, Luminarr will grab a 1080p Director's Cut even though it's the same quality — because the edition is the upgrade.

### 4.4 Rename template

Add `{Edition}` token to the renaming system:
```
"{Movie Title} ({Release Year}) - {Edition} {Quality Full}"
→ "Blade Runner (1982) - Final Cut Bluray-2160p x265 Dolby Vision"
```

If edition is empty/unknown, the `{Edition}` token and its surrounding separator are omitted:
```
"Movie (2020) {Quality Full}"
→ "Movie (2020) Bluray-1080p x265"
```

---

## Phase 5: API Changes

### 5.1 Movie endpoints

`GET /api/v1/movies/{id}` response adds:
```json
{
  "preferred_edition": "Director's Cut"
}
```

`PUT /api/v1/movies/{id}` accepts:
```json
{
  "preferred_edition": "Director's Cut"
}
```

Set to empty string `""` to clear the preference.

`POST /api/v1/movies` accepts:
```json
{
  "preferred_edition": "Director's Cut"
}
```

### 5.2 Movie files endpoint

`GET /api/v1/movies/{id}/files` already returns `edition`. No change needed — just ensure it's populated.

### 5.3 Search results endpoint

`GET /api/v1/movies/{id}/releases` response adds edition to each release:
```json
{
  "releases": [
    {
      "title": "Movie.2020.Directors.Cut.2160p.BluRay.REMUX...",
      "edition": "Director's Cut",
      "edition_match": true,
      ...
    }
  ]
}
```

`edition_match` semantics:
- `true` if release edition matches movie's preferred edition
- `true` if movie has no preferred edition (everything matches)
- `false` if release has a detected edition that differs from preferred
- `null`/absent if release has no detected edition and movie has a preference (neutral)

### 5.4 New endpoint: edition list

```
GET /api/v1/editions
```

Returns the list of canonical edition names for use in dropdown suggestions:
```json
[
  "Theatrical", "Director's Cut", "Extended", "Unrated", "Ultimate",
  "Special Edition", "Criterion", "IMAX", "Remastered", "Anniversary",
  "Final Cut", "Redux", "Rogue Cut", "Black and Chrome", "Open Matte"
]
```

### 5.5 Stats endpoint addition

`GET /api/v1/stats/collection` adds:
```json
{
  "edition_mismatch": 5
}
```

---

## Phase 6: Frontend Changes

### 6.1 Movie detail page

- **Header area:** Show current edition badge next to quality badge
  - If file has edition: show pill like `Director's Cut`
  - If preferred edition set but file doesn't match: show warning pill `Wanted: Director's Cut | Have: Theatrical`
  - If preferred edition set but file has no detected edition: show info pill `Wanted: Director's Cut | Have: Unknown`
- **Edit tab:** Add "Preferred Edition" field — combobox with canonical suggestions but accepts freeform text. Include help text explaining soft preference behavior.
- **Files tab:** Show edition column for each file
- **Releases tab:** Show edition badge on each release; highlight matches with accent color. Untagged releases show no badge (not "Unknown").

### 6.2 Movie list / Dashboard

- Edition mismatch count in stats cards (links to Wanted > Edition Mismatch)

### 6.3 Wanted page

Add "Edition Mismatch" tab alongside "Missing" and "Cutoff Unmet":
- Lists movies where `preferred_edition` is set but file edition doesn't match
- Each row: movie title, preferred edition, current file edition (or "Not detected"), search button
- Bulk search for edition upgrades

### 6.4 Add movie modal

- Optional "Preferred Edition" combobox field (same as edit tab, with canonical suggestions)
- Collapsed/hidden by default to keep the happy path simple

### 6.5 Documentation in UI

Critical: users need to understand how soft edition preference works. Help tooltip on the edition combobox:

> **How edition preference works:**
> When Luminarr searches for releases, it gives a score bonus to releases tagged with your preferred edition. If no matching edition is found, the best quality release is grabbed instead — your movie won't be stuck in "Missing" waiting for a specific edition.
>
> Most releases don't include edition tags in their names. An untagged release is treated neutrally — it may or may not be your preferred edition.

### 6.6 Types

```typescript
interface Movie {
  // ... existing ...
  preferred_edition?: string;
}

interface Release {
  // ... existing ...
  edition?: string;
  edition_match?: boolean;
}

interface MovieFile {
  // ... existing ...
  edition?: string;
}
```

---

## Implementation Order

1. **Edition parser** + tests (pure function, no dependencies)
2. **Database migration** (preferred_edition on movies, edition index on movie_files, release_edition on grab_history)
3. **sqlc queries** + regenerate
4. **Model/service changes** (Movie struct, AddRequest, UpdateRequest)
5. **Search pipeline** (parse edition in search results)
6. **Grab pipeline** (edition scoring bonus in auto-search, +30 pts)
7. **Upgrade logic** (WantRelease considers edition mismatch as upgrade-worthy)
8. **Import pipeline** (populate edition on file import)
9. **Library scan** (backfill editions for existing files)
10. **API endpoints** (movie CRUD, search results, editions list, stats)
11. **Rename template** ({Edition} token)
12. **Frontend** (movie detail, wanted page, add modal, search results, help text)
13. **Stats** (edition mismatch count)

Estimated: ~8-10 implementation sessions.
