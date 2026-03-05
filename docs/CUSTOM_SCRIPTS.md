# Custom Scripts

Luminarr can execute custom scripts when movie events occur (grabs, imports, health changes, etc.). This lets you integrate with home automation, external tools, logging services, or anything else — using any language.

## Setup

1. Place your script in `/config/scripts/` (the Docker volume maps this to your host).
2. Make it executable: `chmod +x /config/scripts/my-script.sh`.
3. In **Settings → Notifications → Add Notification**, choose **Command** and enter the filename (e.g. `my-script.sh`).
4. Select which events should trigger the script.
5. Click **Test** to verify the file exists and is executable.

Scripts must be plain filenames — no path separators (`/`, `\`) or `..` are allowed. Luminarr resolves the path within `/config/scripts/` and rejects anything that would escape it.

## How scripts are called

When a matching event fires, Luminarr:

1. Executes `/config/scripts/<your-script>` as a subprocess.
2. Passes the full event as **JSON on stdin**.
3. Sets **environment variables** for common fields.
4. Captures stdout/stderr and logs them at INFO level.
5. Kills the process if it exceeds the configured timeout (default: 30 seconds).

Your script's exit code determines success or failure. Exit `0` for success; any non-zero exit code is logged as an error.

## Environment variables

| Variable | Description | Example |
|---|---|---|
| `LUMINARR_EVENT_TYPE` | Event type identifier | `grab_started` |
| `LUMINARR_MOVIE_ID` | Movie UUID (empty for non-movie events) | `a1b2c3d4-...` |
| `LUMINARR_MESSAGE` | Human-readable event summary | `Grabbed: Inception (2010)` |
| `LUMINARR_TIMESTAMP` | RFC 3339 timestamp | `2026-03-05T14:30:00Z` |

The full parent environment is also inherited, so any env vars set in your Docker Compose or shell are available.

## Stdin JSON payload

The complete event is piped to stdin as a JSON object:

```json
{
  "Type": "grab_started",
  "Timestamp": "2026-03-05T14:30:00Z",
  "MovieID": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "Message": "Grabbed: Inception (2010) — 1080p BluRay",
  "Data": {
    "movie_title": "Inception",
    "movie_year": 2010,
    "tmdb_id": 27205
  }
}
```

The `Data` field contains event-specific extras (varies by event type). Use stdin when you need the full payload; use env vars for quick shell scripts that only need a few fields.

## Event types

| Type | When it fires |
|---|---|
| `movie_added` | A movie is added to a library |
| `movie_deleted` | A movie is deleted |
| `grab_started` | A release is grabbed from an indexer |
| `download_done` | A download client reports completion |
| `import_done` | A downloaded file is imported into the library |
| `import_failed` | An import attempt failed |
| `health_issue` | A health check detected a problem |
| `health_ok` | A previously failing health check recovered |

## Configuration

| Setting | Default | Description |
|---|---|---|
| `script_name` | *(required)* | Filename in `/config/scripts/` |
| `timeout` | `30` | Max execution time in seconds before the process is killed |

## Examples

### Shell — log events to a file

```bash
#!/bin/sh
echo "$(date): $LUMINARR_EVENT_TYPE — $LUMINARR_MESSAGE" >> /config/luminarr-events.log
```

### Shell — send to a custom HTTP endpoint

```bash
#!/bin/sh
# Forward the full JSON payload to an internal service
cat | curl -s -X POST -H "Content-Type: application/json" \
  -d @- http://homeassistant.local:8123/api/webhook/luminarr
```

### Python — parse stdin and act on specific events

```python
#!/usr/bin/env python3
import json, sys, subprocess

event = json.load(sys.stdin)

if event["Type"] == "import_done":
    title = event.get("Data", {}).get("movie_title", "Unknown")
    subprocess.run(["notify-send", f"Luminarr: {title} imported"])
```

### Shell — trigger a Plex scan after import

```bash
#!/bin/sh
if [ "$LUMINARR_EVENT_TYPE" = "import_done" ]; then
  curl -s -X POST "http://plex.local:32400/library/sections/1/refresh?X-Plex-Token=$PLEX_TOKEN"
fi
```

## Troubleshooting

- **"script not found"** — Check that the file exists at `/config/scripts/<name>` inside the container. If using Docker, verify your volume mount.
- **"script is not executable"** — Run `chmod +x /config/scripts/<name>`.
- **Timeout errors** — Increase the timeout in the notification settings, or make your script faster (e.g. background long-running work with `&`).
- **Script output** — stdout and stderr are captured and logged at INFO level. Check the Luminarr logs to see your script's output.
