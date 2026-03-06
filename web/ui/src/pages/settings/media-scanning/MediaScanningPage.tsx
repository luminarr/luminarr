import { useState } from "react";
import { useMediainfoStatus, useScanAll } from "@/api/mediainfo";

export default function MediaScanningPage() {
  const { data: status, isLoading } = useMediainfoStatus();
  const scanAll = useScanAll();
  const [scanQueued, setScanQueued] = useState(false);

  async function handleScanAll() {
    try {
      await scanAll.mutateAsync();
      setScanQueued(true);
      setTimeout(() => setScanQueued(false), 5000);
    } catch {
      // error handled below
    }
  }

  return (
    <div style={{ maxWidth: 640 }}>
      <h2 style={{ margin: "0 0 6px", fontSize: 18, fontWeight: 600, color: "var(--color-text-primary)" }}>
        Media Scanning
      </h2>
      <p style={{ margin: "0 0 24px", fontSize: 13, color: "var(--color-text-muted)" }}>
        Luminarr can use <strong>ffprobe</strong> to verify the actual codec, resolution, and
        HDR format of imported files — and flag mismatches against the filename-parsed quality.
      </p>

      {/* Status card */}
      <div
        style={{
          background: "var(--color-bg-elevated)",
          border: "1px solid var(--color-border-subtle)",
          borderRadius: 8,
          padding: "16px 20px",
          marginBottom: 16,
        }}
      >
        <div style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)", marginBottom: 10 }}>
          Scanner Status
        </div>

        {isLoading ? (
          <div className="skeleton" style={{ height: 20, width: 200, borderRadius: 4 }} />
        ) : status?.available ? (
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <span style={{ color: "var(--color-success)", fontSize: 16 }}>●</span>
              <span style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500 }}>
                Available
              </span>
            </div>
            <div style={{ fontSize: 12, color: "var(--color-text-muted)", fontFamily: "var(--font-family-mono)" }}>
              {status.ffprobe_path}
            </div>
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <span style={{ color: "var(--color-text-muted)", fontSize: 16 }}>○</span>
              <span style={{ fontSize: 13, color: "var(--color-text-secondary)", fontWeight: 500 }}>
                Unavailable — ffprobe not found
              </span>
            </div>
            <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>
              Install ffprobe to enable media scanning. See the{" "}
              <a
                href="https://github.com/luminarr/luminarr/blob/main/docs/getting-started.md#ffprobe-optional"
                target="_blank"
                rel="noreferrer"
                style={{ color: "var(--color-accent)" }}
              >
                getting started guide
              </a>{" "}
              for setup instructions.
            </p>
          </div>
        )}
      </div>

      {/* Bulk scan */}
      <div
        style={{
          background: "var(--color-bg-elevated)",
          border: "1px solid var(--color-border-subtle)",
          borderRadius: 8,
          padding: "16px 20px",
        }}
      >
        <div style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)", marginBottom: 6 }}>
          Scan All Files
        </div>
        <p style={{ margin: "0 0 14px", fontSize: 12, color: "var(--color-text-muted)" }}>
          Scan every movie file that has not yet been scanned. This runs in the background.
        </p>
        <button
          onClick={handleScanAll}
          disabled={!status?.available || scanAll.isPending || scanQueued}
          style={{
            background: status?.available ? "var(--color-accent)" : "var(--color-bg-surface)",
            color: status?.available ? "var(--color-accent-fg)" : "var(--color-text-muted)",
            border: status?.available ? "none" : "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "6px 16px",
            fontSize: 12,
            fontWeight: 500,
            cursor: (!status?.available || scanAll.isPending || scanQueued) ? "not-allowed" : "pointer",
          }}
        >
          {scanQueued ? "Scan queued ✓" : scanAll.isPending ? "Starting…" : "Scan all unscanned files"}
        </button>
        {scanAll.isError && (
          <p style={{ margin: "8px 0 0", fontSize: 12, color: "var(--color-danger)" }}>
            Failed to start scan: {(scanAll.error as Error).message}
          </p>
        )}
      </div>

      {/* Install guide */}
      <div
        style={{
          marginTop: 24,
          padding: "14px 18px",
          background: "color-mix(in srgb, var(--color-accent) 6%, transparent)",
          border: "1px solid color-mix(in srgb, var(--color-accent) 20%, transparent)",
          borderRadius: 8,
          fontSize: 12,
          color: "var(--color-text-secondary)",
          lineHeight: 1.7,
        }}
      >
        <strong style={{ color: "var(--color-text-primary)" }}>How to install ffprobe</strong>
        <br />
        <strong>Linux (Debian/Ubuntu):</strong>{" "}
        <code style={{ fontFamily: "var(--font-family-mono)", background: "var(--color-bg-elevated)", padding: "1px 4px", borderRadius: 3 }}>
          sudo apt install ffmpeg
        </code>
        <br />
        <strong>macOS (Homebrew):</strong>{" "}
        <code style={{ fontFamily: "var(--font-family-mono)", background: "var(--color-bg-elevated)", padding: "1px 4px", borderRadius: 3 }}>
          brew install ffmpeg
        </code>
        <br />
        <strong>Docker:</strong> use the{" "}
        <code style={{ fontFamily: "var(--font-family-mono)", background: "var(--color-bg-elevated)", padding: "1px 4px", borderRadius: 3 }}>
          latest-full
        </code>{" "}
        image tag — ffprobe is included.
      </div>
    </div>
  );
}
