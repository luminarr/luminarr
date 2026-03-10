import { useState, useCallback, useEffect } from "react";
import { useMovieReleases, useGrabRelease, type GrabReleaseRequest } from "@/api/movies";
import type { Release } from "@/types";
import { formatBytes } from "@/lib/utils";
import ScoreChip from "@/components/ScoreChip";
import IndexerPill from "@/components/IndexerPill";

// ── Quality badge ─────────────────────────────────────────────────────────────

function QualityBadge({ quality }: { quality: Release["quality"] }) {
  const label = [quality.resolution, quality.source].filter(Boolean).join(" ");
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 6px",
        borderRadius: 4,
        fontSize: 10,
        fontWeight: 600,
        textTransform: "uppercase",
        letterSpacing: "0.05em",
        background: "color-mix(in srgb, var(--color-accent) 12%, transparent)",
        color: "var(--color-accent)",
      }}
    >
      {label || "Unknown"}
    </span>
  );
}

// ── Release row ───────────────────────────────────────────────────────────────

interface ReleaseRowProps {
  release: Release;
  grabbed: boolean;
  grabError?: string;
  onGrab: () => void;
  isPending: boolean;
}

function ReleaseRow({ release, grabbed, grabError, onGrab, isPending }: ReleaseRowProps) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 14px",
        background: "var(--color-bg-elevated)",
        borderRadius: 6,
        border: "1px solid var(--color-border-subtle)",
        flexShrink: 0,
      }}
    >
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 12,
            color: "var(--color-text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            fontFamily: "var(--font-family-mono)",
          }}
          title={release.title}
        >
          {release.title}
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 10, marginTop: 4, flexWrap: "wrap" }}>
          <QualityBadge quality={release.quality} />
          <ScoreChip breakdown={release.score_breakdown} />
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
            {formatBytes(release.size)}
          </span>
          {release.seeds !== undefined && (
            <span style={{ fontSize: 11, color: "var(--color-success)" }}>↑{release.seeds}</span>
          )}
          {release.peers !== undefined && (
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>↓{release.peers}</span>
          )}
          <IndexerPill name={release.indexer} />
          {release.age_days !== undefined && release.age_days > 0 && (
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
              {Math.round(release.age_days)}d old
            </span>
          )}
        </div>
        {grabError && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-danger)" }}>{grabError}</p>
        )}
      </div>

      {grabbed ? (
        <span style={{ fontSize: 12, color: "var(--color-success)", flexShrink: 0 }}>Grabbed ✓</span>
      ) : (
        <button
          onClick={onGrab}
          disabled={isPending}
          style={{
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "5px 14px",
            fontSize: 12,
            fontWeight: 500,
            cursor: isPending ? "not-allowed" : "pointer",
            flexShrink: 0,
          }}
          onMouseEnter={(e) => {
            if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)";
          }}
          onMouseLeave={(e) => {
            if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
          }}
        >
          {isPending ? "…" : "Grab"}
        </button>
      )}
    </div>
  );
}

// ── Modal ─────────────────────────────────────────────────────────────────────

interface ManualSearchModalProps {
  movieId: string;
  movieTitle: string;
  onClose: () => void;
}

export function ManualSearchModal({ movieId, movieTitle, onClose }: ManualSearchModalProps) {
  const { data, isLoading, error, refetch } = useMovieReleases(movieId);
  const grab = useGrabRelease();
  const [grabbedGuids, setGrabbedGuids] = useState<Set<string>>(new Set());
  const [pendingGuids, setPendingGuids] = useState<Set<string>>(new Set());
  const [grabErrors, setGrabErrors] = useState<Record<string, string>>({});
  const [sortBySeeders, setSortBySeeders] = useState(false);

  // Close on Escape
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const handleGrab = useCallback((release: Release) => {
    const body: GrabReleaseRequest & { movieId: string } = {
      movieId,
      guid: release.guid,
      title: release.title,
      protocol: release.protocol,
      download_url: release.download_url,
      size: release.size,
    };
    setPendingGuids((prev) => new Set([...prev, release.guid]));
    grab.mutate(body, {
      onSuccess: () => {
        setPendingGuids((prev) => { const n = new Set(prev); n.delete(release.guid); return n; });
        setGrabbedGuids((prev) => new Set([...prev, release.guid]));
      },
      onError: (e) => {
        setPendingGuids((prev) => { const n = new Set(prev); n.delete(release.guid); return n; });
        setGrabErrors((prev) => ({ ...prev, [release.guid]: e.message }));
        setTimeout(() => setGrabErrors((prev) => { const n = { ...prev }; delete n[release.guid]; return n; }), 5000);
      },
    });
  }, [movieId, grab]);

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
        zIndex: 200,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: "var(--color-bg-surface)",
          border: "1px solid var(--color-border-subtle)",
          borderRadius: 12,
          width: 760,
          maxWidth: "calc(100vw - 48px)",
          maxHeight: "85vh",
          boxShadow: "var(--shadow-modal)",
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div
          style={{
            padding: "16px 20px",
            borderBottom: "1px solid var(--color-border-subtle)",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            flexShrink: 0,
          }}
        >
          <div>
            <div style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text-primary)" }}>
              Manual Search
            </div>
            <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 2 }}>
              {movieTitle}
            </div>
          </div>
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
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; }}
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: "auto", padding: "16px 20px" }}>
          {isLoading && (
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="skeleton" style={{ height: 60, borderRadius: 6 }} />
              ))}
            </div>
          )}

          {error && (
            <div style={{ textAlign: "center", padding: "24px 0" }}>
              <p style={{ margin: "0 0 12px", fontSize: 13, color: "var(--color-text-muted)" }}>
                Failed to search indexers: {(error as Error).message}
              </p>
              <button
                onClick={() => refetch()}
                style={{
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "6px 14px",
                  fontSize: 12,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                }}
              >
                Retry
              </button>
            </div>
          )}

          {!isLoading && !error && data?.length === 0 && (
            <div style={{ textAlign: "center", padding: "32px 0" }}>
              <p style={{ margin: 0, fontSize: 14, fontWeight: 500, color: "var(--color-text-secondary)" }}>
                No releases found
              </p>
              <p style={{ margin: "6px 0 16px", fontSize: 13, color: "var(--color-text-muted)" }}>
                No results from any configured indexer.
              </p>
              <button
                onClick={() => refetch()}
                style={{
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "6px 14px",
                  fontSize: 12,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                }}
              >
                Search Again
              </button>
            </div>
          )}

          {data && data.length > 0 && (
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 8 }}>
                <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>
                  {data.length} release{data.length !== 1 ? "s" : ""} found
                </p>
                <button
                  onClick={() => setSortBySeeders((prev) => !prev)}
                  style={{
                    background: sortBySeeders
                      ? "color-mix(in srgb, var(--color-accent) 15%, transparent)"
                      : "var(--color-bg-elevated)",
                    border: `1px solid ${sortBySeeders ? "var(--color-accent)" : "var(--color-border-default)"}`,
                    borderRadius: 6,
                    padding: "3px 10px",
                    fontSize: 11,
                    fontWeight: 500,
                    color: sortBySeeders ? "var(--color-accent)" : "var(--color-text-secondary)",
                    cursor: "pointer",
                  }}
                >
                  {sortBySeeders ? "Seeds ↓" : "Sort: Seeds"}
                </button>
              </div>
              {(sortBySeeders ? [...data].sort((a, b) => (b.seeds ?? 0) - (a.seeds ?? 0)) : data).map((release) => (
                <ReleaseRow
                  key={release.guid}
                  release={release}
                  grabbed={grabbedGuids.has(release.guid)}
                  grabError={grabErrors[release.guid]}
                  onGrab={() => handleGrab(release)}
                  isPending={pendingGuids.has(release.guid)}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
