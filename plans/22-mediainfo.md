# Plan 22 — MediaInfo (ffprobe Integration)

## Problem

Luminarr trusts the filename to determine quality. A file named
`Movie.2160p.x265.HDR10.mkv` is treated as 2160p x265 HDR10 regardless of what
the container actually contains. Re-encodes, mislabelled releases, and codec
surprises are invisible.

This undermines our core differentiator: explicit codec + HDR quality profiles
are only as trustworthy as the data behind them.

---

## Goal

After import, scan each file with `ffprobe` and store the *actual* technical
metadata. Surface it in the Files tab and flag mismatches between claimed and
actual quality.

---

## Dependency Strategy (No Image Bloat)

**ffprobe is optional.** Luminarr checks for it at startup and logs whether
media scanning is available.

| Scenario | Behaviour |
|---|---|
| ffprobe found in `$PATH` or `config.mediainfo.ffprobe_path` | Scanning enabled |
| ffprobe not found | Scanning silently disabled; no errors surfaced to users |
| File scan fails (timeout, corrupt file, etc.) | Logged at debug; not an error |

### Docker image variants

| Tag | Base | Includes ffprobe |
|---|---|---|
| `latest` (current) | scratch | No — stays lean |
| `latest-full` (new) | `alpine:3.21` | Yes — `apk add ffmpeg` |

Users who want media scanning pull `latest-full`. Users who want the smallest
possible image stay on `latest`. Both are published from the same CI workflow.

### Config

```yaml
mediainfo:
  ffprobe_path: ""      # empty = search $PATH; set to /usr/bin/ffprobe to pin
  scan_timeout: 30s     # per-file timeout
  scan_on_import: true  # auto-scan after successful import
```

Env vars: `LUMINARR_MEDIAINFO_FFPROBE_PATH`, `LUMINARR_MEDIAINFO_SCAN_TIMEOUT`.

---

## Data Model

### Migration (`internal/db/migrations/00020_movie_file_mediainfo.sql`)

```sql
-- +goose Up
ALTER TABLE movie_files ADD COLUMN mediainfo_json TEXT NOT NULL DEFAULT '';
ALTER TABLE movie_files ADD COLUMN mediainfo_scanned_at DATETIME;

-- +goose Down
SELECT 1;
```

`mediainfo_json` is a serialised JSON blob stored in the existing `movie_files`
row — no separate table needed.

### Go type (`internal/core/mediainfo/types.go`)

```go
package mediainfo

type Result struct {
    Container    string  `json:"container"`      // "mkv", "mp4"
    DurationSecs float64 `json:"duration_secs"`
    VideoBitrate int64   `json:"video_bitrate"`  // bits/sec
    // Video stream
    Codec        string  `json:"codec"`          // "hevc", "h264", "av1"
    Width        int     `json:"width"`
    Height       int     `json:"height"`
    Resolution   string  `json:"resolution"`     // normalised: "2160p", "1080p", etc.
    ColorSpace   string  `json:"color_space"`    // "bt2020nc", "bt709"
    HDRFormat    string  `json:"hdr_format"`     // "HDR10", "Dolby Vision", "SDR"
    BitDepth     int     `json:"bit_depth"`      // 8, 10, 12
    // Audio (primary stream)
    AudioCodec   string  `json:"audio_codec"`    // "eac3", "dts", "aac"
    AudioChannels int    `json:"audio_channels"` // 2, 6, 8
}
```

---

## ffprobe Scanner (`internal/core/mediainfo/scanner.go`)

```go
package mediainfo

type Scanner struct {
    ffprobePath string  // "" means disabled
    timeout     time.Duration
}

func New(ffprobePath string, timeout time.Duration) *Scanner
func (s *Scanner) Available() bool
func (s *Scanner) Scan(ctx context.Context, filePath string) (*Result, error)
```

Implementation:
1. `exec.CommandContext` with timeout
2. `ffprobe -v quiet -print_format json -show_streams -show_format <path>`
3. Parse JSON output — extract first video stream and first audio stream
4. Normalise `codec_name` → `"hevc"/"h264"/"av1"` etc.
5. Normalise height → `"2160p"/"1080p"/"720p"/"SD"`
6. Detect HDR from `color_transfer` (`"smpte2084"` → HDR10) and `color_primaries`
   (`"bt2020"` = wide gamut). Dolby Vision detected from side data.

---

## Integration with Movie Service

### Post-import scan

In `internal/core/importer/importer.go`, after `AttachFile()` succeeds:

```go
if s.scanner.Available() {
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
        defer cancel()
        if err := s.mediaSvc.ScanFile(ctx, fileID, filePath); err != nil {
            s.logger.Debug("mediainfo scan failed", "file_id", fileID, "err", err)
        }
    }()
}
```

Fire-and-forget goroutine. Import response is not delayed.

### Bulk scan

`internal/core/mediainfo/service.go`:

```go
type Service struct {
    scanner *Scanner
    q       dbsqlite.Querier
}

// ScanFile scans one file and updates movie_files.mediainfo_json.
func (s *Service) ScanFile(ctx context.Context, fileID, filePath string) error

// ScanAll scans every movie_file where mediainfo_json = ''.
// Returns count scanned. Intended for the "Scan all" settings button.
func (s *Service) ScanAll(ctx context.Context) (int, error)
```

### API endpoint for on-demand re-scan

```
POST /api/v1/movies/{id}/files/{fileId}/scan
```

Returns 202. Triggers `ScanFile` in a goroutine. Useful when a file is manually
replaced or the scan previously failed.

Add a "Re-scan" icon button in the Files tab, visible only when scanner is available.

---

## sqlc queries (`internal/db/queries/sqlite/movies.sql` additions)

```sql
-- name: UpdateMovieFileMediainfo :exec
UPDATE movie_files
SET mediainfo_json       = ?,
    mediainfo_scanned_at = ?
WHERE id = ?;

-- name: ListUnscannedMovieFiles :many
SELECT id, path FROM movie_files
WHERE mediainfo_json = ''
ORDER BY imported_at DESC;
```

Run `sqlc generate` after adding.

---

## Frontend Changes

### Files Tab in Movie Detail

Current columns: path, size, quality, imported date, delete/rename buttons.

New columns (when mediainfo available):
- **Actual Quality** — resolved from `mediainfo_json`: e.g. `1080p x264 SDR`
- **Mismatch badge** — shown when claimed quality ≠ actual quality

Mismatch logic (frontend):
```typescript
function hasMismatch(claimed: Quality, actual: MediaInfo): boolean {
  return (
    claimed.codec !== normaliseCodec(actual.codec) ||
    claimed.resolution !== actual.resolution ||
    claimed.hdr !== actual.hdr_format
  )
}
```

When mismatch: show a yellow `⚠ Mismatch` badge. Tooltip:
```
Filename claims: x265 HDR10
Actual:          x264 SDR
```

### Settings page — Media Scanning section

**Settings → Media Scanning**

Shows:
- Scanner status: `● Available (ffprobe 6.1)` or `○ Unavailable — install ffprobe`
- "Scan all unscanned files" button → `POST /api/v1/mediainfo/scan-all` → 202
- Progress (WebSocket event stream or polling)

---

## WebSocket event

When `ScanAll` is running, emit events via the existing event bus:

```go
bus.Publish(ctx, events.Event{
    Type: events.TypeMediainfoScanProgress,
    Payload: MediainfoProgressPayload{
        Scanned: n,
        Total:   total,
    },
})
```

Frontend shows a progress bar on the Settings page while scan runs.

---

## Docker Image Changes

### Dockerfile.full (new file)

```dockerfile
FROM alpine:3.21
RUN apk add --no-cache ffmpeg ca-certificates tzdata
COPY bin/luminarr /luminarr
ENTRYPOINT ["/luminarr"]
```

### CI (.github/workflows/docker.yml additions)

Build and push both tags on every release:
- `ghcr.io/davidfic/luminarr:latest` — existing scratch build
- `ghcr.io/davidfic/luminarr:latest-full` — alpine + ffprobe

---

## Tests

**Unit** (`internal/core/mediainfo/scanner_test.go`):
- `TestScan_realFile` — create a small test video with ffmpeg in CI, verify Result fields
- `TestScan_unavailable` — scanner with empty path returns Available()=false
- `TestScan_timeout` — mock slow ffprobe, verify timeout respected
- `TestNormaliseCodec` — "hevc" → "x265", "h264" → "x264", "av01" → "AV1"
- `TestDetectHDR` — color_transfer="smpte2084" → "HDR10", bt709 → "SDR"

**Integration** (`internal/api/v1/mediainfo_test.go`):
- `TestScanFileEndpoint` — POST returns 202
- `TestFileMediainfoInResponse` — after scan, GET /movies/{id}/files includes mediainfo_json

---

## Open Questions

1. **ffprobe in CI**: The test `TestScan_realFile` requires ffprobe in the CI runner.
   GitHub-hosted runners have ffmpeg pre-installed. The test should be guarded:
   `if testing.Short() { t.Skip() }` so it's skipped in unit-only runs.

2. **Codec normalisation map**: ffprobe returns codec names like `hevc`, `h264`,
   `av01`, `mpeg2video`, `vp9`. We need a canonical map to our internal names
   (`x265`, `x264`, `AV1`, etc.). Define this in `scanner.go` as a package-level
   `var`.

3. **HDR detection accuracy**: Dolby Vision detection requires reading RPU side data
   from the ffprobe stream. This is present in the `-show_streams` output as
   `side_data_list[].side_data_type = "DOVI configuration record"`. Worth
   implementing; not blocking for v1.

4. **Should we scan on library `Scan()` too?** Potentially — if a file already has
   a `movie_files` record but no mediainfo, queue it. Deferred to keep initial
   scope tight.
