# Plan: AI-Powered Command Palette

**Status**: Draft
**Scope**: New AI integration layer, command processing service, and enhanced command palette UI
**Depends on**: Existing command palette (frontend), settings infrastructure (existing)

---

## Summary

Add an optional AI-powered mode to the command palette that lets users type natural language commands like "search for the new Dune movie" or "show me all my 4K remuxes". The AI (Claude) interprets the user's intent and maps it to structured actions. State-modifying actions require explicit confirmation. The feature is purely additive -- when disabled, the command palette works exactly as today.

---

## Current State

- Frontend has a command palette with fuzzy matching for navigation and search
- No natural language understanding -- input must match existing menu items or movie titles
- No Claude API integration anywhere in the stack
- Settings page has app-level configuration but no API key management

---

## Phase 1: Read-Only Actions (Initial Implementation)

### Action Types

- `navigate` -- go to a specific page (e.g., "go to settings" -> navigate to /settings)
- `search_movie` -- search for a movie by title (e.g., "find Inception" -> navigate to search with query)
- `query_library` -- answer questions about the library (e.g., "how many 4K movies do I have?" -> query stats API, return answer)
- `search_releases` -- search for releases of a specific movie (e.g., "search releases for movie 42")
- `explain` -- explain a concept (e.g., "what is a custom format?")
- `fallback` -- AI couldn't determine intent, return helpful message

---

## Step 1: API Key Configuration

### What

Add Claude API key to app settings. Stored as `config.Secret` type (encrypted at rest, never returned to frontend in plain text). Frontend shows masked field with set/clear controls.

### Files

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `AnthropicAPIKey config.Secret` field to AppConfig |
| `internal/config/load.go` | Load API key from config/env |
| `internal/api/v1/settings.go` | Expose set/clear for API key; never return raw key, only `is_set: bool` |
| `web/ui/src/pages/settings/app/AppSettingsPage.tsx` | Add API key field with set/clear UI |

---

## Step 2: Anthropic Client

### What

Minimal Claude API client in `internal/anthropic/client.go`. Thin wrapper around the Messages API -- not a full SDK, just enough for structured command interpretation.

### Capabilities

- Send message with system prompt and user text
- Parse JSON response into typed action struct
- Handle errors (rate limit, auth, network)
- Use `claude-haiku-4-5-20251001` model by default (fast, cheap, ~$0.001/command with prompt caching)

### Files

| File | Change |
|------|--------|
| `internal/anthropic/client.go` | New file: minimal Messages API client |
| `internal/anthropic/client_test.go` | New file: test request formatting and response parsing (mock HTTP) |

---

## Step 3: AI Command Service

### What

New service `internal/core/aicommand/service.go` that orchestrates command processing.

### Flow

1. Receive user text from API handler
2. Build system prompt with available actions, current app state summary
3. Send to Claude via anthropic client
4. Parse structured JSON action from response
5. For read-only actions: execute immediately and return result
6. For state-modifying actions (Phase 2): return action with `requires_confirmation: true`

### System Prompt Design

The system prompt tells Claude:
- Available action types and their JSON schemas
- Current app context (number of movies, active profile names, etc.)
- Instruction to always respond with valid JSON matching the action schema
- Instruction to use `fallback` when uncertain

### Rate Limiting

Token bucket: 10 requests per minute per instance. Returns 429 if exceeded.

### Files

| File | Change |
|------|--------|
| `internal/core/aicommand/service.go` | New file: command processing orchestration |
| `internal/core/aicommand/actions.go` | New file: action type definitions and JSON schemas |
| `internal/core/aicommand/service_test.go` | New file: test action parsing and rate limiting |

---

## Step 4: API Endpoints

### What

- `POST /api/v1/ai/command` -- send user text, receive structured action response
- `POST /api/v1/ai/command/confirm` -- confirm a state-modifying action (Phase 2)

Request: `{ "text": "show me all 4K movies" }`

Response:
```json
{
  "action": "query_library",
  "params": { "filter": "resolution=2160p" },
  "result": { "count": 47, "navigate_to": "/dashboard?quality_resolution=2160p" },
  "explanation": "You have 47 movies in 4K resolution."
}
```

### Files

| File | Change |
|------|--------|
| `internal/api/v1/ai.go` | New file: command and confirm handlers |
| `internal/api/router.go` | Register AI endpoints |
| `internal/registry/registry.go` | Wire AI command service |

---

## Step 5: Frontend Integration

### What

Enhance the existing command palette with an "Ask AI" option that appears when fuzzy matching produces no good results (or always, as a secondary option).

### UX Flow

1. User opens command palette (Cmd+K)
2. Types a query -- fuzzy matching runs as today
3. If no good matches OR user prefixes with "ai:" -- show "Ask AI" option
4. Selecting "Ask AI" sends text to `POST /api/v1/ai/command`
5. Loading indicator while waiting for response
6. Response rendered inline in palette: navigation happens automatically, query results shown as text, search triggers happen immediately
7. For Phase 2 state-modifying actions: confirmation dialog before executing

### Disabled State

When no API key is configured, "Ask AI" option shows "Set up AI in Settings > App" and links to settings.

### Files

| File | Change |
|------|--------|
| `web/ui/src/api/ai.ts` | New file: `sendCommand()`, `confirmAction()` API calls |
| `web/ui/src/components/CommandPalette.tsx` | Add "Ask AI" option, loading state, result rendering |
| `web/ui/src/components/AIConfirmDialog.tsx` | New file (Phase 2): confirmation dialog for destructive actions |

---

## Future Phases (Not in Initial PR)

### Phase 2: Read-Write with Confirmation

New action types:
- `auto_search` -- trigger automatic search for a movie
- `grab` -- grab a specific release
- `run_task` -- trigger a scheduled task

All require confirmation via `POST /api/v1/ai/command/confirm` with the action ID.

### Phase 3: Conversational Context

Maintain short conversation history (last 3 exchanges) for follow-up questions like "search for that one" after "find the new Dune movie".

### Phase 4: Bulk Operations

"Search for all movies missing files" or "upgrade all 720p movies to 1080p" -- with batch confirmation.

---

## Implementation Order

```
Step 1: API key configuration        [independent]
Step 2: Anthropic client             [depends on 1 for key]
Step 3: AI command service           [depends on 2]
Step 4: API endpoints                [depends on 3]
Step 5: Frontend integration         [depends on 4]
```

**PR Strategy**:
- PR 1: Steps 1-2 (config + anthropic client)
- PR 2: Steps 3-4 (service + API)
- PR 3: Step 5 (frontend)

---

## Key Design Decisions

- **Claude Haiku for speed and cost**: At ~$0.001/command, even heavy users won't exceed $1/month. Haiku is fast enough for interactive use (~500ms).
- **Structured output, not free text**: Claude returns JSON actions, not prose. The frontend interprets actions into navigation and display. This keeps the AI layer thin and predictable.
- **Confirmation for state changes**: No grab, search, or task execution without explicit user confirmation. This is non-negotiable for a media management tool.
- **Purely additive**: Zero changes to existing command palette behavior. AI is an additional option, not a replacement. Users who don't set an API key see no difference.
- **Privacy**: Only the user's command text is sent to Claude. No movie titles, no library data, no file paths leave the server. The system prompt contains only action schemas and aggregate counts.
- **Rate limiting at the application level**: 10 RPM token bucket prevents runaway costs and API abuse, regardless of Claude's own rate limits.

---

## Risks Summary

| Risk | Impact | Mitigation |
|------|--------|------------|
| Claude returns invalid JSON | Command fails | Validate response against schema; return fallback action on parse error; retry once |
| Latency too high for interactive use | Poor UX | Haiku is ~500ms; show loading indicator; consider streaming for Phase 3 |
| Cost surprises for users | Unexpected API bills | Show estimated cost in settings; rate limiting caps usage; Haiku is very cheap |
| AI misinterprets intent | Wrong action executed | Read-only actions are harmless; state-modifying actions require confirmation |
| API key security | Key exposure | Stored as config.Secret; never returned to frontend; encrypted at rest |
| Anthropic API outage | Feature unavailable | Graceful degradation to standard command palette; clear error message |
| Prompt injection via movie titles | Unintended actions | Movie titles are not included in prompts; only user command text and action schemas |
