import { useState } from "react";
import type { ScoreBreakdown } from "@/types";

export default function ScoreChip({ breakdown }: { breakdown?: ScoreBreakdown }) {
  const [open, setOpen] = useState(false);

  if (!breakdown) return null;

  const score = breakdown.total;
  const maxScore = breakdown.dimensions.reduce((sum, d) => sum + d.max, 0) || 100;
  const pct = maxScore > 0 ? (score / maxScore) * 100 : 0;
  const color =
    pct >= 80
      ? "var(--color-success)"
      : pct >= 50
        ? "var(--color-warning)"
        : "var(--color-danger)";

  return (
    <div style={{ position: "relative", display: "inline-block" }}>
      <span
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        style={{
          display: "inline-block",
          padding: "2px 7px",
          borderRadius: 4,
          fontSize: 10,
          fontWeight: 700,
          cursor: "default",
          border: `1px solid ${color}`,
          color,
          userSelect: "none",
        }}
      >
        {score}/{maxScore}
      </span>

      {open && (
        <div
          style={{
            position: "absolute",
            bottom: "calc(100% + 6px)",
            left: "50%",
            transform: "translateX(-50%)",
            background: "var(--color-bg-surface)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 8,
            boxShadow: "var(--shadow-modal)",
            padding: "10px 14px",
            minWidth: 220,
            zIndex: 300,
            pointerEvents: "none",
          }}
        >
          {breakdown.dimensions.map((d) => (
            <div
              key={d.name}
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                gap: 12,
                padding: "3px 0",
                borderBottom: "1px solid var(--color-border-subtle)",
                fontSize: 11,
              }}
            >
              <span style={{ color: "var(--color-text-muted)", textTransform: "capitalize", flex: 1 }}>
                {d.name}
              </span>
              <span
                style={{
                  color: d.matched ? "var(--color-success)" : "var(--color-danger)",
                  fontWeight: 600,
                  minWidth: 32,
                  textAlign: "right",
                }}
              >
                {d.score}/{d.max}
              </span>
              <span style={{ color: "var(--color-text-muted)", fontSize: 10, minWidth: 80, textAlign: "right" }}>
                {d.got || "—"}
                {d.want && d.want !== d.got ? ` → ${d.want}` : ""}
              </span>
            </div>
          ))}
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              marginTop: 6,
              paddingTop: 4,
              fontSize: 11,
              fontWeight: 700,
            }}
          >
            <span style={{ color: "var(--color-text-secondary)" }}>Total</span>
            <span style={{ color }}>{score}/{maxScore}</span>
          </div>
        </div>
      )}
    </div>
  );
}
