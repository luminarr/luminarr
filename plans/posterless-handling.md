# Plan: Posterless State Handling

**Status**: Draft
**Scope**: Consistent fallback UI for movies without poster images
**Depends on**: Existing dashboard grid, movie detail page, command palette

---

## Summary

Movies without TMDB posters (unmatched imports, TMDB entries with no artwork, TMDB outages during metadata fetch) currently show broken images or empty space. This plan adds a consistent, attractive placeholder that works everywhere posters appear: dashboard grid, movie detail, command palette results, calendar, collections, and any future poster usage.

---

## Current State

- `Movie.poster_url` is optional — can be `undefined` or empty string
- Dashboard grid: the poster `<img>` renders with an empty `src` or missing URL → broken image icon in some browsers, blank space in others
- Movie detail: the poster section has a basic text fallback (`movie.title`) but it's unstyled
- Command palette: movie search results use TMDB `poster_path` with a hardcoded base URL — missing posters show the lucide `Film` icon (this one actually works reasonably)
- Calendar: poster thumbnails with no fallback
- Collections: poster grid with no fallback

---

## Design

### Placeholder Component

A single reusable `PosterPlaceholder` component used everywhere:

```
┌──────────────────┐
│                  │
│    ◇  (icon)     │
│                  │
│   Movie Title    │
│     (2024)       │
│                  │
└──────────────────┘
```

**Visual design:**
- Background: subtle gradient using the first letter of the title to pick a hue (deterministic — same movie always gets the same color). Range of muted, dark-friendly colors.
- Icon: small film/clapperboard icon centered, very subtle (15% opacity)
- Title: centered, medium weight, max 2 lines with ellipsis
- Year: below title, muted color
- Aspect ratio: matches standard poster ratio (2:3)
- Border: 1px `var(--color-border-subtle)` to maintain the grid visual rhythm

**Color generation:**
```typescript
function placeholderHue(title: string): number {
  let hash = 0;
  for (let i = 0; i < title.length; i++) {
    hash = title.charCodeAt(i) + ((hash << 5) - hash);
  }
  return Math.abs(hash) % 360;
}
// background: hsl(hue, 20%, 18%) — low saturation, dark, always readable
```

This ensures each movie gets a visually distinct but cohesive placeholder. No two adjacent movies look identical, but the overall grid still feels uniform.

### Component API

```typescript
interface PosterPlaceholderProps {
  title: string;
  year?: number;
  width?: number;    // defaults to match parent
  aspectRatio?: string; // defaults to "2/3"
}
```

### Wrapper Component

A `Poster` component that handles the switching:

```typescript
function Poster({ src, title, year, ...props }: PosterProps) {
  const [failed, setFailed] = useState(false);

  if (!src || failed) {
    return <PosterPlaceholder title={title} year={year} {...props} />;
  }

  return (
    <img
      src={src}
      alt={title}
      onError={() => setFailed(true)}
      {...props}
    />
  );
}
```

The `onError` handler catches broken image URLs (TMDB CDN failures, expired URLs). This means even movies that *have* a poster_url but the CDN is down get a clean fallback instead of a broken image.

---

## Files to Change

| File | Change |
|---|---|
| `web/ui/src/components/Poster.tsx` | **New file.** `Poster` wrapper + `PosterPlaceholder` component |
| `web/ui/src/pages/dashboard/Dashboard.tsx` | Replace raw `<img>` in movie grid cards with `<Poster>` |
| `web/ui/src/pages/movies/MovieDetail.tsx` | Replace poster `<img>` with `<Poster>` |
| `web/ui/src/pages/calendar/CalendarPage.tsx` | Replace poster thumbnails with `<Poster>` |
| `web/ui/src/pages/collections/CollectionDetail.tsx` | Replace poster grid with `<Poster>` |
| `web/ui/src/components/command-palette/CommandPalette.tsx` | Replace poster `<img>` in movie results with `<Poster>` (small size) |

---

## Implementation Order

1. Build `Poster` + `PosterPlaceholder` components
2. Replace dashboard grid poster images
3. Replace movie detail poster
4. Replace calendar thumbnails
5. Replace collection detail posters
6. Replace command palette movie result posters
7. Test with movies that have no `poster_url`, empty `poster_url`, and valid `poster_url` pointing to a 404

---

## Testing

### Frontend — Unit Tests

**Poster component — `web/ui/src/components/Poster.test.tsx`:**
- Renders `<img>` when `src` is provided and valid
- Renders `PosterPlaceholder` when `src` is undefined
- Renders `PosterPlaceholder` when `src` is empty string
- Renders `PosterPlaceholder` when `<img>` fires `onError` (simulates broken CDN URL)
- `alt` attribute matches `title` prop on both `<img>` and placeholder
- Placeholder shows movie title text
- Placeholder shows year when provided
- Placeholder omits year when not provided
- Placeholder maintains 2:3 aspect ratio (check computed style)

**Color generation — `web/ui/src/components/Poster.test.tsx`:**
- `placeholderHue("Alien")` returns a number between 0 and 359
- Same title always produces the same hue (deterministic)
- Different titles produce different hues (test with 10 common titles, verify at least 5 distinct hues)
- Empty string title does not throw, returns a valid hue

**Dashboard integration — `web/ui/src/pages/dashboard/Dashboard.test.tsx`:**
- Movie card with `poster_url` set → renders `<img>` with that URL
- Movie card with `poster_url` undefined → renders placeholder with movie title
- Movie card with `poster_url` pointing to non-existent image → after error, renders placeholder

**Command palette — `web/ui/src/components/command-palette/CommandPalette.test.tsx`:**
- Movie search result with `poster_path` → renders poster thumbnail
- Movie search result without `poster_path` → renders placeholder (not broken image)

### Integration Tests

**Visual regression (manual checklist, not automated):**
- Dashboard grid with mix of poster and posterless movies looks cohesive
- Placeholders are visually distinct from each other (different hues)
- Placeholder text is readable at all poster sizes (small grid thumbnails through large detail view)
- Dark theme and light theme both render placeholders correctly

---

## Risks

| Risk | Mitigation |
|---|---|
| Color generation produces clashing or ugly hues | Low saturation (20%) and dark lightness (18%) keeps everything muted. Test with real data. |
| Performance with many placeholders on dashboard | Pure CSS gradient + text — no canvas, no SVG generation. Lighter than loading a real image. |
| Inconsistent sizes across pages | Aspect ratio enforced via CSS `aspect-ratio: 2/3` on the component. Parent controls width. |
