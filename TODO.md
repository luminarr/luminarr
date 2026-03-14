# Luminarr — TODO

## AI Features (Deferred)

- [ ] **AI-assisted collection discovery (Tier 2)** — When smart TMDB search (Tier 1) produces weak results, use Claude API to interpret the query and generate a movie list. Example: user types "Coen Brothers" → Claude knows this means Joel + Ethan's joint filmography and returns the full list → map titles back to TMDB IDs. Requires optional Claude API key configuration in settings. Natural first use of Phase 6 (AI).

## Quality System

- [ ] **Quality grouping** — Radarr lets users group multiple quality tiers together so they're treated as equivalent rank in a profile (e.g., WEBDL-1080p and WEBRip-1080p as interchangeable). Radarr's UX for this (nested drag-and-drop) is clunky. Revisit with a cleaner, more modern approach — possibly a simple "treat as equivalent" toggle or a grouping UI that doesn't require drag reordering of nested items.
