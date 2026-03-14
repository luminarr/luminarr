# Phase D — Media Management Polish

**Goal:** Metadata/NFO generation, recycling bin, alternative titles, and expanded notification events. Fills the remaining media management gaps that block specific user segments from migrating.

**Depends on:** Phases A–C complete (or at least A for tags/events foundation).

---

## D1. Metadata/NFO Generation (Kodi Format)

Write metadata files alongside movie files so media players (Kodi, Jellyfin, Emby) can read movie info without an internet connection.

### Plugin System

Follow the existing plugin pattern. Each metadata consumer is a plugin.

```go
// pkg/plugin/metadata.go

type MetadataConsumer interface {
    // WriteMovieMetadata writes metadata files for a movie.
    WriteMovieMetadata(ctx context.Context, movie MovieInfo, destDir string) error
    // CleanMovieMetadata removes metadata files for a movie.
    CleanMovieMetadata(ctx context.Context, destDir string) error
    // Info returns plugin metadata.
    Info() MetadataConsumerInfo
}

type MovieInfo struct {
    Title         string
    OriginalTitle string
    Year          int
    Overview      string
    TMDbID        int
    IMDbID        string
    Runtime       int         // minutes
    Genres        []string
    Studio        string
    Rating        float64     // TMDb vote average
    Certification string      // e.g., "PG-13"
    PosterURL     string
    FanartURL     string
    Actors        []CastMember
    Director      string
    ReleaseDate   string      // YYYY-MM-DD
}

type CastMember struct {
    Name      string
    Role      string
    ThumbURL  string
}
```

### Registry Extension

```go
func (r *Registry) RegisterMetadataConsumer(kind string, factory func(json.RawMessage) (plugin.MetadataConsumer, error))
```

### Database

**Migration 00031_metadata_consumers.sql:**
```sql
CREATE TABLE metadata_consumer_configs (
    id       TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    kind     TEXT NOT NULL,       -- "kodi", "kometa"
    enabled  BOOLEAN NOT NULL DEFAULT true,
    settings TEXT NOT NULL DEFAULT '{}'
);
```

### Kodi/XBMC NFO Plugin

**Kind:** `kodi`

**Files written per movie:**
- `movie.nfo` — XML metadata file
- `poster.jpg` — Movie poster (downloaded from TMDb)
- `fanart.jpg` — Background art (downloaded from TMDb)

**NFO Format (Kodi standard):**
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<movie>
  <title>Movie Title</title>
  <originaltitle>Original Title</originaltitle>
  <year>2024</year>
  <plot>Movie overview/description...</plot>
  <runtime>120</runtime>
  <genre>Action</genre>
  <genre>Thriller</genre>
  <studio>A24</studio>
  <rating>7.5</rating>
  <mpaa>PG-13</mpaa>
  <uniqueid type="tmdb" default="true">12345</uniqueid>
  <uniqueid type="imdb">tt1234567</uniqueid>
  <premiered>2024-03-15</premiered>
  <director>Director Name</director>
  <actor>
    <name>Actor Name</name>
    <role>Character Name</role>
    <thumb>https://image.tmdb.org/...</thumb>
  </actor>
  <thumb aspect="poster">poster.jpg</thumb>
  <fanart>
    <thumb>fanart.jpg</thumb>
  </fanart>
</movie>
```

**Settings:**
```json
{
  "movie_metadata": true,
  "movie_images": true,
  "add_collection_name": false
}
```

**Plugin:** `plugins/metadata/kodi/kodi.go`

### Integration Points

- **On import:** After file import completes, run all enabled metadata consumers
- **On rename:** Re-write metadata to new location
- **On refresh:** Re-write metadata with updated TMDB data
- **On delete:** Clean up metadata files

Add metadata writing to `internal/core/importer/importer.go` post-import step.

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/metadata` | List metadata consumer configs |
| POST | `/api/v1/metadata` | Create metadata consumer |
| PUT | `/api/v1/metadata/{id}` | Update metadata consumer |
| DELETE | `/api/v1/metadata/{id}` | Delete metadata consumer |
| POST | `/api/v1/metadata/{id}/test` | Test (write to temp dir) |

---

## D2. Recycling Bin

Move deleted files to a trash folder instead of permanent deletion. Auto-cleanup after N days.

### Settings

Add to media management:
```sql
ALTER TABLE media_management ADD COLUMN recycling_bin_path TEXT NOT NULL DEFAULT '';
ALTER TABLE media_management ADD COLUMN recycling_bin_cleanup_days INTEGER NOT NULL DEFAULT 7;
```

Empty string = disabled (permanent delete, current behavior).

### Logic

In every file deletion path (movie file delete, upgrade replacement, library cleanup):

```go
func (s *Service) deleteFile(path string) error {
    settings := s.getMediaManagement()
    if settings.RecyclingBinPath == "" {
        return os.Remove(path)
    }
    // Preserve relative path structure in recycling bin
    relPath := filepath.Base(path)
    dest := filepath.Join(settings.RecyclingBinPath, relPath)
    return os.Rename(path, dest)
}
```

### Cleanup Job

Add scheduled job: `recycling_bin_cleanup`
- Interval: 24 hours
- Logic: walk recycling bin directory, delete files older than `cleanup_days`
- Only runs if recycling bin path is configured

### API Addition

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/media-management/recycling-bin/cleanup` | Trigger manual cleanup |

---

## D3. Alternative Titles

Track movies by alternate names (translations, regional titles) to improve release matching.

### Database

**Migration 00032_alternative_titles.sql:**
```sql
CREATE TABLE alternative_titles (
    id       TEXT PRIMARY KEY,
    movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    title    TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',  -- ISO 639-1 code (e.g., "de", "fr")
    type     TEXT NOT NULL DEFAULT '',  -- "translation", "alias", etc.
    UNIQUE(movie_id, title)
);
CREATE INDEX idx_alt_titles_movie ON alternative_titles(movie_id);
```

### TMDB Integration

TMDb API returns alternative titles via `/movie/{id}/alternative_titles`. Fetch during metadata refresh.

**Response shape:**
```json
{
  "titles": [
    { "iso_3166_1": "DE", "title": "German Title", "type": "" },
    { "iso_3166_1": "FR", "title": "French Title", "type": "translation" }
  ]
}
```

### Matching Integration

When matching a release title to a movie:
1. Try primary title (current behavior)
2. Try original title (current behavior)
3. **NEW:** Try each alternative title

Update the title matching in autosearch to check `alternative_titles` table.

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/movies/{id}/alternative-titles` | List alt titles for movie |

No manual CRUD needed — alt titles are fetched from TMDb automatically.

---

## D4. Expanded Notification Events

Add the 6 missing notification events to match Radarr.

### New Event Types

Add to `internal/events/bus.go`:

```go
const (
    // Existing events...

    // New events
    TypeMovieFileDeleted   Type = "movie_file_deleted"
    TypeMovieFileRenamed   Type = "movie_file_renamed"
    TypeUpgradeComplete    Type = "upgrade_complete"     // import that replaced existing file
    TypeApplicationUpdated Type = "application_updated"
)
```

### Publisher Integration

| Event | Where to Publish |
|-------|-----------------|
| `movie_file_deleted` | Movie file delete handler (`internal/api/v1/movies.go`) |
| `movie_file_renamed` | Rename service (`internal/core/mediamanagement/`) |
| `upgrade_complete` | Importer — when import replaces an existing file |
| `application_updated` | Update check handler (when update is applied) |

### Notification Plugin Updates

Add new events to the notification subscription options. Update notification config forms to include checkboxes for new events.

Update `plugins/notifications/*/` — each plugin's `Send()` method already takes an event; new events just need message formatting.

### Event Data

Each event should include enough data for rich notifications:

**movie_file_deleted:**
```json
{
  "movie_id": "...",
  "movie_title": "...",
  "file_path": "/path/to/file.mkv",
  "reason": "manual|upgrade|cleanup",
  "quality": "1080p Bluray"
}
```

**upgrade_complete:**
```json
{
  "movie_id": "...",
  "movie_title": "...",
  "old_quality": "720p WEBDL",
  "new_quality": "1080p Bluray",
  "old_custom_format_score": 100,
  "new_custom_format_score": 1750
}
```

### Custom Script Environment Variables

For the `command` notification plugin, set environment variables matching Radarr's pattern:

```
LUMINARR_EVENTTYPE=Grab|Import|Upgrade|Rename|MovieFileDelete|...
LUMINARR_MOVIE_ID=12345
LUMINARR_MOVIE_TITLE=Movie Title
LUMINARR_MOVIE_YEAR=2024
LUMINARR_MOVIE_TMDBID=67890
LUMINARR_MOVIE_IMDBID=tt1234567
LUMINARR_MOVIE_PATH=/movies/Movie Title (2024)/
LUMINARR_MOVIEFILE_PATH=/movies/Movie Title (2024)/Movie.mkv
LUMINARR_MOVIEFILE_QUALITY=Bluray-1080p
LUMINARR_RELEASE_TITLE=Movie.Title.2024.1080p.BluRay.x264-GROUP
LUMINARR_RELEASE_GROUP=GROUP
LUMINARR_RELEASE_SIZE=8500000000
LUMINARR_DOWNLOAD_CLIENT=qBittorrent
LUMINARR_DOWNLOAD_ID=abc123...
```

Update `plugins/notifications/command/command.go` to set these vars based on event type.

---

## D5. More Naming Tokens

Add missing naming tokens to match Radarr:

| Token | Source | Priority |
|-------|--------|----------|
| `{Movie TitleThe}` | "The Movie" → "Movie, The" | Medium |
| `{Movie OriginalTitle}` | Original language title | Medium |
| `{Movie Collection}` | TMDb collection name | Medium |
| `{Movie Certification}` | MPAA rating (PG-13, R, etc.) | Low |
| `{ImdbId}` | IMDb ID (tt1234567) | Medium |
| `{TmdbId}` | TMDb numeric ID | Medium |
| `{MediaInfo AudioCodec}` | From ffprobe scan | Medium |
| `{MediaInfo AudioChannels}` | 5.1, 7.1, etc. | Medium |
| `{MediaInfo VideoDynamicRangeType}` | HDR10, DV, SDR | Medium |
| `{Release Group}` | From release title | High |
| `{Edition Tags}` | Director's Cut, Extended, etc. | Medium |
| `{Custom Formats}` | Matched CF names | High (Phase A dep) |
| `{Custom Format Score}` | CF score number | Low |
| `{Original Title}` | Original release filename | Medium |
| `{Original Filename}` | Original filename without ext | Medium |

Update `internal/core/renamer/renamer.go` to support these tokens. Some require additional data passed to the `Apply()` function.

---

## Frontend

### New/Modified Pages

| Page | Description |
|------|-------------|
| Settings > Metadata | CRUD for metadata consumers (Kodi, etc.) |
| Settings > Media Management | Recycling bin path/days, proper/repack pref, hardlinks |
| Settings > Notifications | New event checkboxes |
| Movie Detail | Show alternative titles section |

---

## Build Order

1. **D4 Notification Events** — small, standalone, high value
2. **D2 Recycling Bin** — small, standalone, safety feature
3. **D3 Alternative Titles** — small, improves matching
4. **D5 Naming Tokens** — incremental, can be done token-by-token
5. **D1 Metadata/NFO** — largest piece, new plugin type

---

## Test Strategy

| Component | Test Type | Key Cases |
|-----------|-----------|-----------|
| Kodi NFO generation | Unit | Valid XML output, all fields populated, special chars escaped |
| Image download | Unit | Poster/fanart download, handle missing images, timeout |
| Recycling bin | Unit | Move to trash, cleanup by age, disabled mode, cross-device |
| Alternative titles | Unit | Fetch from TMDb, store, match releases, dedup |
| Notification events | Unit | Each new event type published with correct data |
| Command env vars | Unit | All vars set correctly for each event type |
| Naming tokens | Unit | Each new token resolves correctly, missing data handled |

---

## Estimated Scope

| Sub-phase | New Files | Modified Files | Migrations | Effort |
|-----------|-----------|---------------|------------|--------|
| D1 Metadata/NFO | 5 | 4 | 1 | Medium |
| D2 Recycling Bin | 1 | 3 | 0 | Small |
| D3 Alternative Titles | 2 | 3 | 1 | Small |
| D4 Notification Events | 0 | 6 | 0 | Small |
| D5 Naming Tokens | 0 | 2 | 0 | Small |
| Frontend | 2+ pages | — | — | Small-Medium |
| **Total** | **~10** | **~18** | **2** | **Medium** |
