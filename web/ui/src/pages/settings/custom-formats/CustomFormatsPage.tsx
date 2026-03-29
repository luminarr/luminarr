import { useMemo } from "react";
import { toast } from "sonner";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import { Trash2, Download, Check } from "lucide-react";
import {
  useCustomFormats,
  useCustomFormatPresets,
  useImportPreset,
  useDeleteCustomFormat,
} from "@/api/custom-formats";
import { card, sectionHeader } from "@/lib/styles";
import type { CustomFormatPreset } from "@/types";

// ── Helpers ──────────────────────────────────────────────────────────────────

function ScoreBadge({ score }: { score: number }) {
  const isNeg = score < 0;
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 8px",
        borderRadius: 4,
        fontSize: 11,
        fontWeight: 600,
        fontVariantNumeric: "tabular-nums",
        background: isNeg
          ? "var(--color-danger-muted, rgba(239,68,68,0.1))"
          : "var(--color-success-muted, rgba(34,197,94,0.1))",
        color: isNeg ? "var(--color-danger)" : "var(--color-success)",
      }}
    >
      {isNeg ? "" : "+"}
      {score}
    </span>
  );
}

function PresetCard({
  preset,
  imported,
  onImport,
  importing,
}: {
  preset: CustomFormatPreset;
  imported: boolean;
  onImport: () => void;
  importing: boolean;
}) {
  return (
    <div style={{ ...card, padding: 16, display: "flex", flexDirection: "column", gap: 8 }}>
      <div>
        <div
          style={{
            fontSize: 14,
            fontWeight: 500,
            color: "var(--color-text-primary)",
          }}
        >
          {preset.name}
        </div>
        <div
          style={{
            fontSize: 12,
            color: "var(--color-text-muted)",
            marginTop: 2,
            lineHeight: 1.4,
          }}
        >
          {preset.description}
        </div>
      </div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginTop: "auto",
        }}
      >
        <ScoreBadge score={preset.default_score} />
        {imported ? (
          <span
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 4,
              fontSize: 12,
              fontWeight: 500,
              color: "var(--color-success)",
            }}
          >
            <Check size={14} />
            Imported
          </span>
        ) : (
          <button
            onClick={onImport}
            disabled={importing}
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 5,
              padding: "5px 14px",
              fontSize: 12,
              fontWeight: 500,
              borderRadius: 6,
              border: "1px solid var(--color-accent)",
              background: "var(--color-accent-muted)",
              color: "var(--color-accent)",
              cursor: importing ? "not-allowed" : "pointer",
              opacity: importing ? 0.6 : 1,
              transition: "background 150ms ease",
            }}
          >
            <Download size={13} />
            {importing ? "Importing..." : "Import"}
          </button>
        )}
      </div>
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function CustomFormatsPage() {
  const formats = useCustomFormats();
  const presets = useCustomFormatPresets();
  const importPreset = useImportPreset();
  const deleteCF = useDeleteCustomFormat();

  // Set of imported custom format names for cross-referencing with presets.
  const importedNames = useMemo(() => {
    const set = new Set<string>();
    for (const cf of formats.data ?? []) {
      set.add(cf.name.toLowerCase());
    }
    return set;
  }, [formats.data]);

  // Group presets by category.
  const grouped = useMemo(() => {
    const map = new Map<string, CustomFormatPreset[]>();
    for (const p of presets.data ?? []) {
      const list = map.get(p.category) ?? [];
      list.push(p);
      map.set(p.category, list);
    }
    return map;
  }, [presets.data]);

  async function handleDelete(id: string, name: string) {
    try {
      await deleteCF.mutateAsync(id);
      toast.success(`Deleted "${name}"`);
    } catch {
      toast.error("Failed to delete custom format");
    }
  }

  const isLoading = formats.isLoading || presets.isLoading;
  const error = formats.error || presets.error;

  return (
    <div style={{ padding: 32, maxWidth: 960 }}>
      <PageHeader
        title="Custom Formats"
        description="Scoring rules that influence release selection. Import built-in presets or create your own via TRaSH JSON import."
        docsUrl={DOCS_URLS.customFormats}
      />

      {/* Loading */}
      {isLoading && (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="skeleton" style={{ height: 48, borderRadius: 6 }} />
          ))}
        </div>
      )}

      {/* Error */}
      {error && (
        <div
          style={{
            background: "var(--color-danger-muted, rgba(239,68,68,0.08))",
            border: "1px solid var(--color-danger)",
            borderRadius: 8,
            padding: "16px 20px",
            color: "var(--color-danger)",
            fontSize: 13,
          }}
        >
          Failed to load: {(error as Error).message}
        </div>
      )}

      {!isLoading && !error && (
        <>
          {/* ── Imported Custom Formats ────────────────────────────────── */}
          <h2 style={{ ...sectionHeader, marginTop: 0 }}>Imported Custom Formats</h2>

          {(formats.data?.length ?? 0) === 0 ? (
            <div
              style={{
                ...card,
                textAlign: "center",
                padding: "40px 20px",
                color: "var(--color-text-muted)",
                fontSize: 13,
                marginBottom: 32,
              }}
            >
              No custom formats yet. Import a preset below to get started.
            </div>
          ) : (
            <div
              style={{
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-subtle)",
                borderRadius: 8,
                overflow: "hidden",
                marginBottom: 32,
              }}
            >
              {/* Table header */}
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "1fr 80px 40px",
                  gap: "0 12px",
                  padding: "10px 16px",
                  borderBottom: "1px solid var(--color-border-subtle)",
                  fontSize: 11,
                  fontWeight: 600,
                  color: "var(--color-text-muted)",
                  textTransform: "uppercase",
                  letterSpacing: "0.06em",
                }}
              >
                <span>Name</span>
                <span style={{ textAlign: "right" }}>Specs</span>
                <span />
              </div>

              {/* Rows */}
              {formats.data!.map((cf) => (
                <div
                  key={cf.id}
                  style={{
                    display: "grid",
                    gridTemplateColumns: "1fr 80px 40px",
                    gap: "0 12px",
                    padding: "12px 16px",
                    borderBottom: "1px solid var(--color-border-subtle)",
                    alignItems: "center",
                  }}
                >
                  <div
                    style={{
                      fontSize: 13,
                      color: "var(--color-text-primary)",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {cf.name}
                  </div>
                  <div
                    style={{
                      fontSize: 12,
                      color: "var(--color-text-muted)",
                      textAlign: "right",
                    }}
                  >
                    {cf.specifications?.length ?? 0}
                  </div>
                  <div style={{ display: "flex", justifyContent: "flex-end" }}>
                    <button
                      onClick={() => handleDelete(cf.id, cf.name)}
                      disabled={deleteCF.isPending}
                      title="Delete custom format"
                      style={{
                        background: "none",
                        border: "none",
                        cursor: "pointer",
                        color: "var(--color-text-muted)",
                        display: "flex",
                        alignItems: "center",
                        padding: 4,
                        borderRadius: 4,
                        transition: "color 150ms ease",
                      }}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLButtonElement).style.color =
                          "var(--color-danger)";
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLButtonElement).style.color =
                          "var(--color-text-muted)";
                      }}
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* ── Built-in Presets ───────────────────────────────────────── */}
          <h2 style={{ ...sectionHeader, marginTop: 0 }}>Built-in Presets</h2>

          {Array.from(grouped.entries()).map(([category, items]) => (
            <div key={category} style={{ marginBottom: 24 }}>
              <div
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  color: "var(--color-text-secondary)",
                  marginBottom: 10,
                }}
              >
                {category}
              </div>
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
                  gap: 10,
                }}
              >
                {items.map((preset) => (
                  <PresetCard
                    key={preset.id}
                    preset={preset}
                    imported={importedNames.has(preset.name.toLowerCase())}
                    onImport={() => importPreset.mutate(preset.id)}
                    importing={
                      importPreset.isPending &&
                      importPreset.variables === preset.id
                    }
                  />
                ))}
              </div>
            </div>
          ))}
        </>
      )}
    </div>
  );
}
