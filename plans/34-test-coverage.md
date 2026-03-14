# Plan 34 — Test Coverage Audit & Plan

## Current State

**38 test files** across the codebase. Core services and plugins are reasonably covered. Infrastructure, middleware, API handlers, and several scheduler jobs have zero coverage.

---

## Coverage Report

### Well Covered (no action needed)

| Package | Status |
|---|---|
| `internal/core/quality/` (parser, profile, score, service) | Full CRUD + 45 parser cases + 14 WantRelease scenarios |
| `internal/core/movie/` (service, files, wanted, rename, parser) | CRUD + file ops + rename + 20 parser cases |
| `internal/core/blocklist/` | Full CRUD + pagination |
| `internal/core/downloader/` | Full CRUD + Test + Add |
| `internal/core/indexer/` | Full CRUD + Test + Search + Grab |
| `internal/core/library/` (CRUD only) | Full CRUD + Stats |
| `internal/core/renamer/` | Apply + CleanTitle + FolderName + DestPath |
| `internal/core/dbutil/` | BoolToInt + MergeSettings + IsUniqueViolation |
| `internal/core/mediainfo/` (scanner, export) | NormaliseCodec + NormaliseResolution + DetectHDR |
| `internal/config/` | Load + EnsureAPIKey + WriteConfigKey + Secret redaction |
| `internal/logging/` (ringbuffer, teehandler) | Buffer ops + tee propagation + level filtering |
| `internal/metadata/tmdb/` | Search + Get + StatusMapping + RedactAPIKey |
| `internal/testutil/` | NewTestDB isolation |
| `plugins/downloaders/qbittorrent/` | Full: Test + Add + Queue + Status + Remove + 13 state mappings |
| `plugins/downloaders/nzbget/` | Full: Test + Add + Queue + Status + Remove + state mappings |
| `plugins/downloaders/sabnzbd/` | Full: Test + Add + Queue + Status + Remove + state mappings |
| `plugins/downloaders/transmission/` | Full: Test + Add + Queue + Status + Remove + state mappings |
| `plugins/indexers/torznab/` | Search + Test + GUID fallback + Capabilities + ParseAge |
| `plugins/indexers/newznab/` | Search + Test + GUID fallback + Capabilities + ParseAge |
| `plugins/notifications/command/` | Validate + Notify + Timeout + Test |
| `plugins/notifications/gotify/` | Notify + Test + Sanitizer |
| `plugins/notifications/ntfy/` | Notify + Test + Priority clamping + Sanitizer |
| `plugins/notifications/pushover/` | Notify + Test + Factory validation + Sanitizer |
| `plugins/notifications/telegram/` | Notify + Test + Factory validation + Sanitizer |
| `internal/api/integration_test.go` | 26 tests: health, system, CRUD for profiles/libraries/movies/indexers/downloaders/notifications, queue, tasks, auth |

### Partially Covered (gaps to fill)

| Package | What's tested | What's missing |
|---|---|---|
| `internal/core/movie/` | CRUD, files, wanted, rename, parser | `Update`, `AddUnmatched`, `MatchToTMDB`, `GetByTMDBID`, `RefreshMetadata`, `SuggestMatches` |
| `internal/core/library/` | CRUD + Stats | `ScanDisk`, `Scan`, `matchCandidates`, `ListCandidates`, `DeleteCandidate` |
| `internal/core/importer/` | SingleFile, Directory, MissingGrab, EmptyPath | `copyExtraFiles`, `transferFile` copy fallback, `mediaSvc.ScanFile` path |
| `internal/core/stats/` | Collection, Quality, Snapshot+StorageTrend, GrabPerf, Storage | `DecadeDistribution`, `LibraryGrowth`, `GenreDistribution`, `PruneSnapshots` |
| `internal/core/collection/` | Create, List, Get, Delete | `SearchPeople`, `SearchAll`, `AddMissing`, `AddSelected` |
| `internal/core/indexer/` | CRUD + Test + Search + Grab | `GetRecent`, `ListHistory` filter variants |
| `internal/metadata/tmdb/` | Search + GetMovie | `GetPerson`, `SearchPeople`, `SearchFranchises`, `GetFranchise`, `GetPersonFilmography` |
| `internal/scheduler/jobs/rss_sync` | Private helpers (normalizeTitle, releaseMatchesMovie) | `RssSyncJob.Run()` |
| `internal/api/integration_test.go` | 26 tests | Missing: backup, blocklist, stats, releases, wanted, queue ops, import, history, media servers, plex sync, quality definitions, collections |

### Zero Coverage

| Package | Priority | Notes |
|---|---|---|
| `internal/events/bus.go` | **Critical** | Underpins entire async pipeline. Pub/sub delivery + goroutine safety. |
| `internal/safedialer/safedialer.go` | **Critical** | Security-critical SSRF protection. Table-driven IP validation tests. |
| `internal/core/notification/service.go` | **High** | Fan-out dispatch, error isolation, event filtering. |
| `internal/core/health/service.go` | **High** | Disk space, client connectivity, indexer reachability checks. |
| `internal/core/queue/service.go` | **High** | Queue polling, status mapping, stale item cleanup. |
| `internal/core/downloadhandling/service.go` | **High** | Download handling toggle logic. |
| `internal/core/mediamanagement/service.go` | **Medium** | Media management settings CRUD. |
| `internal/core/mediaserver/service.go` | **Medium** | Media server config CRUD. |
| `internal/core/pathutil/pathutil.go` | **High** | Security: path traversal validation. |
| `internal/core/quality/definitions.go` | **Medium** | Quality definitions getter/setter + defaults. |
| `plugins/downloaders/deluge/` | **High** | Only downloader with no tests. |
| `plugins/notifications/discord/` | **Medium** | Notification plugin, no tests. Pattern exists from gotify/ntfy. |
| `plugins/notifications/email/` | **Medium** | Notification plugin, no tests. |
| `plugins/notifications/slack/` | **Medium** | Notification plugin, no tests. |
| `plugins/notifications/webhook/` | **Medium** | Notification plugin, no tests. |
| `plugins/mediaservers/plex/` | **Medium** | XML parsing, TMDB ID extraction, library refresh. |
| `plugins/mediaservers/emby/` | **Medium** | JSON REST, library sections, refresh. |
| `plugins/mediaservers/jellyfin/` | **Medium** | JSON REST, library sections, refresh. |
| `internal/plexsync/service.go` | **Medium** | Bidirectional sync orchestration. |
| `internal/notifications/dispatcher.go` | **Medium** | Event bus subscriber, fan-out to notifiers. |
| `internal/mediaservers/dispatcher.go` | **Medium** | Import event → media server refresh. |
| `internal/radarrimport/` | **Medium** | Radarr API client + import orchestrator. |
| `internal/registry/registry.go` | **Low** | Plugin registry (simple map). |
| `internal/ratelimit/registry.go` | **Low** | Rate limiter registry. |
| `internal/httpclient/client.go` | **Low** | Shared HTTP client factory. |
| `internal/scheduler/scheduler.go` | **Low** | Ticker-based job runner. |
| `internal/scheduler/jobs/library_scan.go` | **Low** | Job wrapper, logic is in library service. |
| `internal/scheduler/jobs/queue_poll.go` | **Low** | Job wrapper, logic is in queue service. |
| `internal/scheduler/jobs/refresh_metadata.go` | **Low** | Job wrapper, logic is in movie service. |
| `internal/scheduler/jobs/stats_snapshot.go` | **Low** | Job wrapper, logic is in stats service. |
| `internal/api/middleware/` | **Low** | Recovery, request logging, security headers. |
| `internal/api/router.go` | **Low** | Wiring (tested indirectly by integration tests). |
| `internal/api/ws/hub.go` | **Low** | WebSocket hub (hard to unit test). |
| `web/embed.go` | **Low** | Static file serving (tested by running the app). |
| `cmd/luminarr/main.go` | **Low** | Wiring only. |
| `pkg/plugin/` | **Low** | Interfaces + value types (no logic to test). |

---

## Implementation Plan

### Phase 1 — Security Critical (do first)

1. **`internal/safedialer/`** — Table-driven tests for both Strict and LAN transport modes. Test every CIDR range from the security docs (loopback, RFC-1918, link-local, CGNAT, cloud metadata). Verify blocked IPs return errors, allowed IPs connect.

2. **`internal/core/pathutil/`** — Test `ValidateContentPath` with traversal attempts (`../`, symlinks, absolute paths outside allowed dirs). Security boundary code needs explicit coverage.

3. **`internal/events/bus.go`** — Pub/sub delivery test: publish event, verify all subscribers receive it. Concurrent publish test with race detector. Unsubscribe test.

### Phase 2 — Core Service Gaps

4. **`internal/core/notification/service.go`** — CRUD tests (same pattern as other services). Fan-out dispatch: register multiple notifiers, verify all receive event. Error isolation: one notifier errors, others still called. Event filtering: notifier only subscribed to grab events doesn't receive import events.

5. **`internal/core/health/service.go`** — Mock disk stats, download client connectivity, indexer reachability. Test healthy/degraded/unhealthy state transitions.

6. **`internal/core/queue/service.go`** — Mock download client queue responses. Test status mapping, stale item detection.

7. **`internal/core/stats/` gaps** — Add `DecadeDistribution`, `LibraryGrowth`, `GenreDistribution`, `PruneSnapshots` tests. Same in-memory DB pattern as existing stats tests.

8. **`internal/core/movie/` gaps** — Add `Update`, `GetByTMDBID`, `RefreshMetadata` tests. `AddUnmatched` and `MatchToTMDB` if collection feature is stable.

9. **`internal/core/library/` scan gaps** — `ScanDisk` with temp directory + test files. `matchCandidates` with known filenames. `ListCandidates` + `DeleteCandidate` CRUD.

### Phase 3 — Plugin Gaps

10. **`plugins/downloaders/deluge/`** — Follow qbittorrent test pattern: httptest server, Test/Add/Queue/Status/Remove/StateMapping.

11. **`plugins/notifications/discord/`** — Follow gotify pattern: httptest server, Notify/Test/Sanitizer.

12. **`plugins/notifications/slack/`** — Same pattern.

13. **`plugins/notifications/webhook/`** — Same pattern.

14. **`plugins/notifications/email/`** — Mock SMTP server, verify headers (including injection prevention), verify body.

15. **`plugins/mediaservers/plex/`** — httptest with XML responses. Test ListSections, ListMovies (TMDB ID extraction from both agent formats), RefreshLibrary, Test.

16. **`plugins/mediaservers/emby/`** — httptest with JSON responses. Same operations.

17. **`plugins/mediaservers/jellyfin/`** — httptest with JSON responses. Same operations.

### Phase 4 — Integration Test Gaps

18. **Expand `internal/api/integration_test.go`** to cover:
    - Blocklist endpoints (add, list, delete, clear)
    - Wanted endpoints (missing, cutoff unmet)
    - Stats endpoints (collection, quality, storage, grabs)
    - Quality definitions (get, update)
    - Queue operations (remove, blocklist-and-remove)
    - Backup/restore (create backup, restore from backup)
    - Media server CRUD + test
    - History endpoints with filters

### Phase 5 — Lower Priority

19. **`internal/core/importer/`** — `copyExtraFiles` with temp dir + .srt/.nfo files. `transferFile` copy fallback (mock hardlink failure).

20. **`internal/plexsync/service.go`** — Preview + Import with mocked media server plugin. Verify TMDB ID matching, diff calculation.

21. **`internal/radarrimport/`** — Mock Radarr API responses. Test quality name mapping, profile creation, movie deduplication.

22. **`internal/core/downloadhandling/service.go`** — Toggle logic tests.

23. **`internal/core/mediamanagement/service.go`** — Settings CRUD.

24. **`internal/core/mediaserver/service.go`** — Config CRUD (same pattern as other services).

25. **`internal/core/collection/` gaps** — `SearchPeople`, `SearchAll`, `AddMissing`, `AddSelected`.

26. **`internal/metadata/tmdb/` gaps** — `GetPerson`, `SearchPeople`, `SearchFranchises` (if collection feature is active).

---

## Effort Estimate

| Phase | Items | Complexity |
|---|---|---|
| Phase 1 (Security) | 3 | Straightforward table-driven tests |
| Phase 2 (Core gaps) | 6 | Medium — some need mocks, most follow existing patterns |
| Phase 3 (Plugins) | 8 | Formulaic — httptest pattern exists, replicate it |
| Phase 4 (Integration) | 1 (many sub-tests) | Medium — extend existing test harness |
| Phase 5 (Lower priority) | 8 | Mixed — some need filesystem setup, some are simple CRUD |

**Recommended order:** Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5. Security tests first because they protect the most sensitive code and are the easiest to write.
