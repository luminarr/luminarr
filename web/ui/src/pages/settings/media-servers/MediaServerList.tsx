import { useState } from "react";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import {
  useMediaServers,
  useCreateMediaServer,
  useUpdateMediaServer,
  useDeleteMediaServer,
  useTestMediaServer,
} from "@/api/mediaservers";
import { toast } from "sonner";
import type { MediaServerConfig, MediaServerRequest } from "@/types";

// ── Helpers ────────────────────────────────────────────────────────────────────

function strSetting(settings: Record<string, unknown>, key: string): string {
  const v = settings[key];
  return typeof v === "string" ? v : "";
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
  url: string;
  token: string;    // plex
  api_key: string;  // emby / jellyfin
  skip_tls_verify: boolean;
}

const KINDS = [
  { value: "plex",     label: "Plex" },
  { value: "emby",     label: "Emby" },
  { value: "jellyfin", label: "Jellyfin" },
];

function defaultForm(): FormState {
  return { name: "", kind: "plex", enabled: true, url: "", token: "", api_key: "", skip_tls_verify: false };
}

function boolSetting(settings: Record<string, unknown>, key: string): boolean {
  return settings[key] === true;
}

function formFromConfig(cfg: MediaServerConfig): FormState {
  return {
    name: cfg.name,
    kind: cfg.kind,
    enabled: cfg.enabled,
    url: strSetting(cfg.settings, "url"),
    token: strSetting(cfg.settings, "token"),
    api_key: strSetting(cfg.settings, "api_key"),
    skip_tls_verify: boolSetting(cfg.settings, "skip_tls_verify"),
  };
}

function formToRequest(f: FormState): MediaServerRequest {
  const settings: Record<string, unknown> = { url: f.url };
  if (f.kind === "plex") {
    if (f.token) settings.token = f.token;
  } else {
    if (f.api_key) settings.api_key = f.api_key;
  }
  if (f.skip_tls_verify) settings.skip_tls_verify = true;
  return { name: f.name, kind: f.kind, enabled: f.enabled, settings };
}

// ── Kind-specific settings fields ──────────────────────────────────────────────

function PlexFields({ form, setForm }: { form: FormState; setForm: (f: FormState) => void }) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>Token (X-Plex-Token)</label>
      <input
        type="password"
        style={inputStyle}
        placeholder="Your Plex token"
        value={form.token}
        onChange={(e) => setForm({ ...form, token: e.target.value })}
      />
    </div>
  );
}

function EmbyJellyfinFields({ form, setForm }: { form: FormState; setForm: (f: FormState) => void }) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>API Key</label>
      <input
        type="password"
        style={inputStyle}
        placeholder="API key"
        value={form.api_key}
        onChange={(e) => setForm({ ...form, api_key: e.target.value })}
      />
    </div>
  );
}

// ── Card ───────────────────────────────────────────────────────────────────────

function MediaServerCard({
  cfg,
  onEdit,
  onDelete,
  onTest,
  testing,
}: {
  cfg: MediaServerConfig;
  onEdit: () => void;
  onDelete: () => void;
  onTest: () => void;
  testing: boolean;
}) {
  const kindLabel = KINDS.find((k) => k.value === cfg.kind)?.label ?? cfg.kind;
  return (
    <div
      style={{
        background: "var(--color-bg-surface)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        padding: "14px 18px",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 12,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 10, minWidth: 0 }}>
        <span
          style={{
            width: 8,
            height: 8,
            borderRadius: "50%",
            background: cfg.enabled ? "var(--color-success)" : "var(--color-text-muted)",
            flexShrink: 0,
          }}
        />
        <div style={{ minWidth: 0 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {cfg.name}
          </div>
          <div style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 2 }}>
            {kindLabel}
          </div>
        </div>
      </div>
      <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
        <button onClick={onTest} disabled={testing} style={actionBtn("var(--color-accent)", "transparent")}>
          {testing ? "Testing…" : "Test"}
        </button>
        <button onClick={onEdit} style={actionBtn("var(--color-text-secondary)", "transparent")}>
          Edit
        </button>
        <button onClick={onDelete} style={actionBtn("var(--color-danger)", "transparent")}>
          Delete
        </button>
      </div>
    </div>
  );
}

// ── Form modal ─────────────────────────────────────────────────────────────────

function MediaServerForm({
  form,
  setForm,
  onSave,
  onCancel,
  saving,
}: {
  form: FormState;
  setForm: (f: FormState) => void;
  onSave: () => void;
  onCancel: () => void;
  saving: boolean;
}) {
  return (
    <div
      style={{
        background: "var(--color-bg-surface)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        padding: 20,
        display: "flex",
        flexDirection: "column",
        gap: 16,
      }}
    >
      {/* Name */}
      <div style={fieldStyle}>
        <label style={labelStyle}>Name</label>
        <input
          style={inputStyle}
          placeholder="My Plex Server"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
        />
      </div>

      {/* Kind */}
      <div style={fieldStyle}>
        <label style={labelStyle}>Type</label>
        <select
          value={form.kind}
          onChange={(e) => setForm({ ...form, kind: e.target.value })}
          style={{ ...inputStyle, cursor: "pointer" }}
        >
          {KINDS.map((k) => (
            <option key={k.value} value={k.value}>{k.label}</option>
          ))}
        </select>
      </div>

      {/* Enabled */}
      <label style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 13, color: "var(--color-text-primary)", cursor: "pointer" }}>
        <input
          type="checkbox"
          checked={form.enabled}
          onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
          style={{ accentColor: "var(--color-accent)" }}
        />
        Enabled
      </label>

      {/* URL (all kinds) */}
      <div style={fieldStyle}>
        <label style={labelStyle}>URL</label>
        <input
          style={inputStyle}
          placeholder={form.kind === "plex" ? "http://192.168.1.100:32400" : "http://192.168.1.100:8096"}
          value={form.url}
          onChange={(e) => setForm({ ...form, url: e.target.value })}
        />
      </div>

      {/* Kind-specific fields */}
      {form.kind === "plex" ? (
        <PlexFields form={form} setForm={setForm} />
      ) : (
        <EmbyJellyfinFields form={form} setForm={setForm} />
      )}

      {/* TLS verification */}
      <label style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 13, color: "var(--color-text-primary)", cursor: "pointer" }}>
        <input
          type="checkbox"
          checked={form.skip_tls_verify}
          onChange={(e) => setForm({ ...form, skip_tls_verify: e.target.checked })}
          style={{ accentColor: "var(--color-accent)" }}
        />
        Skip TLS certificate verification
        <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>(for self-signed certs)</span>
      </label>

      {/* Actions */}
      <div style={{ display: "flex", gap: 8, justifyContent: "flex-end", marginTop: 4 }}>
        <button onClick={onCancel} style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}>
          Cancel
        </button>
        <button
          onClick={onSave}
          disabled={saving || !form.name || !form.url}
          style={{
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            border: "none",
            borderRadius: 5,
            padding: "6px 16px",
            fontSize: 13,
            fontWeight: 600,
            cursor: saving || !form.name || !form.url ? "not-allowed" : "pointer",
            opacity: saving || !form.name || !form.url ? 0.5 : 1,
          }}
        >
          {saving ? "Saving…" : "Save"}
        </button>
      </div>
    </div>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

export default function MediaServerList() {
  const { data, isLoading, error } = useMediaServers();
  const createMut = useCreateMediaServer();
  const updateMut = useUpdateMediaServer();
  const deleteMut = useDeleteMediaServer();
  const testMut = useTestMediaServer();

  const [editingId, setEditingId] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState<FormState>(defaultForm);
  const [testingId, setTestingId] = useState<string | null>(null);

  const items = data ?? [];

  function startEdit(cfg: MediaServerConfig) {
    setEditingId(cfg.id);
    setForm(formFromConfig(cfg));
    setShowAdd(false);
  }

  function startAdd() {
    setEditingId(null);
    setForm(defaultForm());
    setShowAdd(true);
  }

  function cancel() {
    setEditingId(null);
    setShowAdd(false);
  }

  function save() {
    const req = formToRequest(form);
    if (editingId) {
      updateMut.mutate({ id: editingId, ...req }, { onSuccess: cancel });
    } else {
      createMut.mutate(req, { onSuccess: cancel });
    }
  }

  function handleTest(id: string) {
    setTestingId(id);
    testMut.mutate(id, {
      onSuccess: () => { toast.success("Connection successful"); setTestingId(null); },
      onSettled: () => setTestingId(null),
    });
  }

  function handleDelete(id: string) {
    if (confirm("Delete this media server?")) deleteMut.mutate(id);
  }

  return (
    <div style={{ padding: 24, maxWidth: 720, display: "flex", flexDirection: "column", gap: 20 }}>
      <PageHeader
        title="Media Servers"
        description="Automatically refresh your media server library when a movie is imported."
        docsUrl={DOCS_URLS.mediaServers}
        action={
          !showAdd && !editingId ? (
            <button
              onClick={startAdd}
              style={{
                background: "var(--color-accent)",
                color: "var(--color-accent-fg)",
                border: "none",
                borderRadius: 6,
                padding: "7px 16px",
                fontSize: 13,
                fontWeight: 600,
                cursor: "pointer",
              }}
            >
              + Add
            </button>
          ) : undefined
        }
      />

      {isLoading ? (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          {[1, 2].map((i) => (
            <div key={i} className="skeleton" style={{ height: 56, borderRadius: 8 }} />
          ))}
        </div>
      ) : error ? (
        <div style={{ padding: 32, textAlign: "center", color: "var(--color-danger)", fontSize: 13 }}>
          Failed to load media servers.
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          {items.map((cfg) =>
            editingId === cfg.id ? (
              <MediaServerForm
                key={cfg.id}
                form={form}
                setForm={setForm}
                onSave={save}
                onCancel={cancel}
                saving={updateMut.isPending}
              />
            ) : (
              <MediaServerCard
                key={cfg.id}
                cfg={cfg}
                onEdit={() => startEdit(cfg)}
                onDelete={() => handleDelete(cfg.id)}
                onTest={() => handleTest(cfg.id)}
                testing={testingId === cfg.id}
              />
            ),
          )}
          {items.length === 0 && !showAdd && (
            <div style={{ padding: 40, textAlign: "center", color: "var(--color-text-muted)", fontSize: 14 }}>
              <div style={{ fontSize: 28, marginBottom: 10, opacity: 0.4 }}>📺</div>
              <div style={{ fontWeight: 500, color: "var(--color-text-secondary)", marginBottom: 4 }}>
                No media servers configured
              </div>
              <div style={{ fontSize: 12 }}>
                Add Plex, Emby, or Jellyfin to auto-refresh when movies are imported.
              </div>
            </div>
          )}
        </div>
      )}

      {showAdd && (
        <MediaServerForm
          form={form}
          setForm={setForm}
          onSave={save}
          onCancel={cancel}
          saving={createMut.isPending}
        />
      )}
    </div>
  );
}
