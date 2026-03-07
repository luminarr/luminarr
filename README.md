# Luminarr

> A self-hosted movie collection manager. Monitors your library, searches indexers, and automatically grabs the best available release — with a quality model that makes sense.

**[Getting Started](docs/GETTING_STARTED.md)** · **[Quick Start](#quick-start)** · **[Import from Radarr](#import-from-radarr-in-one-click)** · **[Library Sync](#library-sync--know-whats-where)** · **[Features](#features)** · **[Privacy](#privacy--no-telemetry)** · **[Architecture](docs/ARCHITECTURE.md)**

---

## Why Luminarr?

Radarr is good software. It built the standard for self-hosted movie management and a lot of us have been running it for years. Luminarr starts from the same ideas and fixes the parts that have always felt wrong.

### Quality profiles that actually make sense

Radarr's quality system has a dirty secret: to control **codec** (x265, AV1) or **HDR** (HDR10, Dolby Vision), you have to learn Custom Formats — a scoring system where you define regex patterns, assign weights, and set thresholds. It's powerful but it's also the most common source of confusion in the Radarr community.

Luminarr makes codec and HDR **first-class dimensions** in every quality profile. You pick them from a dropdown. The profile says exactly what you want: "Bluray-1080p, x265, no HDR" — not "Bluray-1080p, plus at least 10 points of custom format score, minus anything tagged CAM."

```
Radarr:   Resolution + Source + Custom Formats (scoring rules you write yourself)
Luminarr: Resolution + Source + Codec + HDR    (just pick what you want)
```

No Custom Formats to configure. No score thresholds to tune. Your quality profile is self-documenting.

### One-click migration from Radarr

Luminarr connects to your live Radarr instance and imports everything — quality profiles, libraries, indexers, download clients, and your entire movie list — in one go. You don't re-enter a single thing. Full details in [Import from Radarr](#import-from-radarr-in-one-click).

### Modern stack, small footprint

| | Radarr | Luminarr |
|---|---|---|
| Backend | .NET / Mono | Go |
| Frontend | Angular | React |
| Database | SQLite / Postgres | SQLite |
| Memory (idle) | ~300–500 MB | ~30–60 MB |
| Startup time | 10–30 s | < 1 s |

The Go backend starts in under a second, idles well under 100 MB, and handles concurrent requests without per-request overhead. The React frontend is a single-page app that loads fast and works on mobile.

### Zero telemetry, ever

Radarr has optional analytics. Luminarr has no analytics at all — optional or otherwise. It phones home to nothing. Full details in [Privacy](#privacy--no-telemetry).

---

## Features

**Library management**
- **Movie gallery** — poster grid or list view with filters by status, quality, library, and search
- **TMDB integration** — search and add movies with metadata, posters, and cast information
- **Minimum availability** — don't search until a movie reaches a configured release status (TBA / Announced / In Cinemas / Released)
- **Bulk actions** — select and delete multiple movies at once from the movie gallery

**Automation**
- **Quality profiles** — explicit codec + HDR dimensions, upgrade rules, and cutoffs
- **Torznab & Newznab indexers** — compatible with Prowlarr and Jackett
- **qBittorrent & Deluge** — send grabs and monitor download progress
- **Automatic RSS sync** — checks indexers every 15 minutes, grabs matching releases
- **Auto-import** — moves or hardlinks completed downloads into your library
- **Blocklist** — automatically prevents failed grabs from being retried; manage the list in Settings

**Discovery**
- **Wanted page** — two tabs: Missing (monitored movies with no file) and Cutoff Unmet (file below your quality target)
- **Calendar view** — monthly grid of movies by release date, colour-coded by download status
- **Manual search** — search all indexers for a specific movie and grab any release from the results

**File management**
- **Files tab** — view every file attached to a movie, delete records or remove from disk
- **File renaming** — rename files on disk to Luminarr's standard format (`Title (Year) Quality.ext`) from the movie detail panel
- **Per-movie history** — full grab and import history per movie in a History tab
- **Media scanning** — optional ffprobe integration verifies actual codec, resolution, and HDR; flags mislabelled releases with a ⚠ Mismatch badge

**Media server integration**
- **Library Sync** — bidirectional comparison between Luminarr and your media server (Plex, Emby, Jellyfin). See what your server has that Luminarr doesn't track, and what Luminarr tracks that isn't on the server yet. Import server-only movies into Luminarr with one click.
- **Auto-refresh** — when Luminarr imports a movie, it tells your media server to refresh the library automatically. No more waiting for scheduled scans.

**Operations**
- **Radarr import** — one-click migration from a running Radarr instance
- **Notifications** — Discord, Slack, webhook, and email alerts for grabs, imports, and health issues
- **Health monitoring** — disk space, download client connectivity, indexer reachability
- **WebSocket live updates** — Queue page updates in real time without polling
- **OpenAPI docs** — interactive API at `/api/docs`

---

## Quick Start

### Docker (recommended)

```bash
docker run -d \
  --name luminarr \
  -p 8282:8282 \
  -v luminarr-data:/config \
  -v /path/to/movies:/movies \
  ghcr.io/luminarr/luminarr:latest
```

Open `http://localhost:8282`. That's it.

> **Running Radarr too?** Luminarr uses port 8282 specifically so you can run both simultaneously during migration. Radarr stays on 7878.

> **Want media scanning?** Use the `latest-full` image tag to get a build that includes ffprobe. Swap `ghcr.io/luminarr/luminarr:latest` for `ghcr.io/luminarr/luminarr:latest-full` — no other changes needed. See the [Media Scanning section](docs/GETTING_STARTED.md#ffprobe-optional) for details.

### Docker Compose

```yaml
services:
  luminarr:
    image: ghcr.io/luminarr/luminarr:latest
    ports:
      - "8282:8282"
    volumes:
      - luminarr-data:/config
      - /path/to/movies:/movies
    restart: unless-stopped

volumes:
  luminarr-data:
```

### Build from source

```bash
git clone https://github.com/luminarr/luminarr
cd luminarr
make build
./bin/luminarr
```

For full setup instructions — libraries, quality profiles, indexers, and download clients — see the **[Getting Started guide](docs/GETTING_STARTED.md)**.

---

## Import from Radarr in One Click

If you're already running Radarr, you don't need to set Luminarr up from scratch.

1. Go to **Settings → Import** in the Luminarr UI
2. Enter your Radarr URL and API key (Settings → General → Security in Radarr)
3. Click **Connect & Preview** — Luminarr shows you what it found
4. Select which categories to import and click **Import**

Luminarr imports (in order, to respect dependencies):
- Quality profiles → mapped to Luminarr's explicit codec/HDR format
- Libraries from your Radarr root folders
- Indexers (Torznab and Newznab only)
- Download clients (qBittorrent and Deluge only)
- Your entire movie list (duplicates skipped by TMDB ID)

Radarr keeps running during the import — there's no cutover moment. Take your time, verify everything looks right, then switch DNS or your port forward when you're ready.

---

## Library Sync — Know What's Where

If you run a media server (Plex, Emby, or Jellyfin), Luminarr can compare its library against yours and show you the difference in both directions:

- **Server has it, Luminarr doesn't** — movies sitting on your server that Luminarr doesn't know about. Select any of them and import them into Luminarr with one click — Luminarr fetches the metadata from TMDB and starts tracking them immediately.
- **Luminarr has it, server doesn't** — movies Luminarr is tracking that haven't made it to your media server yet. Maybe they're still downloading, maybe the file landed in the wrong place. Either way, now you can see the gap.

No background jobs, no scheduled syncs. Go to **Library Sync**, pick your server and library section, hit **Compare**, and you get a full diff in seconds. Matching is by TMDB ID — clean, unambiguous, no false positives from title/year collisions.

When you do import, Luminarr also tells your media server to refresh its library automatically — so new movies show up without waiting for a scheduled scan.

### Setup

1. **Settings → Media Servers** — add your Plex, Emby, or Jellyfin server (URL + token/API key)
2. **Library Sync** in the sidebar — select your server, pick a movie library section, and compare

That's it. No plugins to install, no sync schedules to configure.

---

## Privacy & No Telemetry

Luminarr makes outbound connections **only** to services you explicitly configure:

| Service | When |
|---|---|
| TMDB | When you search for or refresh a movie |
| Your indexers | RSS sync and release search |
| Your download clients | Sending grabs, polling queue |
| Your media servers | Library sync and auto-refresh — Plex, Emby, Jellyfin — only if configured |
| Your notification targets | Discord, Slack, webhook, email — only if configured |

**What Luminarr never does:**
- No telemetry or usage analytics
- No crash reporting to external services
- No update checks phoning home

Credentials are stored in your local `config.yaml` only and never written to logs. The codebase uses a `Secret` type that renders as `***` in all output.

Full details: [PRIVACY.md](PRIVACY.md)

---

## A Note on How This Was Built

Luminarr was built with AI assistance — specifically [Claude](https://claude.ai) (Anthropic) as the primary code generator, with human design and review throughout. We're not hiding it.

What that means in practice:

- Every architectural decision was made by a human and is documented in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Security-sensitive code (auth middleware, credential handling, external HTTP requests) was explicitly designed with security in mind and reviewed
- The codebase follows consistent patterns throughout — the AI didn't switch styles halfway through
- A test suite covers all core services (quality parsing, profile matching, library management, movie CRUD, event dispatch, import logic)
- The code is readable. If you find something that doesn't make sense, open an issue — we consider that a bug

We think AI-assisted development done right produces better software: more consistent patterns, better documentation, faster iteration. The result should stand on its own merits. Read the code. We welcome scrutiny.

---

## Contributing

Bug reports, feature requests, and pull requests are welcome.

- **Bug reports:** use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml)
- **Feature requests:** use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml)
- **Code:** read [CONTRIBUTING.md](.github/CONTRIBUTING.md) before opening a PR

For architectural context — why things are the way they are — see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## License

MIT — see [LICENSE](LICENSE)
