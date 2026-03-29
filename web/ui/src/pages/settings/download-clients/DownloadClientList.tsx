import { useState, useEffect } from "react";
import Modal from "@/components/Modal";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import {
  useDownloadClients,
  useCreateDownloadClient,
  useUpdateDownloadClient,
  useDeleteDownloadClient,
  useTestDownloadClient,
} from "@/api/downloaders";
import {
  useDownloadHandling,
  useUpdateDownloadHandling,
  useRemotePathMappings,
  useCreateRemotePathMapping,
  useDeleteRemotePathMapping,
} from "@/api/download-handling";
import type {
  DownloadClientConfig,
  DownloadClientRequest,
  DownloadHandling,
  RemotePathMapping,
  TestResult,
} from "@/types";

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
  priority: string;
  // qbittorrent
  qb_url: string;
  qb_username: string;
  qb_password: string;
  qb_category: string;
  qb_save_path: string;
  // deluge
  dl_url: string;
  dl_password: string;
  dl_label: string;
  dl_save_path: string;
  // transmission
  trans_url: string;
  trans_username: string;
  trans_password: string;
  // sabnzbd
  sab_url: string;
  sab_api_key: string;
  sab_category: string;
  // nzbget
  nzbget_url: string;
  nzbget_username: string;
  nzbget_password: string;
  nzbget_category: string;
}

function emptyForm(): FormState {
  return {
    name: "", kind: "qbittorrent", enabled: true, priority: "1",
    qb_url: "", qb_username: "", qb_password: "", qb_category: "", qb_save_path: "",
    dl_url: "", dl_password: "", dl_label: "", dl_save_path: "",
    trans_url: "", trans_username: "", trans_password: "",
    sab_url: "", sab_api_key: "", sab_category: "",
    nzbget_url: "", nzbget_username: "", nzbget_password: "", nzbget_category: "",
  };
}

function clientToForm(cfg: DownloadClientConfig): FormState {
  const s = cfg.settings;
  const k = cfg.kind;
  return {
    name: cfg.name,
    kind: k,
    enabled: cfg.enabled,
    priority: String(cfg.priority),
    qb_url: k === "qbittorrent" ? strSetting(s, "url") : "",
    qb_username: k === "qbittorrent" ? strSetting(s, "username") : "",
    qb_password: "",  // never pre-fill; server preserves existing password when omitted
    qb_category: k === "qbittorrent" ? strSetting(s, "category") : "",
    qb_save_path: k === "qbittorrent" ? strSetting(s, "save_path") : "",
    dl_url: k === "deluge" ? strSetting(s, "url") : "",
    dl_password: "",  // never pre-fill
    dl_label: k === "deluge" ? strSetting(s, "label") : "",
    dl_save_path: k === "deluge" ? strSetting(s, "save_path") : "",
    trans_url: k === "transmission" ? strSetting(s, "url") : "",
    trans_username: k === "transmission" ? strSetting(s, "username") : "",
    trans_password: "",  // never pre-fill
    sab_url: k === "sabnzbd" ? strSetting(s, "url") : "",
    sab_api_key: "",  // never pre-fill
    sab_category: k === "sabnzbd" ? strSetting(s, "category") : "",
    nzbget_url: k === "nzbget" ? strSetting(s, "url") : "",
    nzbget_username: k === "nzbget" ? strSetting(s, "username") : "",
    nzbget_password: "",  // never pre-fill
    nzbget_category: k === "nzbget" ? strSetting(s, "category") : "",
  };
}

function formToRequest(f: FormState): DownloadClientRequest {
  let settings: Record<string, unknown>;

  switch (f.kind) {
    case "qbittorrent":
      settings = { url: f.qb_url.trim(), username: f.qb_username.trim() };
      if (f.qb_password.trim()) settings.password = f.qb_password.trim();
      if (f.qb_category.trim()) settings.category = f.qb_category.trim();
      if (f.qb_save_path.trim()) settings.save_path = f.qb_save_path.trim();
      break;
    case "deluge":
      settings = { url: f.dl_url.trim() };
      if (f.dl_password.trim()) settings.password = f.dl_password.trim();
      if (f.dl_label.trim()) settings.label = f.dl_label.trim();
      if (f.dl_save_path.trim()) settings.save_path = f.dl_save_path.trim();
      break;
    case "transmission":
      settings = { url: f.trans_url.trim() };
      if (f.trans_username.trim()) settings.username = f.trans_username.trim();
      if (f.trans_password.trim()) settings.password = f.trans_password.trim();
      break;
    case "sabnzbd":
      settings = { url: f.sab_url.trim() };
      if (f.sab_api_key.trim()) settings.api_key = f.sab_api_key.trim();
      if (f.sab_category.trim()) settings.category = f.sab_category.trim();
      break;
    case "nzbget":
      settings = { url: f.nzbget_url.trim() };
      if (f.nzbget_username.trim()) settings.username = f.nzbget_username.trim();
      if (f.nzbget_password.trim()) settings.password = f.nzbget_password.trim();
      if (f.nzbget_category.trim()) settings.category = f.nzbget_category.trim();
      break;
    default:
      settings = {};
  }

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
  editing: DownloadClientConfig | null;
  onClose: () => void;
}

function DownloadClientModal({ editing, onClose }: ModalProps) {
  const [form, setForm] = useState<FormState>(
    editing ? clientToForm(editing) : emptyForm()
  );
  const [error, setError] = useState<string | null>(null);

  const create = useCreateDownloadClient();
  const update = useUpdateDownloadClient();
  const isPending = create.isPending || update.isPending;

  function set<K extends keyof FormState>(field: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [field]: value }));
    setError(null);
  }

  function handleSubmit() {
    if (!form.name.trim()) { setError("Name is required."); return; }
    const urlMap: Record<string, string> = {
      qbittorrent: form.qb_url, deluge: form.dl_url, transmission: form.trans_url,
      sabnzbd: form.sab_url, nzbget: form.nzbget_url,
    };
    const url = urlMap[form.kind] ?? "";
    if (!url.trim()) { setError("URL is required."); return; }
    if (form.kind === "sabnzbd" && !form.sab_api_key.trim()) { setError("API Key is required."); return; }

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

  const sensitiveHint = editing ? (
    <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
      Masked for security. Enter a new value to update, or leave blank to keep existing.
    </p>
  ) : null;

  return (
    <Modal onClose={onClose} width={540} innerStyle={{ padding: 24, gap: 20, overflowY: "auto" }}>
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
            {editing ? "Edit Download Client" : "Add Download Client"}
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
                placeholder="e.g. qBittorrent Local"
                autoFocus
              />
            </div>
            <div style={fieldStyle}>
              <label style={labelStyle}>Client</label>
              <select
                style={{ ...inputStyle, cursor: "pointer" }}
                value={form.kind}
                onChange={(e) => set("kind", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
              >
                <option value="qbittorrent">qBittorrent</option>
                <option value="deluge">Deluge</option>
                <option value="transmission">Transmission</option>
                <option value="sabnzbd">SABnzbd</option>
                <option value="nzbget">NZBGet</option>
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
              {{ qbittorrent: "qBittorrent", deluge: "Deluge", transmission: "Transmission", sabnzbd: "SABnzbd", nzbget: "NZBGet" }[form.kind] ?? form.kind} Settings
            </p>

            {form.kind === "qbittorrent" && (
              <>
                <div style={fieldStyle}>
                  <label style={labelStyle}>URL *</label>
                  <input
                    style={inputStyle}
                    value={form.qb_url}
                    onChange={(e) => set("qb_url", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="http://localhost:8080"
                  />
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Username</label>
                    <input
                      style={inputStyle}
                      value={form.qb_username}
                      onChange={(e) => set("qb_username", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder="admin"
                      autoComplete="off"
                    />
                  </div>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Password</label>
                    <input
                      style={inputStyle}
                      type="password"
                      value={form.qb_password}
                      onChange={(e) => set("qb_password", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder={editing ? "enter to change" : ""}
                      autoComplete="new-password"
                    />
                    {sensitiveHint}
                  </div>
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Category</label>
                    <input
                      style={inputStyle}
                      value={form.qb_category}
                      onChange={(e) => set("qb_category", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder="luminarr"
                    />
                  </div>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Save Path</label>
                    <input
                      style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12 }}
                      value={form.qb_save_path}
                      onChange={(e) => set("qb_save_path", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder="/downloads/movies"
                    />
                  </div>
                </div>
              </>
            )}

            {form.kind === "deluge" && (
              <>
                <div style={fieldStyle}>
                  <label style={labelStyle}>URL *</label>
                  <input
                    style={inputStyle}
                    value={form.dl_url}
                    onChange={(e) => set("dl_url", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="http://localhost:8112"
                  />
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Password</label>
                  <input
                    style={inputStyle}
                    type="password"
                    value={form.dl_password}
                    onChange={(e) => set("dl_password", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder={editing ? "enter to change" : "deluge (default)"}
                    autoComplete="new-password"
                  />
                  {sensitiveHint}
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Label</label>
                    <input
                      style={inputStyle}
                      value={form.dl_label}
                      onChange={(e) => set("dl_label", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder="luminarr"
                    />
                  </div>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Save Path</label>
                    <input
                      style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12 }}
                      value={form.dl_save_path}
                      onChange={(e) => set("dl_save_path", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder="/downloads/movies"
                    />
                  </div>
                </div>
              </>
            )}

            {form.kind === "transmission" && (
              <>
                <div style={fieldStyle}>
                  <label style={labelStyle}>URL *</label>
                  <input
                    style={inputStyle}
                    value={form.trans_url}
                    onChange={(e) => set("trans_url", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="http://localhost:9091"
                  />
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Username</label>
                    <input
                      style={inputStyle}
                      value={form.trans_username}
                      onChange={(e) => set("trans_username", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      autoComplete="off"
                    />
                  </div>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Password</label>
                    <input
                      style={inputStyle}
                      type="password"
                      value={form.trans_password}
                      onChange={(e) => set("trans_password", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder={editing ? "enter to change" : ""}
                      autoComplete="new-password"
                    />
                    {sensitiveHint}
                  </div>
                </div>
              </>
            )}

            {form.kind === "sabnzbd" && (
              <>
                <div style={fieldStyle}>
                  <label style={labelStyle}>URL *</label>
                  <input
                    style={inputStyle}
                    value={form.sab_url}
                    onChange={(e) => set("sab_url", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="http://localhost:8080"
                  />
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>API Key *</label>
                  <input
                    style={inputStyle}
                    type="password"
                    value={form.sab_api_key}
                    onChange={(e) => set("sab_api_key", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder={editing ? "enter to change" : ""}
                    autoComplete="new-password"
                  />
                  {sensitiveHint}
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Category</label>
                  <input
                    style={inputStyle}
                    value={form.sab_category}
                    onChange={(e) => set("sab_category", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="luminarr"
                  />
                </div>
              </>
            )}

            {form.kind === "nzbget" && (
              <>
                <div style={fieldStyle}>
                  <label style={labelStyle}>URL *</label>
                  <input
                    style={inputStyle}
                    value={form.nzbget_url}
                    onChange={(e) => set("nzbget_url", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="http://localhost:6789"
                  />
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Username</label>
                    <input
                      style={inputStyle}
                      value={form.nzbget_username}
                      onChange={(e) => set("nzbget_username", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder="nzbget"
                      autoComplete="off"
                    />
                  </div>
                  <div style={fieldStyle}>
                    <label style={labelStyle}>Password</label>
                    <input
                      style={inputStyle}
                      type="password"
                      value={form.nzbget_password}
                      onChange={(e) => set("nzbget_password", e.currentTarget.value)}
                      onFocus={focusBorder}
                      onBlur={blurBorder}
                      placeholder={editing ? "enter to change" : ""}
                      autoComplete="new-password"
                    />
                    {sensitiveHint}
                  </div>
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Category</label>
                  <input
                    style={inputStyle}
                    value={form.nzbget_category}
                    onChange={(e) => set("nzbget_category", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="luminarr"
                  />
                </div>
              </>
            )}
          </div>

          {/* Priority + enabled */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
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
            <div style={{ display: "flex", flexDirection: "column", justifyContent: "flex-end", paddingBottom: 2 }}>
              <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
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
            {isPending ? "Saving…" : editing ? "Save Changes" : "Add Client"}
          </button>
        </div>
    </Modal>
  );
}

// ── Row actions ────────────────────────────────────────────────────────────────

interface RowActionsProps {
  client: DownloadClientConfig;
  onEdit: () => void;
}

function RowActions({ client, onEdit }: RowActionsProps) {
  const [confirming, setConfirming] = useState(false);
  const [testResult, setTestResult] = useState<{ ok: boolean; message?: string } | null>(null);

  const del = useDeleteDownloadClient();
  const test = useTestDownloadClient();

  function handleTest() {
    setTestResult(null);
    test.mutate(client.id, {
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
          onClick={() => del.mutate(client.id, { onSuccess: () => setConfirming(false) })}
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
      <button onClick={onEdit} style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}>
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
  const labels: Record<string, string> = {
    qbittorrent: "qBittorrent", deluge: "Deluge", transmission: "Transmission",
    sabnzbd: "SABnzbd", nzbget: "NZBGet",
  };
  const colors: Record<string, string> = {
    qbittorrent: "var(--color-accent)", deluge: "var(--color-accent)",
    transmission: "#B71C1C", sabnzbd: "#F57C00", nzbget: "#388E3C",
  };
  const c = colors[kind] ?? "var(--color-accent)";
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
        background: `color-mix(in srgb, ${c} 12%, transparent)`,
        color: c,
      }}
    >
      {labels[kind] ?? kind}
    </span>
  );
}

// ── Completed & Failed Download Handling ────────────────────────────────────────

function DownloadHandlingSection() {
  const { data, isLoading } = useDownloadHandling();
  const update = useUpdateDownloadHandling();

  const defaultForm: DownloadHandling = {
    enable_completed: true,
    check_interval_minutes: 1,
    redownload_failed: true,
    redownload_failed_interactive: false,
  };

  const [form, setForm] = useState<DownloadHandling>(defaultForm);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (data && !initialized) {
      setForm(data);
      setInitialized(true);
    }
  }, [data, initialized]);

  function set<K extends keyof DownloadHandling>(key: K, val: DownloadHandling[K]) {
    setForm((f) => ({ ...f, [key]: val }));
  }

  const isDirty = initialized && data && (
    form.enable_completed !== data.enable_completed ||
    form.check_interval_minutes !== data.check_interval_minutes ||
    form.redownload_failed !== data.redownload_failed ||
    form.redownload_failed_interactive !== data.redownload_failed_interactive
  );

  const sectionCardStyle: React.CSSProperties = {
    background: "var(--color-bg-surface)",
    border: "1px solid var(--color-border-subtle)",
    borderRadius: 8,
    boxShadow: "var(--shadow-card)",
    overflow: "hidden",
    marginTop: 24,
  };

  const sectionHeaderStyle: React.CSSProperties = {
    padding: "14px 20px",
    borderBottom: "1px solid var(--color-border-subtle)",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
  };

  const sectionBodyStyle: React.CSSProperties = {
    padding: "16px 20px",
    display: "flex",
    flexDirection: "column",
    gap: 14,
  };

  const toggleRowStyle: React.CSSProperties = {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "10px 0",
    borderBottom: "1px solid var(--color-border-subtle)",
  };

  const lastToggleRowStyle: React.CSSProperties = {
    ...toggleRowStyle,
    borderBottom: "none",
    paddingBottom: 0,
  };

  if (isLoading) {
    return (
      <div style={sectionCardStyle}>
        <div style={sectionHeaderStyle}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>Completed Download Handling</span>
        </div>
        <div style={{ padding: 20 }}>
          <div className="skeleton" style={{ height: 36, borderRadius: 4 }} />
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Completed Download Handling */}
      <div style={sectionCardStyle}>
        <div style={sectionHeaderStyle}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Completed Download Handling
          </span>
          {isDirty && (
            <button
              onClick={() => update.mutate(form)}
              disabled={update.isPending}
              style={{
                background: update.isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
                color: update.isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
                border: "none",
                borderRadius: 5,
                padding: "5px 14px",
                fontSize: 12,
                fontWeight: 500,
                cursor: update.isPending ? "not-allowed" : "pointer",
              }}
            >
              {update.isPending ? "Saving…" : "Save Changes"}
            </button>
          )}
        </div>
        <div style={sectionBodyStyle}>
          <div style={toggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>Enable</p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                Automatically import completed downloads into your library
              </p>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
              <input
                type="checkbox"
                checked={form.enable_completed}
                onChange={(e) => set("enable_completed", e.currentTarget.checked)}
                style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
              />
            </label>
          </div>
          <div style={lastToggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: form.enable_completed ? "var(--color-text-primary)" : "var(--color-text-muted)" }}>
                Check For Finished Downloads Interval
              </p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                How often to check for completed downloads (minutes)
              </p>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="number"
                min="1"
                max="60"
                value={form.check_interval_minutes}
                onChange={(e) => set("check_interval_minutes", Math.max(1, parseInt(e.currentTarget.value, 10) || 1))}
                disabled={!form.enable_completed}
                style={{
                  ...inputStyle,
                  width: 72,
                  textAlign: "right",
                  opacity: form.enable_completed ? 1 : 0.4,
                }}
              />
              <span style={{ fontSize: 13, color: "var(--color-text-secondary)", whiteSpace: "nowrap" }}>minutes</span>
            </div>
          </div>
        </div>
      </div>

      {/* Failed Download Handling */}
      <div style={{ ...sectionCardStyle, marginTop: 16 }}>
        <div style={sectionHeaderStyle}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Failed Download Handling
          </span>
        </div>
        <div style={sectionBodyStyle}>
          <div style={toggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>Redownload Failed</p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                Automatically search for a replacement when a download fails
              </p>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
              <input
                type="checkbox"
                checked={form.redownload_failed}
                onChange={(e) => set("redownload_failed", e.currentTarget.checked)}
                style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
              />
            </label>
          </div>
          <div style={lastToggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
                Redownload Failed from Interactive Search
              </p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                Include failures triggered by manual interactive search
              </p>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
              <input
                type="checkbox"
                checked={form.redownload_failed_interactive}
                onChange={(e) => set("redownload_failed_interactive", e.currentTarget.checked)}
                style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
              />
            </label>
          </div>
        </div>
      </div>
    </>
  );
}

// ── Remote Path Mappings ────────────────────────────────────────────────────────

interface AddMappingForm {
  host: string;
  remote_path: string;
  local_path: string;
}

function RemotePathMappingsSection() {
  const { data: mappings, isLoading } = useRemotePathMappings();
  const create = useCreateRemotePathMapping();
  const del = useDeleteRemotePathMapping();

  const [adding, setAdding] = useState(false);
  const [addForm, setAddForm] = useState<AddMappingForm>({ host: "", remote_path: "", local_path: "" });
  const [addError, setAddError] = useState<string | null>(null);

  function setField(key: keyof AddMappingForm, val: string) {
    setAddForm((f) => ({ ...f, [key]: val }));
    setAddError(null);
  }

  function handleAdd() {
    if (!addForm.host.trim()) { setAddError("Host is required."); return; }
    if (!addForm.remote_path.trim()) { setAddError("Remote path is required."); return; }
    if (!addForm.local_path.trim()) { setAddError("Local path is required."); return; }
    create.mutate(
      { host: addForm.host.trim(), remote_path: addForm.remote_path.trim(), local_path: addForm.local_path.trim() },
      {
        onSuccess: () => {
          setAdding(false);
          setAddForm({ host: "", remote_path: "", local_path: "" });
        },
        onError: (e) => setAddError(e.message),
      }
    );
  }

  function focusBorder(e: React.FocusEvent<HTMLInputElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
  }
  function blurBorder(e: React.FocusEvent<HTMLInputElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-border-default)";
  }

  const monoInput: React.CSSProperties = {
    ...inputStyle,
    fontFamily: "var(--font-family-mono)",
    fontSize: 12,
  };

  return (
    <div
      style={{
        background: "var(--color-bg-surface)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        boxShadow: "var(--shadow-card)",
        overflow: "hidden",
        marginTop: 16,
      }}
    >
      <div
        style={{
          padding: "14px 20px",
          borderBottom: "1px solid var(--color-border-subtle)",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <div>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Remote Path Mappings
          </span>
          <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
            Translate download client paths to local filesystem paths
          </p>
        </div>
        {!adding && (
          <button
            onClick={() => setAdding(true)}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 5,
              padding: "5px 14px",
              fontSize: 12,
              fontWeight: 500,
              cursor: "pointer",
              flexShrink: 0,
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
          >
            + Add
          </button>
        )}
      </div>

      {isLoading ? (
        <div style={{ padding: 20 }}>
          <div className="skeleton" style={{ height: 36, borderRadius: 4 }} />
        </div>
      ) : (
        <>
          {/* Existing mappings */}
          {mappings && mappings.length > 0 ? (
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                  {["Host", "Remote Path", "Local Path", ""].map((h) => (
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
                {mappings.map((m: RemotePathMapping, i: number) => (
                  <tr
                    key={m.id}
                    style={{ borderBottom: i < mappings.length - 1 ? "1px solid var(--color-border-subtle)" : "none" }}
                  >
                    <td style={{ padding: "0 16px", height: 48, color: "var(--color-text-primary)", fontWeight: 500 }}>
                      {m.host}
                    </td>
                    <td style={{ padding: "0 16px", height: 48, color: "var(--color-text-secondary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
                      {m.remote_path}
                    </td>
                    <td style={{ padding: "0 16px", height: 48, color: "var(--color-text-secondary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
                      {m.local_path}
                    </td>
                    <td style={{ padding: "0 16px", height: 48, width: 1 }}>
                      <button
                        onClick={() => del.mutate(m.id)}
                        disabled={del.isPending}
                        style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 12%, transparent)")}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : !adding ? (
            <div style={{ padding: 32, textAlign: "center" }}>
              <p style={{ margin: 0, fontSize: 13, color: "var(--color-text-secondary)" }}>No remote path mappings configured</p>
              <p style={{ margin: "4px 0 0", fontSize: 12, color: "var(--color-text-muted)" }}>
                Add a mapping if your download client uses different paths than this host.
              </p>
            </div>
          ) : null}

          {/* Add form */}
          {adding && (
            <div
              style={{
                padding: 16,
                borderTop: mappings && mappings.length > 0 ? "1px solid var(--color-border-subtle)" : undefined,
                display: "flex",
                flexDirection: "column",
                gap: 12,
              }}
            >
              <p style={{ margin: 0, fontSize: 11, fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--color-text-muted)" }}>
                New Mapping
              </p>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 12 }}>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Host *</label>
                  <input
                    style={inputStyle}
                    value={addForm.host}
                    onChange={(e) => setField("host", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="192.168.1.100"
                    autoFocus
                  />
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Remote Path *</label>
                  <input
                    style={monoInput}
                    value={addForm.remote_path}
                    onChange={(e) => setField("remote_path", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="/downloads"
                  />
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Local Path *</label>
                  <input
                    style={monoInput}
                    value={addForm.local_path}
                    onChange={(e) => setField("local_path", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="/mnt/downloads"
                  />
                </div>
              </div>
              {addError && (
                <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>{addError}</p>
              )}
              <div style={{ display: "flex", gap: 8 }}>
                <button
                  onClick={handleAdd}
                  disabled={create.isPending}
                  style={{
                    background: create.isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
                    color: create.isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
                    border: "none",
                    borderRadius: 5,
                    padding: "6px 14px",
                    fontSize: 12,
                    fontWeight: 500,
                    cursor: create.isPending ? "not-allowed" : "pointer",
                  }}
                >
                  {create.isPending ? "Adding…" : "Add Mapping"}
                </button>
                <button
                  onClick={() => { setAdding(false); setAddForm({ host: "", remote_path: "", local_path: "" }); setAddError(null); }}
                  style={{
                    background: "none",
                    border: "1px solid var(--color-border-default)",
                    borderRadius: 5,
                    padding: "6px 14px",
                    fontSize: 12,
                    color: "var(--color-text-secondary)",
                    cursor: "pointer",
                  }}
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

export default function DownloadClientList() {
  const { data, isLoading, error } = useDownloadClients();
  const [modal, setModal] = useState<{ open: boolean; editing: DownloadClientConfig | null }>({
    open: false,
    editing: null,
  });

  function openCreate() { setModal({ open: true, editing: null }); }
  function openEdit(cfg: DownloadClientConfig) { setModal({ open: true, editing: cfg }); }
  function closeModal() { setModal({ open: false, editing: null }); }

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      <PageHeader
        title="Download Clients"
        description="Torrent and Usenet clients used to download releases."
        docsUrl={DOCS_URLS.downloadClients}
        action={
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
            + Add Client
          </button>
        }
      />

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
            {[1, 2].map((i) => (
              <div key={i} className="skeleton" style={{ height: 44, borderRadius: 4 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 24, fontSize: 13, color: "var(--color-text-muted)" }}>
            Failed to load download clients.
          </div>
        ) : !data?.length ? (
          <div style={{ padding: 48, textAlign: "center" }}>
            <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
              No download clients configured
            </p>
            <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
              Add qBittorrent or Deluge to start downloading.
            </p>
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Name", "Client", "Priority", "Status", ""].map((h) => (
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
                    <RowActions client={cfg} onEdit={() => openEdit(cfg)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Download handling settings */}
      <DownloadHandlingSection />

      {/* Remote path mappings */}
      <RemotePathMappingsSection />

      {/* Modal */}
      {modal.open && (
        <DownloadClientModal editing={modal.editing} onClose={closeModal} />
      )}
    </div>
  );
}
