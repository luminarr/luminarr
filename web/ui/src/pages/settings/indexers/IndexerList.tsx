import { useState } from "react";
import {
  useIndexers,
  useCreateIndexer,
  useUpdateIndexer,
  useDeleteIndexer,
  useTestIndexer,
} from "@/api/indexers";
import type { IndexerConfig, IndexerRequest, TestResult } from "@/types";

// ── Helpers ────────────────────────────────────────────────────────────────────

function strSetting(settings: Record<string, unknown>, key: string): string {
  const v = settings[key];
  return typeof v === "string" ? v : "";
}

function numSetting(settings: Record<string, unknown>, key: string): number {
  const v = settings[key];
  return typeof v === "number" ? v : 0;
}

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

// ── Form state ─────────────────────────────────────────────────────────────────

interface FormState {
  name: string;
  kind: string;
  enabled: boolean;
  priority: string;
  url: string;
  api_key: string;
  rate_limit: string;
  seed_ratio: string;
  seed_time_minutes: string;
}

function emptyForm(): FormState {
  return { name: "", kind: "torznab", enabled: true, priority: "1", url: "", api_key: "", rate_limit: "0", seed_ratio: "0", seed_time_minutes: "0" };
}

function indexerToForm(cfg: IndexerConfig): FormState {
  return {
    name: cfg.name,
    kind: cfg.kind,
    enabled: cfg.enabled,
    priority: String(cfg.priority),
    url: strSetting(cfg.settings, "url"),
    api_key: "",  // never pre-fill; server preserves existing key when omitted
    rate_limit: String(numSetting(cfg.settings, "rate_limit")),
    seed_ratio: String(numSetting(cfg.settings, "seed_ratio")),
    seed_time_minutes: String(numSetting(cfg.settings, "seed_time_minutes")),
  };
}

function formToRequest(f: FormState): IndexerRequest {
  const settings: Record<string, unknown> = { url: f.url.trim() };
  if (f.api_key.trim()) settings.api_key = f.api_key.trim();
  const rl = parseInt(f.rate_limit, 10) || 0;
  if (rl > 0) settings.rate_limit = rl;
  const seedRatio = parseFloat(f.seed_ratio) || 0;
  if (seedRatio > 0) settings.seed_ratio = seedRatio;
  const seedTime = parseInt(f.seed_time_minutes, 10) || 0;
  if (seedTime > 0) settings.seed_time_minutes = seedTime;
  return {
    name: f.name.trim(),
    kind: f.kind,
    enabled: f.enabled,
    priority: parseInt(f.priority, 10) || 1,
    settings,
  };
}

// ── Modal ──────────────────────────────────────────────────────────────────────

interface ModalProps {
  editing: IndexerConfig | null;
  onClose: () => void;
}

function IndexerModal({ editing, onClose }: ModalProps) {
  const [form, setForm] = useState<FormState>(
    editing ? indexerToForm(editing) : emptyForm()
  );
  const [error, setError] = useState<string | null>(null);

  const create = useCreateIndexer();
  const update = useUpdateIndexer();
  const isPending = create.isPending || update.isPending;

  function set<K extends keyof FormState>(field: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [field]: value }));
    setError(null);
  }

  function handleSubmit() {
    if (!form.name.trim()) { setError("Name is required."); return; }
    if (!form.url.trim()) { setError("URL is required."); return; }

    const body = formToRequest(form);

    if (editing) {
      update.mutate(
        { id: editing.id, ...body },
        { onSuccess: onClose, onError: (e) => setError(e.message) }
      );
    } else {
      create.mutate(body, { onSuccess: onClose, onError: (e) => setError(e.message) });
    }
  }

  function focusBorder(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
  }
  function blurBorder(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-border-default)";
  }

  const kindLabel = form.kind === "torznab" ? "Torznab" : "Newznab";

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
          width: 520,
          maxWidth: "calc(100vw - 48px)",
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
            {editing ? "Edit Indexer" : "Add Indexer"}
          </h2>
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

        {/* Fields */}
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
            <div style={fieldStyle}>
              <label style={labelStyle}>Name *</label>
              <input
                style={inputStyle}
                value={form.name}
                onChange={(e) => set("name", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
                placeholder="e.g. Prowlarr"
                autoFocus
              />
            </div>
            <div style={fieldStyle}>
              <label style={labelStyle}>Protocol</label>
              <select
                style={{ ...inputStyle, cursor: "pointer" }}
                value={form.kind}
                onChange={(e) => set("kind", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
              >
                <option value="torznab">Torznab (Torrents)</option>
                <option value="newznab">Newznab (Usenet)</option>
              </select>
            </div>
          </div>

          {/* Settings section */}
          <div
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-subtle)",
              borderRadius: 8,
              padding: 16,
              display: "flex",
              flexDirection: "column",
              gap: 14,
            }}
          >
            <p style={{ margin: 0, fontSize: 11, fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--color-text-muted)" }}>
              {kindLabel} Settings
            </p>
            <div style={fieldStyle}>
              <label style={labelStyle}>URL *</label>
              <input
                style={inputStyle}
                value={form.url}
                onChange={(e) => set("url", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
                placeholder="https://indexer.example.com/torznab/api"
              />
            </div>
            <div style={fieldStyle}>
              <label style={labelStyle}>API Key</label>
              <input
                style={inputStyle}
                type="password"
                value={form.api_key}
                onChange={(e) => set("api_key", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
                placeholder={editing ? "leave blank to keep existing key" : "optional"}
                autoComplete="new-password"
              />
              {editing && (
                <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
                  API keys are masked in the UI. Enter a new value to update.
                </p>
              )}
            </div>
          </div>

          {/* Seeding (torrent indexers only) */}
          {form.kind === "torznab" && (
            <div
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-subtle)",
                borderRadius: 8,
                padding: 16,
                display: "flex",
                flexDirection: "column",
                gap: 14,
              }}
            >
              <p style={{ margin: 0, fontSize: 11, fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--color-text-muted)" }}>
                Seeding
              </p>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Seed Ratio</label>
                  <input
                    style={inputStyle}
                    type="number"
                    min="0"
                    step="0.1"
                    value={form.seed_ratio}
                    onChange={(e) => set("seed_ratio", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="0"
                  />
                  <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
                    0 = use download client default
                  </p>
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Seed Time (minutes)</label>
                  <input
                    style={inputStyle}
                    type="number"
                    min="0"
                    value={form.seed_time_minutes}
                    onChange={(e) => set("seed_time_minutes", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="0"
                  />
                  <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
                    0 = no time limit
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Misc */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 16 }}>
            <div style={fieldStyle}>
              <label style={labelStyle}>Priority</label>
              <input
                style={inputStyle}
                type="number"
                min="1"
                value={form.priority}
                onChange={(e) => set("priority", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
              />
            </div>
            <div style={fieldStyle}>
              <label style={labelStyle}>Rate Limit</label>
              <input
                style={inputStyle}
                type="number"
                min="0"
                value={form.rate_limit}
                onChange={(e) => set("rate_limit", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
                placeholder="0"
              />
              <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
                Requests/min (0 = unlimited)
              </p>
            </div>
            <div style={{ display: "flex", flexDirection: "column", justifyContent: "flex-end", paddingBottom: 2 }}>
              <label
                style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}
              >
                <input
                  type="checkbox"
                  checked={form.enabled}
                  onChange={(e) => set("enabled", e.currentTarget.checked)}
                  style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
                />
                <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>Enabled</span>
              </label>
            </div>
          </div>
        </div>

        {/* Error */}
        {error && (
          <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>{error}</p>
        )}

        {/* Footer */}
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "8px 16px",
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
            onClick={handleSubmit}
            disabled={isPending}
            style={{
              background: isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
              color: isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 500,
              cursor: isPending ? "not-allowed" : "pointer",
            }}
            onMouseEnter={(e) => {
              if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
            }}
          >
            {isPending ? "Saving…" : editing ? "Save Changes" : "Add Indexer"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Row actions ────────────────────────────────────────────────────────────────

interface RowActionsProps {
  indexer: IndexerConfig;
  onEdit: () => void;
}

function RowActions({ indexer, onEdit }: RowActionsProps) {
  const [confirming, setConfirming] = useState(false);
  const [testResult, setTestResult] = useState<{ ok: boolean; message?: string } | null>(null);

  const del = useDeleteIndexer();
  const test = useTestIndexer();

  function handleTest() {
    setTestResult(null);
    test.mutate(indexer.id, {
      onSuccess: (r: TestResult) => {
        setTestResult(r);
        setTimeout(() => setTestResult(null), 4000);
      },
      onError: (e) => {
        setTestResult({ ok: false, message: e.message });
        setTimeout(() => setTestResult(null), 4000);
      },
    });
  }

  if (confirming) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
        <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>Delete?</span>
        <button
          onClick={() => del.mutate(indexer.id, { onSuccess: () => setConfirming(false) })}
          disabled={del.isPending}
          style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 15%, transparent)")}
        >
          {del.isPending ? "…" : "Yes"}
        </button>
        <button
          onClick={() => setConfirming(false)}
          style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
        >
          No
        </button>
      </div>
    );
  }

  if (testResult !== null) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
        <span
          style={{
            fontSize: 12,
            color: testResult.ok ? "var(--color-success)" : "var(--color-danger)",
            maxWidth: 200,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
          title={testResult.message}
        >
          {testResult.ok ? "Connected ✓" : `Failed: ${testResult.message ?? "unknown error"}`}
        </span>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
      <button
        onClick={handleTest}
        disabled={test.isPending}
        style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
      >
        {test.isPending ? "Testing…" : "Test"}
      </button>
      <button
        onClick={onEdit}
        style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
      >
        Edit
      </button>
      <button
        onClick={() => setConfirming(true)}
        style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 12%, transparent)")}
      >
        Delete
      </button>
    </div>
  );
}

// ── Badge ──────────────────────────────────────────────────────────────────────

function KindBadge({ kind }: { kind: string }) {
  const isUsenet = kind === "newznab";
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 8px",
        borderRadius: 4,
        fontSize: 11,
        fontWeight: 600,
        textTransform: "uppercase",
        letterSpacing: "0.05em",
        background: isUsenet
          ? "color-mix(in srgb, var(--color-accent) 12%, transparent)"
          : "color-mix(in srgb, var(--color-success) 12%, transparent)",
        color: isUsenet ? "var(--color-accent)" : "var(--color-success)",
      }}
    >
      {kind}
    </span>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

export default function IndexerList() {
  const { data, isLoading, error } = useIndexers();
  const [modal, setModal] = useState<{ open: boolean; editing: IndexerConfig | null }>({
    open: false,
    editing: null,
  });

  function openCreate() { setModal({ open: true, editing: null }); }
  function openEdit(cfg: IndexerConfig) { setModal({ open: true, editing: cfg }); }
  function closeModal() { setModal({ open: false, editing: null }); }

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 24 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", letterSpacing: "-0.01em" }}>
            Indexers
          </h1>
          <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
            Torznab and Newznab sources for searching releases.
          </p>
        </div>
        <button
          onClick={openCreate}
          style={{
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            fontWeight: 500,
            cursor: "pointer",
            flexShrink: 0,
          }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
        >
          + Add Indexer
        </button>
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
          <div style={{ padding: 20, display: "flex", flexDirection: "column", gap: 12 }}>
            {[1, 2, 3].map((i) => (
              <div key={i} className="skeleton" style={{ height: 44, borderRadius: 4 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 24, fontSize: 13, color: "var(--color-text-muted)" }}>
            Failed to load indexers.
          </div>
        ) : !data?.length ? (
          <div style={{ padding: 48, textAlign: "center" }}>
            <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
              No indexers configured
            </p>
            <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
              Add a Torznab or Newznab indexer to start finding releases.
            </p>
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Name", "Protocol", "Priority", "Status", ""].map((h) => (
                  <th
                    key={h}
                    style={{
                      textAlign: "left",
                      padding: "10px 16px",
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
              {data.map((cfg, i) => (
                <tr
                  key={cfg.id}
                  style={{
                    borderBottom: i < data.length - 1 ? "1px solid var(--color-border-subtle)" : "none",
                  }}
                >
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-primary)", fontWeight: 500 }}>
                    {cfg.name}
                  </td>
                  <td style={{ padding: "0 16px", height: 52 }}>
                    <KindBadge kind={cfg.kind} />
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)" }}>
                    {cfg.priority}
                  </td>
                  <td style={{ padding: "0 16px", height: 52 }}>
                    <span
                      style={{
                        display: "inline-flex",
                        alignItems: "center",
                        gap: 6,
                        fontSize: 12,
                        color: cfg.enabled ? "var(--color-success)" : "var(--color-text-muted)",
                      }}
                    >
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: "50%",
                          background: cfg.enabled ? "var(--color-success)" : "var(--color-text-muted)",
                          flexShrink: 0,
                        }}
                      />
                      {cfg.enabled ? "Enabled" : "Disabled"}
                    </span>
                  </td>
                  <td style={{ padding: "0 16px", height: 52, width: 1 }}>
                    <RowActions indexer={cfg} onEdit={() => openEdit(cfg)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Modal */}
      {modal.open && (
        <IndexerModal editing={modal.editing} onClose={closeModal} />
      )}
    </div>
  );
}
