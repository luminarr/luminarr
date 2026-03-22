# Luminarr — TODO

## AI Features

- [x] **AI command palette** — Natural language commands via Cmd+K. Navigate, search, query library stats, grab movies with quality preferences, run tasks. State-modifying actions require explicit confirmation. Uses Claude Haiku. Shipped in v0.9.0.
- [ ] **AI-assisted collection discovery (Tier 2)** — When smart TMDB search (Tier 1) produces weak results, use Claude API to interpret the query and generate a movie list. Example: user types "Coen Brothers" → Claude knows this means Joel + Ethan's joint filmography and returns the full list → map titles back to TMDB IDs.
- [ ] **Conversational context (Phase 3)** — Maintain short conversation history (last 3 exchanges) for follow-up questions like "search for that one" after "find the new Dune movie".
- [ ] **Bulk operations (Phase 4)** — "Search for all movies missing files" or "upgrade all 720p movies to 1080p" with batch confirmation.

## Quality System

- [ ] **Quality grouping** — Radarr lets users group multiple quality tiers together so they're treated as equivalent rank in a profile (e.g., WEBDL-1080p and WEBRip-1080p as interchangeable). Radarr's UX for this (nested drag-and-drop) is clunky. Revisit with a cleaner, more modern approach — possibly a simple "treat as equivalent" toggle or a grouping UI that doesn't require drag reordering of nested items.
