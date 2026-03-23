# Luminarr — Architecture & Technical Reference

This document covers internals, configuration, API reference, and development setup. The [README](../README.md) covers the user-facing pitch.

---

## Table of Contents

- [Architecture overview](#architecture-overview)
- [Project structure](#project-structure)
- [Configuration](#configuration)
- [API reference](#api-reference)
- [Plugin system](#plugin-system)
- [Quality model](#quality-model)
- [Event system](#event-system)
- [Database](#database)
- [Auth model](#auth-model)
- [Development](#development)
- [Testing](#testing)

---

## Architecture Overview

Luminarr is a single Go binary that embeds the React frontend as a static file system. At runtime it serves both the API and the UI from the same port.

```
┌─────────────────────────────────────────────────────────┐
│  Browser                                                │
│  React SPA (embedded in binary at /web/static/)        │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP
┌──────────────────────▼──────────────────────────────────┐
│  Chi router + Huma (OpenAPI)                           │
│  Auth middleware (X-Api-Key header)                    │
├─────────────────────────────────────────────────────────┤
│  Core services                                         │
│  movie · quality · library · indexer · downloader      │
│  queue · notification · health · importer · plexsync   │
│  aicommand · autosearch · stats                        │
├─────────────────────────────────────────────────────────┤
│  Plugin registry                                       │
│  torznab · newznab                                     │
│  qbittorrent · deluge · transmission · sabnzbd · nzbget│
│  plex · emby · jellyfin                                │
│  discord · slack · telegram · gotify · ntfy · pushover │
│  webhook · email · command                             │
├─────────────────────────────────────────────────────────┤
│  Infrastructure                                        │
│  SQLite/Postgres (sqlc) · Events bus · Scheduler       │
│  TMDB client · Anthropic client · Radarr import       │
└─────────────────────────────────────────────────────────┘
```

**Request flow:** Browser → Chi router → Huma auth middleware → handler → core service → sqlc querier → SQLite.

**Background flow:** Scheduler ticks → job calls service → service publishes event → bus delivers to subscribers (importer, notification dispatcher).

---

## Project Structure

```
cmd/luminarr/          Main entrypoint + wiring
internal/
  api/                 HTTP router, middleware, Huma setup
  api/v1/              REST handlers (one file per domain)
  config/              Config loading (Viper), Secret type
  anthropic/           Minimal Claude Messages API client
  core/
    aicommand/         AI command palette service (intent parsing, confirmation, execution)
    downloader/        Download client config + grab dispatch
    health/            System health checks (disk, clients, indexers)
    importer/          File import after download completes
    indexer/           Indexer config + search orchestration
    library/           Library CRUD + disk stats
    movie/             Movie CRUD + TMDB metadata
    notification/      Notification config management
    quality/           Quality profile CRUD + name parsing
    mediaserver/       Media server config CRUD (Plex, Emby, Jellyfin)
    queue/             Download queue polling
    renamer/           Naming template engine
    seedenforcer/      Post-import seed limit enforcement
  db/                  DB connection, migrations (goose), sqlc wiring
  db/generated/sqlite/ sqlc-generated query code — do not edit
  db/migrations/       Numbered goose SQL migration files
  events/              In-process pub/sub bus
  logging/             Structured logger setup (slog)
  metadata/tmdb/       TMDB API client
  notifications/       Event bus → notification dispatcher
  plexsync/            Bidirectional library sync (media server ↔ Luminarr)
  radarrimport/        One-time Radarr migration client + orchestrator
  registry/            Plugin registry (indexers, downloaders, notifiers)
  scheduler/           Background job scheduler
  scheduler/jobs/      Built-in jobs: rss_sync, queue_poll, library_scan, refresh_metadata
  version/             Version constants (injected at build time)
pkg/plugin/            Public plugin interfaces + shared value types
plugins/
  downloaders/deluge/
  downloaders/qbittorrent/
  downloaders/transmission/
  downloaders/sabnzbd/
  downloaders/nzbget/
  indexers/newznab/
  indexers/torznab/
  mediaservers/plex/
  mediaservers/emby/
  mediaservers/jellyfin/
  notifications/command/
  notifications/discord/
  notifications/email/
  notifications/gotify/
  notifications/ntfy/
  notifications/pushover/
  notifications/slack/
  notifications/telegram/
  notifications/webhook/
web/
  embed.go             Embeds /web/static/ into the binary; injects API key into HTML
  static/              Built React SPA (generated by `npm run build`)
web/ui/                React source (Vite + TypeScript)
  src/
    api/               React Query hooks for each domain
    components/        Shared UI components
    layouts/Shell.tsx  Sidebar navigation + layout
    pages/             Page components (dashboard, settings/*)
    types/index.ts     TypeScript interfaces matching Go API shapes
```

---

## Configuration

All settings can be set via `config.yaml` or environment variables. Environment variables are prefixed with `LUMINARR_` and use underscores for dots: `server.port` → `LUMINARR_SERVER_PORT`.

### Core settings

| Key | Default | Env var | Description |
|-----|---------|---------|-------------|
| `server.host` | `0.0.0.0` | `LUMINARR_SERVER_HOST` | Listen address |
| `server.port` | `8282` | `LUMINARR_SERVER_PORT` | HTTP port |
| `database.driver` | `sqlite` | `LUMINARR_DATABASE_DRIVER` | `sqlite` or `postgres` |
| `database.path` | `~/.config/luminarr/luminarr.db` | `LUMINARR_DATABASE_PATH` | SQLite path |
| `database.dsn` | — | `LUMINARR_DATABASE_DSN` | Postgres connection string |
| `auth.api_key` | auto-generated | `LUMINARR_AUTH_API_KEY` | Required for all API calls |
| `tmdb.api_key` | — | `LUMINARR_TMDB_API_KEY` | Movie metadata (optional) |
| `ai.api_key` | — | `LUMINARR_AI_API_KEY` | Anthropic key for AI command palette (optional) |
| `log.level` | `info` | `LUMINARR_LOG_LEVEL` | `debug`, `info`, `warn`, `error` |
| `log.format` | `json` | `LUMINARR_LOG_FORMAT` | `json` or `text` |

See [`config.example.yaml`](../config.example.yaml) for a fully-commented reference.

### Docker note

The scratch Docker image has no `$HOME`, so the config file default path (`~/.config/luminarr/`) doesn't resolve. Always set `LUMINARR_AUTH_API_KEY` explicitly, or the key changes on every container restart (any open browser tab will get 401 errors).

---

## API Reference

All endpoints require `X-Api-Key: <your-key>` header except `GET /health`.

Interactive docs with full request/response schemas are available at `/api/docs` when the server is running.

### Endpoints by domain

| Domain | Endpoints |
|--------|-----------|
| **System** | `GET /api/v1/system/status` · `GET /api/v1/system/health` |
| **Tasks** | `GET /api/v1/tasks` · `POST /api/v1/tasks/{name}/run` |
| **Movies** | `GET /POST /api/v1/movies` · `GET/PUT/DELETE /api/v1/movies/{id}` · `POST /api/v1/movies/lookup` · `POST /api/v1/movies/{id}/refresh` |
| **Libraries** | `GET/POST /api/v1/libraries` · `GET/PUT/DELETE /api/v1/libraries/{id}` · `GET /api/v1/libraries/{id}/stats` · `POST /api/v1/libraries/{id}/scan` |
| **Quality Profiles** | `GET/POST /api/v1/quality-profiles` · `GET/PUT/DELETE /api/v1/quality-profiles/{id}` |
| **Indexers** | `GET/POST /api/v1/indexers` · `GET/PUT/DELETE /api/v1/indexers/{id}` · `POST /api/v1/indexers/{id}/test` |
| **Download Clients** | `GET/POST /api/v1/download-clients` · `GET/PUT/DELETE /api/v1/download-clients/{id}` · `POST /api/v1/download-clients/{id}/test` |
| **Releases** | `GET /api/v1/movies/{id}/releases` · `POST /api/v1/movies/{id}/releases/{guid}/grab` |
| **Queue** | `GET /api/v1/queue` · `DELETE /api/v1/queue/{id}` |
| **Notifications** | `GET/POST /api/v1/notifications` · `GET/PUT/DELETE /api/v1/notifications/{id}` · `POST /api/v1/notifications/{id}/test` |
| **Media Servers** | `GET/POST /api/v1/media-servers` · `GET/PUT/DELETE /api/v1/media-servers/{id}` · `POST /api/v1/media-servers/{id}/test` |
| **Library Sync** | `GET /api/v1/media-servers/{id}/sections` · `POST /api/v1/media-servers/{id}/sync/preview` · `POST /api/v1/media-servers/{id}/sync/import` |
| **AI** | `POST /api/v1/ai/command` · `POST /api/v1/ai/command/confirm` |
| **Import** | `POST /api/v1/import/radarr/preview` · `POST /api/v1/import/radarr/execute` |

---

## Plugin System

Plugins implement one of four interfaces defined in `pkg/plugin/`:

- `Indexer` — `Search(ctx, query, categories) ([]Release, error)`
- `DownloadClient` — `Add(ctx, release) (itemID string, error)` · `Status(ctx, itemID) (DownloadStatus, error)` · etc.
- `SeedLimiter` (optional) — `SetSeedLimits(ctx, clientItemID, ratioLimit, seedTimeSecs) error` — implemented by torrent clients (qBittorrent, Deluge, Transmission) for per-indexer seed enforcement
- `MediaServer` — `RefreshLibrary(ctx, moviePath) error` · `Test(ctx) error`
- `Notifier` — `Notify(ctx, event) error` · `Test(ctx) error`

**Registration:** each plugin calls `registry.Default.Register*()` from its `init()` function. Plugins are activated by blank-importing their package in `cmd/luminarr/main.go`.

**Settings:** plugin settings are stored as opaque `json.RawMessage` per config record. The registry validates and sanitizes settings when a config is created or updated — unrecognized fields are rejected, sensitive fields (passwords, API keys) are redacted from API responses.

### Adding a plugin

1. Create `plugins/{kind}/{kind}.go`
2. Define a struct implementing the interface
3. Call `registry.Default.Register*(kind, factory)` in `init()`
4. Add `_ "github.com/luminarr/luminarr/plugins/{kind}/{kind}"` to `cmd/luminarr/main.go`
5. Add the settings shape to the UI's settings sub-form component

---

## Media Server Integration & Library Sync

### Media server plugins

Media server plugins (`plugins/mediaservers/`) handle two responsibilities:

1. **Library refresh** — after Luminarr imports a movie, it calls `RefreshLibrary(ctx, moviePath)` on all configured media servers. The plugin finds the matching library section by path prefix and triggers a targeted refresh via the server's API.
2. **Library listing** — Plex plugins expose `ListSections(ctx)` and `ListMovies(ctx, sectionKey)` for the library sync feature.

| Plugin | API format | Auth | TLS |
|--------|-----------|------|-----|
| Plex | XML (sections, movies) | `X-Plex-Token` header | Self-signed accepted |
| Emby | JSON REST | `?api_key=` query param | Self-signed accepted |
| Jellyfin | JSON REST | `Authorization: MediaBrowser Token=` | Self-signed accepted |

All media server plugins use `safedialer.LANTransport()` with `InsecureSkipVerify: true` to support self-signed certificates common on LAN servers.

### Library sync architecture

The `internal/plexsync` service orchestrates the bidirectional library comparison:

```
┌──────────────┐     ListMovies()     ┌──────────────────┐
│ Media Server │ ◄──────────────────► │  plexsync.Service │
│ (Plex API)   │    XML/JSON → Movie  │                  │
└──────────────┘                      │  Preview():      │
                                      │   1. Fetch all   │
┌──────────────┐  ListMovieSummaries  │      server      │
│ SQLite DB    │ ◄──────────────────► │      movies      │
│ (movies tbl) │   TMDB ID + status   │   2. Fetch all   │
└──────────────┘                      │      Luminarr    │
                                      │      movies      │
                                      │   3. Set diff    │
                                      │      by TMDB ID  │
                                      │                  │
                                      │  Import():       │
                                      │   movie.Add()    │
                                      │   per TMDB ID    │
                                      └──────────────────┘
```

**Matching strategy:** TMDB ID only. Plex stores TMDB IDs in two formats:
- New agent: `<Guid id="tmdb://12345"/>` child elements (requires `?includeGuids=1` on the API request)
- Legacy agent: `guid="com.plexapp.agents.themoviedb://12345?lang=en"` top-level attribute

Movies without a TMDB GUID are counted as "unmatched" and excluded from the diff.

**Data flow for preview:**
1. `plexsync.Service.Preview()` loads the media server config from DB
2. Instantiates the Plex plugin with the stored URL + token
3. Calls `ListMovies()` → builds `map[tmdbID]PlexMovie`
4. Queries `ListMovieSummaries` from SQLite → builds `map[tmdbID]LuminarrMovie`
5. Iterates both maps to produce `inPlexOnly`, `inLuminarrOnly`, and `alreadySynced` counts

**Data flow for import:**
1. For each selected TMDB ID, calls `movie.Add()` which fetches metadata from TMDB and creates the movie record
2. Reports imported/skipped/failed counts — skipped means the movie already exists (duplicate TMDB ID)

### API endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/media-servers/{id}/sections` | GET | List movie library sections from the media server |
| `/api/v1/media-servers/{id}/sync/preview` | POST | Compare server library against Luminarr; body: `{"section_key":"1"}` |
| `/api/v1/media-servers/{id}/sync/import` | POST | Import selected movies; body: `{"tmdb_ids":[...],"library_id":"...","quality_profile_id":"...","monitored":true}` |

---

## Quality Model

Luminarr's `plugin.Quality` is a value type with four explicit dimensions:

```go
type Quality struct {
    Resolution Resolution // "sd", "720p", "1080p", "2160p"
    Source     Source     // "cam", "dvd", "hdtv", "webrip", "webdl", "bluray", "remux"
    Codec      Codec      // "unknown", "x264", "x265", "av1", "xvid"
    HDR        HDRFormat  // "none", "hdr10", "dolby_vision", "hlg", "hdr10plus"
    Name       string     // human-readable label, e.g. "Bluray-1080p x265"
}
```

**Scoring:** `Score() = resolutionScore*100 + sourceScore*10 + codecScore`. Used for upgrade decisions and cutoff comparisons.

**Quality profiles** store:
- `Cutoff` — minimum acceptable quality; anything below is always grabbed if available
- `Qualities` — ordered list of acceptable qualities
- `UpgradeAllowed` — whether to grab a better release if one becomes available
- `UpgradeUntil` — stop upgrading once this quality is reached

**Radarr import mapping:** Radarr quality names (e.g. `"Bluray-1080p"`) are translated to Luminarr Quality structs via a static lookup table in `internal/radarrimport/service.go`. Codec defaults to `"unknown"` and HDR defaults to `"none"` since Radarr expresses these via Custom Formats rather than quality profiles.

---

## Event System

`internal/events/Bus` is an in-process pub/sub system. Events are published by services and delivered to registered handlers in separate goroutines.

**Event types** (defined in `internal/events/`):
- `TypeGrabStarted` — a release was sent to a download client
- `TypeGrabFailed`
- `TypeDownloadDone` — download client reports completion
- `TypeImportComplete` — file moved/hardlinked into library
- `TypeImportFailed`
- `TypeHealthIssue`
- `TypeHealthOK`

**Subscribers:**
- `internal/core/importer` — subscribes to `TypeDownloadDone`, imports the file
- `internal/core/seedenforcer` — subscribes to `TypeImportComplete`, applies per-indexer seed ratio/time limits to the torrent via the download client
- `internal/notifications/Dispatcher` — subscribes to all event types, fans out to enabled notifiers

---

## Database

Luminarr supports SQLite (default) and PostgreSQL.

**Schema management:** [goose](https://github.com/pressly/goose) numbered SQL migrations in `internal/db/migrations/`. Migrations run automatically on startup.

**Query layer:** [sqlc](https://sqlc.dev/) generates type-safe Go from SQL in `internal/db/queries/`. The generated code lives in `internal/db/generated/sqlite/` — do not edit it by hand. To add or modify a query: edit the `.sql` file, then run `sqlc generate` or `make generate`.

**ID strategy:** all records use UUID v4 strings as primary keys.

---

## Auth Model

1. `config.EnsureAPIKey()` in `internal/config/load.go` generates a random 32-byte hex key if none is configured
2. `web.ServeIndex(apiKey)` in `web/embed.go` substitutes `__LUMINARR_KEY__` in the HTML template once at startup — the key is baked into the served page
3. The browser stores the key in `window.__LUMINARR_KEY__`; every `apiFetch()` sends it as `X-Api-Key`
4. Huma middleware in `internal/api/router.go` rejects any request where the header does not match the in-memory key

**Important:** In Docker (scratch image, no `$HOME`), the key is regenerated on every container start unless `LUMINARR_AUTH_API_KEY` is fixed. If a browser tab holds a stale key, all API calls return 401. Hard-refresh after restart, or set a fixed key.

---

## Development

### Prerequisites

- Go 1.23+
- Node.js 20+
- (optional) `sqlc`, `goose`, `air` for SQL generation and hot reload

```bash
go install github.com/air-verse/air@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### Running locally

Two terminals:

```bash
# Terminal 1: Go backend with hot reload
make dev
# or: go run ./cmd/luminarr

# Terminal 2: React dev server (proxies /api to localhost:8282)
cd web/ui && npm run dev
```

Visit `http://localhost:5173` (Vite dev server) for the UI with hot module replacement.

### Common make targets

```bash
make build        # build binary to ./bin/luminarr
make test         # run all Go tests
make test/race    # run tests with race detector
make generate     # regenerate sqlc code
make migrate      # run goose migrations
make docker/run   # docker compose up --build
```

### Frontend build

```bash
cd web/ui
npm run build     # production build → web/static/
npm run dev       # dev server on :5173
```

The Go binary embeds `web/static/` at compile time via `go:embed`. Running `go build` after `npm run build` produces a self-contained binary.

---

## Testing

All core services have table-driven unit tests using `testing` and `testify`. Tests use in-memory SQLite (via the same migration path as production) — no mocks for the DB layer.

```bash
make test         # short tests only
make test/race    # full suite with race detector
go test ./internal/core/... -v  # core services only
```

Test coverage targets:
- `internal/core/quality` — parsing, scoring, profile matching
- `internal/core/movie` — CRUD, duplicate detection, metadata stub handling
- `internal/core/library` — CRUD, disk stats
- `internal/core/indexer` — config CRUD, search dispatch
- `internal/core/downloader` — config CRUD, grab dispatch
- `internal/core/notification` — config CRUD, event dispatch
- `internal/events` — bus publish/subscribe, concurrent delivery
- `internal/radarrimport` — quality name mapping, field extraction
- `internal/core/seedenforcer` — event-driven seed limit enforcement (mock providers)
- `plugins/downloaders/*` — download client integration (httptest servers), including `SetSeedLimits`
- `plugins/notifications/*` — notification delivery (httptest servers, script execution)

Integration tests (full HTTP stack) are on the roadmap.
