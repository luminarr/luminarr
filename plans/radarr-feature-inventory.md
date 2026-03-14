# Radarr Feature Inventory — Comprehensive Gap Analysis Reference

Generated 2026-03-12 from Radarr wiki, source code, OpenAPI spec, and TRaSH Guides.

---

## 1. Media Management

### 1.1 Movie Naming
| Setting | Values/Options | Notes |
|---------|---------------|-------|
| Rename Movies | bool (default: false) | If disabled, uses existing/client-provided names |
| Replace Illegal Characters | bool (default: true) | Characters: `: \ / > < ? * \| "` |
| Colon Replacement Format | Delete, Dash, Space, Space-Dash-Space, Smart (default) | Controls how `:` is replaced |
| Standard Movie Format | Token string | Default: `{Movie Title} ({Release Year}) {Quality Full}` |
| Movie Folder Format | Token string | Default: `{Movie Title} ({Release Year})` |

#### Naming Tokens Available
- `{Movie Title}`, `{Movie CleanTitle}`, `{Movie TitleThe}`, `{Movie OriginalTitle}`
- `{Movie Collection}`, `{Movie Certification}`
- `{Release Year}`
- `{ImdbId}`, `{TmdbId}`
- `{Quality Full}`, `{Quality Title}`
- `{MediaInfo AudioCodec}`, `{MediaInfo AudioChannels}`, `{MediaInfo VideoCodec}`
- `{MediaInfo VideoDynamicRangeType}`, `{MediaInfo 3D}`
- `{Release Group}`
- `{Edition Tags}`
- `{Custom Formats}`
- `{Original Title}`, `{Original Filename}`

### 1.2 Folders
| Setting | Default | Description |
|---------|---------|-------------|
| Create Empty Media Folders | false | Auto-create movie folders during disk scan |
| Delete Empty Folders | false | Remove empty folders when media deleted |

### 1.3 Importing
| Setting | Default | Description |
|---------|---------|-------------|
| Skip Free Space Check | false | Bypass disk space detection |
| Minimum Free Space When Importing | 100 MB | Prevent import if insufficient space |
| Copy Using Hardlinks | true | Use hardlinks for torrents still seeding |
| Import Extra Files | false | Include subtitles, NFO after import |
| Extra File Extensions | "srt" | Extensions to import alongside media |
| Use Script Import | false | Custom import script handling |
| Script Import Path | null | Path to custom import script |

### 1.4 File Management
| Setting | Default | Description |
|---------|---------|-------------|
| Auto Unmonitor Previously Downloaded Movies | false | Unmonitor when files removed |
| Download Propers & Repacks | PreferAndUpgrade | Options: Prefer and Upgrade, Do Not Upgrade Auto, Do Not Prefer |
| Enable MediaInfo (Analyse Video Files) | true | Extract resolution, runtime, codec data |
| Rescan Movie Folder After Refresh | Always | Options: Always, After Manual Refresh, Never |
| Change File Date | None | Options: None, In Cinemas Date, Physical Release Date |
| Recycling Bin | "" | Trash location path |
| Recycling Bin Cleanup Days | 7 | Days before permanent deletion |

### 1.5 Permissions (Linux)
| Setting | Default | Description |
|---------|---------|-------------|
| Set Permissions | false | Enable chmod on import/rename |
| chmod Folder | "755" | Octal folder permissions |
| chown Group | "" | Group name or GID |

### 1.6 Root Folders
- Path to media library
- Free space reporting
- Unmapped folders detection
- Add/remove root folders

---

## 2. Quality Profiles

### 2.1 Quality Profile Fields
| Field | Description |
|-------|-------------|
| Name | Unique identifier (required) |
| Upgrades Allowed | bool — toggle automatic quality upgrades |
| Upgrade Until Quality | Target quality threshold (cutoff) |
| Upgrade Until Custom Format Score | Stop upgrading at this CF score |
| Minimum Custom Format Score | Reject releases below this CF score |
| Qualities | Ordered list with grouping; checked items are allowed |
| Language | Preferred language (Original, specific, or Any) |
| Custom Format Scores | Per-format scoring within the profile |

### 2.2 Quality Tiers (31 total)
| ID | Name | Source | Resolution |
|----|------|--------|-----------|
| 0 | Unknown | — | — |
| 24 | WORKPRINT | — | — |
| 25 | CAM | — | — |
| 26 | TELESYNC | — | — |
| 27 | TELECINE | — | — |
| 28 | DVDSCR | DVD | 480p |
| 29 | REGIONAL | DVD | 480p |
| 1 | SDTV | TV | 480p |
| 2 | DVD | DVD | — |
| 23 | DVDR | DVD | 480p |
| 8 | WEBDL-480p | Web-DL | 480p |
| 12 | WEBRip-480p | Web-Rip | 480p |
| 20 | Bluray-480p | Blu-ray | 480p |
| 21 | Bluray-576p | Blu-ray | 576p |
| 4 | HDTV-720p | TV | 720p |
| 5 | WEBDL-720p | Web-DL | 720p |
| 14 | WEBRip-720p | Web-Rip | 720p |
| 6 | Bluray-720p | Blu-ray | 720p |
| 9 | HDTV-1080p | TV | 1080p |
| 3 | WEBDL-1080p | Web-DL | 1080p |
| 15 | WEBRip-1080p | Web-Rip | 1080p |
| 7 | Bluray-1080p | Blu-ray | 1080p |
| 30 | Remux-1080p | Blu-ray | 1080p |
| 16 | HDTV-2160p | TV | 2160p |
| 18 | WEBDL-2160p | Web-DL | 2160p |
| 17 | WEBRip-2160p | Web-Rip | 2160p |
| 19 | Bluray-2160p | Blu-ray | 2160p |
| 31 | Remux-2160p | Blu-ray | 2160p |
| 22 | BR-DISK | Blu-ray | 1080p |
| 10 | Raw-HD | TV | 1080p |

### 2.3 Quality Definitions
Per quality tier:
| Field | Type | Description |
|-------|------|-------------|
| Title | string | Display name (editable) |
| GroupName | string | For grouping qualities |
| Weight | int | Ordering/ranking |
| MinSize | double? | Min MB per minute of runtime |
| MaxSize | double? | Max MB per minute of runtime |
| PreferredSize | double? | Preferred MB per minute |

### 2.4 Delay Profiles
| Field | Description |
|-------|-------------|
| Preferred Protocol | Usenet or Torrent |
| Usenet Delay | Minutes before download |
| Torrent Delay | Minutes before download |
| Bypass if Highest Quality | Skip delay for top-tier releases |
| Tags | Link profiles to specific movies |
| Order | Priority ordering (reorderable) |

Timer begins from release upload timestamp, not discovery time.

### 2.5 Release Profiles (deprecated in v5, replaced by Custom Formats)
| Field | Description |
|-------|-------------|
| Must Contain | Regex/text requirement |
| Must Not Contain | Regex/text exclusion |
| Tags | Apply to tagged movies only |

---

## 3. Custom Formats

### 3.1 Condition Types (from source code)
| Type | File | Description |
|------|------|-------------|
| Release Title | ReleaseTitleSpecification.cs | Regex match on release title |
| Edition | EditionSpecification.cs | Match edition tags |
| Language | LanguageSpecification.cs | Match audio language |
| Indexer Flag | IndexerFlagSpecification.cs | Match indexer flags (freeleech etc.) |
| Source | SourceSpecification.cs | Match source type (TV, Web-DL, Blu-ray, etc.) |
| Resolution | ResolutionSpecification.cs | Match resolution (480p, 720p, 1080p, 2160p) |
| Quality Modifier | QualityModifierSpecification.cs | Match quality modifiers (Remux, etc.) |
| Size | SizeSpecification.cs | Match file size range (GB) |
| Release Group | ReleaseGroupSpecification.cs | Match release group name |
| Year | YearSpecification.cs | Match release year |

### 3.2 Condition Modifiers
- **Negate** — invert match logic
- **Required** — must satisfy when multiple conditions of same type exist

### 3.3 Scoring System
- Each custom format gets a score per quality profile
- **Minimum Custom Format Score** — reject releases below threshold
- **Upgrade Until Custom Format Score** — stop upgrading at this point
- Score of 0 = informational only (no preference effect)
- Supports JSON import/export for sharing

### 3.4 Custom Format Categories (from TRaSH Guides — community standard)

**Audio Formats (16):**
TrueHD ATMOS, DTS X, ATMOS (undefined), DD+ ATMOS, TrueHD, DTS-HD MA, FLAC, PCM, DTS-HD HRA, DD+, DTS-ES, DTS, AAC, DD, MP3, Opus

**Audio Channels (7):**
1.0 Mono, 2.0 Stereo, 3.0, 4.0, 5.1, 6.1, 7.1

**HDR Formats (3+):**
HDR, DV Boost, HDR10+ Boost, DV (Disk), DV (w/o HDR fallback), SDR, SDR (no WEBDL)

**Movie Versions (11):**
Hybrid, Remaster, 4K Remaster, Criterion Collection, Masters of Cinema, Vinegar Syndrome, Theatrical Cut, Special Edition, IMAX, IMAX Enhanced, Open Matte

**Unwanted (10):**
AV1, BR-DISK, Generated Dynamic HDR, LQ, LQ (Release Title), Sing-Along, 3D, x265 (HD), Upscaled, Extras

**Streaming Services (30+):**
Amazon, Apple TV+, Disney+, HBO Max, Hulu, Netflix, Paramount+, Peacock, etc.

**HQ Release Groups:**
Remux Tier 01-03, UHD Bluray Tier 01-03, HD Bluray Tier 01-03, WEB Tier 01-03

**Miscellaneous (20+):**
Resolution tags, Bad Dual Groups, No-RlsGroup, Obfuscated, Retags, Scene, codecs (x264/x265/x266/VC-1/VP9/MPEG2), Repack/Proper, FreeLeech, HFR, Multi, etc.

---

## 4. Indexers

### 4.1 Indexer Types
| Type | Protocol | Description |
|------|----------|-------------|
| Newznab | Usenet | Standard Usenet API |
| omgwtfnzbs | Usenet | Deprecated private indexer |
| Torznab | Torrent | Newznab-compatible torrent API (Prowlarr/Jackett) |
| FileList | Torrent | Private tracker |
| HDBits | Torrent | Private tracker |
| IP Torrents | Torrent | Private tracker |
| Nyaa | Torrent | Anime tracker |
| Pass The Popcorn | Torrent | Private tracker |
| Torrent RSS Feed | Torrent | Generic RSS parser |
| TorrentPotato | Torrent | Legacy CouchPotato format |

### 4.2 Per-Indexer Settings
| Setting | Description |
|---------|-------------|
| Name | Identifier |
| Enable RSS | Monitor for missing/wanted |
| Enable Automatic Search | Use in auto-search |
| Enable Interactive Search | Include in manual search |
| URL | API endpoint |
| API Path | Typically `/api` |
| Multi Languages | Define MULTI language set |
| API Key | Authentication |
| Categories | Query categories |
| Additional Parameters | Extra Newznab params (Advanced) |
| Remove Year from Search | Strip year for text queries |
| Indexer Priority | 1-50 (1=highest) |
| Download Client | Assign specific client |
| Tags | Restrict to tagged movies |

**Torrent-specific additions:**
| Setting | Description |
|---------|-------------|
| Minimum Seeders | Min peer requirement (default: 1) |
| Seed Ratio | Min ratio before pause/removal |
| Seed Time | Min seeding duration (minutes) |
| Required Flags | Mandatory indexer flags |

### 4.3 Indexer Flags
| Flag | Value | Description |
|------|-------|-------------|
| G_Freeleech | 1 | Download doesn't count |
| G_Halfleech | 2 | Download counts 50% |
| G_DoubleUpload | 4 | Upload doubled |
| PTP_Golden | 8 | PTP staff-designated HQ encode |
| PTP_Approved | 16 | PTP staff/checker verified |
| G_Internal | 32 | Internal release group |
| G_Scene | 128 | Scene release |
| G_Freeleech75 | 256 | Download counts 75% |
| G_Freeleech25 | 512 | Download counts 25% |
| Nuked | 2048 | Release is nuked |

### 4.4 Global Indexer Options
| Setting | Default | Description |
|---------|---------|-------------|
| Minimum Age | 0 | Usenet delay minutes before grab |
| Retention | 0 | Usenet retention (0=unlimited) |
| Maximum Size | 0 | Max download size (0=unlimited) |
| Prefer Indexer Flags | false | Prioritize special flag releases |
| Availability Delay | 0 | Days before/after release to search |
| RSS Sync Interval | 30 | Minutes (10-120; 0=disabled) |
| Whitelisted Subtitle Tags | "" | Exclude hardcoded sub detection |
| Allow Hardcoded Subs | false | Permit hardcoded subtitles |

---

## 5. Download Clients

### 5.1 Supported Clients (17)
**Torrent:**
- Aria2
- Deluge
- Download Station (Synology)
- Flood
- Freebox Download
- Hadouken
- qBittorrent
- rTorrent
- Transmission
- uTorrent (discouraged — adware)
- Vuze
- Torrent Blackhole

**Usenet:**
- NZBGet
- NZBVortex
- Pneumatic
- SABnzbd
- Usenet Blackhole

### 5.2 Per-Client Settings (qBittorrent example — most popular)
| Field | Default | Description |
|-------|---------|-------------|
| Host | "localhost" | Client address |
| Port | 8080 | Connection port |
| Use SSL | false | HTTPS connection |
| URL Base | "" | Reverse proxy prefix (Advanced) |
| Username | "" | Authentication |
| Password | "" | Authentication |
| Category | "radarr" | Label for downloads |
| Post-Import Category | "" | Category after import (Advanced) |
| Recent Priority | normal | Priority for new releases |
| Older Priority | normal | Priority for older releases |
| Initial State | Start | Start, Force Start, Pause |
| Sequential Order | false | Download pieces sequentially |
| First and Last First | false | Prioritize first/last pieces |
| Content Layout | Default | Default, Original, Subfolder |

### 5.3 Seed Settings (per indexer)
| Client | Seed Ratio | Seed Time |
|--------|-----------|-----------|
| Aria2 | Yes | No |
| Deluge | Yes | No |
| Flood | Yes | Yes |
| qBittorrent | Yes | Yes |
| rTorrent | Yes | Yes |
| Transmission | Yes | Yes (Idle Limit) |
| uTorrent | Yes | Yes |
| Vuze | Yes | Yes |

### 5.4 Global Download Client Settings
| Setting | Default | Description |
|---------|---------|-------------|
| Enable Completed Download Handling | true | Auto-import from client |
| Check for Finished Download Interval | 1 | Query frequency (minutes, min 1) |
| Remove Completed Downloads | varies | Clear after import |
| Auto Redownload Failed | true | Re-search on failure |
| Auto Redownload Failed From Interactive Search | true | Re-search after manual search failure |
| Download Client Working Folders | "_UNPACK_\|_FAILED_" | In-progress folder patterns |
| Download Client History Limit | 60 | History retention days |

### 5.5 Remote Path Mappings
Maps remote paths to local equivalents for Docker/network setups. Fields:
- Host
- Remote Path
- Local Path

---

## 6. Import Lists

### 6.1 Supported List Types (18+)
| Type | Description |
|------|-------------|
| CouchPotato | Legacy list import |
| Custom Lists | Radarr native |
| IMDb Lists | Public lists, requires list ID |
| Plex Watchlist | Per-user auth, v4.1+ |
| Radarr | Instance-to-instance sync |
| RSS List | XML format with title/year parsing |
| StevenLu Custom | JSON format (title/imdb_id) |
| StevenLu List | Predefined list variant |
| TMDb Collection | Collection-based import |
| TMDb Company | Studio-based lists |
| TMDb Keyword | Tag-based lists |
| TMDb List | User-curated TMDb lists |
| TMDb Person | Actor/director filmography |
| TMDb Popular | Trending/top-rated/upcoming |
| TMDb User | User ratings/watchlists |
| Trakt List | Username/list imports |
| Trakt Popular | Community trending |
| Trakt User | Personal watchlist |
| Simkl User | Simkl user lists |

### 6.2 Per-List Settings
| Setting | Description |
|---------|-------------|
| Search on Add | Auto-search when list adds movies |
| Monitor | Monitoring type for imported movies |
| Minimum Availability | Default availability for list imports |
| Quality Profile | Default profile for list imports |
| Root Folder | Default root folder |
| Tags | Tags to apply to imported movies |

### 6.3 Global Import List Settings
| Setting | Default | Description |
|---------|---------|-------------|
| List Update Interval | 24 hours | Poll frequency (Advanced) |
| Clean Library Level | Disabled | Options: Disabled, Log Only, Keep and Unmonitor, Remove and Keep Files, Remove and Delete Files |

### 6.4 Import Exclusions
- Prevent specific movies from re-importing by TMDb ID
- Paged list with bulk add/delete
- Can restore excluded movies to lists

---

## 7. Notifications/Connections

### 7.1 Supported Notification Agents (28)
| Agent | Key Settings |
|-------|-------------|
| Apprise | URL |
| Boxcar | Access Token |
| Custom Script | Path, arguments |
| Discord | Webhook URL, Username, Avatar, Author, configurable Grab/Import/ManualInteraction fields |
| Email | Server, Port, SSL, From/To, Username/Password |
| Emby/Jellyfin | Host, API Key |
| Gotify | Server, App Token, Priority |
| Join | API Key, Device IDs |
| Kodi/XBMC | Host, Port, Username, Password |
| Mailgun | API Key, Domain, From, Recipients |
| Notifiarr | API Key |
| Ntfy | Server URL, Topic, Priority, Username/Password, Access Token |
| Plex Media Server | Host, Auth Token |
| Prowl | API Key, Priority |
| PushBullet | API Key, Device IDs, Channel Tag |
| Pushcut | Webhook URL |
| Pushover | User Key, API Key, Priority, Sound, Devices |
| Pushsafer | API Key |
| SendGrid | API Key, From, Recipients |
| Signal | Host, Port, Sender Number, Receiver IDs |
| Simplepush | Event Token |
| Slack | Webhook URL, Username, Icon, Channel |
| Synology Indexer | — |
| Telegram | Bot Token, Chat ID, Topic ID, Send Silently, Include App Name, Metadata Links |
| Trakt | Auth Token |
| Twitter | Consumer Key/Secret, Access Token/Secret |
| Webhook | URL, Method (POST/PUT), Username, Password, Custom Headers |

### 7.2 Notification Event Triggers (13)
| Event | Description |
|-------|-------------|
| On Grab | Release grabbed from indexer |
| On Download/Import | Movie imported |
| On Upgrade | Existing file upgraded |
| On Rename | Movie files renamed |
| On Movie Added | New movie added to library |
| On Movie Delete | Movie removed from library |
| On Movie File Delete | Movie file deleted |
| On Movie File Delete For Upgrade | File deleted because upgrade available |
| On Health Issue | Health check failure |
| Include Health Warnings | Include warning-level health notifications |
| On Health Restored | Previous health issue resolved |
| On Application Update | App updated |
| On Manual Interaction Required | Manual user action needed |

### 7.3 Discord Configurable Embed Fields
**Grab fields:** Overview, Rating, Genres, Quality, Group, Size, Links, Release, Poster, Fanart, Indexer, CustomFormats, CustomFormatScore, Tags

**Import fields:** Overview, Rating, Genres, Quality, Codecs, Group, Size, Languages, Subtitles, Links, Release, Poster, Fanart, Tags, CustomFormats, CustomFormatScore

**Manual Interaction fields:** Overview, Rating, Genres, Quality, Group, Size, Links, DownloadTitle, Poster, Fanart, Tags

---

## 8. Metadata

### 8.1 Global Settings
| Setting | Default | Description |
|---------|---------|-------------|
| Certification Country | US | Region for film ratings (R, PG-13, 12A, etc.) |
| Clean Up Metadata Images | true | Remove unused artwork |

### 8.2 Metadata Consumers (5)
| Consumer | Description |
|----------|-------------|
| Kodi/Emby (XBMC) | NFO files, images, collection names, configurable language |
| Emby (Legacy) | NFO file generation, movie-specific metadata |
| Kometa | Metadata for Kometa (formerly Plex Meta Manager) |
| Roksbox | XML metadata, JPG poster images |
| WDTV | XML metadata, folder artwork |

---

## 9. Tags

### 9.1 Tag System
- Tags link: Delay Profiles, Restrictions/Release Profiles, Indexers, Download Clients, Notifications, Import Lists, and Movies
- Movies use both matching-tag resources AND no-tag (untagged) resources
- Tags do NOT influence Custom Formats or Quality Profiles directly
- CRUD operations: create, read, update, delete, bulk delete

### 9.2 Auto-Tagging (newer feature)
Automatically apply tags based on conditions:
| Condition Type | Source File | Description |
|---------------|------------|-------------|
| Genre | GenreSpecification.cs | Match movie genre |
| Keyword | KeywordSpecification.cs | Match movie keywords |
| Monitored | MonitoredSpecification.cs | Match monitoring status |
| Original Language | OriginalLanguageSpecification.cs | Match original language |
| Quality Profile | QualityProfileSpecification.cs | Match assigned quality profile |
| Root Folder | RootFolderSpecification.cs | Match root folder path |
| Runtime | RuntimeSpecification.cs | Match movie runtime |
| Status | StatusSpecification.cs | Match movie status (TBA/Announced/InCinemas/Released) |
| Studio | StudioSpecification.cs | Match production studio |
| Tag | TagSpecification.cs | Match existing tags |
| Year | YearSpecification.cs | Match release year |

---

## 10. General Settings

### 10.1 Host/Binding
| Setting | Default | Description |
|---------|---------|-------------|
| Bind Address | "*" (all interfaces) | Options: *, 0.0.0.0, 127.0.0.1, specific IP |
| Port | 7878 | HTTP port |
| SSL Port | 9898 | HTTPS port |
| URL Base | "" | Reverse proxy path (e.g. `/radarr`) |
| Enable SSL | false | HTTPS with certificate |
| SSL Cert Path | "" | Certificate file |
| SSL Cert Password | "" | Certificate password |
| Instance Name | "Radarr" | Application display name |
| Application URL | "" | Base URL for links |

### 10.2 Security/Authentication
| Setting | Default | Description |
|---------|---------|-------------|
| Authentication Method | None (v5: mandatory) | None (deprecated), Basic (popup), Forms (login page), External |
| Authentication Required | Enabled | Enabled, Disabled for Local Addresses |
| Trust CGNAT IP Addresses | false | Trust carrier-grade NAT addresses as local |
| API Key | Auto-generated | For program-to-program communication |
| Certificate Validation | Enabled | Enabled, Disabled for Local, Disabled |

### 10.3 Proxy
| Setting | Default | Description |
|---------|---------|-------------|
| Use Proxy | false | Enable proxy routing |
| Proxy Type | HTTP | HTTP, HTTPS, Socks4, Socks5 |
| Hostname | "" | Proxy address (no protocol prefix) |
| Port | 8080 | Proxy port |
| Username | "" | Proxy auth |
| Password | "" | Proxy auth |
| Bypass Filter | "" | Comma-separated bypass patterns |
| Bypass for Local Addresses | true | Skip proxy for local URIs |

### 10.4 Logging
| Setting | Default | Description |
|---------|---------|-------------|
| Log Level | Debug | Info, Debug, Trace |
| Console Log Level | "" | Console output level |
| Console Log Format | Standard | Standard, JSON |
| Log SQL | false | Database query logging |
| Log Rotate | 50 | Rolling log count |
| Log Size Limit | 1 MB | Max log file size (0-10 MB) |
| Log DB Enabled | true | Database logging |
| Syslog Server | "" | Syslog server address |
| Syslog Port | 514 | Syslog port |
| Syslog Level | (follows LogLevel) | Syslog severity |

### 10.5 Analytics
| Setting | Default | Description |
|---------|---------|-------------|
| Analytics Enabled | true | Send anonymous usage/error data |

### 10.6 Updates
| Setting | Default | Description |
|---------|---------|-------------|
| Branch | "master" | Release channel: master, develop, nightly |
| Update Automatically | true (Windows) | Auto-download and install |
| Update Mechanism | BuiltIn | BuiltIn, Script, Docker |
| Script Path | "" | Custom update script |

### 10.7 Backups
| Setting | Default | Description |
|---------|---------|-------------|
| Backup Folder | "Backups" | Relative to appdata or absolute |
| Backup Interval | 7 | Days between backups |
| Backup Retention | 28 | Days to keep (manual backups: forever) |

---

## 11. UI Settings

| Setting | Default | Description |
|---------|---------|-------------|
| First Day of Week | System default | Calendar start day |
| Calendar Week Column Header | "ddd M/D" | Week view date format |
| Movie Runtime Format | HoursMinutes | HoursMinutes or Minutes |
| Short Date Format | "MMM D YYYY" | Abbreviated date |
| Long Date Format | "dddd, MMMM D YYYY" | Full date |
| Time Format | "h(:mm)a" | 12-hour or 24-hour |
| Show Relative Dates | true | Today/Yesterday vs absolute |
| Enable Color-Impaired Mode | false | Accessibility mode |
| Movie Info Language | English | Metadata display language |
| UI Language | English | Interface language |
| Theme | "auto" | UI theme |

---

## 12. System

### 12.1 Status
- Version info, .NET version, AppData directory
- Disk space per root folder (Docker shows container-level)
- About section with links

### 12.2 Health Checks (Categories)
**System:** Branch validity, .NET version, SQLite version, database integrity, update capability, SignalR connectivity, system time deviation, proxy health

**Download Clients:** Client availability, communication, path mapping, permissions, completed download handling, root folder conflicts

**Indexers:** Search capability, RSS configuration, indexer health/availability, Jackett `/all` warning

**Movie Folders:** Root folder availability, mount permissions (read-only detection)

**Movies:** TMDb-deleted movies, list communication failures

**Notifications:** Webhook configuration warnings

### 12.3 Scheduled Tasks
| Task | Description |
|------|-------------|
| Application Check Update | Check for and install updates |
| Backup | Database backup execution |
| Check Health | Comprehensive health assessment |
| Clean Up Recycle Bin | Empty recycling bin |
| Housekeeping | Maintenance and cleanup (not trash) |
| Import List Sync | Execute configured lists |
| Messaging Cleanup | Remove UI notification messages |
| Refresh Monitored Downloads | Check download client status |
| Refresh Movie | Update metadata for all movies |
| RSS Sync | Monitor RSS feeds |

All tasks support manual execution.

### 12.4 Backup System
- Manual backup trigger
- Restore from backup (file selection or upload)
- Download, restore, or delete previous backups
- Restores across different OS not supported (path differences)
- Database must be on local storage (not NFS/SMB)

### 12.5 Updates
- Historical record of past 5 updates
- Current version display
- Release notes

### 12.6 Events & Logs
**Events:** INFO-level log viewer, 50-per-page, clear/refresh

**Log Files:**
- Standard: `radarr.txt` + rolling `radarr.0.txt` through `radarr.51.txt`
- Debug: `radarr.debug.txt` (~40-hour coverage)
- Trace: `radarr.trace.txt` (couple-hour coverage)
- Updater logs
- Download/delete/refresh controls

---

## 13. Movie Management

### 13.1 Add Movie Options
| Field | Options |
|-------|---------|
| Root Folder | Select from configured root folders |
| Monitor | MovieOnly, MovieAndCollection, None |
| Minimum Availability | TBA, Announced, InCinemas, Released |
| Quality Profile | Select from configured profiles |
| Tags | Assign tags |
| Search on Add | bool — trigger search immediately |
| Add Method | Manual, List, Collection |

### 13.2 Movie Status Types
| Value | ID | Description |
|-------|----|-------------|
| Deleted | -1 | Deleted from TMDb |
| TBA | 0 | Only rumors, has IMDb page |
| Announced | 1 | Announced but cinema date future/unknown |
| InCinemas | 2 | In cinemas < 3 months |
| Released | 3 | Physical/Web release or > 3 months in cinemas |

### 13.3 Movie Metadata (MovieMetadata model)
TmdbId, ImdbId, Title, CleanTitle, SortTitle, OriginalTitle, CleanOriginalTitle, OriginalLanguage, Year, SecondaryYear, Overview, Certification, Runtime, Website, Studio, Images, Genres, Keywords, Ratings (TMDb/IMDb/Rotten Tomatoes), CollectionTmdbId, CollectionTitle, YouTubeTrailerId, InCinemas, PhysicalRelease, DigitalRelease, Status, AlternativeTitles, Translations, Recommendations, Popularity

### 13.4 Movie Collections
| Field | Description |
|-------|-------------|
| Title, CleanTitle, SortTitle | Collection name variants |
| TmdbId | TMDb collection ID |
| Overview | Collection description |
| Monitored | bool |
| QualityProfileId | Assigned profile |
| RootFolderPath | Storage path |
| SearchOnAdd | Auto-search new collection movies |
| MinimumAvailability | Default for collection movies |
| Images | Cover art |
| Movies | List of movies in collection |
| Tags | Associated tags |

### 13.5 Movie Editor (Bulk Edit)
Bulk operations on selected movies:
- Change monitoring status
- Change quality profile
- Change root folder (with move option)
- Change minimum availability
- Change tags (add/remove)
- Delete multiple movies

### 13.6 Library Views
- **Table**: List view with configurable columns
- **Posters**: Visual grid with poster images
- **Overview**: Detailed view with descriptions

### 13.7 Filtering
Predefined: Monitored Only, Unmonitored, Missing, Wanted, Cut-off Unmet

Custom filters on: monitored status, availability, quality profiles, release dates, ratings (TMDb, IMDb, Rotten Tomatoes), genres, certifications, disk storage, tags, path, and more

### 13.8 MediaInfo Tracked Fields (29)
Container format, video format/codec/profile/bitrate/bitdepth/FPS/3D, HDR format, Dolby Vision config, resolution (height/width), scan type, audio format/codec/profile/bitrate/channels/channel positions/stream count/languages, subtitles, runtime

### 13.9 Supported File Extensions
`.webm, .m4v, .3gp, .nsv, .ty, .strm, .rm, .rmvb, .m3u, .ifo, .mov, .qt, .divx, .xvid, .bivx, .nrg, .pva, .wmv, .asf, .asx, .ogm, .ogv, .m2v, .avi, .bin, .dat, .dvr-ms, .mpg, .mpeg, .mp4, .avc, .vp3, .svq3, .nuv, .viv, .dv, .fli, .flv, .wpl, .img, .iso, .vob, .mkv, .mk3d, .ts, .wtv, .m2ts`

---

## 14. Calendar

### 14.1 Calendar View
- Displays upcoming and recently released monitored movies by week
- Shows In Cinemas / Physical Release / Digital Release dates

### 14.2 iCal Feed
- Endpoint: `/feed/v3/calendar/radarr.ics`
- Parameters: date range filtering
- Provides calendar subscription for external apps

---

## 15. Wanted

### 15.1 Missing Movies
- Movies that are monitored but have no file
- Bulk search capability
- Paged list with sorting

### 15.2 Cutoff Unmet
- Movies that have a file but below the quality profile cutoff
- Bulk search for upgrades
- Paged list with sorting/filtering

---

## 16. Queue

### 16.1 Queue Management
- Displays actively downloading items not yet imported
- Shows items in download client's specified category
- Show Unknown releases option
- Usenet: only scans 60 items deep

### 16.2 Queue Actions
- Remove items from queue or download client
- Blocklist releases to prevent re-download
- Manually import releases
- Resend releases to download client
- Change priority

### 16.3 Queue Statuses
- Grey clock: Release Pending (awaiting delay rules)
- Yellow: Warning — Unable to Import
- Purple: Download Importing (active import)

### 16.4 Queue API
| Endpoint | Description |
|----------|-------------|
| GET /api/v3/queue | View queue (paged, filtered) |
| DELETE /api/v3/queue/{id} | Remove from queue |
| GET /api/v3/queue/status | Queue status summary |

---

## 17. History

### 17.1 History Event Types
| Type | ID | Description |
|------|----|-------------|
| Unknown | 0 | Unknown event |
| Grabbed | 1 | Release grabbed from indexer |
| DownloadFolderImported | 3 | Imported from download folder |
| DownloadFailed | 4 | Download failed |
| MovieFileDeleted | 6 | Movie file deleted |
| MovieFolderImported | 7 | Imported from movie folder (unused) |
| MovieFileRenamed | 8 | Movie file renamed |
| DownloadIgnored | 9 | Download ignored |

### 17.2 History Data Fields
MovieId, SourceTitle, Quality, Date, EventType, DownloadId, Languages, plus metadata dict (download client, movie match type, release source, release group, size, indexer)

### 17.3 History Features
- Filtering by event type
- Per-movie history view
- Mark as failed (triggers removal + blocklist + re-search)
- History since timestamp
- Configurable columns

---

## 18. Blocklist

- Prevents re-downloading failed/unwanted releases
- Items remain permanently unless manually removed
- Info reveals automatic vs manual failure
- Remove items to allow re-grabbing
- Bulk delete support
- Per-movie blocklist view

---

## 19. API

### 19.1 API Key
- Auto-generated on first run
- Used for all external API access via `X-Api-Key` header or `apikey` query parameter
- Redacted in logs
- Configurable via config.xml or environment variable

### 19.2 API Endpoints (comprehensive — 100+ endpoints)
Major endpoint groups:
- `/api/v3/movie` — CRUD, lookup, import, bulk operations
- `/api/v3/movie/editor` — bulk edit
- `/api/v3/qualityprofile` — CRUD
- `/api/v3/qualitydefinition` — CRUD
- `/api/v3/customformat` — CRUD, bulk, schema
- `/api/v3/indexer` — CRUD, bulk, test, schema
- `/api/v3/downloadclient` — CRUD, bulk, test, schema
- `/api/v3/importlist` — CRUD, bulk, test, schema
- `/api/v3/notification` — CRUD, test, schema
- `/api/v3/metadata` — CRUD, test, schema
- `/api/v3/delayprofile` — CRUD, reorder
- `/api/v3/tag` — CRUD, bulk
- `/api/v3/autotagging` — CRUD, schema
- `/api/v3/history` — paged, since, per-movie, mark failed
- `/api/v3/queue` — view, remove, status
- `/api/v3/blocklist` — paged, per-movie, remove, bulk remove
- `/api/v3/wanted/cutoff` — cutoff unmet list
- `/api/v3/calendar` — date range query
- `/feed/v3/calendar/radarr.ics` — iCal feed
- `/api/v3/collection` — CRUD
- `/api/v3/command` — execute, status, cancel
- `/api/v3/release` — search, push
- `/api/v3/interactivesearch` — manual search results
- `/api/v3/manualimport` — list, create
- `/api/v3/rename` — suggestions, bulk execute
- `/api/v3/remotepathmap` — CRUD
- `/api/v3/rootfolder` — CRUD
- `/api/v3/diskspace` — disk info
- `/api/v3/health` — health checks
- `/api/v3/system/status` — system status
- `/api/v3/system/backup` — list, delete, restore, upload
- `/api/v3/system/restart` — restart app
- `/api/v3/system/shutdown` — shutdown app
- `/api/v3/log` — paged logs
- `/api/v3/log/file` — log files, download
- `/api/v3/exclusions` — import exclusions CRUD, paged, bulk
- `/api/v3/credit` — movie credits
- `/api/v3/alttitle` — alternative titles
- `/api/v3/extrafile` — extra files per movie
- `/api/v3/filesystem` — browse, type check, media files
- `/api/v3/language` — available languages
- `/api/v3/localization` — localization data
- `/api/v3/customfilter` — saved filters CRUD
- `/api/v3/indexerflag` — indexer flag options
- `/api/v3/config/*` — mediamanagement, host, downloadclient, indexer, importlist, ui configs
- `/api/v3/update` — check for updates
- `/login` / `/logout` — authentication

---

## 20. Custom Scripts

### 20.1 Event Types
On Grab, On Import/Upgrade, On Rename, On Health Check, On Application Update, On Test

### 20.2 Environment Variables
**On Grab (19 vars):** movie IDs (radarr_movie_id, imdbid, tmdbid), release details (title, quality, size), download info (client, ID), metadata (year, dates, tags, indexer flags)

**On Import/Upgrade (24 vars):** movie/file paths, quality, release groups, isupgrade flag, deleted file paths (pipe-delimited)

**On Rename (15 vars):** previous and current file paths (comma/pipe-delimited), movie metadata

**On Health Check:** issue level (Ok/Notice/Warning/Error), message, type, wiki URL

**On Application Update:** new version, previous version, update message

### 20.3 Logging
- stdout logs as Debug
- stderr logs as Info
- Script triggers log as Trace

---

## 21. Environment Variables

### 21.1 Namespaces
| Namespace | Settings |
|-----------|----------|
| APP | InstanceName, Theme, LaunchBrowser |
| AUTH | ApiKey, Enabled, Method, Required |
| LOG | Level, FilterSentryEvents, Rotate, SizeLimit, SQL, ConsoleLevel, ConsoleFormat, AnalyticsEnabled, SyslogServer/Port/Level, DbEnabled |
| POSTGRES | Host, Port, User, Password, MainDb, LogDb |
| SERVER | UrlBase, BindAddress, Port, EnableSsl, SslPort, SslCertPath, SslCertPassword |
| UPDATE | Mechanism, Automatically, ScriptPath, Branch |

Variables override config.xml, case-sensitive, require restart.

---

## 22. Database

### 22.1 SQLite (Default)
- `radarr.db` — main database
- `logs.db` — log database
- Must be on local storage (not NFS/SMB)
- Minimum SQLite 3.9.0

### 22.2 PostgreSQL (Alternative)
- Supported since v4.1+
- Separate main and log databases
- Configured via environment variables or config.xml
- No migration path from SQLite to PostgreSQL provided

---

## 23. Discover / Recommendations
- Recommendation engine based on existing library
- Import list integration for discovery
- TMDb Popular (Trending, Top Rated, Upcoming, Now Playing)
- Collection following (actors, directors, franchises)

---

## Summary: Feature Count by Category

| Category | Count |
|----------|-------|
| Quality Tiers | 31 |
| Custom Format Condition Types | 10 |
| Download Client Types | 17 |
| Indexer Types | 10 |
| Notification Agent Types | 28 |
| Notification Event Triggers | 13 |
| Import List Types | 18+ |
| Metadata Consumer Types | 5 |
| Auto-Tag Condition Types | 11 |
| Config Settings (config.xml) | ~30 |
| Config Settings (database) | ~60 |
| API Endpoints | 100+ |
| Scheduled Tasks | 10 |
| History Event Types | 8 |
| Naming Tokens | 15+ |
| Indexer Flags | 10 |
| Movie Status Types | 5 |
| Health Check Categories | 5 |
