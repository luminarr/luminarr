# Plan: Release Decision Explainability ("Why was this grabbed/skipped?")

**Status**: Draft
**Scope**: New explainability layer over the autosearch pipeline
**Depends on**: Autosearch pipeline (existing), custom format scoring (release-group-audio-presets Step 4)

---

## Summary

Add a "Why?" feature that explains why each release candidate was grabbed, skipped, or rejected during an automatic or manual search. Decisions are computed on demand (not stored in DB), keeping the system stateless and simple.

---

## Current State

- `autosearch/service.go` iterates release candidates, applies quality profile checks, and grabs the best match
- Rejection reasons are implicit in control flow (early returns, boolean checks) -- not surfaced to the user
- The API returns only the final grab result, not per-candidate reasoning
- Frontend shows release lists but gives no insight into why a release was or wasn't chosen

---

## Step 1: Decision Types

### What

New file `internal/core/autosearch/decision.go` defining the `ReleaseDecision` type and typed skip reasons.

### New Types

`ReleaseDecision` struct containing:
- `Release` -- the candidate release
- `Outcome` -- `grabbed`, `skipped`, or `rejected`
- `Reason` -- typed skip reason (see below)
- `Detail` -- human-readable context string
- `ScoreBreakdown` -- the full scoring breakdown (quality + CF + edition)

`SkipReason` constants:
- `blocklisted` -- release is on the user's blocklist
- `cf_score_below_minimum` -- custom format score below profile's MinCustomFormatScore
- `quality_not_in_profile` -- resolution/source/codec not allowed by quality profile
- `no_upgrade_needed` -- existing file already meets or exceeds this quality
- `upgrade_disabled` -- quality profile has upgrades disabled
- `upgrade_ceiling_reached` -- score already at or above UpgradeUntil threshold
- `download_client_rejected` -- download client returned an error
- `grabbed` -- this was the selected release

### Files

| File | Change |
|------|--------|
| `internal/core/autosearch/decision.go` | New file: `ReleaseDecision`, `SkipReason` constants |

---

## Step 2: Typed Rejection from Quality Profile

### What

Add a `RejectReason(candidate Quality) SkipReason` method to `quality.Profile` that returns a typed reason instead of a bare `bool`. The existing `Allowed()` method can delegate to this internally.

### Files

| File | Change |
|------|--------|
| `internal/core/quality/profile.go` | Add `RejectReason()` method returning `SkipReason` or empty string |
| `internal/core/quality/profile_test.go` | Test each rejection path returns the correct reason |

---

## Step 3: Collect Decisions in SearchMovie

### What

Refactor `SearchMovie` to build a `[]ReleaseDecision` as it iterates candidates. Each rejection point populates a decision with the appropriate reason and detail string. The final grab also gets a decision entry.

This is the core refactor -- every early return or `continue` in the candidate loop becomes a decision append.

### Files

| File | Change |
|------|--------|
| `internal/core/autosearch/service.go` | Refactor candidate loop to collect `[]ReleaseDecision`; add `SearchMovieExplain()` variant that returns decisions without grabbing |

---

## Step 4: Explanation Text Generation

### What

New file `internal/core/autosearch/explain.go` with a `func Explain(d ReleaseDecision) string` that generates human-readable explanations from decision context using text templates.

Examples:
- "Skipped: quality 720p WEB-DL is not in your quality profile"
- "Skipped: custom format score -150 is below minimum 0"
- "Skipped: existing file (1080p BluRay, score 85) is already at upgrade ceiling"
- "Grabbed: best candidate with score 92 (quality 72 + CF 20 + edition 0)"

### Files

| File | Change |
|------|--------|
| `internal/core/autosearch/explain.go` | New file: `Explain()` function with text templates per `SkipReason` |
| `internal/core/autosearch/explain_test.go` | Test each reason produces expected explanation text |

---

## Step 5: API Endpoint

### What

New endpoint `GET /api/v1/movies/{id}/releases/explain` -- performs a dry-run search that returns all candidate decisions without actually grabbing anything.

Response shape:
```json
[{
  "release": { "title": "...", "size": 1234 },
  "outcome": "skipped",
  "reason": "cf_score_below_minimum",
  "detail": "CF score -150 < minimum 0",
  "explanation": "Skipped: custom format score..."
}]
```

### Files

| File | Change |
|------|--------|
| `internal/api/v1/movies.go` | Add `handleExplainReleases` handler |
| `internal/api/router.go` | Register `GET /api/v1/movies/{id}/releases/explain` |

---

## Step 6: Frontend

### What

- "Why?" button on each release row in ManualSearchModal and in grab history
- Clicking opens a `DecisionPanel` modal showing the full decision list
- Each decision row shows: release title, outcome badge (grabbed/skipped), reason, and expandable detail
- `ScoreChip` extended with CF info in tooltip (matched format names and scores)
- Client-side explanation helper in `web/ui/src/lib/explain.ts` for formatting

### Files

| File | Change |
|------|--------|
| `web/ui/src/api/movies.ts` | Add `explainReleases(movieId)` API call |
| `web/ui/src/lib/explain.ts` | New file: client-side explanation formatting helpers |
| `web/ui/src/components/DecisionPanel.tsx` | New file: modal displaying decision list |
| `web/ui/src/components/ScoreChip.tsx` | Add CF breakdown tooltip |
| ManualSearchModal component | Add "Why?" button wired to explain endpoint |

---

## Implementation Order

```
Step 1: Decision types                    [independent, pure types]
Step 2: Typed rejection from profile      [independent, extends existing]
Step 3: Collect decisions in SearchMovie  [depends on 1+2, core refactor]
Step 4: Explanation text generation       [depends on 1]
Step 5: API endpoint                      [depends on 3+4]
Step 6: Frontend                          [depends on 5]
```

**PR Strategy**:
- PR 1: Steps 1-4 (backend types, collection, and text generation)
- PR 2: Steps 5-6 (API + frontend)

---

## Key Design Decisions

- **Transient, not stored**: Decisions are computed on demand via dry-run search. No new DB tables. This avoids schema churn and keeps explanations always consistent with current profile settings.
- **Dry-run reuses real pipeline**: `SearchMovieExplain()` calls the same candidate evaluation logic as `SearchMovie` but skips the grab step. This guarantees explanations match actual behavior.
- **Typed reasons, not free text**: Using `SkipReason` constants makes it possible for the frontend to render localized or styled explanations.

---

## Risks Summary

| Risk | Impact | Mitigation |
|------|--------|------------|
| Dry-run hits indexers | Extra API calls to indexer on each explain request | Cache recent search results for short TTL; reuse existing RSS cache if available |
| Refactoring SearchMovie breaks grabs | Regression in core grab logic | Existing autosearch tests must pass; add explicit test for decision collection |
| Explanation text drift from actual logic | "Why?" lies to the user | Both share the same code path; explain is a strict superset of grab |
| Performance on large candidate sets | Slow explain response | Indexers already limit results; decisions are cheap structs |
