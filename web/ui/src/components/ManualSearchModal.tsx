import { useState, useCallback, useMemo } from "react";
import { TriangleAlert, HelpCircle, ChevronDown, ChevronUp } from "lucide-react";
import { useMovieReleases, useGrabRelease, useExplainReleases, type GrabReleaseRequest } from "@/api/movies";
import type { Release, QualityConflict, ReleaseDecision, ExplainResult } from "@/types";
import { formatBytes, sortReleases, RELEASE_SORT_LABELS, type ReleaseSortField } from "@/lib/utils";
import IndexerPill from "@/components/IndexerPill";
import QualityBadge from "@/components/QualityBadge";
import Modal from "@/components/Modal";

// ── Conflict pills ─────────────────────────────────────────────────────────────

function ConflictPills({ conflicts }: { conflicts: QualityConflict[] }) {
  if (conflicts.length === 0) return null;
  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: 4, marginTop: 6 }}>
      {conflicts.map((c, i) => {
        const isWarning = c.severity === "warning";
        return (
          <span
            key={i}
            title={`${c.dimension}: ${c.current} → ${c.candidate}`}
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 4,
              padding: "2px 7px",
              borderRadius: 4,
              fontSize: 10,
              fontWeight: 500,
              background: isWarning
                ? "color-mix(in srgb, var(--color-warning) 14%, transparent)"
                : "color-mix(in srgb, var(--color-text-muted) 10%, transparent)",
              color: isWarning ? "var(--color-warning)" : "var(--color-text-muted)",
              border: `1px solid ${isWarning ? "color-mix(in srgb, var(--color-warning) 30%, transparent)" : "var(--color-border-subtle)"}`,
            }}
          >
            {isWarning && <TriangleAlert size={10} strokeWidth={2} style={{ flexShrink: 0 }} />}
            {c.summary}
          </span>
        );
      })}
    </div>
  );
}

// ── Release row ────────────────────────────────────────────────────────────────

interface ReleaseRowProps {
  release: Release;
  grabbed: boolean;
  grabError?: string;
  onGrab: () => void;
  isPending: boolean;
}

function ReleaseRow({ release, grabbed, grabError, onGrab, isPending }: ReleaseRowProps) {
  const hasConflicts = (release.conflicts?.length ?? 0) > 0;
  return (
    <div
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: 12,
        padding: "10px 14px",
        background: "var(--color-bg-elevated)",
        borderRadius: 6,
        border: `1px solid ${hasConflicts ? "color-mix(in srgb, var(--color-warning) 25%, var(--color-border-subtle))" : "var(--color-border-subtle)"}`,
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
          {release.edition && (
            <span
              style={{
                display: "inline-block",
                padding: "1px 6px",
                borderRadius: 4,
                fontSize: 10,
                fontWeight: 600,
                background: "color-mix(in srgb, var(--color-info, #3b82f6) 15%, transparent)",
                color: "var(--color-info, #3b82f6)",
              }}
            >
              {release.edition}
            </span>
          )}
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
        {release.conflicts && release.conflicts.length > 0 && (
          <ConflictPills conflicts={release.conflicts} />
        )}
        {grabError && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-danger)" }}>{grabError}</p>
        )}
      </div>

      {grabbed ? (
        <span style={{ fontSize: 12, color: "var(--color-success)", flexShrink: 0, marginTop: 2 }}>Grabbed ✓</span>
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
            marginTop: 2,
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

// ── Decision panel ─────────────────────────────────────────────────────────────

function OutcomeBadge({ outcome }: { outcome: ReleaseDecision["outcome"] }) {
  const isGrabbed = outcome === "grabbed";
  return (
    <span
      style={{
        display: "inline-block",
        padding: "1px 7px",
        borderRadius: 4,
        fontSize: 10,
        fontWeight: 600,
        textTransform: "uppercase",
        letterSpacing: "0.05em",
        flexShrink: 0,
        background: isGrabbed
          ? "color-mix(in srgb, var(--color-success) 15%, transparent)"
          : "color-mix(in srgb, var(--color-text-muted) 12%, transparent)",
        color: isGrabbed ? "var(--color-success)" : "var(--color-text-muted)",
      }}
    >
      {outcome}
    </span>
  );
}

function DecisionRow({ decision }: { decision: ReleaseDecision }) {
  return (
    <div
      style={{
        padding: "8px 0",
        borderBottom: "1px solid var(--color-border-subtle)",
        display: "flex",
        alignItems: "flex-start",
        gap: 10,
      }}
    >
      <OutcomeBadge outcome={decision.outcome} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 11,
            color: "var(--color-text-primary)",
            fontFamily: "var(--font-family-mono)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
          title={decision.title}
        >
          {decision.title}
        </div>
        <div style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 2 }}>
          {decision.explanation}
        </div>
      </div>
    </div>
  );
}

function DecisionPanel({ movieId, explain }: { movieId: string; explain: ExplainResult | undefined }) {
  const [expanded, setExpanded] = useState(false);

  if (!explain) return null;

  return (
    <div
      style={{
        margin: "0 0 16px",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        background: "var(--color-bg-surface)",
        overflow: "hidden",
      }}
    >
      {/* Summary header — always visible */}
      <button
        onClick={() => setExpanded((v) => !v)}
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          width: "100%",
          padding: "10px 14px",
          background: "none",
          border: "none",
          cursor: "pointer",
          textAlign: "left",
        }}
      >
        <div style={{ flex: 1, minWidth: 0 }}>
          <span style={{ fontSize: 12, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Profile: {explain.profile_name}
          </span>
          {explain.current_file && (
            <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginLeft: 10 }}>
              Current: {explain.current_file.name || `${explain.current_file.resolution} ${explain.current_file.source}`}
            </span>
          )}
        </div>
        <span style={{ fontSize: 11, color: "var(--color-text-muted)", flexShrink: 0 }}>
          {explain.decisions.length} decision{explain.decisions.length !== 1 ? "s" : ""}
        </span>
        {expanded
          ? <ChevronUp size={14} style={{ color: "var(--color-text-muted)", flexShrink: 0 }} />
          : <ChevronDown size={14} style={{ color: "var(--color-text-muted)", flexShrink: 0 }} />
        }
      </button>

      {/* Decision list */}
      {expanded && (
        <div
          style={{
            padding: "0 14px 10px",
            borderTop: "1px solid var(--color-border-subtle)",
            maxHeight: 300,
            overflowY: "auto",
          }}
        >
          {explain.decisions.length === 0 ? (
            <p style={{ margin: "10px 0 0", fontSize: 12, color: "var(--color-text-muted)" }}>
              No decisions recorded.
            </p>
          ) : (
            explain.decisions.map((d) => (
              <DecisionRow key={d.guid} decision={d} />
            ))
          )}
        </div>
      )}
    </div>
  );
}

// ── Modal ──────────────────────────────────────────────────────────────────────

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
  const [sortField, setSortField] = useState<ReleaseSortField>("seeds");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");
  const [showExplain, setShowExplain] = useState(false);

  const explain = useExplainReleases(movieId, showExplain);

  const sortedReleases = useMemo(
    () => (data ? sortReleases(data, sortField, sortDir) : []),
    [data, sortField, sortDir]
  );

  function toggleSort(field: ReleaseSortField) {
    if (sortField === field) {
      setSortDir((d) => (d === "desc" ? "asc" : "desc"));
    } else {
      setSortField(field);
      setSortDir("desc");
    }
  }

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
    <Modal onClose={onClose} width={760} maxHeight="85vh">
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
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            {/* Why? button */}
            <button
              onClick={() => setShowExplain((v) => !v)}
              title="Show scoring decisions"
              style={{
                display: "flex",
                alignItems: "center",
                gap: 5,
                background: showExplain
                  ? "color-mix(in srgb, var(--color-accent) 12%, transparent)"
                  : "none",
                border: showExplain
                  ? "1px solid color-mix(in srgb, var(--color-accent) 30%, transparent)"
                  : "1px solid transparent",
                borderRadius: 6,
                padding: "5px 10px",
                fontSize: 12,
                fontWeight: 500,
                color: showExplain ? "var(--color-accent)" : "var(--color-text-muted)",
                cursor: "pointer",
                transition: "color 120ms ease, background 120ms ease, border-color 120ms ease",
              }}
              onMouseEnter={(e) => {
                if (!showExplain) {
                  (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)";
                }
              }}
              onMouseLeave={(e) => {
                if (!showExplain) {
                  (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)";
                }
              }}
            >
              <HelpCircle size={14} strokeWidth={1.75} />
              Why?
              {explain.isLoading && <span style={{ marginLeft: 2 }}>…</span>}
            </button>

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
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: "auto", padding: "16px 20px" }}>
          {/* Decision panel (Feature 2) */}
          {showExplain && (
            explain.isLoading ? (
              <div className="skeleton" style={{ height: 44, borderRadius: 8, marginBottom: 16 }} />
            ) : explain.error ? (
              <div
                style={{
                  padding: "10px 14px",
                  marginBottom: 16,
                  borderRadius: 8,
                  border: "1px solid var(--color-border-subtle)",
                  fontSize: 12,
                  color: "var(--color-danger)",
                }}
              >
                Could not load decisions: {(explain.error as Error).message}
              </div>
            ) : explain.data ? (
              <DecisionPanel movieId={movieId} explain={explain.data} />
            ) : null
          )}

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
                <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                  <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginRight: 4 }}>Sort:</span>
                  {(Object.keys(RELEASE_SORT_LABELS) as ReleaseSortField[]).map((field) => (
                    <button
                      key={field}
                      onClick={() => toggleSort(field)}
                      aria-label={`Sort by ${RELEASE_SORT_LABELS[field]}`}
                      style={{
                        background: sortField === field ? "var(--color-bg-elevated)" : "transparent",
                        border: sortField === field ? "1px solid var(--color-border-default)" : "1px solid transparent",
                        borderRadius: 4,
                        padding: "2px 8px",
                        fontSize: 11,
                        color: sortField === field ? "var(--color-text-primary)" : "var(--color-text-muted)",
                        cursor: "pointer",
                        fontWeight: sortField === field ? 600 : 400,
                      }}
                    >
                      {RELEASE_SORT_LABELS[field]} {sortField === field ? (sortDir === "desc" ? "↓" : "↑") : ""}
                    </button>
                  ))}
                </div>
              </div>
              {sortedReleases.map((release) => (
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
    </Modal>
  );
}
