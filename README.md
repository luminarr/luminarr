<p align="center">
  <img src="web/ui/public/luminarr-512.png" alt="Luminarr" width="120">
</p>
<h1 align="center">Luminarr</h1>
<p align="center">A self-hosted movie collection manager built for simplicity.</p>
<p align="center">
  <img src="https://github.com/luminarr/luminarr/actions/workflows/ci.yml/badge.svg" alt="CI">
  <a href="https://github.com/luminarr/luminarr/releases/latest"><img src="https://img.shields.io/github/v/release/luminarr/luminarr" alt="Release"></a>
  <a href="https://github.com/luminarr/luminarr/blob/main/LICENSE"><img src="https://img.shields.io/github/license/luminarr/luminarr" alt="License"></a>
  <img src="https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white" alt="Go 1.25">
</p>
<p align="center">
  <a href="https://luminarr.video">Website</a> ·
  <a href="https://github.com/luminarr/luminarr/wiki">Documentation</a> ·
  <a href="https://github.com/luminarr/luminarr/issues">Bug Reports</a> ·
  <a href="https://github.com/luminarr/luminarr/wiki/Getting-Started">Getting Started</a>
</p>

---

**Luminarr** monitors your movie library, searches indexers, and automatically grabs the best available release. It's written in Go and React, starts in under a second, and idles under 60 MB of RAM.

If you're coming from Radarr, Luminarr can import your entire setup — quality profiles, libraries, indexers, download clients, and movie list — in one click.

<p align="center">
  <img src="docs/screenshots/dashboard.png" alt="Luminarr Dashboard" width="800">
</p>

## Current Features

- **Quality profiles** with explicit codec + HDR dimensions — no Custom Formats to configure
- **TMDB integration** — search, add, and manage movies with full metadata
- **Torznab & Newznab indexers** — compatible with Prowlarr and Jackett
- **qBittorrent, Deluge, Transmission, SABnzbd, NZBGet** download clients
- **Automatic RSS sync** — checks indexers on a schedule and grabs matching releases
- **Auto-import** — moves or hardlinks completed downloads into your library
- **Radarr import** — one-click migration from a running Radarr instance
- **Radarr v3 API compatibility** — use Overseerr, Homepage, Home Assistant, and other tools without changes
- **Media server integration** — Plex, Emby, and Jellyfin library sync with auto-refresh
- **Library Sync** — compare your media server library against Luminarr and import the difference
- **Wanted page** — missing movies and cutoff-unmet in one view
- **Calendar** — monthly grid of movies by release date
- **Manual search** — search all indexers for a specific movie and pick a release
- **File management** — view, rename, and delete files per movie
- **Media scanning** — optional ffprobe integration to verify actual codec, resolution, and HDR
- **Notifications** — Discord, Slack, Telegram, Pushover, Gotify, ntfy, webhook, email, and custom scripts
- **Health monitoring** — disk space, download client connectivity, indexer reachability
- **AI command palette** — optional Claude-powered natural language commands (Cmd+K): "grab Dune in 4K", "how many movies am I missing?", "go to quality profiles". State-modifying actions require explicit confirmation. Requires an Anthropic API key (Settings → App)
- **WebSocket live updates** — real-time queue updates without polling
- **OpenAPI docs** — interactive API at `/api/docs`
- **Zero telemetry** — no analytics, no crash reporting, no phoning home

## Preview

| Library | Quality Profiles |
|:-:|:-:|
| ![Library](docs/screenshots/dashboard.png) | ![Quality Profiles](docs/screenshots/quality-profiles.png) |

## Getting Started

### Docker (recommended)

```bash
docker run -d \
  --name luminarr \
  -p 8282:8282 \
  -v /path/to/config:/config \
  -v /path/to/movies:/movies \
  ghcr.io/luminarr/luminarr:latest
```

Open `http://localhost:8282`. That's it.

### Docker Compose

```yaml
services:
  luminarr:
    image: ghcr.io/luminarr/luminarr:latest
    ports:
      - "8282:8282"
    volumes:
      - /path/to/config:/config
      - /path/to/movies:/movies
    restart: unless-stopped
```

### Build from source

```bash
git clone https://github.com/luminarr/luminarr
cd luminarr
make build
./bin/luminarr
```

> **Running Radarr too?** Luminarr uses port 8282 so you can run both side by side during migration.

> **Media scanning** works out of the box — the default Docker image includes ffprobe. A `latest-minimal` tag (scratch-based, no ffprobe) is also available.

For full setup instructions, see the **[Getting Started guide](https://github.com/luminarr/luminarr/wiki/Getting-Started)**.

## Radarr API Compatibility

Luminarr exposes a Radarr v3 compatible API at `/api/v3/`. External tools with a "Radarr" integration — Overseerr, Jellyseerr, Homepage, Home Assistant, LunaSea, and others — can point directly at Luminarr with no changes on their side.

| Field | Value |
|-------|-------|
| **URL** | `http://<luminarr-host>:8282` |
| **API Key** | Your Luminarr API key (Settings → App Settings) |

Full details: [Radarr API Compatibility](https://github.com/luminarr/luminarr/wiki/Radarr-API-Compatibility)

## Privacy

Luminarr makes outbound connections **only** to services you explicitly configure (TMDB, your indexers, your download clients, your media servers, your notification targets, and optionally the Anthropic API for AI features). No telemetry, no analytics, no crash reporting, no update checks.

Credentials are stored in your local `config.yaml` only and never written to logs. When AI features are enabled, only your command text and aggregate library stats are sent to Claude — no movie titles, file paths, or personal data.

Full details: [Privacy](https://github.com/luminarr/luminarr/wiki/Privacy)

## How This Was Built

Luminarr was built with AI assistance — specifically [Claude](https://claude.ai) (Anthropic) as the primary code generator, with human design and review throughout. Every architectural decision was made by a human. The code is readable and tested. If something doesn't make sense, that's a bug — [open an issue](https://github.com/luminarr/luminarr/issues).

## Contributing

Bug reports, feature requests, and pull requests are welcome.

- **Bug reports:** [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml)
- **Feature requests:** [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml)
- **Code:** read [CONTRIBUTING.md](.github/CONTRIBUTING.md) before opening a PR

## License

MIT — see [LICENSE](LICENSE)
