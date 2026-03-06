# Getting Started with Luminarr

This guide walks you through installing Luminarr, configuring it, and getting your first movies tracked. If you're coming from Radarr, there's a one-click import — skip to [Migrating from Radarr](#migrating-from-radarr).

---

## Prerequisites

You need one thing before you start:

- **A TMDB API key** — free at [themoviedb.org/settings/api](https://www.themoviedb.org/settings/api). Sign up, request an API key (choose "Developer"), and you'll have it in two minutes. Without this key, Luminarr runs but can't search for or fetch movie metadata.

---

## Installation

### Docker (recommended)

```bash
docker run -d \
  --name luminarr \
  -p 8282:8282 \
  -v luminarr-data:/config \
  -v /path/to/movies:/movies \
  -e LUMINARR_TMDB_API_KEY=your-tmdb-key \
  ghcr.io/davidfic/luminarr:latest
```

Open `http://localhost:8282`. Done.

On first run, Luminarr generates an API key and saves it to the `/config` volume. It persists across restarts automatically.

### Docker Compose

Create a `docker-compose.yml`:

```yaml
services:
  luminarr:
    image: ghcr.io/davidfic/luminarr:latest
    ports:
      - "8282:8282"
    environment:
      LUMINARR_TMDB_API_KEY: your-tmdb-key
      # Optional: set a fixed API key instead of auto-generating one
      # LUMINARR_AUTH_API_KEY: my-secret-key
    volumes:
      - luminarr-data:/config
      - /path/to/movies:/movies
    restart: unless-stopped

volumes:
  luminarr-data:
```

```bash
docker compose up -d
```

> **Port choice:** Luminarr uses 8282 so it can run alongside Radarr (7878) during migration.

### Build from source

Requires Go 1.22+ and Node.js 20+.

```bash
git clone https://github.com/davidfic/luminarr
cd luminarr
cd web/ui && npm install && npm run build && cd ../..
make build
./bin/luminarr
```

The binary is fully self-contained — it embeds the React frontend. Config defaults to `~/.config/luminarr/config.yaml` and the database to `~/.config/luminarr/luminarr.db`.

---

## Initial Setup

After starting Luminarr, open the UI at `http://localhost:8282` and configure four things:

### 1. Add a library

**Settings → Libraries → Add Library**

A library is a root folder where your movie files live. Each library maps to a directory on disk (e.g. `/movies`). You'll assign a quality profile and optionally set a minimum free space threshold.

### 2. Create a quality profile

**Settings → Quality Profiles → Add Profile**

Quality profiles define what you want. Unlike Radarr's Custom Formats, Luminarr has four explicit dimensions:

| Dimension | Examples |
|-----------|----------|
| Resolution | 720p, 1080p, 2160p |
| Source | WebDL, Bluray, Remux |
| Codec | x264, x265, AV1 |
| HDR | None, HDR10, Dolby Vision |

Pick a preset (e.g. "HD-1080p x265") or build a custom profile. Set a **cutoff** — the quality level where Luminarr stops looking for upgrades.

### 3. Add an indexer

**Settings → Indexers → Add Indexer**

Luminarr supports **Torznab** and **Newznab** protocols. If you use Prowlarr or Jackett, add each indexer with its URL and API key. Click **Test** to verify the connection.

### 4. Add a download client

**Settings → Download Clients → Add Client**

Supported clients:

- **qBittorrent** — host, port, username, password
- **Deluge** — host, port, password
- **Transmission** — URL, username, password
- **SABnzbd** — URL, API key, category
- **NZBGet** — URL, username, password, category

Click **Test** to verify. Luminarr polls the download client for progress and auto-imports completed downloads into your library.

---

## Adding Movies

Once setup is complete:

1. Go to the **Movies** page
2. Click **Add Movie**
3. Search by title — results come from TMDB
4. Pick a quality profile and library
5. Set **Minimum Availability** — Luminarr won't search for the movie until it reaches this release status:
   - **TBA** — search as soon as the movie exists in TMDB
   - **Announced** — search once production is announced
   - **In Cinemas** — search once theatrical release begins
   - **Released** (default) — search only after the movie has a home-media release
6. Choose whether to start monitoring immediately

Luminarr will search your indexers for available releases during the next RSS sync (every 15 minutes by default), or you can trigger a manual search from the movie detail panel.

---

## How the Grab Pipeline Works

1. **RSS sync** (every 15 min) or **manual search** finds releases on your indexers
2. Luminarr scores each release against your quality profile
3. The best matching release is sent to your download client
4. Luminarr polls the download client for progress (visible on the **Queue** page, with live WebSocket updates)
5. When the download completes, the **importer** moves or hardlinks the file into your library
6. If notifications are configured, you get alerts at each stage

If a grab fails, Luminarr adds it to the **blocklist** automatically so the same release isn't retried. You can review and clear the blocklist at **Settings → Blocklist**.

---

## Movie Detail Panel

Click any movie to open its detail panel. It has three tabs:

### Overview

Metadata, poster, cast, and the grab controls. The **Manual Search** button searches all your indexers immediately and shows results in a table — you can review quality, size, age, and seeds, then grab whichever release you want.

### Files

Lists every file attached to the movie. From here you can:
- **Delete** a file record from the database only, or delete it from disk as well
- **Rename** files to Luminarr's standard format: `Title (Year) Quality.ext` — preview changes before committing
- **View actual codec/HDR metadata** (if ffprobe is configured) — the "Actual" row shows what the container really contains, with a ⚠ Mismatch badge when the filename claims a different quality than the file
- **Re-scan** a file with ffprobe using the ↻ Scan button (visible when media scanning is available)

### History

A log of every grab and import event for this movie — release title, quality, indexer, timestamp, and outcome. Useful for diagnosing why a particular release was or wasn't grabbed.

---

## Wanted Page

**Wanted** in the sidebar shows movies that need attention:

- **Missing** — monitored movies with no file at all
- **Cutoff Unmet** — monitored movies where the existing file is below the quality cutoff in your profile

Both tabs have a **Search** button per movie that opens the Manual Search modal directly. Use the Wanted page as your daily work queue when the automation hasn't found something yet.

---

## Calendar

The **Calendar** page shows a monthly grid of movies by their release date. Colour coding tells you at a glance what needs attention:

- **Green** — movie has a file
- **Yellow** — monitored, no file yet
- **Grey** — unmonitored

Click any movie to open its detail panel. Use Prev/Next to navigate months.

---

## Migrating from Radarr

If you already run Radarr, you can import everything in one step.

1. Go to **Settings → Import**
2. Enter your Radarr URL (e.g. `http://localhost:7878`) and API key (found in Radarr → Settings → General → Security)
3. Click **Connect & Preview** — Luminarr shows what it found
4. Select categories to import and click **Import**

Luminarr imports in dependency order:
- Quality profiles (mapped to Luminarr's explicit codec/HDR format)
- Libraries (from Radarr root folders)
- Indexers (Torznab and Newznab only)
- Download clients (qBittorrent, Deluge, Transmission, SABnzbd, NZBGet)
- Movies (duplicates skipped by TMDB ID)

Radarr keeps running during import. Switch over when you're ready.

---

## Media Scanning with ffprobe (optional) {#ffprobe-optional}

Luminarr can verify the **actual** technical metadata of imported files using `ffprobe` — codec, resolution, HDR format, audio. It catches mislabelled releases: a file named `Movie.2160p.x265.HDR10.mkv` might actually contain `x264 SDR`.

### Install ffprobe

`ffprobe` is part of the `ffmpeg` package.

**Linux (Debian/Ubuntu):**
```bash
sudo apt install ffmpeg
```

**macOS (Homebrew):**
```bash
brew install ffmpeg
```

**Arch/Manjaro:**
```bash
sudo pacman -S ffmpeg
```

**Windows:** Not officially supported yet. Advanced users can set `LUMINARR_MEDIAINFO_FFPROBE_PATH` to a full path.

### Docker: use the `latest-full` image

The standard `latest` image is built from scratch and has no shell or extra binaries. To include ffprobe, use the `latest-full` variant:

```yaml
services:
  luminarr:
    image: ghcr.io/davidfic/luminarr:latest-full   # ← change this line
    ports:
      - "8282:8282"
    environment:
      LUMINARR_TMDB_API_KEY: your-tmdb-key
    volumes:
      - luminarr-data:/config
      - /path/to/movies:/movies
    restart: unless-stopped
```

The `latest-full` image is Alpine-based with `ffmpeg` included. It's larger (~80 MB vs ~20 MB) but requires zero extra setup.

### Verify it works

Go to **Settings → Media Scanning**. The status card shows either:
- ● **Available** (with the resolved ffprobe path)
- ○ **Unavailable** — ffprobe not found

Once available, Luminarr scans new imports automatically. For existing files, use the **Scan all unscanned files** button.

### Configuration

| Setting | Default | Env var | Description |
|---------|---------|---------|-------------|
| `mediainfo.ffprobe_path` | `` (search $PATH) | `LUMINARR_MEDIAINFO_FFPROBE_PATH` | Full path to ffprobe binary |
| `mediainfo.scan_timeout` | `30s` | `LUMINARR_MEDIAINFO_SCAN_TIMEOUT` | Per-file timeout |
| `mediainfo.scan_on_import` | `true` | `LUMINARR_MEDIAINFO_SCAN_ON_IMPORT` | Auto-scan after import |

---

## Notifications (optional)

**Settings → Notifications → Add Notification**

Supported channels: **Discord**, **Slack**, **Telegram**, **Gotify**, **ntfy**, **Pushover**, **Webhook**, **Email**, and **Command** (custom scripts). Each can subscribe to specific events:

- Grab started / failed
- Download complete
- Import complete / failed
- Health issue / resolved

The **Command** notifier executes scripts from `/config/scripts/` with the event payload on stdin. See [Custom Scripts](CUSTOM_SCRIPTS.md) for details.

---

## Media Server Integration (optional)

**Settings → Media Servers → Add Media Server**

Connect Luminarr to your media server so it can automatically refresh your library when movies are imported — no more waiting for scheduled scans.

Supported servers:

| Server | Required settings |
|--------|-------------------|
| **Plex** | Server URL, X-Plex-Token |
| **Emby** | Server URL, API key |
| **Jellyfin** | Server URL, API key |

Click **Test** to verify the connection. Once connected, Luminarr sends a library refresh to your media server every time it imports a movie file.

> **Finding your Plex token:** Open any media item in the Plex web app, click "Get Info" → "View XML", and look for `X-Plex-Token=` in the URL. Or see the [Plex support article](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/).

> **Self-signed certificates:** Luminarr accepts self-signed TLS certificates from media servers on your local network. No extra configuration needed.

---

## Library Sync

**Library Sync** (in the sidebar) lets you compare your media server's library against Luminarr's — in both directions.

### How to use it

1. Go to **Library Sync** in the sidebar
2. Select a configured media server from the dropdown
3. Pick a movie library section (Luminarr auto-loads them from your server)
4. Click **Compare**

Luminarr fetches every movie from that section, extracts TMDB IDs, and compares them against its own database. You get two views:

### Server Only

Movies your media server has that Luminarr doesn't track. These are candidates for import — maybe you added them outside of Luminarr, or they predate your Luminarr setup.

Select any (or all), choose a library and quality profile, and click **Import Selected**. Luminarr fetches metadata from TMDB and starts tracking each movie immediately.

### Luminarr Only

Movies Luminarr tracks that aren't on your media server. This is informational — it helps you spot movies that are still downloading, files that landed in the wrong directory, or imports that didn't trigger a library scan.

Each movie shows its current status (monitored, downloaded, missing) so you can quickly assess what needs attention.

### How matching works

Matching is by **TMDB ID only**. Luminarr reads the TMDB GUID that your media server stores for each movie (both new-style and legacy Plex agent formats are supported). Movies where the server doesn't have a TMDB ID are reported as "unmatched" and excluded from the comparison.

This approach is unambiguous — no false positives from movies that share a title or year.

---

## Configuration Reference

All settings can live in `config.yaml` or as environment variables (prefixed with `LUMINARR_`, dots become underscores).

| Setting | Default | Env var | Description |
|---------|---------|---------|-------------|
| `server.host` | `0.0.0.0` | `LUMINARR_SERVER_HOST` | Listen address |
| `server.port` | `8282` | `LUMINARR_SERVER_PORT` | HTTP port |
| `database.driver` | `sqlite` | `LUMINARR_DATABASE_DRIVER` | `sqlite` only |
| `database.path` | `~/.config/luminarr/luminarr.db` | `LUMINARR_DATABASE_PATH` | SQLite file path |
| `auth.api_key` | auto-generated | `LUMINARR_AUTH_API_KEY` | API key for all requests |
| `tmdb.api_key` | — | `LUMINARR_TMDB_API_KEY` | TMDB metadata key |
| `log.level` | `info` | `LUMINARR_LOG_LEVEL` | `debug`, `info`, `warn`, `error` |
| `log.format` | `json` | `LUMINARR_LOG_FORMAT` | `json` or `text` |
| `mediainfo.ffprobe_path` | `` | `LUMINARR_MEDIAINFO_FFPROBE_PATH` | Path to ffprobe binary; empty = search $PATH |
| `mediainfo.scan_timeout` | `30s` | `LUMINARR_MEDIAINFO_SCAN_TIMEOUT` | Per-file scan timeout |
| `mediainfo.scan_on_import` | `true` | `LUMINARR_MEDIAINFO_SCAN_ON_IMPORT` | Auto-scan imported files |

Config file search order:
1. `/config/config.yaml` (Docker volume mount)
2. `~/.config/luminarr/config.yaml`
3. `/etc/luminarr/config.yaml`
4. `./config.yaml`

A fully commented example is at [`config.example.yaml`](../config.example.yaml).

---

## API Key

Every Luminarr instance has its own randomly generated API key. This is intentional.

**Why one key per instance, not a shared key?**
Tools like Radarr ship with one API key that's visible in the settings UI — it's the same key for every client that talks to that instance, and you copy-paste it wherever needed. That works fine when you're the only user, but it means anyone who has ever seen the key has permanent access.

Luminarr takes the same one-key-per-instance approach, but removes the copy-paste step entirely: the key is generated once on first start and **baked directly into the HTML** that the browser receives. The browser stores it in memory and sends it with every API request automatically. You never see it, never manage it, never paste it anywhere — it just works.

The practical consequence: if you restart the container without a persistent key, the key changes and your open browser tabs get a 401 until you hard-refresh. Fix this with `LUMINARR_AUTH_API_KEY` or by mounting a `config.yaml`. See [Troubleshooting](#troubleshooting) below.

**Using the API from outside the browser** (scripts, Home Assistant, etc.): find your key in the container logs on startup (`api key: ...`), or set a fixed key via `LUMINARR_AUTH_API_KEY` and use that value in the `X-Api-Key` header.

Interactive OpenAPI docs are available at `/api/docs` when the server is running.

---

## Troubleshooting

### 401 errors after restarting Docker

The API key changed. Either:
- Hard-refresh the browser tab (Ctrl+Shift+R) to pick up the new key
- Set `LUMINARR_AUTH_API_KEY` in your Docker config so the key is stable across restarts

### "TMDB API key not configured" warning

Movie search and metadata are disabled. Set `LUMINARR_TMDB_API_KEY` via environment variable or `tmdb.api_key` in config.yaml.

### Download client connection fails

- Verify the host/port are reachable from the Luminarr container
- In Docker, use the host's IP or Docker network alias — not `localhost`
- Check that your download client's web UI is enabled and credentials are correct

### Indexer test fails

- Check the indexer URL includes the full API path (e.g. `http://prowlarr:9696/1/api`)
- Verify the API key matches what your indexer expects
- Ensure the indexer is reachable from the Luminarr container

---

## Next Steps

- Browse the [Architecture docs](ARCHITECTURE.md) for internals
- Check the [API docs](/api/docs) for automation
- Report bugs or request features on [GitHub](https://github.com/davidfic/luminarr/issues)
