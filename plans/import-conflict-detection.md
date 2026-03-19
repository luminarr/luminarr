# Plan: Import Conflict Detection

**Status**: Draft
**Scope**: New pure conflict comparison package, API integration, and frontend warnings
**Depends on**: Audio parsing (release-group-audio-presets Step 2), edition parsing (existing)

---

## Summary

Detect and surface quality trade-offs when a candidate release would replace the existing file. For example: "Audio downgrade: TrueHD Atmos -> AC3 5.1" or "HDR lost: Dolby Vision -> none". Conflicts are warnings, not blockers -- V1 is warn-only and does not prevent grabs.

---

## Current State

- Autosearch compares candidates by overall quality score but does not flag individual dimension regressions
- A release with higher resolution but worse audio will be grabbed silently
- Manual search shows release metadata but gives no comparison against the current file
- Users have no visibility into what they lose when upgrading

---

## Step 1: Conflict Comparison Package

### What

New pure package `internal/core/conflict/conflict.go` with no service dependencies -- pure functions only.

### Core Function

`Compare(current, candidate Quality, currentEditions, candidateEditions []string) []Conflict`

### Conflict Type

```go
type Conflict struct {
    Dimension string   // "audio_codec", "audio_channels", "hdr", "edition", "resolution", "codec"
    Severity  string   // "warning" (significant) or "caution" (minor)
    Current   string   // human-readable current value
    Candidate string   // human-readable candidate value
    Summary   string   // "Audio downgrade: TrueHD Atmos -> AC3 5.1"
}
```

### Dimension Checks (6 total)

Each dimension has a rank function that maps values to integers. A conflict is detected when the candidate ranks lower than current.

1. **Resolution**: 2160p > 1080p > 720p > 480p. Severity: warning if dropping more than one tier, caution otherwise.
2. **Video Codec**: x265/HEVC > x264/AVC > XviD. Severity: caution.
3. **HDR**: DV+HDR10 > DV > HDR10+ > HDR10 > HDR > SDR. Severity: warning (losing HDR is significant).
4. **Audio Codec**: TrueHD Atmos > DTS-X > TrueHD > DTS-HD MA > EAC3 Atmos > EAC3 > AC3 > AAC. Severity: warning.
5. **Audio Channels**: 7.1 > 5.1 > 2.0 > 1.0. Severity: warning if dropping from surround to stereo, caution otherwise.
6. **Edition**: flag when current has edition tags (Director's Cut, Extended) that candidate lacks. Severity: caution.

### Skip Rule

If either value is unknown/empty for a dimension, skip that comparison entirely. This avoids false positives from unparseable metadata.

### Files

| File | Change |
|------|--------|
| `internal/core/conflict/conflict.go` | New file: `Compare()`, rank functions, `Conflict` type |
| `internal/core/conflict/conflict_test.go` | New file: test each dimension, skip-on-unknown, mixed upgrades/downgrades |

---

## Step 2: Summary Text Templates

### What

Generate human-readable conflict summaries from the dimension comparison results.

Format: `"{Dimension} downgrade: {Current} -> {Candidate}"`

Examples:
- "Audio downgrade: TrueHD Atmos -> AC3 5.1"
- "HDR lost: Dolby Vision -> none"
- "Resolution downgrade: 2160p -> 1080p"
- "Edition lost: Director's Cut -> Theatrical"
- "Channel downgrade: 7.1 -> 5.1"

### Files

| File | Change |
|------|--------|
| `internal/core/conflict/conflict.go` | Summary generation as part of `Compare()` output |

---

## Step 3: API Integration

### What

Add a `conflicts` field to the release response body in the search API. When a movie has an existing file, run `conflict.Compare()` against each candidate and include the results inline.

### Response Shape Addition

```json
{
  "title": "Movie.2024.1080p.BluRay...",
  "quality": { ... },
  "conflicts": [
    {
      "dimension": "audio_codec",
      "severity": "warning",
      "current": "TrueHD Atmos",
      "candidate": "AC3 5.1",
      "summary": "Audio downgrade: TrueHD Atmos -> AC3 5.1"
    }
  ]
}
```

When the movie has no existing file, `conflicts` is an empty array.

### Files

| File | Change |
|------|--------|
| `internal/api/v1/movies.go` | In release search handler, call `conflict.Compare()` for each candidate if movie has existing file |

---

## Step 4: Autosearch Integration (Warn-Only)

### What

In the autosearch pipeline, compute conflicts for the selected candidate before grabbing. Log conflicts as warnings. Include in the score breakdown for the explainability feature. Do NOT block grabs based on conflicts in V1.

### Files

| File | Change |
|------|--------|
| `internal/core/autosearch/service.go` | After selecting best candidate, call `conflict.Compare()`; log warnings; include in decision context |

---

## Step 5: Frontend

### What

Show conflict warning pills inline in ManualSearchModal, below the release metadata row for each candidate.

- Warning severity: orange pill with warning icon
- Caution severity: yellow pill with info icon
- Each pill shows the summary text
- Pill tooltip shows full "Current: X -> Candidate: Y" detail

### Files

| File | Change |
|------|--------|
| `web/ui/src/components/ConflictPill.tsx` | New file: styled pill component for conflict display |
| ManualSearchModal component | Add ConflictPill rendering below release rows |
| `web/ui/src/types/index.ts` | Add `Conflict` type |

---

## Implementation Order

```
Step 1: Conflict comparison package    [independent, pure functions]
Step 2: Summary text templates         [part of Step 1]
Step 3: API integration                [depends on 1]
Step 4: Autosearch integration         [depends on 1, can parallel with 3]
Step 5: Frontend                       [depends on 3]
```

**PR Strategy**: Single PR -- the package is pure, the API change is additive (new field), and autosearch is warn-only.

---

## Key Design Decisions

- **Pure package, no dependencies**: `conflict.Compare()` takes values in, returns conflicts out. No DB, no services, no side effects. Easy to test, easy to reuse.
- **Warn-only for V1**: Conflicts do not block grabs. Users may intentionally want a higher-resolution release even with audio downgrade. Blocking can be added later as a profile setting.
- **Skip on unknown**: If audio codec is empty on either side, no audio conflict is reported. This is critical to avoid false positives from releases with unparseable metadata.
- **Two severity levels**: "warning" for significant regressions (losing HDR, losing lossless audio, major resolution drop) and "caution" for minor ones (codec change, small channel reduction). Helps users focus on what matters.

---

## Risks Summary

| Risk | Impact | Mitigation |
|------|--------|------------|
| Rank ordering is opinionated | Users may disagree with HDR or audio rankings | Rankings match community consensus (TRaSH guides); can be made configurable later |
| False positives from bad parsing | Incorrect conflict warnings | Skip-on-unknown rule eliminates most false positives |
| Noisy warnings on every search | User fatigue | Only show for actual downgrades, not lateral moves; severity helps triage |
| Performance on large candidate sets | Slow search response | Compare is O(1) per dimension * 6 dimensions -- negligible |
| Existing file quality unknown | Can't compare against disk | If movie_file has no parsed quality, skip conflict detection entirely |
