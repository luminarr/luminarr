import { useState, useMemo } from "react";
import { Link } from "react-router-dom";
import { LayoutGrid, List } from "lucide-react";
import {
  useMovies,
  useDeleteMovie,
  useAddMovie,
  useLookupMovies,
} from "@/api/movies";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import type { Movie, TMDBResult } from "@/types";

// ── Shared styles ─────────────────────────────────────────────────────────────

const ctrlStyle: React.CSSProperties = {
  background: "var(--color-bg-elevated)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  padding: "7px 11px",
  fontSize: 13,
  color: "var(--color-text-primary)",
  outline: "none",
};

const thStyle: React.CSSProperties = {
  textAlign: "left",
  padding: "10px 16px",
  fontSize: 11,
  fontWeight: 600,
  letterSpacing: "0.08em",
  textTransform: "uppercase",
  color: "var(--color-text-muted)",
  whiteSpace: "nowrap",
};

// ── Types ────────────────────────────────────────────────────────────────────

type MonitoredFilter = "all" | "monitored" | "unmonitored";
type OnDiskFilter = "all" | "on_disk" | "missing";
type SortField = "title" | "year" | "added_at";
type SortDir = "asc" | "desc";
type ViewMode = "grid" | "list";

// ── StatusBadge ───────────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: string }) {
  const isReleased = status === "released";
  return (
    <span
      style={{
        display: "inline-block",
        padding: "1px 6px",
        borderRadius: 3,
        fontSize: 10,
        fontWeight: 600,
        textTransform: "capitalize",
        letterSpacing: "0.04em",
        background: isReleased
          ? "color-mix(in srgb, var(--color-success) 12%, transparent)"
          : "color-mix(in srgb, var(--color-warning) 12%, transparent)",
        color: isReleased ? "var(--color-success)" : "var(--color-warning)",
        whiteSpace: "nowrap",
      }}
    >
      {status}
    </span>
  );
}

// ── PosterCard ────────────────────────────────────────────────────────────────

function PosterCard({ movie }: { movie: Movie }) {
  const [hovered, setHovered] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const del = useDeleteMovie();
  const onDisk = !!movie.path;

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => {
        setHovered(false);
        setConfirming(false);
      }}
    >
      {/* Poster */}
      <div
        style={{
          paddingBottom: "150%",
          position: "relative",
          borderRadius: 8,
          overflow: "hidden",
          background: "var(--color-bg-subtle)",
          border: "1px solid var(--color-border-subtle)",
        }}
      >
        {movie.poster_url ? (
          <img
            src={movie.poster_url}
            alt={movie.title}
            loading="lazy"
            style={{
              position: "absolute",
              inset: 0,
              width: "100%",
              height: "100%",
              objectFit: "cover",
              display: "block",
            }}
          />
        ) : (
          <div
            style={{
              position: "absolute",
              inset: 0,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              padding: 12,
            }}
          >
            <span
              style={{
                fontSize: 11,
                color: "var(--color-text-muted)",
                textAlign: "center",
                lineHeight: 1.4,
              }}
            >
              {movie.title}
            </span>
          </div>
        )}

        {/* Hover overlay */}
        {hovered && (
          <div
            style={{
              position: "absolute",
              inset: 0,
              background: "rgba(0,0,0,0.72)",
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              justifyContent: "center",
              gap: 8,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {!confirming ? (
              <>
                <Link
                  to={`/movies/${movie.id}`}
                  style={{
                    background: "var(--color-accent)",
                    color: "var(--color-accent-fg)",
                    borderRadius: 6,
                    padding: "7px 0",
                    fontSize: 13,
                    fontWeight: 500,
                    textDecoration: "none",
                    width: "72%",
                    textAlign: "center",
                  }}
                >
                  View
                </Link>
                <button
                  onClick={() => setConfirming(true)}
                  style={{
                    background: "transparent",
                    border: "1px solid rgba(255,255,255,0.25)",
                    borderRadius: 6,
                    padding: "7px 0",
                    fontSize: 13,
                    color: "rgba(255,255,255,0.65)",
                    cursor: "pointer",
                    width: "72%",
                  }}
                >
                  Delete
                </button>
              </>
            ) : (
              <div style={{ textAlign: "center", padding: "0 12px" }}>
                <p
                  style={{
                    margin: "0 0 10px",
                    fontSize: 12,
                    color: "rgba(255,255,255,0.85)",
                    lineHeight: 1.4,
                  }}
                >
                  Remove from library?
                </p>
                <div style={{ display: "flex", gap: 6, justifyContent: "center" }}>
                  <button
                    onClick={() => del.mutate(movie.id)}
                    disabled={del.isPending}
                    style={{
                      background: "var(--color-danger)",
                      border: "none",
                      borderRadius: 6,
                      padding: "6px 16px",
                      fontSize: 13,
                      color: "white",
                      cursor: del.isPending ? "not-allowed" : "pointer",
                      opacity: del.isPending ? 0.7 : 1,
                    }}
                  >
                    {del.isPending ? "…" : "Yes"}
                  </button>
                  <button
                    onClick={() => setConfirming(false)}
                    style={{
                      background: "transparent",
                      border: "1px solid rgba(255,255,255,0.25)",
                      borderRadius: 6,
                      padding: "6px 16px",
                      fontSize: 13,
                      color: "rgba(255,255,255,0.65)",
                      cursor: "pointer",
                    }}
                  >
                    No
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Corner badge — rendered after overlay so it sits on top of it.
            pointerEvents:none so it doesn't block the overlay's click handler. */}
        <div style={{ position: "absolute", top: 6, right: 6, pointerEvents: "none" }}>
          {onDisk ? (
            <span
              style={{
                display: "inline-flex",
                width: 18,
                height: 18,
                borderRadius: "50%",
                background: "var(--color-success)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 10,
                color: "white",
                fontWeight: 700,
                boxShadow: "0 1px 3px rgba(0,0,0,0.4)",
              }}
            >
              ✓
            </span>
          ) : movie.monitored ? (
            <span
              style={{
                display: "inline-flex",
                width: 18,
                height: 18,
                borderRadius: "50%",
                background: "var(--color-warning)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 9,
                color: "white",
                fontWeight: 700,
                boxShadow: "0 1px 3px rgba(0,0,0,0.4)",
              }}
            >
              ●
            </span>
          ) : null}
        </div>
      </div>

      {/* Card footer */}
      <div style={{ paddingTop: 8 }}>
        <p
          style={{
            margin: 0,
            fontSize: 13,
            fontWeight: 500,
            color: "var(--color-text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {movie.title}
        </p>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 5,
            marginTop: 3,
            flexWrap: "wrap",
          }}
        >
          {movie.year > 0 && (
            <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
              {movie.year}
            </span>
          )}
          <StatusBadge status={movie.status} />
        </div>
      </div>
    </div>
  );
}

// ── ListRow ───────────────────────────────────────────────────────────────────

function ListRow({ movie, isLast }: { movie: Movie; isLast: boolean }) {
  const [confirming, setConfirming] = useState(false);
  const del = useDeleteMovie();

  return (
    <tr
      style={{
        borderBottom: isLast ? "none" : "1px solid var(--color-border-subtle)",
      }}
    >
      {/* Poster thumb */}
      <td style={{ padding: "0 0 0 16px", height: 60, width: 44 }}>
        <div
          style={{
            width: 32,
            height: 48,
            borderRadius: 3,
            overflow: "hidden",
            background: "var(--color-bg-subtle)",
            flexShrink: 0,
          }}
        >
          {movie.poster_url && (
            <img
              src={movie.poster_url}
              alt=""
              loading="lazy"
              style={{
                width: "100%",
                height: "100%",
                objectFit: "cover",
                display: "block",
              }}
            />
          )}
        </div>
      </td>

      {/* Title */}
      <td style={{ padding: "0 8px 0 12px", height: 60 }}>
        <Link
          to={`/movies/${movie.id}`}
          style={{
            color: "var(--color-text-primary)",
            fontWeight: 500,
            textDecoration: "none",
            fontSize: 13,
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLAnchorElement).style.color =
              "var(--color-accent)";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLAnchorElement).style.color =
              "var(--color-text-primary)";
          }}
        >
          {movie.title}
        </Link>
      </td>

      {/* Year */}
      <td
        style={{
          padding: "0 16px",
          height: 60,
          color: "var(--color-text-muted)",
          fontSize: 12,
          whiteSpace: "nowrap",
        }}
      >
        {movie.year || "—"}
      </td>

      {/* Status */}
      <td style={{ padding: "0 16px", height: 60 }}>
        <StatusBadge status={movie.status} />
      </td>

      {/* Monitored */}
      <td style={{ padding: "0 16px", height: 60 }}>
        <span
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 5,
            fontSize: 12,
            color: movie.monitored
              ? "var(--color-success)"
              : "var(--color-text-muted)",
          }}
        >
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: "50%",
              background: movie.monitored
                ? "var(--color-success)"
                : "var(--color-text-muted)",
            }}
          />
          {movie.monitored ? "Yes" : "No"}
        </span>
      </td>

      {/* On disk */}
      <td style={{ padding: "0 16px", height: 60, fontSize: 12 }}>
        {movie.path ? (
          <span style={{ color: "var(--color-success)" }}>On disk</span>
        ) : (
          <span
            style={{
              color: "var(--color-text-muted)",
              fontStyle: "italic",
            }}
          >
            Missing
          </span>
        )}
      </td>

      {/* Actions */}
      <td
        style={{
          padding: "0 16px",
          height: 60,
          width: 1,
          whiteSpace: "nowrap",
        }}
      >
        {confirming ? (
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span
              style={{ fontSize: 12, color: "var(--color-text-secondary)" }}
            >
              Delete?
            </span>
            <button
              onClick={() =>
                del.mutate(movie.id, { onSuccess: () => setConfirming(false) })
              }
              disabled={del.isPending}
              style={{
                background:
                  "color-mix(in srgb, var(--color-danger) 15%, transparent)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 5,
                padding: "3px 10px",
                fontSize: 12,
                color: "var(--color-danger)",
                cursor: del.isPending ? "not-allowed" : "pointer",
              }}
            >
              {del.isPending ? "…" : "Yes"}
            </button>
            <button
              onClick={() => setConfirming(false)}
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 5,
                padding: "3px 10px",
                fontSize: 12,
                color: "var(--color-text-secondary)",
                cursor: "pointer",
              }}
            >
              No
            </button>
          </div>
        ) : (
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <Link
              to={`/movies/${movie.id}`}
              style={{
                display: "inline-block",
                textDecoration: "none",
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 5,
                padding: "3px 10px",
                fontSize: 12,
                color: "var(--color-text-secondary)",
              }}
            >
              View
            </Link>
            <button
              onClick={() => setConfirming(true)}
              style={{
                background:
                  "color-mix(in srgb, var(--color-danger) 12%, transparent)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 5,
                padding: "3px 10px",
                fontSize: 12,
                color: "var(--color-danger)",
                cursor: "pointer",
              }}
            >
              Delete
            </button>
          </div>
        )}
      </td>
    </tr>
  );
}

// ── Add Movie Dialog ──────────────────────────────────────────────────────────

const dialogInputStyle: React.CSSProperties = {
  width: "100%",
  background: "var(--color-bg-elevated)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  padding: "8px 12px",
  fontSize: 13,
  color: "var(--color-text-primary)",
  outline: "none",
  boxSizing: "border-box",
};

const dialogLabelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 12,
  fontWeight: 500,
  color: "var(--color-text-secondary)",
  marginBottom: 6,
};

function focusBorder(e: React.FocusEvent<HTMLElement>) {
  (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
}
function blurBorder(e: React.FocusEvent<HTMLElement>) {
  (e.currentTarget as HTMLElement).style.borderColor =
    "var(--color-border-default)";
}

function SearchResultRow({
  result,
  onSelect,
}: {
  result: TMDBResult;
  onSelect: (r: TMDBResult) => void;
}) {
  const posterSrc = result.poster_path
    ? `https://image.tmdb.org/t/p/w92${result.poster_path}`
    : null;

  return (
    <button
      onClick={() => onSelect(result)}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 14px",
        background: "none",
        border: "none",
        borderBottom: "1px solid var(--color-border-subtle)",
        cursor: "pointer",
        textAlign: "left",
        width: "100%",
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLButtonElement).style.background =
          "var(--color-bg-surface)";
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLButtonElement).style.background = "none";
      }}
    >
      {posterSrc ? (
        <img
          src={posterSrc}
          alt={result.title}
          style={{
            width: 36,
            height: 54,
            objectFit: "cover",
            borderRadius: 3,
            flexShrink: 0,
          }}
        />
      ) : (
        <div
          style={{
            width: 36,
            height: 54,
            borderRadius: 3,
            background: "var(--color-bg-subtle)",
            flexShrink: 0,
          }}
        />
      )}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 13,
            fontWeight: 500,
            color: "var(--color-text-primary)",
          }}
        >
          {result.title}
          {result.year > 0 && (
            <span
              style={{
                marginLeft: 6,
                fontSize: 12,
                color: "var(--color-text-muted)",
                fontWeight: 400,
              }}
            >
              {result.year}
            </span>
          )}
        </div>
        {result.overview && (
          <div
            style={{
              fontSize: 12,
              color: "var(--color-text-muted)",
              marginTop: 2,
              overflow: "hidden",
              display: "-webkit-box",
              WebkitLineClamp: 2,
              WebkitBoxOrient: "vertical",
            }}
          >
            {result.overview}
          </div>
        )}
      </div>
      <span
        style={{
          fontSize: 11,
          color: "var(--color-accent)",
          fontWeight: 600,
          flexShrink: 0,
        }}
      >
        Select →
      </span>
    </button>
  );
}

function SearchStep({
  onSelect,
  onClose,
}: {
  onSelect: (result: TMDBResult) => void;
  onClose: () => void;
}) {
  const [query, setQuery] = useState("");
  const lookup = useLookupMovies();

  function handleSearch() {
    if (!query.trim()) return;
    lookup.mutate({ query: query.trim() });
  }

  return (
    <>
      <div style={{ display: "flex", gap: 8 }}>
        <input
          style={{ ...dialogInputStyle, flex: 1 }}
          value={query}
          onChange={(e) => setQuery(e.currentTarget.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSearch()}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="Search TMDB — title or year…"
          autoFocus
        />
        <button
          onClick={handleSearch}
          disabled={lookup.isPending || !query.trim()}
          style={{
            background:
              lookup.isPending || !query.trim()
                ? "var(--color-bg-subtle)"
                : "var(--color-accent)",
            color:
              lookup.isPending || !query.trim()
                ? "var(--color-text-muted)"
                : "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            fontWeight: 500,
            cursor:
              lookup.isPending || !query.trim() ? "not-allowed" : "pointer",
            whiteSpace: "nowrap",
          }}
        >
          {lookup.isPending ? "Searching…" : "Search"}
        </button>
      </div>

      {lookup.isError && (
        <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>
          {lookup.error.message}
        </p>
      )}

      {lookup.data && lookup.data.length === 0 && (
        <p
          style={{
            margin: 0,
            fontSize: 13,
            color: "var(--color-text-muted)",
            textAlign: "center",
            padding: "24px 0",
          }}
        >
          No results found. Try a different search.
        </p>
      )}

      {lookup.data && lookup.data.length > 0 && (
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 1,
            maxHeight: 360,
            overflowY: "auto",
            borderRadius: 6,
            border: "1px solid var(--color-border-subtle)",
            background: "var(--color-bg-elevated)",
          }}
        >
          {lookup.data.map((result) => (
            <SearchResultRow
              key={result.tmdb_id}
              result={result}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}

      <div style={{ display: "flex", justifyContent: "flex-end" }}>
        <button
          onClick={onClose}
          style={{
            background: "none",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            color: "var(--color-text-secondary)",
            cursor: "pointer",
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background =
              "var(--color-bg-elevated)";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = "none";
          }}
        >
          Cancel
        </button>
      </div>
    </>
  );
}

function ConfigureStep({
  result,
  onBack,
  onSuccess,
  onClose,
}: {
  result: TMDBResult;
  onBack: () => void;
  onSuccess: () => void;
  onClose: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const addMovie = useAddMovie();

  const [libraryId, setLibraryId] = useState(libraries?.[0]?.id ?? "");
  const [profileId, setProfileId] = useState(profiles?.[0]?.id ?? "");
  const [monitored, setMonitored] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const posterSrc = result.poster_path
    ? `https://image.tmdb.org/t/p/w185${result.poster_path}`
    : null;

  function handleAdd() {
    if (!libraryId) {
      setError("Select a library.");
      return;
    }
    if (!profileId) {
      setError("Select a quality profile.");
      return;
    }
    addMovie.mutate(
      {
        tmdb_id: result.tmdb_id,
        library_id: libraryId,
        quality_profile_id: profileId,
        monitored,
      },
      {
        onSuccess,
        onError: (e) => {
          const msg = (e as Error).message;
          if (msg.includes("409") || msg.toLowerCase().includes("already")) {
            setError("This movie is already in the library.");
          } else {
            setError(msg);
          }
        },
      }
    );
  }

  return (
    <>
      {/* Movie preview */}
      <div
        style={{
          display: "flex",
          gap: 16,
          padding: 16,
          background: "var(--color-bg-elevated)",
          borderRadius: 8,
          border: "1px solid var(--color-border-subtle)",
        }}
      >
        {posterSrc ? (
          <img
            src={posterSrc}
            alt={result.title}
            style={{
              width: 64,
              height: 96,
              objectFit: "cover",
              borderRadius: 4,
              flexShrink: 0,
            }}
          />
        ) : (
          <div
            style={{
              width: 64,
              height: 96,
              background: "var(--color-bg-subtle)",
              borderRadius: 4,
              flexShrink: 0,
            }}
          />
        )}
        <div style={{ flex: 1, minWidth: 0 }}>
          <p
            style={{
              margin: 0,
              fontSize: 15,
              fontWeight: 600,
              color: "var(--color-text-primary)",
            }}
          >
            {result.title}
          </p>
          {result.year > 0 && (
            <p
              style={{
                margin: "2px 0 0",
                fontSize: 13,
                color: "var(--color-text-muted)",
              }}
            >
              {result.year}
            </p>
          )}
          {result.overview && (
            <p
              style={{
                margin: "8px 0 0",
                fontSize: 12,
                color: "var(--color-text-secondary)",
                lineHeight: 1.5,
                overflow: "hidden",
                display: "-webkit-box",
                WebkitLineClamp: 3,
                WebkitBoxOrient: "vertical",
              }}
            >
              {result.overview}
            </p>
          )}
        </div>
      </div>

      {/* Config fields */}
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <div>
          <label style={dialogLabelStyle}>Library *</label>
          <select
            style={{ ...dialogInputStyle, cursor: "pointer" }}
            value={libraryId || (libraries?.[0]?.id ?? "")}
            onChange={(e) => {
              setLibraryId(e.currentTarget.value);
              setError(null);
            }}
            onFocus={focusBorder}
            onBlur={blurBorder}
          >
            {!libraries?.length && (
              <option value="">No libraries configured</option>
            )}
            {libraries?.map((lib) => (
              <option key={lib.id} value={lib.id}>
                {lib.name} — {lib.root_path}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label style={dialogLabelStyle}>Quality Profile *</label>
          <select
            style={{ ...dialogInputStyle, cursor: "pointer" }}
            value={profileId || (profiles?.[0]?.id ?? "")}
            onChange={(e) => {
              setProfileId(e.currentTarget.value);
              setError(null);
            }}
            onFocus={focusBorder}
            onBlur={blurBorder}
          >
            {!profiles?.length && (
              <option value="">No profiles configured</option>
            )}
            {profiles?.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>

        <label
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            cursor: "pointer",
            userSelect: "none",
          }}
        >
          <input
            type="checkbox"
            checked={monitored}
            onChange={(e) => setMonitored(e.currentTarget.checked)}
            style={{
              width: 16,
              height: 16,
              cursor: "pointer",
              accentColor: "var(--color-accent)",
            }}
          />
          <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
            Monitor for releases
          </span>
        </label>
      </div>

      {error && (
        <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>
          {error}
        </p>
      )}

      <div style={{ display: "flex", justifyContent: "space-between" }}>
        <button
          onClick={onBack}
          style={{
            background: "none",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            color: "var(--color-text-secondary)",
            cursor: "pointer",
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background =
              "var(--color-bg-elevated)";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = "none";
          }}
        >
          ← Back
        </button>
        <div style={{ display: "flex", gap: 8 }}>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-bg-elevated)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = "none";
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleAdd}
            disabled={addMovie.isPending}
            style={{
              background: addMovie.isPending
                ? "var(--color-bg-subtle)"
                : "var(--color-accent)",
              color: addMovie.isPending
                ? "var(--color-text-muted)"
                : "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 500,
              cursor: addMovie.isPending ? "not-allowed" : "pointer",
            }}
            onMouseEnter={(e) => {
              if (!addMovie.isPending)
                (e.currentTarget as HTMLButtonElement).style.background =
                  "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              if (!addMovie.isPending)
                (e.currentTarget as HTMLButtonElement).style.background =
                  "var(--color-accent)";
            }}
          >
            {addMovie.isPending ? "Adding…" : "Add Movie"}
          </button>
        </div>
      </div>
    </>
  );
}

function AddMovieDialog({ onClose }: { onClose: () => void }) {
  const [selected, setSelected] = useState<TMDBResult | null>(null);

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.6)",
        backdropFilter: "blur(2px)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 100,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: "var(--color-bg-surface)",
          border: "1px solid var(--color-border-subtle)",
          borderRadius: 12,
          padding: 24,
          width: 580,
          maxWidth: "calc(100vw - 48px)",
          maxHeight: "calc(100vh - 80px)",
          overflowY: "auto",
          boxShadow: "var(--shadow-modal)",
          display: "flex",
          flexDirection: "column",
          gap: 20,
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <h2
            style={{
              margin: 0,
              fontSize: 16,
              fontWeight: 600,
              color: "var(--color-text-primary)",
            }}
          >
            {selected ? "Configure Movie" : "Add Movie"}
          </h2>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              fontSize: 18,
              lineHeight: 1,
              padding: "4px 6px",
              borderRadius: 4,
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color =
                "var(--color-text-primary)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color =
                "var(--color-text-muted)";
            }}
          >
            ✕
          </button>
        </div>

        {selected ? (
          <ConfigureStep
            result={selected}
            onBack={() => setSelected(null)}
            onSuccess={onClose}
            onClose={onClose}
          />
        ) : (
          <SearchStep onSelect={setSelected} onClose={onClose} />
        )}
      </div>
    </div>
  );
}

// ── Skeleton grid ─────────────────────────────────────────────────────────────

function GridSkeleton() {
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))",
        gap: 16,
      }}
    >
      {Array.from({ length: 24 }).map((_, i) => (
        <div key={i}>
          <div
            style={{
              paddingBottom: "150%",
              position: "relative",
              borderRadius: 8,
              overflow: "hidden",
            }}
          >
            <div className="skeleton" style={{ position: "absolute", inset: 0 }} />
          </div>
          <div
            className="skeleton"
            style={{ height: 14, borderRadius: 3, marginTop: 8, width: "85%" }}
          />
          <div
            className="skeleton"
            style={{ height: 12, borderRadius: 3, marginTop: 4, width: "55%" }}
          />
        </div>
      ))}
    </div>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

export default function Dashboard() {
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    return (localStorage.getItem("gallery-view") as ViewMode) || "grid";
  });

  const [search, setSearch] = useState("");
  const [monitoredFilter, setMonitoredFilter] =
    useState<MonitoredFilter>("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [onDiskFilter, setOnDiskFilter] = useState<OnDiskFilter>("all");
  const [libraryFilter, setLibraryFilter] = useState("");
  const [sortField, setSortField] = useState<SortField>("title");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [showAdd, setShowAdd] = useState(false);

  const { data, isLoading, error } = useMovies({ per_page: 2000 });
  const { data: libraries } = useLibraries();

  // Derive unique status values from actual data
  const statuses = useMemo(() => {
    if (!data?.movies) return [] as string[];
    return [...new Set(data.movies.map((m) => m.status))].sort();
  }, [data?.movies]);

  // Client-side filtering + sorting
  const filtered = useMemo(() => {
    if (!data?.movies) return [] as Movie[];
    let result = data.movies;

    if (search.trim()) {
      const q = search.toLowerCase();
      result = result.filter(
        (m) =>
          m.title.toLowerCase().includes(q) ||
          m.original_title.toLowerCase().includes(q)
      );
    }

    if (monitoredFilter !== "all") {
      const want = monitoredFilter === "monitored";
      result = result.filter((m) => m.monitored === want);
    }

    if (statusFilter !== "all") {
      result = result.filter((m) => m.status === statusFilter);
    }

    if (onDiskFilter !== "all") {
      const want = onDiskFilter === "on_disk";
      result = result.filter((m) => !!m.path === want);
    }

    if (libraryFilter) {
      result = result.filter((m) => m.library_id === libraryFilter);
    }

    return [...result].sort((a, b) => {
      let cmp = 0;
      if (sortField === "title") cmp = a.title.localeCompare(b.title);
      else if (sortField === "year") cmp = (a.year || 0) - (b.year || 0);
      else if (sortField === "added_at")
        cmp = a.added_at.localeCompare(b.added_at);
      return sortDir === "asc" ? cmp : -cmp;
    });
  }, [
    data?.movies,
    search,
    monitoredFilter,
    statusFilter,
    onDiskFilter,
    libraryFilter,
    sortField,
    sortDir,
  ]);

  function handleViewMode(mode: ViewMode) {
    setViewMode(mode);
    localStorage.setItem("gallery-view", mode);
  }

  function clearFilters() {
    setSearch("");
    setMonitoredFilter("all");
    setStatusFilter("all");
    setOnDiskFilter("all");
    setLibraryFilter("");
  }

  const totalCount = data?.total ?? 0;
  const hasFilters =
    !!search ||
    monitoredFilter !== "all" ||
    statusFilter !== "all" ||
    onDiskFilter !== "all" ||
    !!libraryFilter;

  return (
    <div style={{ padding: 24 }}>
      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          justifyContent: "space-between",
          marginBottom: 20,
          gap: 16,
        }}
      >
        <div>
          <h1
            style={{
              margin: 0,
              fontSize: 20,
              fontWeight: 600,
              color: "var(--color-text-primary)",
              letterSpacing: "-0.01em",
            }}
          >
            Library
          </h1>
          {!isLoading && data && (
            <p
              style={{
                margin: "4px 0 0",
                fontSize: 13,
                color: "var(--color-text-secondary)",
              }}
            >
              {hasFilters
                ? `${filtered.length.toLocaleString()} of ${totalCount.toLocaleString()} movies`
                : `${totalCount.toLocaleString()} ${totalCount === 1 ? "movie" : "movies"}`}
            </p>
          )}
        </div>
        <button
          onClick={() => setShowAdd(true)}
          style={{
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            fontWeight: 500,
            cursor: "pointer",
            flexShrink: 0,
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background =
              "var(--color-accent-hover)";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background =
              "var(--color-accent)";
          }}
        >
          + Add Movie
        </button>
      </div>

      {/* Filter bar */}
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: 8,
          marginBottom: 20,
          alignItems: "center",
        }}
      >
        {/* Search */}
        <input
          type="search"
          placeholder="Search by title…"
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          style={{ ...ctrlStyle, minWidth: 180 }}
          onFocus={(e) => {
            (e.currentTarget as HTMLInputElement).style.borderColor =
              "var(--color-accent)";
          }}
          onBlur={(e) => {
            (e.currentTarget as HTMLInputElement).style.borderColor =
              "var(--color-border-default)";
          }}
        />

        {/* Monitored */}
        <select
          value={monitoredFilter}
          onChange={(e) =>
            setMonitoredFilter(e.currentTarget.value as MonitoredFilter)
          }
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="all">All monitored</option>
          <option value="monitored">Monitored</option>
          <option value="unmonitored">Unmonitored</option>
        </select>

        {/* Status */}
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.currentTarget.value)}
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="all">All statuses</option>
          {statuses.map((s) => (
            <option key={s} value={s}>
              {s.charAt(0).toUpperCase() + s.slice(1).replace(/_/g, " ")}
            </option>
          ))}
        </select>

        {/* On disk */}
        <select
          value={onDiskFilter}
          onChange={(e) =>
            setOnDiskFilter(e.currentTarget.value as OnDiskFilter)
          }
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="all">All files</option>
          <option value="on_disk">On disk</option>
          <option value="missing">Missing</option>
        </select>

        {/* Library — only shown when multiple libraries exist */}
        {libraries && libraries.length > 1 && (
          <select
            value={libraryFilter}
            onChange={(e) => setLibraryFilter(e.currentTarget.value)}
            style={{ ...ctrlStyle, cursor: "pointer" }}
          >
            <option value="">All libraries</option>
            {libraries.map((lib) => (
              <option key={lib.id} value={lib.id}>
                {lib.name}
              </option>
            ))}
          </select>
        )}

        {/* Sort field */}
        <select
          value={sortField}
          onChange={(e) => setSortField(e.currentTarget.value as SortField)}
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="title">Title</option>
          <option value="year">Year</option>
          <option value="added_at">Date Added</option>
        </select>

        {/* Sort direction */}
        <button
          onClick={() => setSortDir((d) => (d === "asc" ? "desc" : "asc"))}
          title={sortDir === "asc" ? "Ascending — click to flip" : "Descending — click to flip"}
          style={{
            ...ctrlStyle,
            cursor: "pointer",
            padding: "7px 10px",
            fontFamily: "var(--font-family-mono)",
            fontSize: 12,
            color: "var(--color-text-secondary)",
          }}
        >
          {sortDir === "asc" ? "A→Z" : "Z→A"}
        </button>

        {/* Spacer */}
        <div style={{ flex: 1 }} />

        {/* View toggle */}
        <div
          style={{
            display: "flex",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            overflow: "hidden",
          }}
        >
          <button
            onClick={() => handleViewMode("grid")}
            title="Grid view"
            style={{
              background:
                viewMode === "grid"
                  ? "var(--color-bg-elevated)"
                  : "transparent",
              border: "none",
              padding: "6px 10px",
              cursor: "pointer",
              color:
                viewMode === "grid"
                  ? "var(--color-text-primary)"
                  : "var(--color-text-muted)",
              display: "flex",
              alignItems: "center",
            }}
          >
            <LayoutGrid size={16} strokeWidth={1.5} />
          </button>
          <button
            onClick={() => handleViewMode("list")}
            title="List view"
            style={{
              background:
                viewMode === "list"
                  ? "var(--color-bg-elevated)"
                  : "transparent",
              border: "none",
              borderLeft: "1px solid var(--color-border-default)",
              padding: "6px 10px",
              cursor: "pointer",
              color:
                viewMode === "list"
                  ? "var(--color-text-primary)"
                  : "var(--color-text-muted)",
              display: "flex",
              alignItems: "center",
            }}
          >
            <List size={16} strokeWidth={1.5} />
          </button>
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        viewMode === "grid" ? (
          <GridSkeleton />
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {[1, 2, 3, 4, 5].map((i) => (
              <div
                key={i}
                className="skeleton"
                style={{ height: 60, borderRadius: 4 }}
              />
            ))}
          </div>
        )
      ) : error ? (
        <div
          style={{
            padding: 48,
            textAlign: "center",
            fontSize: 13,
            color: "var(--color-text-muted)",
          }}
        >
          Failed to load movies.
        </div>
      ) : !data?.movies.length ? (
        <div style={{ padding: 80, textAlign: "center" }}>
          <p
            style={{
              margin: 0,
              fontSize: 15,
              fontWeight: 500,
              color: "var(--color-text-secondary)",
            }}
          >
            No movies in your library
          </p>
          <p
            style={{
              margin: "8px 0 20px",
              fontSize: 13,
              color: "var(--color-text-muted)",
            }}
          >
            Search TMDB to add your first movie.
          </p>
          <button
            onClick={() => setShowAdd(true)}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 500,
              cursor: "pointer",
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-accent)";
            }}
          >
            + Add Movie
          </button>
        </div>
      ) : filtered.length === 0 ? (
        <div style={{ padding: 48, textAlign: "center" }}>
          <p
            style={{
              margin: 0,
              fontSize: 14,
              color: "var(--color-text-muted)",
            }}
          >
            No movies match the current filters.
          </p>
          <button
            onClick={clearFilters}
            style={{
              marginTop: 12,
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "6px 14px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
          >
            Clear filters
          </button>
        </div>
      ) : viewMode === "grid" ? (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))",
            gap: 16,
          }}
        >
          {filtered.map((movie) => (
            <PosterCard key={movie.id} movie={movie} />
          ))}
        </div>
      ) : (
        <div
          style={{
            background: "var(--color-bg-surface)",
            border: "1px solid var(--color-border-subtle)",
            borderRadius: 8,
            boxShadow: "var(--shadow-card)",
            overflow: "hidden",
          }}
        >
          <table
            style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}
          >
            <thead>
              <tr
                style={{
                  borderBottom: "1px solid var(--color-border-subtle)",
                }}
              >
                <th style={thStyle} />
                <th style={thStyle}>Title</th>
                <th style={thStyle}>Year</th>
                <th style={thStyle}>Status</th>
                <th style={thStyle}>Monitored</th>
                <th style={thStyle}>File</th>
                <th style={thStyle} />
              </tr>
            </thead>
            <tbody>
              {filtered.map((movie, i) => (
                <ListRow
                  key={movie.id}
                  movie={movie}
                  isLast={i === filtered.length - 1}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Legend — only shown in grid view with at least one movie */}
      {viewMode === "grid" && filtered.length > 0 && (
        <div style={{ display: "flex", gap: 20, alignItems: "center" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span
              style={{
                display: "inline-flex",
                width: 14,
                height: 14,
                borderRadius: "50%",
                background: "var(--color-success)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 8,
                color: "white",
                fontWeight: 700,
                flexShrink: 0,
              }}
            >
              ✓
            </span>
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>On disk</span>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span
              style={{
                display: "inline-flex",
                width: 14,
                height: 14,
                borderRadius: "50%",
                background: "var(--color-warning)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 8,
                color: "white",
                fontWeight: 700,
                flexShrink: 0,
              }}
            >
              ●
            </span>
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>Monitored, missing</span>
          </div>
        </div>
      )}

      {showAdd && <AddMovieDialog onClose={() => setShowAdd(false)} />}
    </div>
  );
}
