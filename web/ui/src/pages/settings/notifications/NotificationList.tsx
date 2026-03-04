import { useState } from "react";
import {
  useNotifications,
  useCreateNotification,
  useUpdateNotification,
  useDeleteNotification,
  useTestNotification,
} from "@/api/notifications";
import type { NotificationConfig, NotificationRequest } from "@/types";

// ── Constants ──────────────────────────────────────────────────────────────────

const ALL_EVENTS: { value: string; label: string }[] = [
  { value: "grab_started",    label: "Grab Started" },
  { value: "grab_failed",     label: "Grab Failed" },
  { value: "download_done",   label: "Download Done" },
  { value: "import_complete", label: "Import Complete" },
  { value: "import_failed",   label: "Import Failed" },
  { value: "health_issue",    label: "Health Issue" },
  { value: "health_ok",       label: "Health Restored" },
];

// ── Helpers ────────────────────────────────────────────────────────────────────

function strSetting(settings: Record<string, unknown>, key: string): string {
  const v = settings[key];
  return typeof v === "string" ? v : "";
}

function boolSetting(settings: Record<string, unknown>, key: string): boolean {
  return settings[key] === true;
}

function numSetting(settings: Record<string, unknown>, key: string, fallback: number): number {
  const v = settings[key];
  return typeof v === "number" ? v : fallback;
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
  on_events: Set<string>;
  // discord
  dc_webhook_url: string;
  dc_username: string;
  dc_avatar_url: string;
  // slack
  sl_webhook_url: string;
  sl_username: string;
  sl_icon_emoji: string;
  // webhook
  wh_url: string;
  wh_method: string;
  // email
  em_host: string;
  em_port: string;
  em_username: string;
  em_password: string;
  em_from: string;
  em_to: string; // comma-separated
  em_tls: boolean;
}

function emptyForm(): FormState {
  return {
    name: "", kind: "discord", enabled: true,
    on_events: new Set(["grab_started", "download_done", "import_complete"]),
    dc_webhook_url: "", dc_username: "", dc_avatar_url: "",
    sl_webhook_url: "", sl_username: "", sl_icon_emoji: "",
    wh_url: "", wh_method: "POST",
    em_host: "", em_port: "587", em_username: "", em_password: "",
    em_from: "", em_to: "", em_tls: false,
  };
}

function notifToForm(cfg: NotificationConfig): FormState {
  const s = cfg.settings;
  return {
    name: cfg.name,
    kind: cfg.kind,
    enabled: cfg.enabled,
    on_events: new Set(cfg.on_events ?? []),
    dc_webhook_url: cfg.kind === "discord" ? strSetting(s, "webhook_url") : "",
    dc_username: cfg.kind === "discord" ? strSetting(s, "username") : "",
    dc_avatar_url: cfg.kind === "discord" ? strSetting(s, "avatar_url") : "",
    sl_webhook_url: cfg.kind === "slack" ? strSetting(s, "webhook_url") : "",
    sl_username: cfg.kind === "slack" ? strSetting(s, "username") : "",
    sl_icon_emoji: cfg.kind === "slack" ? strSetting(s, "icon_emoji") : "",
    wh_url: cfg.kind === "webhook" ? strSetting(s, "url") : "",
    wh_method: cfg.kind === "webhook" ? (strSetting(s, "method") || "POST") : "POST",
    em_host: cfg.kind === "email" ? strSetting(s, "host") : "",
    em_port: cfg.kind === "email" ? String(numSetting(s, "port", 587)) : "587",
    em_username: cfg.kind === "email" ? strSetting(s, "username") : "",
    em_password: cfg.kind === "email" ? strSetting(s, "password") : "",
    em_from: cfg.kind === "email" ? strSetting(s, "from") : "",
    em_to: cfg.kind === "email" ? arrayToStr(s["to"]) : "",
    em_tls: cfg.kind === "email" ? boolSetting(s, "tls") : false,
  };
}

function arrayToStr(v: unknown): string {
  if (Array.isArray(v)) return v.filter((x) => typeof x === "string").join(", ");
  return "";
}

function formToRequest(f: FormState): NotificationRequest {
  let settings: Record<string, unknown>;

  if (f.kind === "discord") {
    settings = { webhook_url: f.dc_webhook_url.trim() };
    if (f.dc_username.trim()) settings.username = f.dc_username.trim();
    if (f.dc_avatar_url.trim()) settings.avatar_url = f.dc_avatar_url.trim();
  } else if (f.kind === "slack") {
    settings = { webhook_url: f.sl_webhook_url.trim() };
    if (f.sl_username.trim()) settings.username = f.sl_username.trim();
    if (f.sl_icon_emoji.trim()) settings.icon_emoji = f.sl_icon_emoji.trim();
  } else if (f.kind === "webhook") {
    settings = { url: f.wh_url.trim(), method: f.wh_method };
  } else {
    // email
    settings = {
      host: f.em_host.trim(),
      port: parseInt(f.em_port, 10) || 587,
      from: f.em_from.trim(),
      to: f.em_to.split(",").map((s) => s.trim()).filter(Boolean),
      tls: f.em_tls,
    };
    if (f.em_username.trim()) settings.username = f.em_username.trim();
    if (f.em_password.trim()) settings.password = f.em_password.trim();
  }

  return {
    name: f.name.trim(),
    kind: f.kind,
    enabled: f.enabled,
    settings,
    on_events: [...f.on_events],
  };
}

// ── Settings sub-forms ─────────────────────────────────────────────────────────

interface SubFormProps {
  form: FormState;
  set: <K extends keyof FormState>(field: K, value: FormState[K]) => void;
  editing: boolean;
  focusBorder: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => void;
  blurBorder: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => void;
}

function DiscordSettings({ form, set, editing, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>Webhook URL *</label>
        <input
          style={inputStyle}
          type="password"
          value={form.dc_webhook_url}
          onChange={(e) => set("dc_webhook_url", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder={editing ? "enter to change" : "https://discord.com/api/webhooks/…"}
          autoComplete="new-password"
        />
        {editing && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
            Webhook URL is masked. Enter a new value to update.
          </p>
        )}
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
        <div style={fieldStyle}>
          <label style={labelStyle}>Bot Name</label>
          <input
            style={inputStyle}
            value={form.dc_username}
            onChange={(e) => set("dc_username", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder="Luminarr"
          />
        </div>
        <div style={fieldStyle}>
          <label style={labelStyle}>Avatar URL</label>
          <input
            style={inputStyle}
            value={form.dc_avatar_url}
            onChange={(e) => set("dc_avatar_url", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder="https://…/avatar.png"
          />
        </div>
      </div>
    </>
  );
}

function SlackSettings({ form, set, editing, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>Webhook URL *</label>
        <input
          style={inputStyle}
          type="password"
          value={form.sl_webhook_url}
          onChange={(e) => set("sl_webhook_url", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder={editing ? "enter to change" : "https://hooks.slack.com/services/…"}
          autoComplete="new-password"
        />
        {editing && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
            Webhook URL is masked. Enter a new value to update.
          </p>
        )}
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
        <div style={fieldStyle}>
          <label style={labelStyle}>Bot Name</label>
          <input
            style={inputStyle}
            value={form.sl_username}
            onChange={(e) => set("sl_username", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder="Luminarr"
          />
        </div>
        <div style={fieldStyle}>
          <label style={labelStyle}>Icon Emoji</label>
          <input
            style={inputStyle}
            value={form.sl_icon_emoji}
            onChange={(e) => set("sl_icon_emoji", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder=":clapper:"
          />
        </div>
      </div>
    </>
  );
}

function WebhookSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>URL *</label>
        <input
          style={inputStyle}
          value={form.wh_url}
          onChange={(e) => set("wh_url", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="https://hooks.example.com/…"
        />
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>Method</label>
        <select
          style={{ ...inputStyle, cursor: "pointer" }}
          value={form.wh_method}
          onChange={(e) => set("wh_method", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
        >
          <option value="POST">POST</option>
          <option value="GET">GET</option>
          <option value="PUT">PUT</option>
        </select>
      </div>
    </>
  );
}

function EmailSettings({ form, set, editing, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 80px", gap: 14 }}>
        <div style={fieldStyle}>
          <label style={labelStyle}>SMTP Host *</label>
          <input
            style={inputStyle}
            value={form.em_host}
            onChange={(e) => set("em_host", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder="smtp.example.com"
          />
        </div>
        <div style={fieldStyle}>
          <label style={labelStyle}>Port</label>
          <input
            style={inputStyle}
            type="number"
            min="1"
            max="65535"
            value={form.em_port}
            onChange={(e) => set("em_port", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
          />
        </div>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
        <div style={fieldStyle}>
          <label style={labelStyle}>Username</label>
          <input
            style={inputStyle}
            value={form.em_username}
            onChange={(e) => set("em_username", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder="user@example.com"
            autoComplete="off"
          />
        </div>
        <div style={fieldStyle}>
          <label style={labelStyle}>Password</label>
          <input
            style={inputStyle}
            type="password"
            value={form.em_password}
            onChange={(e) => set("em_password", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder={editing ? "enter to change" : ""}
            autoComplete="new-password"
          />
          {editing && (
            <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
              Password is masked. Enter a new value to update.
            </p>
          )}
        </div>
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>From Address *</label>
        <input
          style={inputStyle}
          type="email"
          value={form.em_from}
          onChange={(e) => set("em_from", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="luminarr@example.com"
        />
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>To Addresses *</label>
        <input
          style={inputStyle}
          value={form.em_to}
          onChange={(e) => set("em_to", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="you@example.com, other@example.com"
        />
        <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
          Comma-separated list of recipient addresses.
        </p>
      </div>
      <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
        <input
          type="checkbox"
          checked={form.em_tls}
          onChange={(e) => set("em_tls", e.currentTarget.checked)}
          style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
        />
        <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>Use TLS (port 465)</span>
        <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>— leave unchecked for STARTTLS</span>
      </label>
    </>
  );
}

// ── Modal ──────────────────────────────────────────────────────────────────────

interface ModalProps {
  editing: NotificationConfig | null;
  onClose: () => void;
}

function NotificationModal({ editing, onClose }: ModalProps) {
  const [form, setForm] = useState<FormState>(
    editing ? notifToForm(editing) : emptyForm()
  );
  const [error, setError] = useState<string | null>(null);

  const create = useCreateNotification();
  const update = useUpdateNotification();
  const isPending = create.isPending || update.isPending;

  function set<K extends keyof FormState>(field: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [field]: value }));
    setError(null);
  }

  function toggleEvent(event: string) {
    setForm((f) => {
      const next = new Set(f.on_events);
      if (next.has(event)) next.delete(event);
      else next.add(event);
      return { ...f, on_events: next };
    });
    setError(null);
  }

  function handleSubmit() {
    if (!form.name.trim()) { setError("Name is required."); return; }
    if (form.kind === "discord" && !form.dc_webhook_url.trim()) {
      setError("Webhook URL is required."); return;
    }
    if (form.kind === "slack" && !form.sl_webhook_url.trim()) {
      setError("Webhook URL is required."); return;
    }
    if (form.kind === "webhook" && !form.wh_url.trim()) {
      setError("URL is required."); return;
    }
    if (form.kind === "email") {
      if (!form.em_host.trim()) { setError("SMTP host is required."); return; }
      if (!form.em_from.trim()) { setError("From address is required."); return; }
      if (!form.em_to.trim()) { setError("At least one recipient is required."); return; }
    }

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

  const subFormProps: SubFormProps = { form, set, editing: !!editing, focusBorder, blurBorder };

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
          width: 560,
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
            {editing ? "Edit Notification" : "Add Notification"}
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

        {/* Basic fields */}
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
                placeholder="e.g. My Discord"
                autoFocus
              />
            </div>
            <div style={fieldStyle}>
              <label style={labelStyle}>Type</label>
              <select
                style={{ ...inputStyle, cursor: "pointer" }}
                value={form.kind}
                onChange={(e) => set("kind", e.currentTarget.value)}
                onFocus={focusBorder}
                onBlur={blurBorder}
              >
                <option value="discord">Discord</option>
                <option value="slack">Slack</option>
                <option value="webhook">Webhook</option>
                <option value="email">Email</option>
              </select>
            </div>
          </div>

          {/* Events */}
          <div
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-subtle)",
              borderRadius: 8,
              padding: 16,
            }}
          >
            <p style={{ margin: "0 0 12px", fontSize: 11, fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--color-text-muted)" }}>
              Trigger Events
            </p>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
              {ALL_EVENTS.map(({ value, label }) => (
                <label
                  key={value}
                  style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}
                >
                  <input
                    type="checkbox"
                    checked={form.on_events.has(value)}
                    onChange={() => toggleEvent(value)}
                    style={{ width: 15, height: 15, cursor: "pointer", accentColor: "var(--color-accent)" }}
                  />
                  <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>{label}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Plugin settings */}
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
              {{ discord: "Discord Settings", slack: "Slack Settings", webhook: "Webhook Settings", email: "Email Settings" }[form.kind] ?? `${form.kind} Settings`}
            </p>
            {form.kind === "discord" && <DiscordSettings {...subFormProps} />}
            {form.kind === "slack" && <SlackSettings {...subFormProps} />}
            {form.kind === "webhook" && <WebhookSettings {...subFormProps} />}
            {form.kind === "email" && <EmailSettings {...subFormProps} />}
          </div>

          {/* Enabled */}
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
            {isPending ? "Saving…" : editing ? "Save Changes" : "Add Notification"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Row actions ────────────────────────────────────────────────────────────────

interface RowActionsProps {
  notif: NotificationConfig;
  onEdit: () => void;
}

function RowActions({ notif, onEdit }: RowActionsProps) {
  const [confirming, setConfirming] = useState(false);
  const [testState, setTestState] = useState<"idle" | "testing" | "ok" | "fail">("idle");
  const [testMsg, setTestMsg] = useState<string>("");

  const del = useDeleteNotification();
  const test = useTestNotification();

  function handleTest() {
    setTestState("testing");
    test.mutate(notif.id, {
      onSuccess: () => {
        setTestState("ok");
        setTimeout(() => setTestState("idle"), 3000);
      },
      onError: (e) => {
        setTestState("fail");
        setTestMsg(e.message);
        setTimeout(() => setTestState("idle"), 4000);
      },
    });
  }

  if (confirming) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
        <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>Delete?</span>
        <button
          onClick={() => del.mutate(notif.id, { onSuccess: () => setConfirming(false) })}
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

  if (testState === "ok") {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
        <span style={{ fontSize: 12, color: "var(--color-success)" }}>Sent ✓</span>
      </div>
    );
  }

  if (testState === "fail") {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
        <span
          style={{ fontSize: 12, color: "var(--color-danger)", maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
          title={testMsg}
        >
          Failed: {testMsg || "unknown error"}
        </span>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
      <button
        onClick={handleTest}
        disabled={testState === "testing"}
        style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
      >
        {testState === "testing" ? "Sending…" : "Test"}
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
  const colors: Record<string, string> = {
    discord: "var(--color-accent)",
    slack: "#E01E5A",
    webhook: "var(--color-success)",
    email: "var(--color-warning)",
  };
  const color = colors[kind] ?? "var(--color-text-secondary)";
  const labels: Record<string, string> = { discord: "Discord", slack: "Slack", webhook: "Webhook", email: "Email" };

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
        background: `color-mix(in srgb, ${color} 12%, transparent)`,
        color,
      }}
    >
      {labels[kind] ?? kind}
    </span>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

export default function NotificationList() {
  const { data, isLoading, error } = useNotifications();
  const [modal, setModal] = useState<{ open: boolean; editing: NotificationConfig | null }>({
    open: false,
    editing: null,
  });

  function openCreate() { setModal({ open: true, editing: null }); }
  function openEdit(cfg: NotificationConfig) { setModal({ open: true, editing: cfg }); }
  function closeModal() { setModal({ open: false, editing: null }); }

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 24 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", letterSpacing: "-0.01em" }}>
            Notifications
          </h1>
          <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
            Discord, Slack, webhook, and email alerts for movie events.
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
          + Add Notification
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
            {[1, 2].map((i) => (
              <div key={i} className="skeleton" style={{ height: 44, borderRadius: 4 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 24, fontSize: 13, color: "var(--color-text-muted)" }}>
            Failed to load notifications.
          </div>
        ) : !data?.length ? (
          <div style={{ padding: 48, textAlign: "center" }}>
            <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
              No notifications configured
            </p>
            <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
              Add Discord, Slack, webhook, or email alerts for movie events.
            </p>
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Name", "Type", "Events", "Status", ""].map((h) => (
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
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)", maxWidth: 220 }}>
                    {cfg.on_events?.length > 0 ? (
                      <span
                        style={{
                          display: "block",
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}
                        title={cfg.on_events.join(", ")}
                      >
                        {cfg.on_events.length} event{cfg.on_events.length !== 1 ? "s" : ""}
                      </span>
                    ) : (
                      <span style={{ color: "var(--color-text-muted)" }}>None</span>
                    )}
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
                    <RowActions notif={cfg} onEdit={() => openEdit(cfg)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Modal */}
      {modal.open && (
        <NotificationModal editing={modal.editing} onClose={closeModal} />
      )}
    </div>
  );
}
