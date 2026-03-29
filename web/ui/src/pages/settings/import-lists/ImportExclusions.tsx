import { useState } from "react";
import Modal from "@/components/Modal";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import {
  useImportExclusions,
  useCreateImportExclusion,
  useDeleteImportExclusion,
} from "@/api/importlists";
import type { ImportExclusion } from "@/types";

// ── Shared styles ──────────────────────────────────────────────────────────────

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

const fieldStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 0,
};

function actionBtn(color: string, bg: string): React.CSSProperties {
  return {
    background: bg,
    border: "1px solid var(--color-border-default)",
    borderRadius: 5,
    padding: "3px 10px",
    fontSize: 12,
    color,
    cursor: "pointer",
    whiteSpace: "nowrap",
  };
}

// ── Main component ─────────────────────────────────────────────────────────────

export default function ImportExclusions() {
  const { data: exclusions, isLoading, error } = useImportExclusions();
  const createMut = useCreateImportExclusion();
  const deleteMut = useDeleteImportExclusion();

  const [showModal, setShowModal] = useState(false);
  const [tmdbId, setTmdbId] = useState("");
  const [title, setTitle] = useState("");
  const [year, setYear] = useState("");

  function openCreate() {
    setTmdbId("");
    setTitle("");
    setYear("");
    setShowModal(true);
  }

  function handleSave() {
    createMut.mutate(
      { tmdb_id: parseInt(tmdbId, 10) || 0, title: title.trim(), year: parseInt(year, 10) || 0 },
      { onSuccess: () => setShowModal(false) },
    );
  }

  function focusBorder(e: React.FocusEvent<HTMLInputElement>) {
    e.currentTarget.style.borderColor = "var(--color-accent)";
  }
  function blurBorder(e: React.FocusEvent<HTMLInputElement>) {
    e.currentTarget.style.borderColor = "var(--color-border-default)";
  }

  // ── Loading / error ──────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div style={{ padding: "32px 40px", maxWidth: 800 }}>
        <h2 style={{ fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", margin: 0 }}>Import Exclusions</h2>
        <div style={{ display: "flex", flexDirection: "column", gap: 12, marginTop: 20 }}>
          {[1, 2, 3].map((i) => (
            <div key={i} className="skeleton" style={{ height: 48, borderRadius: 8 }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: "32px 40px", maxWidth: 800 }}>
        <h2 style={{ fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", margin: 0 }}>Import Exclusions</h2>
        <p style={{ color: "var(--color-text-danger)", marginTop: 16 }}>
          Failed to load exclusions: {(error as Error).message}
        </p>
      </div>
    );
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div style={{ padding: "32px 40px", maxWidth: 800 }}>
      <PageHeader
        title="Import Exclusions"
        description="Movies in this list will never be added by import list syncs. Deleting a movie automatically adds it here."
        docsUrl={DOCS_URLS.importExclusions}
        action={
          <button
            style={{
              ...actionBtn("white", "var(--color-accent)"),
              border: "none",
              fontWeight: 500,
            }}
            onClick={openCreate}
          >
            + Add
          </button>
        }
      />

      {/* Empty state */}
      {(!exclusions || exclusions.length === 0) && (
        <div
          style={{
            textAlign: "center",
            padding: "48px 24px",
            color: "var(--color-text-muted)",
            background: "var(--color-bg-elevated)",
            border: "1px dashed var(--color-border-default)",
            borderRadius: 8,
          }}
        >
          <p style={{ fontSize: 14, margin: 0 }}>No import exclusions.</p>
          <p style={{ fontSize: 12, marginTop: 6 }}>
            Deleted movies are automatically excluded. You can also manually add exclusions.
          </p>
        </div>
      )}

      {/* List */}
      {exclusions && exclusions.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          {exclusions.map((excl) => (
            <ExclusionRow
              key={excl.id}
              excl={excl}
              onDelete={() => deleteMut.mutate(excl.id)}
            />
          ))}
        </div>
      )}

      {/* Add modal */}
      {showModal && (
        <Modal onClose={() => setShowModal(false)} width={420}>
          <div style={{ padding: "20px 24px 0", borderBottom: "1px solid var(--color-border-subtle)" }}>
            <h3 style={{ margin: "0 0 16px", fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
              Add Exclusion
            </h3>
          </div>
          <div style={{ padding: "20px 24px", display: "flex", flexDirection: "column", gap: 16 }}>
            <div style={fieldStyle}>
              <label style={labelStyle}>TMDb ID *</label>
              <input
                style={inputStyle}
                type="number"
                min="1"
                value={tmdbId}
                onChange={(e) => setTmdbId(e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
                placeholder="e.g. 27205"
              />
            </div>
            <div style={fieldStyle}>
              <label style={labelStyle}>Title</label>
              <input
                style={inputStyle}
                value={title}
                onChange={(e) => setTitle(e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
                placeholder="e.g. Inception"
              />
            </div>
            <div style={fieldStyle}>
              <label style={labelStyle}>Year</label>
              <input
                style={inputStyle}
                type="number"
                min="1900"
                max="2100"
                value={year}
                onChange={(e) => setYear(e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
                placeholder="e.g. 2010"
              />
            </div>
          </div>
          <div
            style={{
              display: "flex",
              justifyContent: "flex-end",
              gap: 8,
              padding: "16px 24px",
              borderTop: "1px solid var(--color-border-subtle)",
            }}
          >
            <button
              style={actionBtn("var(--color-text-secondary)", "transparent")}
              onClick={() => setShowModal(false)}
            >
              Cancel
            </button>
            <button
              style={{
                ...actionBtn("white", "var(--color-accent)"),
                border: "none",
                fontWeight: 500,
                opacity: createMut.isPending ? 0.6 : 1,
              }}
              onClick={handleSave}
              disabled={createMut.isPending || !tmdbId}
            >
              {createMut.isPending ? "Saving..." : "Save"}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

// ── Row component ──────────────────────────────────────────────────────────────

function ExclusionRow({ excl, onDelete }: { excl: ImportExclusion; onDelete: () => void }) {
  const [hovered, setHovered] = useState(false);

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        padding: "10px 16px",
        background: hovered ? "var(--color-bg-elevated)" : "var(--color-bg-surface)",
        border: "1px solid var(--color-border-default)",
        borderRadius: 8,
        transition: "background 0.15s",
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 12, minWidth: 0 }}>
        <span style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
          {excl.title || `TMDb #${excl.tmdb_id}`}
        </span>
        {excl.year > 0 && (
          <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>({excl.year})</span>
        )}
        <span
          style={{
            fontSize: 11,
            color: "var(--color-text-muted)",
            background: "var(--color-bg-elevated)",
            padding: "1px 6px",
            borderRadius: 4,
          }}
        >
          TMDb {excl.tmdb_id}
        </span>
      </div>
      <button
        style={actionBtn("var(--color-text-danger)", "transparent")}
        onClick={onDelete}
      >
        Remove
      </button>
    </div>
  );
}
