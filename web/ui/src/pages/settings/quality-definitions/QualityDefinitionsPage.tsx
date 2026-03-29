import { useState, useEffect } from "react";
import { useQualityDefinitions, useUpdateQualityDefinitions } from "@/api/quality-definitions";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import { RangeSlider } from "@/components/RangeSlider";
import type { QualityDefinition } from "@/types";

// preferred = max (TRaSH Guides: set preferred as high as possible within range)

const DEFAULTS: Record<string, { min: number; max: number; preferred: number }> = {
  "sd-dvd-xvid-none":        { min: 0,  max: 3,   preferred: 3   },
  "sd-hdtv-x264-none":       { min: 0,  max: 3,   preferred: 3   },
  "720p-hdtv-x264-none":     { min: 2,  max: 20,  preferred: 20  },
  "720p-webdl-x264-none":    { min: 2,  max: 20,  preferred: 20  },
  "720p-webrip-x264-none":   { min: 2,  max: 20,  preferred: 20  },
  "720p-bluray-x264-none":   { min: 2,  max: 30,  preferred: 30  },
  "1080p-hdtv-x264-none":    { min: 4,  max: 40,  preferred: 40  },
  "1080p-webdl-x264-none":   { min: 4,  max: 40,  preferred: 40  },
  "1080p-webrip-x265-none":  { min: 4,  max: 40,  preferred: 40  },
  "1080p-bluray-x265-none":  { min: 4,  max: 95,  preferred: 95  },
  "1080p-remux-x265-none":   { min: 17, max: 400, preferred: 400 },
  "2160p-webdl-x265-hdr10":  { min: 15, max: 250, preferred: 250 },
  "2160p-bluray-x265-hdr10": { min: 15, max: 250, preferred: 250 },
  "2160p-remux-x265-hdr10":  { min: 35, max: 800, preferred: 800 },
};

function resolutionColor(resolution: string): string {
  switch (resolution) {
    case "2160p": return "var(--color-accent)";
    case "1080p": return "var(--color-success)";
    case "720p":  return "var(--color-warning)";
    default:      return "var(--color-text-muted)";
  }
}

function ResolutionBadge({ resolution }: { resolution: string }) {
  const color = resolutionColor(resolution);
  return (
    <span style={{
      display: "inline-block",
      background: `color-mix(in srgb, ${color} 15%, transparent)`,
      color,
      borderRadius: 3,
      padding: "1px 5px",
      fontSize: 10,
      fontWeight: 700,
      letterSpacing: "0.05em",
      textTransform: "uppercase",
      verticalAlign: "middle",
      marginLeft: 7,
    }}>
      {resolution}
    </span>
  );
}

interface RowState {
  min: number;
  max: number;
  preferred: number;
}

interface DefinitionRowProps {
  def: QualityDefinition;
  row: RowState;
  isLast: boolean;
  onChange: (id: string, min: number, max: number, preferred: number) => void;
  onReset: (id: string) => void;
}

function DefinitionRow({ def, row, isLast, onChange, onReset }: DefinitionRowProps) {
  const defaults = DEFAULTS[def.id];
  const atDefault = defaults
    && row.min === defaults.min
    && row.max === defaults.max
    && row.preferred === defaults.preferred;

  const sourceInfo = [
    def.source,
    def.codec !== "unknown" ? def.codec : null,
    def.hdr !== "none" ? def.hdr : null,
  ].filter(Boolean).join(" · ");

  return (
    <tr style={{ borderBottom: isLast ? "none" : "1px solid var(--color-border-subtle)" }}>
      {/* Quality name + resolution badge */}
      <td style={{ padding: "14px 20px", verticalAlign: "middle", whiteSpace: "nowrap" }}>
        <span style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
          {def.name}
        </span>
        <ResolutionBadge resolution={def.resolution} />
      </td>

      {/* Source / codec / HDR */}
      <td style={{ padding: "14px 20px", verticalAlign: "middle", whiteSpace: "nowrap" }}>
        <span style={{ fontSize: 11, color: "var(--color-text-muted)", letterSpacing: "0.02em" }}>
          {sourceInfo}
        </span>
      </td>

      {/* Range slider */}
      <td style={{ padding: "14px 20px 2px", verticalAlign: "top", width: "100%", minWidth: 300 }}>
        <RangeSlider
          minValue={row.min}
          maxValue={row.max}
          preferredValue={row.preferred}
          onChange={(min, max, preferred) => onChange(def.id, min, max, preferred)}
        />
      </td>

      {/* Reset */}
      <td style={{ padding: "14px 20px", verticalAlign: "middle", whiteSpace: "nowrap" }}>
        {defaults && !atDefault && (
          <button
            onClick={() => onReset(def.id)}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 4,
              padding: "3px 8px",
              fontSize: 11,
              color: "var(--color-text-muted)",
              cursor: "pointer",
            }}
          >
            Reset
          </button>
        )}
      </td>
    </tr>
  );
}

export default function QualityDefinitionsPage() {
  const { data, isLoading, error } = useQualityDefinitions();
  const updateMutation = useUpdateQualityDefinitions();

  const [rows, setRows] = useState<Record<string, RowState>>({});
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (!data) return;
    const initial: Record<string, RowState> = {};
    for (const d of data) {
      initial[d.id] = { min: d.min_size, max: d.max_size, preferred: d.preferred_size };
    }
    setRows(initial);
    setDirty(false);
  }, [data]);

  function handleChange(id: string, min: number, max: number, preferred: number) {
    setRows((prev) => ({ ...prev, [id]: { min, max, preferred } }));
    setDirty(true);
  }

  function handleReset(id: string) {
    const defaults = DEFAULTS[id];
    if (!defaults) return;
    setRows((prev) => ({ ...prev, [id]: { min: defaults.min, max: defaults.max, preferred: defaults.preferred } }));
    setDirty(true);
  }

  function handleResetAll() {
    if (!data) return;
    const reset: Record<string, RowState> = {};
    for (const d of data) {
      const def = DEFAULTS[d.id];
      reset[d.id] = def
        ? { min: def.min, max: def.max, preferred: def.preferred }
        : { min: d.min_size, max: d.max_size, preferred: d.preferred_size };
    }
    setRows(reset);
    setDirty(true);
  }

  async function handleSave() {
    if (!data) return;
    const updates = data.map((d) => ({
      id: d.id,
      min_size:       rows[d.id]?.min       ?? d.min_size,
      max_size:       rows[d.id]?.max       ?? d.max_size,
      preferred_size: rows[d.id]?.preferred ?? d.preferred_size,
    }));
    await updateMutation.mutateAsync(updates);
    setDirty(false);
  }

  const defs = data ?? [];

  return (
    <div style={{ padding: 24, maxWidth: 1000, display: "flex", flexDirection: "column", gap: 24 }}>
      <PageHeader
        title="Quality Definitions"
        description="Acceptable file-size range (MB per minute of runtime) for each quality level. The blue diamond marks the preferred size within that range."
        docsUrl={DOCS_URLS.qualityDefinitions}
        action={
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <button
              onClick={handleResetAll}
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                padding: "7px 14px",
                fontSize: 13,
                color: "var(--color-text-secondary)",
                cursor: "pointer",
              }}
            >
              Reset All
            </button>
            <button
              onClick={handleSave}
              disabled={!dirty || updateMutation.isPending}
              style={{
                background: dirty ? "var(--color-accent)" : "var(--color-bg-elevated)",
                border: `1px solid ${dirty ? "var(--color-accent)" : "var(--color-border-default)"}`,
                borderRadius: 6,
                padding: "7px 18px",
                fontSize: 13,
                fontWeight: 600,
                color: dirty ? "var(--color-accent-fg)" : "var(--color-text-muted)",
                cursor: dirty ? "pointer" : "default",
                transition: "background 0.15s, border-color 0.15s",
              }}
            >
              {updateMutation.isPending ? "Saving…" : "Save Changes"}
            </button>
          </div>
        }
      />

      {/* Table card */}
      <div style={{ background: "var(--color-bg-surface)", border: "1px solid var(--color-border-subtle)", borderRadius: 8, boxShadow: "var(--shadow-card)", overflow: "hidden" }}>
        {isLoading ? (
          <div style={{ padding: 20, display: "flex", flexDirection: "column", gap: 20 }}>
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="skeleton" style={{ height: 14, borderRadius: 3 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 32, textAlign: "center", color: "var(--color-danger)", fontSize: 13 }}>
            Failed to load quality definitions. Please try again.
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Quality", "Source", "Size Range", ""].map((h) => (
                  <th
                    key={h || "_action"}
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
              {defs.map((def, idx) => (
                <DefinitionRow
                  key={def.id}
                  def={def}
                  row={rows[def.id] ?? { min: def.min_size, max: def.max_size, preferred: def.preferred_size }}
                  isLast={idx === defs.length - 1}
                  onChange={handleChange}
                  onReset={handleReset}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Legend */}
      {!isLoading && !error && defs.length > 0 && (
        <div style={{ display: "flex", gap: 20, fontSize: 12, color: "var(--color-text-muted)", alignItems: "center" }}>
          <span style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span style={{ display: "inline-block", width: 10, height: 10, borderRadius: "50%", background: "var(--color-accent)" }} />
            Min / Max
          </span>
          <span style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span style={{ display: "inline-block", width: 8, height: 8, borderRadius: 1, background: "var(--color-info)", transform: "rotate(45deg)" }} />
            Preferred
          </span>
          <span style={{ color: "var(--color-text-muted)" }}>
            Scale is logarithmic · sizes in MB per minute of runtime
          </span>
        </div>
      )}
    </div>
  );
}
