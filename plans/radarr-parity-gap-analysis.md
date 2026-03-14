# Radarr Parity Gap Analysis

Generated 2026-03-12. Comprehensive comparison of Radarr features vs Luminarr's current state.

**Goal:** Luminarr is a modern, drop-in replacement for Radarr. This document identifies what's missing, what matters most, and what to build next.

---

## Feature Comparison Summary

| Category | Radarr | Luminarr | Gap |
|----------|--------|----------|-----|
| Quality Tiers | 31 | 14 | -17 (missing pre-release, 480p/576p variants) |
| Custom Formats | Full system (10 condition types, scoring) | None | **Critical gap** |
| Delay Profiles | Yes (protocol preference + timer) | No | Missing |
| Download Clients | 17 | 5 | -12 (missing rTorrent, Aria2, Flood, blackholes) |
| Indexer Types | 10 | 2 (Torznab/Newznab) | -8 (but Prowlarr covers most) |
| Notification Agents | 28 | 9 | -19 (missing many niche agents) |
| Notification Events | 13 | 7 | -6 |
| Import Lists | 18+ types | 0 | **Critical gap** |
| Metadata Consumers | 5 (NFO, Kometa, etc.) | 0 | Missing |
| Tags (full system) | Links to all entities | Library-level only | Incomplete |
| Auto-Tagging | 11 condition types | None | Missing |
| UI Settings | 10 options | 0 | Missing |
| iCal Feed | Yes | No | Missing |
| Propers/Repacks | Prefer & Upgrade setting | No differentiation | Missing |
| Hardlinks | Yes (copy using hardlinks) | No | Missing |
| Recycling Bin | Yes (with auto-cleanup) | No | Missing |
| File Permissions | chmod/chown on import | Auto-chmod config only | Partial |
| Custom Scripts | Full env var system | Command notification only | Partial |
| Proxy Support | HTTP/HTTPS/SOCKS4/SOCKS5 | None | Missing |
| URL Base | Yes (reverse proxy) | No | Missing |
| Logs UI | Full viewer + file download | Backend only, no frontend | Partial |
| Import Exclusions | Yes (TMDb ID blocklist) | No | Missing |
| Movie Collections (TMDb) | Franchise tracking + auto-add | Director/actor only | Partial |
| Alternative Titles | Yes | No | Missing |
| Saved/Custom Filters | Yes (per-user) | No | Missing |

---

## Tier 1 — Must Have (Core Parity)

These are features that experienced Radarr users will immediately notice are missing. Without them, Luminarr cannot credibly be called a Radarr replacement.

### 1. Custom Formats ★★★★★

**What it is:** Radarr's most powerful feature. A rule-based scoring system that replaces the old release profiles. Users define "custom formats" with conditions (regex on title, edition, language, source, resolution, release group, file size, year, indexer flags). Each format gets a score per quality profile. Releases are scored and the highest-scoring release wins.

**Why it matters:** This is how power users control quality. TRaSH Guides (the most popular Radarr resource) are built entirely around custom formats. Categories include:
- Audio formats (TrueHD ATMOS, DTS-X, DD+, etc.)
- HDR formats (HDR10+, Dolby Vision, SDR)
- Movie versions (Criterion, IMAX, Open Matte, Theatrical Cut)
- Unwanted (AV1, BR-DISK, LQ, 3D, upscaled)
- Streaming services (Amazon, Netflix, Disney+, etc.)
- HQ release groups (tiered by reputation)
- Miscellaneous (Repack/Proper, FreeLeech, Scene, Obfuscated)

**What Luminarr needs:**
- Custom format CRUD (name, conditions list, include_when_renaming)
- 10 condition types: release_title (regex), edition, language, indexer_flag, source, resolution, quality_modifier, size, release_group, year
- Negate + Required modifiers per condition
- Per-quality-profile scoring (score per format)
- Minimum CF score (reject below) and upgrade-until CF score (stop upgrading)
- JSON import/export (for TRaSH Guides compatibility)
- Integrate into release scoring pipeline (autosearch already scores — add CF dimension)
- Show CF matches in manual search results
- `{Custom Formats}` and `{Custom Format Score}` naming tokens

**Effort:** Large. This touches scoring, quality profiles, searching, naming, and needs a full UI. But it's THE differentiator for power users.

**Recommendation: BUILD THIS NEXT.** It's the single biggest gap. Without custom formats, advanced users won't switch.

---

### 2. Import Lists ★★★★★

**What it is:** Automatically add movies to your library from external sources. Radarr supports 18+ list types:
- **Trakt** — Watchlist, custom lists, trending, popular
- **TMDb** — Popular, upcoming, now playing, user lists, person filmography, company, keyword
- **Plex Watchlist** — Add movies from Plex watchlist
- **IMDb** — Public IMDb lists
- **Radarr** — Sync from another Radarr instance
- **RSS/Custom** — Generic RSS or JSON feeds
- **Simkl** — Simkl user lists

Each list has: search on add, default monitor type, minimum availability, quality profile, root folder, tags.

Global settings: list update interval, clean library level (disabled / log only / unmonitor / remove).

**Why it matters:** This is how most users discover and add movies. "Add my Trakt watchlist" and "auto-add trending movies" are the #1 and #2 most requested workflows. Plex Watchlist integration is also extremely popular (Overseerr alternative).

**What Luminarr needs:**
- Import list plugin system (similar to existing indexer/downloader/notification plugin pattern)
- Start with: Trakt (watchlist + lists), TMDb (popular + lists + person), Plex Watchlist, IMDb Lists
- Per-list settings: search_on_add, monitor, min_availability, quality_profile_id, root_folder, tags
- Global: update interval, clean library level
- Import exclusions (TMDb IDs to never auto-add)
- Scheduled job: ImportListSync (runs on interval)
- UI: Settings page for list CRUD + exclusions page

**Effort:** Medium-large. Plugin system already exists — this is a new plugin type. Start with Trakt + TMDb.

**Recommendation: BUILD.** Second highest priority after custom formats. This is table-stakes for a media manager.

---

### 3. Delay Profiles ★★★★☆

**What it is:** Wait N minutes before downloading a release, with protocol preference (prefer Usenet over Torrent or vice versa). Timer starts from when the release was uploaded (not discovered). Bypass if highest quality is found immediately.

Linked to movies via tags. Multiple profiles with priority ordering.

**Why it matters:** Users who have both Usenet and torrent indexers use this to prefer one protocol. Common setup: "Wait 120 minutes for Usenet before falling back to torrents." Also used to avoid grabbing pre-release/early uploads that may be low quality.

**What Luminarr needs:**
- delay_profiles table (preferred_protocol, usenet_delay, torrent_delay, bypass_if_highest, tags, order)
- CRUD API + UI
- Integration into grab pipeline: when a release is found, check delay profile before grabbing
- Pending queue state (release found but waiting for delay timer)

**Effort:** Medium. Requires changes to the grab pipeline.

**Recommendation: BUILD.** Important for dual-protocol users (Usenet + Torrent).

---

### 4. Tags (Full System) ★★★★☆

**What it is:** Radarr's tags link multiple entity types together. A tag can connect: movies, delay profiles, indexers, download clients, notifications, and import lists. When a movie has a tag, it uses matching-tag resources (plus untagged resources).

Example: Tag "4K" → links to specific indexer (4K-only tracker), specific download client (separate seedbox), specific quality profile, specific notification.

**Why it matters:** Tags are the glue that holds Radarr's advanced configuration together. Without proper tags, delay profiles, per-movie indexer assignment, and per-movie download client routing don't work.

**What Luminarr needs:**
- tags table (id, name)
- Tag CRUD API
- Add tag_ids to: movies, indexers, download_clients, notifications, delay_profiles, import_lists
- Tag matching logic: when processing a movie, filter resources by tag overlap
- UI: tag management, tag assignment on all relevant entities
- Bulk tag operations in movie editor

**Effort:** Medium. Schema changes + filtering logic across multiple services.

**Recommendation: BUILD.** Required foundation for delay profiles and advanced routing. Build before or alongside delay profiles.

---

### 5. More Quality Tiers ★★★☆☆

**What it is:** Radarr has 31 quality tiers. Luminarr has 14. Missing tiers:

**Pre-release (important for some users):**
- WORKPRINT, CAM, TELESYNC, TELECINE, DVDSCR, REGIONAL

**480p/576p variants (completeness):**
- WEBDL-480p, WEBRip-480p, Bluray-480p, Bluray-576p

**Other:**
- DVDR, BR-DISK, Raw-HD
- Separate WEBRip vs WEBDL at each resolution (Luminarr may already distinguish these)

**Why it matters:** Most users don't want pre-release quality, but they need to be in the system so the parser can identify and reject them. Without CAM/TELESYNC detection, a poorly-named CAM rip could slip through as "Unknown" quality.

**What Luminarr needs:**
- Add missing quality tiers to quality_definitions seed
- Update release parser to detect these qualities
- Quality grouping (Radarr lets you group qualities together in a profile)

**Effort:** Small-medium. Mostly seed data + parser updates.

**Recommendation: BUILD.** Low effort, important for correctness.

---

### 6. Notification Events (Complete Set) ★★★☆☆

**What it is:** Radarr has 13 notification events. Luminarr has 7. Missing events:

| Missing Event | Description | Importance |
|--------------|-------------|------------|
| On Upgrade | File replaced with higher quality | High — users want to know about upgrades |
| On Rename | Files renamed by rename job | Medium |
| On Movie Added | New movie added to library | High — useful for Discord bots |
| On Movie Delete | Movie removed | Medium |
| On Movie File Delete | File deleted (manual or upgrade) | Medium |
| On Application Update | Luminarr updated | Low |

**Effort:** Small. Event bus already exists — add new event types + wire up publishers.

**Recommendation: BUILD.** Low effort, high completeness value. "On Movie Added" and "On Upgrade" are the most requested.

---

### 7. Metadata/NFO Generation ★★★☆☆

**What it is:** Write metadata files alongside movie files for media players to read. Radarr supports:
- **Kodi/Emby (XBMC)** — NFO files (XML), poster/fanart images
- **Kometa** — Metadata for Plex Meta Manager
- **Roksbox / WDTV** — Legacy formats

**Why it matters:** Kodi users depend on NFO files. Without them, Kodi can't read movie metadata offline. This is a common migration blocker — "Radarr writes my NFO files, can Luminarr?"

**What Luminarr needs:**
- Metadata plugin system (similar to notification plugins)
- Start with Kodi/XBMC NFO format (most popular by far)
- Write .nfo file + download poster.jpg + fanart.jpg on import
- Option to clean up orphaned metadata images
- Settings: certification country, metadata consumers list

**Effort:** Medium. New plugin type, but well-defined output format.

**Recommendation: BUILD.** Important for Kodi users (large segment of the *arr community).

---

## Tier 2 — Should Have (Competitive Parity)

These make Luminarr feel complete. Not deal-breakers, but noticeable gaps.

### 8. Propers & Repacks Handling ★★★☆☆

**What it is:** Radarr can automatically upgrade to Proper/Repack releases (fixed versions of original releases). Three modes: Prefer and Upgrade, Do Not Upgrade Auto, Do Not Prefer.

**Why it matters:** When a release group fixes an encoding issue and puts out a "PROPER" or "REPACK", users want to automatically get the fix.

**What Luminarr needs:**
- Detect PROPER/REPACK in release title (parser may already do this)
- Setting: proper_repack_preference (prefer_and_upgrade / do_not_upgrade / do_not_prefer)
- Scoring adjustment: PROPER/REPACK of same quality scores higher than original

**Effort:** Small. Parser + scoring tweak.

**Recommendation: BUILD.** Low effort, good quality-of-life.

---

### 9. Hardlink Support ★★★☆☆

**What it is:** When importing a torrent that's still seeding, use a hardlink instead of copying. Saves disk space (file exists once on disk but appears in both locations).

**Why it matters:** Without hardlinks, users need 2x disk space during seeding. This is the #1 requested feature for torrent users.

**What Luminarr needs:**
- Setting: use_hardlinks (bool, default true)
- On import: attempt hardlink first, fall back to copy
- Only works on same filesystem (detect and warn)

**Effort:** Small. Single import path change.

**Recommendation: BUILD.** Very low effort, huge quality-of-life for torrent users.

---

### 10. Recycling Bin ★★★☆☆

**What it is:** Instead of permanently deleting files, move them to a configurable trash folder. Auto-cleanup after N days.

**Why it matters:** Safety net. Users accidentally delete movies and want them back. Radarr's recycling bin has saved many users.

**What Luminarr needs:**
- Settings: recycling_bin_path, recycling_bin_cleanup_days
- On file delete: move to recycling bin instead of permanent delete
- Scheduled job: clean up files older than N days

**Effort:** Small.

**Recommendation: BUILD.** Low effort, high safety value.

---

### 11. iCal Feed ★★★☆☆

**What it is:** `/feed/calendar/luminarr.ics` endpoint that external calendar apps can subscribe to. Shows upcoming releases.

**Why it matters:** Users subscribe in Google Calendar, Apple Calendar, etc. to see when movies release. Simple feature, frequently requested.

**What Luminarr needs:**
- Single endpoint that generates .ics format
- Filter by: in_cinemas, physical_release, digital_release dates
- Include movie title, overview, poster URL in event

**Effort:** Small. Single endpoint, well-defined format.

**Recommendation: BUILD.** Trivial effort, nice feature.

---

### 12. Saved/Custom Filters ★★★☆☆

**What it is:** Save filter combinations in the movie library view. Radarr has predefined filters (Monitored, Unmonitored, Missing, Wanted, Cutoff Unmet) plus user-defined custom filters on any field.

**Why it matters:** Power users with large libraries need quick access to filtered views ("Show me all 4K movies missing", "Show me movies added this month").

**What Luminarr needs:**
- custom_filters table (name, type=movie, conditions JSON)
- API: CRUD for saved filters
- UI: filter builder + saved filter dropdown in movie list

**Effort:** Medium.

**Recommendation: BUILD.** Good power-user feature, differentiator for "more modern" claim.

---

### 13. TMDb Collections (Franchise) ★★★☆☆

**What it is:** Radarr tracks TMDb collections (franchises like "The Lord of the Rings", "Marvel Cinematic Universe"). Users can monitor a collection and auto-add new movies when announced.

Luminarr has director/actor collections but NOT TMDb franchise collections.

**Why it matters:** "Monitor the MCU and auto-add new movies" is a very common workflow. This is distinct from import lists — it's a first-class concept in Radarr's UI.

**What Luminarr needs:**
- Fetch TMDb collection data (already available from TMDB API)
- collection tracking: monitored flag, quality profile, root folder
- Auto-add new movies when collection membership changes (via metadata refresh)
- UI: collection browse/detail page with franchise poster

**Effort:** Medium. TMDb data already available, needs new entity + auto-add logic.

**Recommendation: BUILD.** Natural extension of existing collection work.

---

### 14. UI Settings ★★★☆☆

**What it is:** Date format, time format, first day of week, runtime format, movie info language, UI language, theme selection, color-impaired mode.

**Why it matters:** International users need date/time customization. Theme is already partially there. These are table-stakes for a polished app.

**What Luminarr needs:**
- ui_settings table or localStorage-based preferences
- Settings: date_format, time_format, first_day_of_week, runtime_format, movie_info_language
- Apply formats across all pages

**Effort:** Medium (mostly frontend work).

**Recommendation: BUILD.** Polish feature, supports "more modern" positioning.

---

### 15. Logs UI ★★☆☆☆

**What it is:** View application logs in the web UI. Radarr has: events page (INFO+ logs), log files list with download, different verbosity levels.

Luminarr has the backend endpoint (`GET /api/v1/logs`) but no frontend page.

**What Luminarr needs:**
- Frontend page: LogsPage.tsx
- Log level filtering, search, auto-refresh
- Log file download (for support/debugging)

**Effort:** Small-medium (frontend only, backend exists).

**Recommendation: BUILD.** Already have the backend, just need the UI.

---

### 16. URL Base (Reverse Proxy) ★★☆☆☆

**What it is:** Serve Luminarr under a path prefix like `/luminarr` for reverse proxy setups (e.g., `https://myserver.com/luminarr`).

**Why it matters:** Many users run multiple *arr apps behind a single reverse proxy. Without URL base support, they need subdomain-per-app.

**What Luminarr needs:**
- Config: `server.url_base` (e.g., "/luminarr")
- Prefix all API routes and static asset paths
- Update frontend to use base path for all requests

**Effort:** Medium. Touches router + frontend asset paths.

**Recommendation: BUILD.** Common deployment pattern.

---

### 17. More Download Clients ★★☆☆☆

**What it is:** Radarr supports 17 clients. Luminarr has 5 (qBittorrent, Deluge, Transmission, NZBGet, SABnzbd). Missing popular ones:

| Client | Protocol | Popularity | Priority |
|--------|----------|-----------|----------|
| rTorrent/ruTorrent | Torrent | High (seedbox users) | High |
| Aria2 | Torrent | Medium | Medium |
| Flood | Torrent | Medium (modern rTorrent UI) | Medium |
| Torrent Blackhole | Torrent | Medium (watch folder) | Medium |
| Usenet Blackhole | Usenet | Low | Low |
| Download Station | Torrent | Low (Synology only) | Low |
| Others (Vuze, Freebox, Hadouken, NZBVortex, Pneumatic, uTorrent) | Mixed | Low | Skip |

**Recommendation: ADD rTorrent + Blackholes.** rTorrent is heavily used by seedbox users. Blackhole (watch folder) is a universal fallback that works with any client.

---

### 18. Alternative Titles ★★☆☆☆

**What it is:** Track movies by alternate names (translations, regional titles). Helps with matching releases that use non-English titles.

**Why it matters:** International releases often use local titles. Without alt titles, these releases won't match.

**What Luminarr needs:**
- Fetch alt titles from TMDb (already available in API)
- Store in alt_titles table
- Include in release matching logic

**Effort:** Small-medium.

**Recommendation: BUILD.** Improves matching accuracy for international users.

---

## Tier 3 — Nice to Have (Completeness)

Lower priority. Build when core is solid.

### 19. Proxy Support ★★☆☆☆
HTTP/HTTPS/SOCKS proxy for all outbound connections. Useful for users behind corporate firewalls or for privacy. Medium effort.

### 20. Auto-Tagging ★★☆☆☆
Automatically apply tags based on rules (genre, year, studio, etc.). Requires full tag system first. Medium effort.

### 21. More Notification Agents ★☆☆☆☆
Radarr has 28, Luminarr has 9. Most missing agents are niche (Boxcar, Join, Prowl, Mailgun, SendGrid, Signal, Simplepush, Pushsafer, Pushcut, Twitter, Synology Indexer, Trakt scrobble). Add on-demand when users request specific ones.

### 22. More Indexer Types ★☆☆☆☆
Radarr has native support for FileList, HDBits, IPTorrents, Nyaa, PassThePopcorn, TorrentPotato, Torrent RSS. However, **Prowlarr covers all of these via Torznab/Newznab**. Only add native support if specific API features are needed. Nyaa (anime) is the most commonly requested.

### 23. Custom Scripts (Full Env Vars) ★☆☆☆☆
Radarr passes 20+ environment variables to custom scripts on each event. Luminarr's Command notification exists but may not pass the same variables. Parity here matters for automation-heavy users.

### 24. File Date Modification ★☆☆☆☆
Change file modified date to in-cinemas or physical release date. Niche but some users depend on it for media player sorting.

### 25. PostgreSQL Log Database ★☆☆☆☆
Separate log database (Radarr has radarr.db + logs.db). Luminarr uses single SQLite. Only relevant at scale.

### 26. SSL/TLS Built-in ★☆☆☆☆
Built-in HTTPS with certificate configuration. Most users use a reverse proxy instead. Low priority.

### 27. Restart/Shutdown API ★☆☆☆☆
Remote restart/shutdown endpoints. Niche, usually handled by systemd/Docker.

### 28. Quality Grouping ★★☆☆☆
Radarr lets you group qualities together in a profile (e.g., treat "WEBDL-1080p" and "WEBRip-1080p" as equivalent). Luminarr's profiles are ordered lists but may not support grouping.

### 29. Empty Folder Management ★☆☆☆☆
Create empty movie folders on add, delete empty folders on remove. Minor convenience.

### 30. Indexer Flags ★★☆☆☆
Freeleech, half-leech, double-upload, PTP Golden/Approved, internal, scene. Used in custom format conditions. Build alongside custom formats.

---

## Recommended Build Order

Based on impact, effort, and dependencies:

### Phase A — Custom Formats + Tags Foundation
1. **Tags (full system)** — foundation for everything else
2. **Custom Formats** — the killer feature
3. **Indexer Flags** — feeds into custom formats
4. **Quality tiers expansion** — 14 → 31 tiers

### Phase B — Import Lists + Discovery
5. **Import Lists** (Trakt, TMDb, Plex Watchlist, IMDb)
6. **Import Exclusions**
7. **TMDb Collections (franchise tracking)**

### Phase C — Download Pipeline Polish
8. **Delay Profiles** — requires tags
9. **Propers/Repacks handling**
10. **Hardlink support**
11. **More download clients** (rTorrent, Blackholes)

### Phase D — Media Management Polish
12. **Metadata/NFO generation** (Kodi format)
13. **Recycling Bin**
14. **Alternative Titles**
15. **Notification events expansion**

### Phase E — UI & Settings Completeness
16. **UI Settings** (date/time format, language)
17. **Logs UI page**
18. **Saved/Custom Filters**
19. **iCal Feed**
20. **URL Base**

### Phase F — Long Tail
21. Auto-tagging
22. Proxy support
23. Additional notification agents (on demand)
24. Additional indexer types (on demand)
25. Custom script env vars
26. Quality grouping

---

## What NOT to Build

These Radarr features are either obsolete, niche, or better handled differently:

| Feature | Reason to Skip |
|---------|---------------|
| Release Profiles | Deprecated in Radarr v5, replaced by Custom Formats |
| uTorrent support | Radarr itself discourages it (adware) |
| Pneumatic / NZBVortex | Very niche Usenet clients |
| Roksbox / WDTV metadata | Legacy formats, nearly unused |
| Analytics/telemetry | Privacy-first approach is a selling point |
| .NET version checks | Not applicable (Go) |
| SignalR | Using WebSocket instead (already done) |
| Emby Legacy metadata | Superseded by Kodi/XBMC format |

---

## Summary

**Luminarr today covers ~65-70% of Radarr's feature surface.** The core workflow (add movie → search → grab → import → monitor) is solid. The biggest gaps are:

1. **Custom Formats** — THE power-user feature. Without it, advanced users won't switch.
2. **Import Lists** — THE discovery feature. Without it, users can't auto-populate libraries.
3. **Tags + Delay Profiles** — THE routing feature. Without it, multi-protocol setups are limited.
4. **Metadata/NFO** — THE Kodi feature. Blocks migration for a significant user segment.

Building Phases A + B would bring Luminarr to ~85-90% parity and make it viable as a real Radarr replacement for most users.
