import { useState } from "react";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import {
  useQualityProfiles,
  useCreateQualityProfile,
  useUpdateQualityProfile,
  useDeleteQualityProfile,
} from "@/api/quality-profiles";
import { useQualityDefinitions } from "@/api/quality-definitions";
import Modal from "@/components/Modal";
import type { Quality, QualityDefinition, QualityProfile, QualityProfileRequest } from "@/types";

// ── Quality helpers ──────────────────────────────────────────────────────────

/**
 * Find the definition that matches a stored quality.
 *
 * Real DB data often has resolution/source/codec = "unknown" (from Radarr
 * import or older schema). The `name` field is the most reliable identifier:
 * stored as "Bluray-1080p", definitions use "1080p Bluray" — same tokens,
 * different order. We try field-based matching first, then fall back to
 * name-based token matching.
 */
function findMatchingDef(q: Quality, defs: QualityDefinition[]): QualityDefinition | undefined {
  // Strategy 1: resolution + source (works when fields aren't "unknown")
  if (q.resolution !== "unknown" && q.resolution !== "" && q.source !== "unknown" && q.source !== "") {
    const byFields = defs.find((d) => d.resolution === q.resolution && d.source === q.source);
    if (byFields) return byFields;
  }

  // Strategy 2: name contains both the definition's resolution and source tokens
  // e.g. stored "Remux-1080p" contains "1080p" and "remux" → matches def 1080p-remux-*
  const nameLower = q.name.toLowerCase();
  const byName = defs.find((d) => {
    const res = d.resolution.toLowerCase();
    const src = d.source.toLowerCase();
    return res !== "unknown" && src !== "unknown" && nameLower.includes(res) && nameLower.includes(src);
  });
  if (byName) return byName;

  // Strategy 3: single-word names like "CAM", "TELESYNC" — match by source alone
  return defs.find((d) => d.source.toLowerCase() === nameLower);
}

function defToQuality(d: QualityDefinition): Quality {
  return { resolution: d.resolution, source: d.source, codec: d.codec, hdr: d.hdr, name: d.name };
}

/** Simple key for definition-level grouping (resolution + source). */
function defKey(d: QualityDefinition): string {
  return `${d.resolution}-${d.source}`;
}

/** The curated set of popular qualities shown in simple mode. */
const POPULAR_KEYS = new Set([
  "2160p-remux", "2160p-bluray", "2160p-webdl", "2160p-webrip",
  "1080p-remux", "1080p-bluray", "1080p-webdl", "1080p-webrip",
  "720p-bluray", "720p-webdl", "720p-webrip",
]);

function isPopular(d: QualityDefinition): boolean {
  return POPULAR_KEYS.has(defKey(d));
}

// ── Shared styles ─────────────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
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

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 12,
  fontWeight: 500,
  color: "var(--color-text-secondary)",
  marginBottom: 6,
};

function onFocus(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
  (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
}
function onBlur(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
  (e.currentTarget as HTMLElement).style.borderColor = "var(--color-border-default)";
}

// ── Toggle slider ────────────────────────────────────────────────────────────

function ToggleSlider({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <div
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", fontSize: 12, color: "var(--color-text-muted)", userSelect: "none" }}
    >
      <div
        style={{
          width: 34,
          height: 18,
          borderRadius: 9,
          background: checked ? "var(--color-accent)" : "var(--color-bg-elevated)",
          border: `1px solid ${checked ? "var(--color-accent)" : "var(--color-border-default)"}`,
          position: "relative",
          transition: "background 150ms ease, border-color 150ms ease",
          flexShrink: 0,
        }}
      >
        <div
          style={{
            width: 14,
            height: 14,
            borderRadius: "50%",
            background: checked ? "var(--color-accent-fg)" : "var(--color-text-muted)",
            position: "absolute",
            top: 1,
            left: checked ? 17 : 1,
            transition: "left 150ms ease, background 150ms ease",
          }}
        />
      </div>
      <span>{label}</span>
    </div>
  );
}

// ── Modal ─────────────────────────────────────────────────────────────────────

interface ModalProps {
  editing: QualityProfile | null;
  definitions: QualityDefinition[];
  onClose: () => void;
}

function QualityProfileModal({ editing, definitions, onClose }: ModalProps) {
  // Build initial selected keys from existing qualities.
  // Uses multi-strategy matching (fields → name tokens → source-only)
  // to handle legacy data where resolution/source/codec are "unknown".
  const initialKeys = new Set<string>();

  for (const q of editing?.qualities ?? []) {
    const def = findMatchingDef(q, definitions);
    if (def) initialKeys.add(def.id);
  }

  let resolvedCutoffKey = "";
  if (editing?.cutoff) {
    const def = findMatchingDef(editing.cutoff, definitions);
    if (def) resolvedCutoffKey = def.id;
  }

  let resolvedUpgradeUntilKey = "";
  const editingUpgradeUntil = editing?.upgrade_until;
  if (editingUpgradeUntil) {
    const def = findMatchingDef(editingUpgradeUntil, definitions);
    if (def) resolvedUpgradeUntilKey = def.id;
  }

  // Auto-flip to advanced if the profile contains non-popular qualities
  const hasNonPopular = [...initialKeys].some((id) => {
    const def = definitions.find((d) => d.id === id);
    return def && !isPopular(def);
  });

  const [name, setName] = useState(editing?.name ?? "");
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(initialKeys);
  const [cutoffKey, setCutoffKey] = useState<string>(resolvedCutoffKey);
  const [upgradeAllowed, setUpgradeAllowed] = useState(editing?.upgrade_allowed ?? false);
  const [upgradeUntilKey, setUpgradeUntilKey] = useState<string>(resolvedUpgradeUntilKey);
  const [advanced, setAdvanced] = useState(hasNonPopular);
  const [error, setError] = useState<string | null>(null);

  const createProfile = useCreateQualityProfile();
  const updateProfile = useUpdateQualityProfile();
  const isPending = createProfile.isPending || updateProfile.isPending;

  // Filter definitions based on mode
  const visibleDefs = advanced ? definitions : definitions.filter(isPopular);

  // Ordered selected definitions (preserving sort_order, from full list)
  const selectedDefs = definitions.filter((d) => selectedKeys.has(d.id));

  // Table columns differ by mode
  const headers = advanced
    ? ["Name", "Resolution", "Source", "Codec", "HDR", ""]
    : ["Name", "Resolution", "Source", ""];

  function toggleDef(key: string) {
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
        if (cutoffKey === key) setCutoffKey("");
        if (upgradeUntilKey === key) setUpgradeUntilKey("");
      } else {
        next.add(key);
      }
      return next;
    });
    setError(null);
  }

  function handleSubmit() {
    if (!name.trim()) { setError("Name is required."); return; }
    if (selectedDefs.length === 0) { setError("Select at least one quality."); return; }
    if (!cutoffKey) { setError("Select a cutoff quality."); return; }

    const cutoffDef = definitions.find((d) => d.id === cutoffKey);
    const upgradeUntilDef = upgradeAllowed && upgradeUntilKey
      ? definitions.find((d) => d.id === upgradeUntilKey)
      : undefined;

    if (!cutoffDef) { setError("Invalid cutoff selection."); return; }

    const body: QualityProfileRequest = {
      name: name.trim(),
      cutoff: defToQuality(cutoffDef),
      qualities: selectedDefs.map(defToQuality),
      upgrade_allowed: upgradeAllowed,
      upgrade_until: upgradeUntilDef ? defToQuality(upgradeUntilDef) : undefined,
    };

    if (editing) {
      updateProfile.mutate(
        { id: editing.id, ...body },
        { onSuccess: onClose, onError: (e) => setError(e.message) }
      );
    } else {
      createProfile.mutate(body, { onSuccess: onClose, onError: (e) => setError(e.message) });
    }
  }

  return (
    <Modal onClose={onClose} width={640} innerStyle={{ padding: 24, gap: 20, overflowY: "auto" }}>
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
            {editing ? "Edit Profile" : "Add Quality Profile"}
          </h2>
          <button
            onClick={onClose}
            style={{ background: "none", border: "none", cursor: "pointer", color: "var(--color-text-muted)", fontSize: 18, padding: "4px 6px", borderRadius: 4 }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; }}
          >
            ✕
          </button>
        </div>

        {/* Name */}
        <div>
          <label style={labelStyle}>Name *</label>
          <input
            style={inputStyle}
            value={name}
            onChange={(e) => { setName(e.currentTarget.value); setError(null); }}
            onFocus={onFocus}
            onBlur={onBlur}
            placeholder="e.g. HD Standard"
            autoFocus
          />
        </div>

        {/* Qualities */}
        <div>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 10 }}>
            <label style={{ ...labelStyle, marginBottom: 0 }}>Allowed Qualities *</label>
            <ToggleSlider checked={advanced} onChange={setAdvanced} label="Advanced" />
          </div>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {headers.map((h) => (
                  <th key={h} style={{ textAlign: "left", padding: "6px 10px", fontSize: 11, fontWeight: 600, letterSpacing: "0.08em", textTransform: "uppercase", color: "var(--color-text-muted)", whiteSpace: "nowrap" }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {visibleDefs.map((def) => {
                const checked = selectedKeys.has(def.id);
                return (
                  <tr
                    key={def.id}
                    onClick={() => toggleDef(def.id)}
                    style={{
                      cursor: "pointer",
                      background: checked ? "var(--color-accent-muted)" : "transparent",
                      transition: "background 100ms ease",
                    }}
                  >
                    <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)", fontWeight: checked ? 500 : 400 }}>{def.name}</td>
                    <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{def.resolution}</td>
                    <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{def.source}</td>
                    {advanced && (
                      <>
                        <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{def.codec}</td>
                        <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{def.hdr === "none" ? "—" : def.hdr}</td>
                      </>
                    )}
                    <td style={{ padding: "6px 10px", textAlign: "center", width: 32 }}>
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleDef(def.id)}
                        onClick={(e) => e.stopPropagation()}
                        style={{ accentColor: "var(--color-accent)", width: 14, height: 14 }}
                      />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* Cutoff + Upgrade */}
        {selectedDefs.length > 0 && (
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
            <div>
              <label style={labelStyle}>Cutoff (minimum) *</label>
              <select
                style={{ ...inputStyle, cursor: "pointer" }}
                value={cutoffKey}
                onChange={(e) => { setCutoffKey(e.currentTarget.value); setError(null); }}
                onFocus={onFocus}
                onBlur={onBlur}
              >
                <option value="">Select…</option>
                {selectedDefs.map((d) => (
                  <option key={d.id} value={d.id}>{d.name}</option>
                ))}
              </select>
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              <label style={labelStyle}>Upgrades</label>
              <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", fontSize: 13, color: "var(--color-text-secondary)" }}>
                <input
                  type="checkbox"
                  checked={upgradeAllowed}
                  onChange={(e) => {
                    setUpgradeAllowed(e.currentTarget.checked);
                    if (!e.currentTarget.checked) setUpgradeUntilKey("");
                  }}
                  style={{ accentColor: "var(--color-accent)", width: 14, height: 14 }}
                />
                Allow upgrades
              </label>
              {upgradeAllowed && (
                <select
                  style={{ ...inputStyle, cursor: "pointer", marginTop: 4 }}
                  value={upgradeUntilKey}
                  onChange={(e) => setUpgradeUntilKey(e.currentTarget.value)}
                  onFocus={onFocus}
                  onBlur={onBlur}
                >
                  <option value="">No ceiling</option>
                  {selectedDefs.map((d) => (
                    <option key={d.id} value={d.id}>{d.name}</option>
                  ))}
                </select>
              )}
            </div>
          </div>
        )}

        {error && (
          <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>{error}</p>
        )}

        {/* Footer */}
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <button
            onClick={onClose}
            style={{ background: "none", border: "1px solid var(--color-border-default)", borderRadius: 6, padding: "8px 16px", fontSize: 13, color: "var(--color-text-secondary)", cursor: "pointer" }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "none"; }}
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={isPending}
            style={{
              background: isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
              color: isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
              border: "none", borderRadius: 6, padding: "8px 20px", fontSize: 13, fontWeight: 500,
              cursor: isPending ? "not-allowed" : "pointer",
            }}
            onMouseEnter={(e) => { if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
            onMouseLeave={(e) => { if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
          >
            {isPending ? "Saving…" : editing ? "Save Changes" : "Add Profile"}
          </button>
        </div>
    </Modal>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function QualityProfileList() {
  const { data, isLoading, error } = useQualityProfiles();
  const { data: definitions } = useQualityDefinitions();
  const deleteProfile = useDeleteQualityProfile();
  const [modal, setModal] = useState<{ open: boolean; editing: QualityProfile | null }>({
    open: false,
    editing: null,
  });
  const [confirming, setConfirming] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  function openCreate() { setModal({ open: true, editing: null }); }
  function openEdit(p: QualityProfile) { setModal({ open: true, editing: p }); }
  function closeModal() { setModal({ open: false, editing: null }); }

  function handleDelete(id: string) {
    setDeleteError(null);
    deleteProfile.mutate(id, {
      onSuccess: () => setConfirming(null),
      onError: (e) => {
        setConfirming(null);
        setDeleteError(e.message);
      },
    });
  }

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      <PageHeader
        title="Quality Profiles"
        description="Define which quality tiers are acceptable and when to upgrade."
        docsUrl={DOCS_URLS.qualityProfiles}
        action={
          <button
            onClick={openCreate}
            style={{ background: "var(--color-accent)", color: "var(--color-accent-fg)", border: "none", borderRadius: 6, padding: "8px 16px", fontSize: 13, fontWeight: 500, cursor: "pointer", flexShrink: 0 }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
          >
            + Add Profile
          </button>
        }
      />

      {deleteError && (
        <div style={{ marginBottom: 16, padding: "10px 14px", background: "color-mix(in srgb, var(--color-danger) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--color-danger) 30%, transparent)", borderRadius: 6, fontSize: 13, color: "var(--color-danger)" }}>
          {deleteError}
        </div>
      )}

      <div style={{ background: "var(--color-bg-surface)", border: "1px solid var(--color-border-subtle)", borderRadius: 8, boxShadow: "var(--shadow-card)", overflow: "hidden" }}>
        {isLoading ? (
          <div style={{ padding: 20, display: "flex", flexDirection: "column", gap: 12 }}>
            {[1, 2, 3].map((i) => (
              <div key={i} className="skeleton" style={{ height: 44, borderRadius: 4 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 24, fontSize: 13, color: "var(--color-text-muted)" }}>
            Failed to load quality profiles.
          </div>
        ) : !data?.length ? (
          <div style={{ padding: 48, textAlign: "center" }}>
            <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>No quality profiles</p>
            <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>Add a profile to control which releases get grabbed.</p>
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Name", "Cutoff", "Qualities", "Upgrades", ""].map((h) => (
                  <th key={h} style={{ textAlign: "left", padding: "10px 16px", fontSize: 11, fontWeight: 600, letterSpacing: "0.08em", textTransform: "uppercase", color: "var(--color-text-muted)", whiteSpace: "nowrap" }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.map((profile, i) => (
                <tr key={profile.id} style={{ borderBottom: i < data.length - 1 ? "1px solid var(--color-border-subtle)" : "none" }}>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-primary)", fontWeight: 500 }}>
                    {profile.name}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)" }}>
                    {profile.cutoff.name}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)" }}>
                    {profile.qualities.length}
                  </td>
                  <td style={{ padding: "0 16px", height: 52 }}>
                    {profile.upgrade_allowed ? (
                      <span style={{ fontSize: 12, color: "var(--color-success)", background: "color-mix(in srgb, var(--color-success) 12%, transparent)", padding: "2px 8px", borderRadius: 4, fontWeight: 500 }}>
                        {profile.upgrade_until ? `Until ${profile.upgrade_until.name}` : "Yes"}
                      </span>
                    ) : (
                      <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>No</span>
                    )}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, width: 1 }}>
                    {confirming === profile.id ? (
                      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
                        <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>Delete?</span>
                        <button
                          onClick={() => handleDelete(profile.id)}
                          disabled={deleteProfile.isPending}
                          style={{ background: "color-mix(in srgb, var(--color-danger) 15%, transparent)", border: "1px solid var(--color-border-default)", borderRadius: 5, padding: "3px 10px", fontSize: 12, color: "var(--color-danger)", cursor: "pointer" }}
                        >
                          {deleteProfile.isPending ? "…" : "Yes"}
                        </button>
                        <button
                          onClick={() => setConfirming(null)}
                          style={{ background: "var(--color-bg-elevated)", border: "1px solid var(--color-border-default)", borderRadius: 5, padding: "3px 10px", fontSize: 12, color: "var(--color-text-secondary)", cursor: "pointer" }}
                        >
                          No
                        </button>
                      </div>
                    ) : (
                      <div style={{ display: "flex", gap: 6, justifyContent: "flex-end" }}>
                        <button
                          onClick={() => openEdit(profile)}
                          style={{ background: "var(--color-bg-elevated)", border: "1px solid var(--color-border-default)", borderRadius: 5, padding: "3px 10px", fontSize: 12, color: "var(--color-text-secondary)", cursor: "pointer" }}
                          onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
                          onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)"; }}
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => { setDeleteError(null); setConfirming(profile.id); }}
                          style={{ background: "color-mix(in srgb, var(--color-danger) 12%, transparent)", border: "1px solid var(--color-border-default)", borderRadius: 5, padding: "3px 10px", fontSize: 12, color: "var(--color-danger)", cursor: "pointer" }}
                        >
                          Delete
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {modal.open && definitions && (
        <QualityProfileModal editing={modal.editing} definitions={definitions} onClose={closeModal} />
      )}
    </div>
  );
}
