import { useState } from "react";
import { useFsBrowse } from "@/api/filesystem";
import Modal from "@/components/Modal";

interface DirPickerProps {
  open: boolean;
  /** Current text-field value — used as the initial browsing path. */
  value: string;
  onSelect: (path: string) => void;
  onClose: () => void;
}

/** Split an absolute path into breadcrumb segments. "/" produces one empty segment. */
function pathSegments(p: string): { label: string; path: string }[] {
  const parts = p.split("/").filter(Boolean);
  const segs: { label: string; path: string }[] = [{ label: "/", path: "/" }];
  for (let i = 0; i < parts.length; i++) {
    segs.push({ label: parts[i], path: "/" + parts.slice(0, i + 1).join("/") });
  }
  return segs;
}

export function DirPicker({ open, value, onSelect, onClose }: DirPickerProps) {
  const [currentPath, setCurrentPath] = useState<string>(() =>
    value && value.startsWith("/") ? value : "/"
  );

  const { data, isLoading, error } = useFsBrowse(open ? currentPath : "");

  if (!open) return null;

  const segs = pathSegments(currentPath);

  function navigate(path: string) {
    setCurrentPath(path);
  }

  return (
    <Modal onClose={onClose} width={520} maxHeight="80vh">
        {/* Header */}
        <div
          style={{
            padding: "16px 20px 12px",
            borderBottom: "1px solid var(--color-border-subtle)",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            flexShrink: 0,
          }}
        >
          <span style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Browse for Folder
          </span>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              fontSize: 18,
              lineHeight: 1,
              padding: "2px 6px",
              borderRadius: 4,
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; }}
          >
            ✕
          </button>
        </div>

        {/* Breadcrumb */}
        <div
          style={{
            padding: "10px 20px",
            borderBottom: "1px solid var(--color-border-subtle)",
            display: "flex",
            alignItems: "center",
            flexWrap: "wrap",
            gap: 2,
            flexShrink: 0,
            background: "var(--color-bg-subtle)",
          }}
        >
          {segs.map((seg, i) => {
            const isLast = i === segs.length - 1;
            return (
              <span key={seg.path} style={{ display: "flex", alignItems: "center", gap: 2 }}>
                {i > 0 && (
                  <span style={{ color: "var(--color-text-muted)", fontSize: 11, padding: "0 2px" }}>›</span>
                )}
                <button
                  onClick={() => !isLast && navigate(seg.path)}
                  style={{
                    background: "none",
                    border: "none",
                    cursor: isLast ? "default" : "pointer",
                    padding: "2px 4px",
                    borderRadius: 4,
                    fontSize: 12,
                    fontFamily: "var(--font-family-mono)",
                    color: isLast ? "var(--color-text-primary)" : "var(--color-accent)",
                    fontWeight: isLast ? 600 : 400,
                  }}
                  onMouseEnter={(e) => {
                    if (!isLast) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)";
                  }}
                  onMouseLeave={(e) => {
                    if (!isLast) (e.currentTarget as HTMLButtonElement).style.background = "none";
                  }}
                >
                  {seg.label}
                </button>
              </span>
            );
          })}
        </div>

        {/* Directory list */}
        <div style={{ flex: 1, overflowY: "auto", padding: "8px 0" }}>
          {/* Up directory row */}
          {data?.parent != null && (
            <button
              onClick={() => navigate(data.parent!)}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                width: "100%",
                background: "none",
                border: "none",
                cursor: "pointer",
                padding: "8px 20px",
                textAlign: "left",
                color: "var(--color-text-secondary)",
                fontSize: 13,
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "none"; }}
            >
              <span style={{ fontSize: 15 }}>📁</span>
              <span style={{ fontFamily: "var(--font-family-mono)", fontSize: 12 }}>..</span>
            </button>
          )}

          {isLoading && (
            <>
              {[...Array(6)].map((_, i) => (
                <div
                  key={i}
                  className="skeleton"
                  style={{
                    margin: "6px 20px",
                    height: 14,
                    borderRadius: 4,
                    width: `${55 + (i % 3) * 15}%`,
                  }}
                />
              ))}
            </>
          )}

          {error && (
            <p style={{ margin: "12px 20px", fontSize: 13, color: "var(--color-danger)" }}>
              Could not read directory: {(error as Error).message}
            </p>
          )}

          {!isLoading && !error && data?.dirs.length === 0 && (
            <p style={{ margin: "12px 20px", fontSize: 13, color: "var(--color-text-muted)" }}>
              No subdirectories found.
            </p>
          )}

          {data?.dirs.map((dir) => (
            <button
              key={dir.path}
              onClick={() => navigate(dir.path)}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                width: "100%",
                background: "none",
                border: "none",
                cursor: "pointer",
                padding: "8px 20px",
                textAlign: "left",
                color: "var(--color-text-primary)",
                fontSize: 13,
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "none"; }}
            >
              <span style={{ fontSize: 15 }}>📁</span>
              <span style={{ fontFamily: "var(--font-family-mono)", fontSize: 12 }}>{dir.name}</span>
            </button>
          ))}
        </div>

        {/* Footer */}
        <div
          style={{
            padding: "12px 20px",
            borderTop: "1px solid var(--color-border-subtle)",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            flexShrink: 0,
            background: "var(--color-bg-subtle)",
            gap: 8,
          }}
        >
          <span
            style={{
              fontSize: 11,
              fontFamily: "var(--font-family-mono)",
              color: "var(--color-text-muted)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
              flex: 1,
            }}
          >
            {currentPath}
          </span>
          <div style={{ display: "flex", gap: 8, flexShrink: 0 }}>
            <button
              onClick={onClose}
              style={{
                background: "none",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                padding: "7px 14px",
                fontSize: 13,
                color: "var(--color-text-secondary)",
                cursor: "pointer",
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "none"; }}
            >
              Cancel
            </button>
            <button
              onClick={() => { onSelect(currentPath); onClose(); }}
              style={{
                background: "var(--color-accent)",
                color: "var(--color-accent-fg)",
                border: "none",
                borderRadius: 6,
                padding: "7px 16px",
                fontSize: 13,
                fontWeight: 500,
                cursor: "pointer",
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
            >
              Select This Folder
            </button>
          </div>
        </div>
    </Modal>
  );
}
