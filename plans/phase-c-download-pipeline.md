# Phase C — Download Pipeline Polish

**Goal:** Delay profiles, propers/repacks handling, hardlink support, and additional download clients. Makes Luminarr's download pipeline match Radarr for dual-protocol and torrent-heavy users.

**Depends on:** Phase A (tags — delay profiles are linked via tags).

---

## C1. Delay Profiles

Wait N minutes after a release appears before grabbing, with protocol preference. Lets users prefer Usenet over torrents (or vice versa) by giving the preferred protocol a head start.

### Database

**Migration 00029_delay_profiles.sql:**
```sql
CREATE TABLE delay_profiles (
    id                    TEXT PRIMARY KEY,
    preferred_protocol    TEXT NOT NULL DEFAULT 'usenet',   -- 'usenet' | 'torrent'
    usenet_delay_minutes  INTEGER NOT NULL DEFAULT 0,
    torrent_delay_minutes INTEGER NOT NULL DEFAULT 0,
    bypass_if_highest     BOOLEAN NOT NULL DEFAULT false,   -- skip delay if release matches highest quality
    sort_order            INTEGER NOT NULL DEFAULT 0,
    tags                  TEXT NOT NULL DEFAULT '[]'        -- JSON array of tag IDs
);

-- Default profile: no delay, no tags (applies to all untagged movies)
INSERT INTO delay_profiles (id, preferred_protocol, usenet_delay_minutes, torrent_delay_minutes, bypass_if_highest, sort_order, tags)
VALUES ('default', 'usenet', 0, 0, false, 999, '[]');
```

### How Delay Profiles Work

1. Movie has tags → find first delay profile (by sort_order) that shares a tag with the movie. If none match, use the default (untagged) profile.
2. When a release is found:
   - Check if it matches the highest quality in the profile AND `bypass_if_highest` is true → grab immediately.
   - Check release age (from indexer `publishDate` or `pubDate`): `age = now - publishDate`
   - If release protocol == preferred_protocol AND `age >= preferred_delay` → grab.
   - If release protocol != preferred_protocol AND `age >= other_delay` → grab.
   - Otherwise → **pending** (don't grab yet, re-check on next RSS sync).

### Pending Releases

New concept: releases that passed quality checks but are waiting on delay timer.

```sql
CREATE TABLE pending_releases (
    id           TEXT PRIMARY KEY,
    movie_id     TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    release_json TEXT NOT NULL,       -- full plugin.Release serialized
    indexer_id   TEXT NOT NULL,
    added_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    grab_after   DATETIME NOT NULL    -- when delay expires
);
```

### Pending Release Processing

Add to RSS sync job (or separate "process pending" job):
1. Query pending releases where `grab_after <= now`
2. For each: re-validate (still passes quality? not blocklisted? not already downloaded?)
3. If valid → grab. If invalid → delete pending record.
4. Also: when a better release arrives for same movie, update the pending record.

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/delay-profiles` | List all delay profiles (ordered) |
| POST | `/api/v1/delay-profiles` | Create delay profile |
| PUT | `/api/v1/delay-profiles/{id}` | Update delay profile |
| DELETE | `/api/v1/delay-profiles/{id}` | Delete delay profile |
| PUT | `/api/v1/delay-profiles/reorder` | Reorder profiles |

### Service: `internal/core/delayprofile/service.go`

```go
type Service struct { q dbsqlite.Querier }

func (s *Service) List(ctx) ([]DelayProfile, error)
func (s *Service) Create(ctx, req) (DelayProfile, error)
func (s *Service) Update(ctx, id, req) (DelayProfile, error)
func (s *Service) Delete(ctx, id) error
func (s *Service) Reorder(ctx, ids []string) error
func (s *Service) ForMovie(ctx, movieTags []string) (DelayProfile, error)  // find matching profile
func (s *Service) ShouldGrab(profile DelayProfile, release Release, publishDate time.Time) (grab bool, grabAfter time.Time)
```

### Integration

| Component | Change |
|-----------|--------|
| `autosearch/service.go` | After quality + CF checks, check delay profile. If delayed → add to pending_releases. |
| `scheduler/jobs/rss_sync.go` | Process pending releases on each sync cycle. |
| Queue API | Show pending releases in queue with "pending" status + countdown timer. |

---

## C2. Propers & Repacks

Automatically upgrade to PROPER/REPACK releases (fixed versions from the same release group).

### Setting

Add to media management settings:
```sql
ALTER TABLE media_management ADD COLUMN proper_repack_preference TEXT NOT NULL DEFAULT 'prefer_and_upgrade';
-- Values: 'prefer_and_upgrade', 'do_not_upgrade', 'do_not_prefer'
```

### Logic

1. **Parser:** Detect PROPER, REPACK, RERIP in release title. Store as `release_type` field on Release.
2. **Scoring:** When comparing releases of the **same quality**:
   - `prefer_and_upgrade`: PROPER/REPACK scores higher → triggers upgrade even at same quality level
   - `do_not_upgrade`: PROPER/REPACK noted but won't trigger automatic upgrade
   - `do_not_prefer`: Ignore PROPER/REPACK entirely

3. **Group matching:** A REPACK is only valid if it's from the **same release group** as the existing file. Otherwise it's just a different release, not a fix.

### Files to Modify

| Action | File |
|--------|------|
| Modify | `pkg/plugin/types.go` — add ReleaseType (proper/repack/rerip/none) to Release |
| Modify | Release title parser — detect PROPER/REPACK/RERIP tokens |
| Modify | `internal/core/autosearch/service.go` — proper/repack scoring logic |
| Modify | `internal/core/quality/profile.go` — proper/repack in WantRelease |
| Modify | `internal/api/v1/mediamanagement.go` — expose proper_repack_preference |
| Create | `internal/db/migrations/00030_proper_repack.sql` (or bundle with another migration) |

---

## C3. Hardlink Support

Use filesystem hardlinks instead of file copies when importing torrents that are still seeding. Saves disk space — the file exists once on disk but appears in both the download client's location and the library.

### Setting

Add to media management settings:
```sql
ALTER TABLE media_management ADD COLUMN use_hardlinks BOOLEAN NOT NULL DEFAULT true;
```

### Logic

In `internal/core/importer/importer.go`, when moving/copying a file:

```go
if settings.UseHardlinks && sameFilesystem(src, dst) {
    err = os.Link(src, dst)  // hardlink
    if err != nil {
        // fallback to copy
        err = copyFile(src, dst)
    }
} else {
    err = copyFile(src, dst)
}
```

**Same filesystem detection:** Compare `os.Stat().Sys().(*syscall.Stat_t).Dev` for source and destination. If same device → same filesystem → hardlinks work.

**When to hardlink vs move:**
- Torrent still seeding → hardlink (keep original for seeding)
- Usenet (no seeding) → move (rename)
- Torrent done seeding → move

**Detect seeding status:** Check download client for item status. If `seeding` or `downloading` → hardlink. If `completed` (done seeding) → move.

### Files to Modify

| Action | File |
|--------|------|
| Modify | `internal/core/importer/importer.go` — hardlink logic |
| Modify | `internal/api/v1/mediamanagement.go` — expose use_hardlinks setting |
| Create | `internal/core/importer/fs.go` — filesystem helpers (sameFilesystem, hardlink, copy) |

---

## C4. Additional Download Clients

### C4a. rTorrent (XMLRPC)

**Priority: High** — heavily used by seedbox users.

**Plugin:** `plugins/downloaders/rtorrent/rtorrent.go`

**Settings:**
```json
{
  "url": "https://mybox.example.com/RPC2",
  "username": "",
  "password": "",
  "category": "luminarr",
  "directory": ""
}
```

**Protocol:** XML-RPC. Key methods:
- `load.raw_start` — add torrent
- `d.multicall2` — list torrents
- `d.get_hash`, `d.get_name`, `d.get_size_bytes`, `d.get_completed_bytes`
- `d.get_ratio`, `d.get_state`, `d.is_active`
- `d.set_custom1` — set label/category
- `d.erase` — remove torrent

**Implementation notes:**
- Use `safedialer.LANTransport()` for HTTP client
- XMLRPC library: `github.com/kolo/xmlrpc` or roll minimal client
- Support both HTTP and HTTPS
- Support SCGI (unix socket) as alternative to HTTP

### C4b. Torrent Blackhole

**Priority: Medium** — universal fallback, works with any torrent client.

**Plugin:** `plugins/downloaders/blackhole_torrent/blackhole.go`

**Settings:**
```json
{
  "torrent_folder": "/path/to/watch",
  "watch_folder": "/path/to/completed",
  "save_magnet_files": true
}
```

**Logic:**
- On grab: save .torrent file to `torrent_folder` (or .magnet file with magnet URI)
- Queue poll: scan `watch_folder` for completed downloads
- No direct client communication — relies on file system

### C4c. Usenet Blackhole

**Priority: Low** — similar pattern to torrent blackhole but for .nzb files.

**Plugin:** `plugins/downloaders/blackhole_usenet/blackhole.go`

**Settings:**
```json
{
  "nzb_folder": "/path/to/watch",
  "watch_folder": "/path/to/completed"
}
```

### C4d. Aria2

**Priority: Medium** — modern multi-protocol downloader.

**Plugin:** `plugins/downloaders/aria2/aria2.go`

**Settings:**
```json
{
  "url": "http://localhost:6800/jsonrpc",
  "secret": "",
  "directory": ""
}
```

**Protocol:** JSON-RPC. Key methods:
- `aria2.addTorrent`, `aria2.addUri` — add downloads
- `aria2.tellStatus` — get status
- `aria2.tellActive`, `aria2.tellWaiting`, `aria2.tellStopped` — list downloads
- `aria2.remove`, `aria2.forceRemove` — remove

---

## Frontend

### New/Modified Pages

| Page | Description |
|------|-------------|
| Settings > Delay Profiles | CRUD, drag-to-reorder, tag assignment |
| Settings > Media Management | Add proper/repack preference + hardlinks toggle |
| Settings > Download Clients | Add rTorrent, Blackhole forms |
| Queue page | Show pending releases with delay countdown |

---

## Build Order

1. **C3 Hardlinks** — small, standalone, high value
2. **C2 Propers/Repacks** — small, touches parser + scoring
3. **C4a rTorrent** — new plugin, standalone
4. **C1 Delay Profiles** — medium, requires pending release concept
5. **C4b-d Blackholes + Aria2** — standalone plugins

---

## Test Strategy

| Component | Test Type | Key Cases |
|-----------|-----------|-----------|
| Delay profile matching | Unit | Tag overlap, default fallback, sort order |
| ShouldGrab logic | Unit | Both protocols, delays, bypass_if_highest, edge times |
| Pending releases | Unit | Add, process on timer, replace with better, expire |
| Proper/Repack detection | Unit | Parser detects PROPER, REPACK, RERIP, handles case variations |
| Proper/Repack scoring | Unit | Same-group upgrade, different-group ignore, preference modes |
| Hardlink | Unit | Same filesystem detection, fallback to copy, cross-device |
| rTorrent | Unit | XMLRPC request building, response parsing, error handling |
| Blackhole | Unit | File writing, watch folder scanning, magnet file format |

---

## Estimated Scope

| Sub-phase | New Files | Modified Files | Migrations | Effort |
|-----------|-----------|---------------|------------|--------|
| C1 Delay Profiles | 5 | 4 | 1 | Medium |
| C2 Propers/Repacks | 1 | 5 | 1 | Small |
| C3 Hardlinks | 1 | 2 | 0 | Small |
| C4 Download Clients | 4 | 2 | 0 | Medium |
| Frontend | 2+ pages | — | — | Small-Medium |
| **Total** | **~13** | **~13** | **2** | **Medium-Large** |
