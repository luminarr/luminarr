import { Link } from "react-router-dom";
import { useHistory } from "@/api/history";
import { formatBytes } from "@/lib/utils";
import type { GrabHistory } from "@/types";

// ── Helpers ─────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// ── Status badge ─────────────────────────────────────────────────────────────

const statusStyles: Record<string, { bg: string; color: string; label: string }> = {
  completed: {
    bg: "color-mix(in srgb, var(--color-success) 15%, transparent)",
    color: "var(--color-success)",
    label: "Completed",
  },
  downloading: {
    bg: "color-mix(in srgb, var(--color-accent) 15%, transparent)",
    color: "var(--color-accent)",
    label: "Downloading",
  },
  queued: {
    bg: "color-mix(in srgb, var(--color-warning) 15%, transparent)",
    color: "var(--color-warning)",
    label: "Queued",
  },
  failed: {
    bg: "color-mix(in srgb, var(--color-danger) 15%, transparent)",
    color: "var(--color-danger)",
    label: "Failed",
  },
  removed: {
    bg: "color-mix(in srgb, var(--color-text-muted) 15%, transparent)",
    color: "var(--color-text-muted)",
    label: "Removed",
  },
};

function StatusBadge({ status }: { status: string }) {
  const style = statusStyles[status] ?? {
    bg: "color-mix(in srgb, var(--color-text-muted) 15%, transparent)",
    color: "var(--color-text-muted)",
    label: status,
  };
  return (
    <span
      style={{
        display: "inline-block",
        background: style.bg,
        color: style.color,
        borderRadius: 4,
        padding: "2px 8px",
        fontSize: 11,
        fontWeight: 600,
        letterSpacing: "0.04em",
        textTransform: "capitalize",
        whiteSpace: "nowrap",
      }}
    >
      {style.label}
    </span>
  );
}

// ── Quality badge ────────────────────────────────────────────────────────────

function QualityBadge({ source, resolution }: { source?: string; resolution?: string }) {
  const label = [resolution, source].filter(Boolean).join(" ");
  if (!label) return <span style={{ color: "var(--color-text-muted)", fontSize: 11 }}>—</span>;
  return (
    <span
      style={{
        display: "inline-block",
        background: "var(--color-bg-elevated)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 4,
        padding: "2px 6px",
        fontSize: 10,
        fontWeight: 600,
        letterSpacing: "0.05em",
        textTransform: "uppercase",
        color: "var(--color-text-secondary)",
        whiteSpace: "nowrap",
      }}
    >
      {label}
    </span>
  );
}

// ── Table row ────────────────────────────────────────────────────────────────

function HistoryRow({ item, isLast }: { item: GrabHistory; isLast: boolean }) {
  return (
    <tr style={{ borderBottom: isLast ? "none" : "1px solid var(--color-border-subtle)" }}>
      {/* Release title */}
      <td style={{ padding: "12px 20px", verticalAlign: "middle" }}>
        <Link
          to={`/movies/${item.movie_id}`}
          style={{
            fontSize: 13,
            color: "var(--color-text-primary)",
            fontWeight: 500,
            textDecoration: "none",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            maxWidth: 420,
            display: "block",
          }}
          title={item.release_title}
        >
          {item.release_title}
        </Link>
      </td>

      {/* Quality */}
      <td style={{ padding: "12px 20px", verticalAlign: "middle", whiteSpace: "nowrap" }}>
        <QualityBadge source={item.release_source} resolution={item.release_resolution} />
      </td>

      {/* Protocol */}
      <td
        style={{
          padding: "12px 20px",
          verticalAlign: "middle",
          fontSize: 12,
          color: "var(--color-text-muted)",
          textTransform: "capitalize",
          whiteSpace: "nowrap",
        }}
      >
        {item.protocol || "—"}
      </td>

      {/* Size */}
      <td
        style={{
          padding: "12px 20px",
          verticalAlign: "middle",
          fontSize: 12,
          color: "var(--color-text-muted)",
          fontFamily: "var(--font-family-mono)",
          whiteSpace: "nowrap",
        }}
      >
        {item.size > 0 ? formatBytes(item.size) : "—"}
      </td>

      {/* Status */}
      <td style={{ padding: "12px 20px", verticalAlign: "middle", whiteSpace: "nowrap" }}>
        <StatusBadge status={item.download_status} />
      </td>

      {/* Grabbed at */}
      <td
        style={{
          padding: "12px 20px",
          verticalAlign: "middle",
          fontSize: 12,
          color: "var(--color-text-muted)",
          whiteSpace: "nowrap",
        }}
      >
        {formatDate(item.grabbed_at)}
      </td>
    </tr>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function HistoryPage() {
  const { data, isLoading, error } = useHistory(200);
  const items = data ?? [];

  return (
    <div style={{ padding: 24, maxWidth: 1200, display: "flex", flexDirection: "column", gap: 24 }}>
      {/* Header */}
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
          History
        </h1>
        <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
          {isLoading
            ? "Loading…"
            : error
            ? "Failed to load history."
            : items.length === 0
            ? "No grabs recorded yet."
            : `${items.length} grab${items.length !== 1 ? "s" : ""}`}
        </p>
      </div>

      {/* Table card */}
      <div
        style={{
          background: "var(--color-bg-surface)",
          border: "1px solid var(--color-border-subtle)",
          borderRadius: 8,
          boxShadow: "var(--shadow-card)",
          overflow: "hidden",
        }}
      >
        {isLoading ? (
          <div style={{ padding: 20, display: "flex", flexDirection: "column", gap: 16 }}>
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="skeleton" style={{ height: 14, borderRadius: 3 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 32, textAlign: "center", color: "var(--color-danger)", fontSize: 13 }}>
            Failed to load history. Please try again.
          </div>
        ) : items.length === 0 ? (
          <div
            style={{
              padding: 48,
              textAlign: "center",
              color: "var(--color-text-muted)",
              fontSize: 14,
            }}
          >
            <div style={{ fontSize: 32, marginBottom: 12, opacity: 0.4 }}>⏳</div>
            <div style={{ fontWeight: 500, color: "var(--color-text-secondary)", marginBottom: 4 }}>
              No history yet
            </div>
            <div style={{ fontSize: 12 }}>
              Grab a release from a movie detail page to get started.
            </div>
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Release", "Quality", "Protocol", "Size", "Status", "Grabbed"].map((h) => (
                  <th
                    key={h}
                    style={{
                      textAlign: "left",
                      padding: "8px 20px",
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: "0.08em",
                      textTransform: "uppercase",
                      color: "var(--color-text-muted)",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {items.map((item, idx) => (
                <HistoryRow key={item.id} item={item} isLast={idx === items.length - 1} />
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
