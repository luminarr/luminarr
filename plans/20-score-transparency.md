# Plan 20 — Score Transparency

## Problem

When a release is grabbed or skipped, Luminarr computes a quality score internally
but throws it away. Users have no way to know why a particular release won/lost or
why an expected grab never happened.

Radarr's scoring is also opaque. Making ours visible is a direct differentiator and
a support burden reducer — most "why didn't it grab?" questions answer themselves once
you can see the breakdown.

---

## Scope

Two surfaces:

1. **Manual Search modal** — show a score column and per-release breakdown inline,
   so users can see exactly why one result ranks above another before they grab it.

2. **Grab History** — store the score breakdown when a grab is dispatched, so users
   can audit past decisions from the History tab.

Non-scope: logging _skipped_ releases (too noisy — every RSS sync evaluates hundreds
of releases and most are skipped). We score what we grab, not what we reject.

---

## Score Breakdown Type

```go
// pkg/plugin/score.go (new file)

// ScoreBreakdown records how a release was evaluated against a quality profile.
// Each dimension is independently scored; Total is the sum.
type ScoreBreakdown struct {
    Total      int              `json:"total"`       // 0–100
    Dimensions []ScoreDimension `json:"dimensions"`
}

type ScoreDimension struct {
    Name    string `json:"name"`    // "resolution", "source", "codec", "hdr"
    Score   int    `json:"score"`   // points awarded for this dimension
    Max     int    `json:"max"`     // maximum possible for this dimension
    Matched bool   `json:"matched"` // did it meet the profile requirement?
    Got     string `json:"got"`     // what we found (e.g. "x264")
    Want    string `json:"want"`    // what the profile requires (e.g. "x265")
}
```

Weights (out of 100):

| Dimension  | Max points |
|------------|-----------|
| Resolution | 40        |
| Source     | 30        |
| Codec      | 20        |
| HDR        | 10        |

Partial credit: if resolution exceeds cutoff, still award full resolution points.
If codec is "any" in the profile, codec always matches. Same for HDR.

---

## Backend Changes

### 1. Quality scoring returns a breakdown

Currently `quality.Profile.Score(release)` returns `int`. Change to:

```go
func (p *Profile) ScoreWithBreakdown(q plugin.Quality) (int, plugin.ScoreBreakdown)
```

Keep the existing `Score()` method as a thin wrapper for callers that don't need the breakdown.

### 2. Store breakdown in grab_history

#### Migration (`internal/db/migrations/00018_grab_score.sql`)

```sql
-- +goose Up
ALTER TABLE grab_history ADD COLUMN score_breakdown TEXT NOT NULL DEFAULT '';

-- +goose Down
SELECT 1;
```

#### sqlc query update

In `internal/db/queries/sqlite/grabhistory.sql`, update `CreateGrabHistory` (or
`InsertGrabHistory`) to accept the new column.

Run `sqlc generate` after.

### 3. Populate on grab

In `internal/core/indexer/service.go`, where a grab is dispatched:

```go
score, breakdown := profile.ScoreWithBreakdown(release.Quality)
// serialize breakdown to JSON
// pass to CreateGrabHistory params
```

### 4. Manual search API includes breakdown

`GET /api/v1/movies/{id}/releases` currently returns a list of releases.
Add `score_breakdown` to each result. The handler already calls the scorer —
just include the breakdown in the response:

```go
type releaseBody struct {
    // ... existing fields ...
    Score          int                   `json:"score"`
    ScoreBreakdown plugin.ScoreBreakdown `json:"score_breakdown"`
}
```

No new endpoint needed.

---

## Frontend Changes

### Manual Search modal

Add a **Score** column (e.g. `85/100`). On hover or click, expand a tooltip/popover:

```
Resolution  ✓  40/40  (1080p — matches)
Source      ✓  30/30  (Bluray — matches)
Codec       ✗   0/20  (got x264, need x265)
HDR         ✓  10/10  (SDR — profile allows any)
─────────────────────
Total       80/100
```

Default sort in the modal changes to score descending (already the intent; now explicit).

### History tab

Add a **Score** column showing `total/100`. Expand row (or tooltip) shows
the same dimension breakdown. Stored as JSON in `grab_history.score_breakdown` —
parse and render on the frontend.

---

## Tests

**Unit** (`internal/core/quality/score_breakdown_test.go`):
- `TestScoreBreakdown_perfectMatch` — all 4 dimensions match → 100
- `TestScoreBreakdown_codecMismatch` — codec wrong → 80, codec dimension shows got/want
- `TestScoreBreakdown_anyCodec` — profile codec = "any" → codec always matches
- `TestScoreBreakdown_resolutionExceedsCutoff` — 2160p file, 1080p cutoff → full resolution points

**Integration** (`internal/api/v1/releases_test.go`):
- `TestReleasesIncludeScoreBreakdown` — GET releases returns score_breakdown with correct shape

---

## Open Questions

1. Should we retroactively score existing grab_history rows? Probably not — no release
   data to score against. Leave `score_breakdown` empty for pre-migration rows.

2. Should skipped releases ever be logged? Deferred — revisit if users ask for it.
   The noise-to-signal ratio is too high to include in v1.
