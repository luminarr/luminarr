import { useState } from "react";
import { Film } from "lucide-react";

// ── Color generation ────────────────────────────────────────────────────────

/**
 * Deterministic hue from a title string. Same title always produces the same
 * color. Used for placeholder backgrounds so each movie is visually distinct.
 */
export function placeholderHue(title: string): number {
  let hash = 0;
  for (let i = 0; i < title.length; i++) {
    hash = title.charCodeAt(i) + ((hash << 5) - hash);
  }
  return Math.abs(hash) % 360;
}

// ── PosterPlaceholder ───────────────────────────────────────────────────────

interface PosterPlaceholderProps {
  title: string;
  year?: number;
  style?: React.CSSProperties;
}

export function PosterPlaceholder({ title, year, style }: PosterPlaceholderProps) {
  const hue = placeholderHue(title);

  return (
    <div
      aria-label={title}
      data-testid="poster-placeholder"
      style={{
        aspectRatio: "2/3",
        width: "100%",
        background: `linear-gradient(135deg, hsl(${hue}, 20%, 18%) 0%, hsl(${hue}, 15%, 14%) 100%)`,
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 6,
        padding: 12,
        overflow: "hidden",
        ...style,
      }}
    >
      <Film
        size={24}
        strokeWidth={1.5}
        style={{ opacity: 0.15, color: `hsl(${hue}, 30%, 50%)`, flexShrink: 0 }}
      />
      <span
        style={{
          fontSize: 11,
          fontWeight: 500,
          color: "var(--color-text-secondary)",
          textAlign: "center",
          lineHeight: 1.3,
          overflow: "hidden",
          display: "-webkit-box",
          WebkitLineClamp: 2,
          WebkitBoxOrient: "vertical",
          wordBreak: "break-word",
        }}
      >
        {title}
      </span>
      {year != null && year > 0 && (
        <span
          style={{
            fontSize: 10,
            color: "var(--color-text-muted)",
          }}
        >
          {year}
        </span>
      )}
    </div>
  );
}

// ── Poster ──────────────────────────────────────────────────────────────────

interface PosterProps {
  src?: string | null;
  title: string;
  year?: number;
  style?: React.CSSProperties;
  imgStyle?: React.CSSProperties;
  loading?: "lazy" | "eager";
}

/**
 * Renders a poster image with automatic fallback to PosterPlaceholder when:
 * - src is null/undefined/empty
 * - the image fails to load (CDN outage, 404, etc.)
 */
export function Poster({ src, title, year, style, imgStyle, loading = "lazy" }: PosterProps) {
  const [failed, setFailed] = useState(false);

  if (!src || failed) {
    return <PosterPlaceholder title={title} year={year} style={style} />;
  }

  return (
    <img
      src={src}
      alt={title}
      loading={loading}
      onError={() => setFailed(true)}
      data-testid="poster-img"
      style={{
        aspectRatio: "2/3",
        width: "100%",
        objectFit: "cover",
        borderRadius: 8,
        display: "block",
        ...style,
        ...imgStyle,
      }}
    />
  );
}
