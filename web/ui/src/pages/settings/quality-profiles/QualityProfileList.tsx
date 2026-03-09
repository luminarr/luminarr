import { useState } from "react";
import {
  useQualityProfiles,
  useCreateQualityProfile,
  useUpdateQualityProfile,
  useDeleteQualityProfile,
} from "@/api/quality-profiles";
import type { Quality, QualityProfile, QualityProfileRequest } from "@/types";

// ── Quality presets ───────────────────────────────────────────────────────────
// Ordered from lowest to highest quality. Users pick a subset; cutoff and
// upgrade_until are selected from that subset.

const PRESETS: Quality[] = [
  { resolution: "sd",    source: "dvd",    codec: "xvid", hdr: "none",  name: "SD DVD" },
  { resolution: "sd",    source: "hdtv",   codec: "x264", hdr: "none",  name: "SD HDTV" },
  { resolution: "720p",  source: "hdtv",   codec: "x264", hdr: "none",  name: "720p HDTV" },
  { resolution: "720p",  source: "webdl",  codec: "x264", hdr: "none",  name: "720p WEBDL" },
  { resolution: "720p",  source: "webrip", codec: "x264", hdr: "none",  name: "720p WEBRip" },
  { resolution: "720p",  source: "bluray", codec: "x264", hdr: "none",  name: "720p Bluray" },
  { resolution: "1080p", source: "hdtv",   codec: "x264", hdr: "none",  name: "1080p HDTV" },
  { resolution: "1080p", source: "webdl",  codec: "x264", hdr: "none",  name: "1080p WEBDL" },
  { resolution: "1080p", source: "webrip", codec: "x265", hdr: "none",  name: "1080p WEBRip" },
  { resolution: "1080p", source: "bluray", codec: "x265", hdr: "none",  name: "1080p Bluray" },
  { resolution: "1080p", source: "remux",  codec: "x265", hdr: "none",  name: "1080p Remux" },
  { resolution: "2160p", source: "webdl",  codec: "x265", hdr: "hdr10", name: "2160p WEBDL HDR" },
  { resolution: "2160p", source: "bluray", codec: "x265", hdr: "hdr10", name: "2160p Bluray HDR" },
  { resolution: "2160p", source: "remux",  codec: "x265", hdr: "hdr10", name: "2160p Remux HDR" },
];

function presetKey(q: Quality): string {
  return `${q.resolution}-${q.source}-${q.codec}-${q.hdr}`;
}

function presetLabel(q: Quality): string {
  return `${q.resolution} · ${q.source} · ${q.codec} · ${q.hdr}`;
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

// ── Modal ─────────────────────────────────────────────────────────────────────

interface ModalProps {
  editing: QualityProfile | null;
  onClose: () => void;
}

function QualityProfileModal({ editing, onClose }: ModalProps) {
  // Build initial selected preset keys from existing qualities (match by key)
  const existingKeys = new Set((editing?.qualities ?? []).map(presetKey));

  const [name, setName] = useState(editing?.name ?? "");
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(existingKeys);
  const [cutoffKey, setCutoffKey] = useState<string>(
    editing?.cutoff ? presetKey(editing.cutoff) : ""
  );
  const [upgradeAllowed, setUpgradeAllowed] = useState(editing?.upgrade_allowed ?? false);
  const [upgradeUntilKey, setUpgradeUntilKey] = useState<string>(
    editing?.upgrade_until ? presetKey(editing.upgrade_until) : ""
  );
  const [error, setError] = useState<string | null>(null);

  const createProfile = useCreateQualityProfile();
  const updateProfile = useUpdateQualityProfile();
  const isPending = createProfile.isPending || updateProfile.isPending;

  // Ordered selected presets (preserving PRESETS order)
  const selectedPresets = PRESETS.filter((p) => selectedKeys.has(presetKey(p)));

  function togglePreset(key: string) {
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
        // Clear cutoff/upgradeUntil if they referenced this preset
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
    if (selectedPresets.length === 0) { setError("Select at least one quality."); return; }
    if (!cutoffKey) { setError("Select a cutoff quality."); return; }

    const cutoff = PRESETS.find((p) => presetKey(p) === cutoffKey);
    const upgradeUntil = upgradeAllowed && upgradeUntilKey
      ? PRESETS.find((p) => presetKey(p) === upgradeUntilKey)
      : undefined;

    if (!cutoff) { setError("Invalid cutoff selection."); return; }

    const body: QualityProfileRequest = {
      name: name.trim(),
      cutoff,
      qualities: selectedPresets,
      upgrade_allowed: upgradeAllowed,
      upgrade_until: upgradeUntil,
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
          width: 640,
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
          <label style={{ ...labelStyle, marginBottom: 10 }}>Allowed Qualities *</label>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Resolution", "Source", "Codec", "HDR", ""].map((h) => (
                  <th key={h} style={{ textAlign: "left", padding: "6px 10px", fontSize: 11, fontWeight: 600, letterSpacing: "0.08em", textTransform: "uppercase", color: "var(--color-text-muted)", whiteSpace: "nowrap" }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {PRESETS.map((preset) => {
                const key = presetKey(preset);
                const checked = selectedKeys.has(key);
                return (
                  <tr
                    key={key}
                    onClick={() => togglePreset(key)}
                    style={{
                      cursor: "pointer",
                      background: checked ? "var(--color-accent-muted)" : "transparent",
                      transition: "background 100ms ease",
                    }}
                  >
                    <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{preset.resolution}</td>
                    <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{preset.source}</td>
                    <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{preset.codec}</td>
                    <td style={{ padding: "6px 10px", color: checked ? "var(--color-text-primary)" : "var(--color-text-secondary)" }}>{preset.hdr}</td>
                    <td style={{ padding: "6px 10px", textAlign: "center", width: 32 }}>
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => togglePreset(key)}
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
        {selectedPresets.length > 0 && (
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
                {selectedPresets.map((p) => (
                  <option key={presetKey(p)} value={presetKey(p)}>{presetLabel(p)}</option>
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
                  {selectedPresets.map((p) => (
                    <option key={presetKey(p)} value={presetKey(p)}>{presetLabel(p)}</option>
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
      </div>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function QualityProfileList() {
  const { data, isLoading, error } = useQualityProfiles();
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
      {/* Header */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 24 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", letterSpacing: "-0.01em" }}>
            Quality Profiles
          </h1>
          <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
            Define which quality tiers are acceptable and when to upgrade.
          </p>
        </div>
        <button
          onClick={openCreate}
          style={{ background: "var(--color-accent)", color: "var(--color-accent-fg)", border: "none", borderRadius: 6, padding: "8px 16px", fontSize: 13, fontWeight: 500, cursor: "pointer", flexShrink: 0 }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
        >
          + Add Profile
        </button>
      </div>

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
                    {presetLabel(profile.cutoff)}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)" }}>
                    {profile.qualities.length}
                  </td>
                  <td style={{ padding: "0 16px", height: 52 }}>
                    {profile.upgrade_allowed ? (
                      <span style={{ fontSize: 12, color: "var(--color-success)", background: "color-mix(in srgb, var(--color-success) 12%, transparent)", padding: "2px 8px", borderRadius: 4, fontWeight: 500 }}>
                        {profile.upgrade_until ? `Until ${presetLabel(profile.upgrade_until)}` : "Yes"}
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

      {modal.open && (
        <QualityProfileModal editing={modal.editing} onClose={closeModal} />
      )}
    </div>
  );
}
