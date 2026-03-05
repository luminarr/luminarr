# Migrating from Radarr

This guide is for users familiar with Radarr who want to understand where
settings live in Luminarr and what the key differences are.

---

## Import Wizard

Use **Settings → Import** to pull your existing Radarr configuration (quality
profiles, indexers, download clients, root folders, and movie library) into
Luminarr in one step. You will need your Radarr URL and API key.

---

## Settings mapping

### Media Management → Movie Naming

| Radarr | Luminarr | Notes |
|---|---|---|
| Rename Movies | Settings → Media Management → Rename Movies | Same toggle |
| Standard Movie Format | Settings → Media Management → Standard Movie Format | Same tokens, same default |
| Movie Folder Format | Settings → Media Management → Movie Folder Format | Same |
| Colon Replacement | Settings → Media Management → Colon Replacement | Same four strategies |

**Per-library overrides:** Luminarr treats each library as an independent entity.
You can override the file naming format and folder format per library in
**Settings → Libraries → Edit**. Leave the fields blank to inherit the global
default from Media Management.

### Media Management → Importing

| Radarr | Luminarr | Notes |
|---|---|---|
| Use Hardlinks instead of Copy | Always on | Luminarr always tries a hardlink first and falls back to copy — there is no toggle |
| Import Extra Files | Settings → Media Management → Import Extra Files | Same |
| Extra File Extensions | Settings → Media Management → Extra File Extensions | Same comma-separated format |
| Skip Free Space Check | Not present | Set Min Free Space to 0 on the library instead |
| Minimum Free Space | Settings → Libraries → Min Free Space (GB) | Configured per library in Luminarr, not globally |
| Import Using Script | Not present | Out of scope |

### Media Management → File Management

| Radarr | Luminarr | Notes |
|---|---|---|
| Unmonitor Deleted Movies | Settings → Media Management → Unmonitor Deleted Movies | Same |
| Recycling Bin | Not present | Deleted files are permanently removed; planned for a future release |
| Propers and Repacks | Not present | Grab priority policy — planned |
| Rescan Movie Folder after Refresh | Automatic | Luminarr's library scanner runs on a schedule and picks up file changes |
| Analyze Video Files | Not present | No mediainfo dependency; quality metadata comes from the grab |
| Change File Date | Not present | Out of scope |

### Media Management → Permissions

| Radarr | Luminarr | Notes |
|---|---|---|
| Set Permissions / chmod / chown | Not present | Let your download client or Docker user set permissions instead |

---

### Quality Profiles

Radarr quality profiles map directly to **Settings → Quality Profiles** in
Luminarr. The Import Wizard copies them automatically.

### Quality Definitions (size constraints)

Radarr's Quality Definitions (min/max file size per quality) are at
**Settings → Quality Definitions** in Luminarr. The Import Wizard does not copy
size settings from Radarr; Luminarr seeds sensible defaults based on TRaSH
Guides recommendations.

### Indexers

**Settings → Indexers.** Torznab and Newznab are supported. The Import Wizard
copies indexer URLs and API keys from Radarr automatically.

### Download Clients

**Settings → Download Clients.** qBittorrent, Deluge, Transmission, SABnzbd, and NZBGet are supported. The Import Wizard copies connection details automatically.

### Notifications

**Settings → Notifications.** Discord, Slack, Telegram, Gotify, ntfy, Pushover, Webhook, Email, and Command (custom scripts) are supported. Radarr notifications are not imported — configure them manually. See [Custom Scripts](CUSTOM_SCRIPTS.md) for the Command notifier.

### Blocklist

**Settings → Blocklist.** Works the same way: blocked releases are skipped
during future grabs.

---

## Things Radarr has that Luminarr does not (yet)

- Custom Formats and scoring
- Recycling Bin
- Propers/Repacks grab policy
- Rescan policy (Always / Never / After Manual Refresh)
- Rename existing files in bulk (only renames on new imports)
- Movie collections / series grouping
- Calendar integration (iCal export)
- API key management UI
- Backup and restore
