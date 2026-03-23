import { useState } from "react";
import { toast } from "sonner";
import { Copy, Monitor, Moon, Sun } from "lucide-react";
import { useSystemStatus, useSystemConfig, useSaveConfig, useRevealApiKey } from "@/api/system";
import {
  THEME_PRESETS,
  getStoredMode,
  getStoredPreset,
  getTooltipsEnabled,
  setTooltipsEnabled,
  setThemeMode,
  setThemePreset,
  resolveMode,
} from "@/theme";
import type { ThemeMode } from "@/theme";
import { card, sectionHeader } from "@/lib/styles";
import Pill from "@/components/Pill";

function AppearanceSection() {
  const [mode, setMode] = useState<ThemeMode>(getStoredMode);
  const resolved = resolveMode(mode);
  const [darkPreset, setDarkPreset] = useState(() => getStoredPreset("dark"));
  const [lightPreset, setLightPreset] = useState(() => getStoredPreset("light"));

  const currentPresetId = resolved === "dark" ? darkPreset : lightPreset;

  function handleModeChange(next: ThemeMode) {
    setMode(next);
    setThemeMode(next);
  }

  function handlePresetSelect(presetId: string, presetMode: "dark" | "light") {
    if (presetMode === "dark") setDarkPreset(presetId);
    else setLightPreset(presetId);
    setThemePreset(presetMode, presetId);
  }

  const modeBtn = (m: ThemeMode, Icon: React.ElementType, label: string) => {
    const active = mode === m;
    return (
      <button
        key={m}
        onClick={() => handleModeChange(m)}
        style={{
          display: "flex",
          alignItems: "center",
          gap: 6,
          padding: "6px 14px",
          borderRadius: 6,
          border: active
            ? "1px solid var(--color-accent)"
            : "1px solid var(--color-border-default)",
          background: active ? "var(--color-accent-muted)" : "var(--color-bg-elevated)",
          color: active ? "var(--color-accent-hover)" : "var(--color-text-secondary)",
          fontSize: 13,
          fontWeight: 500,
          cursor: "pointer",
          transition: "background 120ms ease, border-color 120ms ease, color 120ms ease",
        }}
        onMouseEnter={(e) => {
          if (!active) {
            (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-border-strong)";
            (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
          }
        }}
        onMouseLeave={(e) => {
          if (!active) {
            (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-border-default)";
            (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)";
          }
        }}
      >
        <Icon size={14} strokeWidth={2} />
        {label}
      </button>
    );
  };

  const darkPresets = THEME_PRESETS.filter((p) => p.mode === "dark");
  const lightPresets = THEME_PRESETS.filter((p) => p.mode === "light");

  const presetGrid = (presets: typeof THEME_PRESETS) => (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(130px, 1fr))",
        gap: 10,
        marginTop: 12,
      }}
    >
      {presets.map((preset) => {
        const selected = preset.id === currentPresetId;
        return (
          <button
            key={preset.id}
            onClick={() => handlePresetSelect(preset.id, preset.mode)}
            title={preset.label}
            style={{
              display: "flex",
              flexDirection: "column",
              gap: 0,
              borderRadius: 8,
              border: selected
                ? "2px solid var(--color-accent)"
                : "2px solid var(--color-border-subtle)",
              overflow: "hidden",
              cursor: "pointer",
              background: "none",
              padding: 0,
              transition: "border-color 120ms ease, box-shadow 120ms ease",
              boxShadow: selected ? "0 0 0 1px var(--color-accent)" : "none",
            }}
            onMouseEnter={(e) => {
              if (!selected)
                (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-border-strong)";
            }}
            onMouseLeave={(e) => {
              if (!selected)
                (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-border-subtle)";
            }}
          >
            {/* Colour preview strip */}
            <div style={{ display: "flex", height: 40 }}>
              <div style={{ flex: 1, background: preset.preview.bg }} />
              <div style={{ flex: 1, background: preset.preview.surface }} />
              <div
                style={{
                  width: 12,
                  background: preset.preview.accent,
                  flexShrink: 0,
                }}
              />
            </div>
            {/* Label */}
            <div
              style={{
                padding: "6px 8px",
                background: preset.preview.surface,
                display: "flex",
                alignItems: "center",
                gap: 6,
              }}
            >
              {selected && (
                <span
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: "50%",
                    background: preset.preview.accent,
                    flexShrink: 0,
                  }}
                />
              )}
              <span
                style={{
                  fontSize: 11,
                  fontWeight: selected ? 600 : 500,
                  color: preset.preview.text,
                  whiteSpace: "nowrap",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  minWidth: 0,
                }}
              >
                {preset.label}
              </span>
            </div>
          </button>
        );
      })}
    </div>
  );

  return (
    <div style={card}>
      <p style={sectionHeader}>Appearance</p>

      {/* Mode toggle */}
      <div style={{ marginBottom: 20 }}>
        <span
          style={{
            display: "block",
            fontSize: 13,
            fontWeight: 500,
            color: "var(--color-text-primary)",
            marginBottom: 10,
          }}
        >
          Color mode
        </span>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          {modeBtn("dark", Moon, "Dark")}
          {modeBtn("light", Sun, "Light")}
          {modeBtn("system", Monitor, "System")}
        </div>
      </div>

      {/* Preset grids */}
      {(mode === "dark" || mode === "system") && (
        <div style={{ marginBottom: mode === "system" ? 20 : 0 }}>
          {mode === "system" && (
            <span
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--color-text-secondary)",
                marginBottom: 4,
              }}
            >
              Dark theme
            </span>
          )}
          {presetGrid(darkPresets)}
        </div>
      )}

      {(mode === "light" || mode === "system") && (
        <div>
          {mode === "system" && (
            <span
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--color-text-secondary)",
                marginBottom: 4,
              }}
            >
              Light theme
            </span>
          )}
          {presetGrid(lightPresets)}
        </div>
      )}
    </div>
  );
}

// ── Section 2: UI Preferences ─────────────────────────────────────────────────

function ToggleRow({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 16,
        paddingBottom: 16,
        marginBottom: 16,
        borderBottom: "1px solid var(--color-border-subtle)",
      }}
    >
      <div>
        <span
          style={{
            display: "block",
            fontSize: 13,
            fontWeight: 500,
            color: "var(--color-text-primary)",
            marginBottom: 2,
          }}
        >
          {label}
        </span>
        <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>{description}</span>
      </div>
      <button
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        style={{
          width: 40,
          height: 22,
          borderRadius: 11,
          border: "none",
          background: checked ? "var(--color-accent)" : "var(--color-bg-subtle)",
          cursor: "pointer",
          position: "relative",
          flexShrink: 0,
          transition: "background 150ms ease",
        }}
      >
        <span
          style={{
            position: "absolute",
            top: 3,
            left: checked ? 21 : 3,
            width: 16,
            height: 16,
            borderRadius: "50%",
            background: "var(--color-bg-base)",
            transition: "left 150ms ease",
          }}
        />
      </button>
    </div>
  );
}

function UIPreferencesSection() {
  const [tooltips, setTooltips] = useState(getTooltipsEnabled);

  function handleTooltipsChange(v: boolean) {
    setTooltips(v);
    setTooltipsEnabled(v);
  }

  return (
    <div style={card}>
      <p style={sectionHeader}>UI Preferences</p>
      <ToggleRow
        label="Tooltips"
        description="Show informational tooltips when hovering over UI elements."
        checked={tooltips}
        onChange={handleTooltipsChange}
      />
    </div>
  );
}

// ── Configuration ─────────────────────────────────────────────────────────────

function ConfigSection() {
  const { data: status } = useSystemStatus();
  const { data: sysConfig } = useSystemConfig();
  const saveConfig = useSaveConfig();
  const [key, setKey] = useState("");
  const [show, setShow] = useState(false);
  const [saved, setSaved] = useState(false);

  // AI API key state
  const [aiKey, setAiKey] = useState("");
  const [showAiKey, setShowAiKey] = useState(false);
  const [aiSaved, setAiSaved] = useState(false);

  function handleSave() {
    if (!key.trim()) return;
    saveConfig.mutate(
      { tmdb_api_key: key.trim() },
      {
        onSuccess: () => {
          setSaved(true);
          setKey("");
          setTimeout(() => setSaved(false), 2000);
        },
      }
    );
  }

  function handleAiSave() {
    if (!aiKey.trim()) return;
    saveConfig.mutate(
      { ai_api_key: aiKey.trim() },
      {
        onSuccess: () => {
          setAiSaved(true);
          setAiKey("");
          setTimeout(() => setAiSaved(false), 2000);
        },
      }
    );
  }

  const keySource = sysConfig?.tmdb_key_source ?? "none";

  const keyInputStyle: React.CSSProperties = {
    background: "var(--color-bg-elevated)",
    border: "1px solid var(--color-border-default)",
    borderRadius: 6,
    padding: "8px 12px",
    fontSize: 13,
    color: "var(--color-text-primary)",
    width: 320,
    outline: "none",
    fontFamily: "var(--font-family-mono)",
  };

  const showHideBtnStyle: React.CSSProperties = {
    background: "none",
    border: "none",
    cursor: "pointer",
    fontSize: 12,
    color: "var(--color-text-muted)",
    padding: "4px 6px",
  };

  function saveBtnStyle(enabled: boolean): React.CSSProperties {
    return {
      background: enabled ? "var(--color-accent)" : "var(--color-bg-subtle)",
      color: enabled ? "var(--color-accent-fg)" : "var(--color-text-muted)",
      border: "none",
      borderRadius: 6,
      padding: "8px 16px",
      fontSize: 13,
      fontWeight: 500,
      cursor: enabled ? "pointer" : "not-allowed",
    };
  }

  return (
    <div style={card}>
      <p style={sectionHeader}>Configuration</p>

      {/* TMDB API Key */}
      <div style={{ display: "flex", flexDirection: "column", gap: 8, marginBottom: 24 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span style={{ fontSize: 13, color: "var(--color-text-secondary)", minWidth: 100 }}>
            TMDB API Key
          </span>
          {status && (
            <Pill
              ok={status.tmdb_enabled}
              labelTrue={keySource === "default" ? "Using built-in key" : "Configured"}
              labelFalse="Not configured"
            />
          )}
        </div>
        {keySource === "default" && (
          <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: 0 }}>
            A default TMDB key is included. You can optionally use your own below.
          </p>
        )}

        <div style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4, flexWrap: "wrap" }}>
          <input
            type={show ? "text" : "password"}
            placeholder="Enter new TMDB API key…"
            value={key}
            onChange={(e) => setKey(e.currentTarget.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSave()}
            style={keyInputStyle}
            onFocus={(e) => {
              (e.currentTarget as HTMLInputElement).style.borderColor = "var(--color-accent)";
            }}
            onBlur={(e) => {
              (e.currentTarget as HTMLInputElement).style.borderColor =
                "var(--color-border-default)";
            }}
          />
          <button onClick={() => setShow((s) => !s)} style={showHideBtnStyle}>
            {show ? "hide" : "show"}
          </button>
          <button
            disabled={!key.trim() || saveConfig.isPending}
            onClick={handleSave}
            style={saveBtnStyle(!!key.trim() && !saveConfig.isPending)}
            onMouseEnter={(e) => {
              if (key.trim() && !saveConfig.isPending)
                (e.currentTarget as HTMLButtonElement).style.background =
                  "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              if (key.trim() && !saveConfig.isPending)
                (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
            }}
          >
            {saveConfig.isPending ? "Saving…" : "Save"}
          </button>
          {saved && (
            <span style={{ fontSize: 12, color: "var(--color-success)" }}>Saved ✓</span>
          )}
        </div>
      </div>

      {/* AI API Key */}
      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span style={{ fontSize: 13, color: "var(--color-text-secondary)", minWidth: 100 }}>
            Claude API Key
          </span>
          {status && (
            <Pill
              ok={status.ai_enabled}
              labelTrue="Configured"
              labelFalse="Not configured"
            />
          )}
        </div>
        <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: 0 }}>
          Enables AI-powered commands in the command palette (Cmd+K). Get a key from{" "}
          <a
            href="https://console.anthropic.com/settings/keys"
            target="_blank"
            rel="noopener noreferrer"
            style={{ color: "var(--color-accent)" }}
          >
            console.anthropic.com
          </a>.
        </p>

        <div style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4, flexWrap: "wrap" }}>
          <input
            type={showAiKey ? "text" : "password"}
            placeholder="sk-ant-..."
            value={aiKey}
            onChange={(e) => setAiKey(e.currentTarget.value)}
            onKeyDown={(e) => e.key === "Enter" && handleAiSave()}
            style={keyInputStyle}
            onFocus={(e) => {
              (e.currentTarget as HTMLInputElement).style.borderColor = "var(--color-accent)";
            }}
            onBlur={(e) => {
              (e.currentTarget as HTMLInputElement).style.borderColor =
                "var(--color-border-default)";
            }}
          />
          <button onClick={() => setShowAiKey((s) => !s)} style={showHideBtnStyle}>
            {showAiKey ? "hide" : "show"}
          </button>
          <button
            disabled={!aiKey.trim() || saveConfig.isPending}
            onClick={handleAiSave}
            style={saveBtnStyle(!!aiKey.trim() && !saveConfig.isPending)}
            onMouseEnter={(e) => {
              if (aiKey.trim() && !saveConfig.isPending)
                (e.currentTarget as HTMLButtonElement).style.background =
                  "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              if (aiKey.trim() && !saveConfig.isPending)
                (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
            }}
          >
            {saveConfig.isPending ? "Saving…" : "Save"}
          </button>
          {aiSaved && (
            <span style={{ fontSize: 12, color: "var(--color-success)" }}>Saved ✓</span>
          )}
        </div>

        {saveConfig.error && (
          <p style={{ fontSize: 12, color: "var(--color-danger)", margin: 0 }}>
            {saveConfig.error instanceof Error ? saveConfig.error.message : "Failed to save."}
          </p>
        )}
      </div>
    </div>
  );
}

// ── Section 5: API Key ────────────────────────────────────────────────────────

function APIKeySection() {
  const { data: sysConfig } = useSystemConfig();
  const reveal = useRevealApiKey();
  const [fullKey, setFullKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const maskedKey = sysConfig?.api_key ?? "";
  const show = fullKey !== null;

  function handleToggle() {
    if (fullKey) {
      setFullKey(null);
      return;
    }
    reveal.mutate(undefined, {
      onSuccess: (data) => setFullKey(data.api_key),
      onError: (err) => toast.error((err as Error).message),
    });
  }

  function handleCopy() {
    if (fullKey) {
      void navigator.clipboard.writeText(fullKey).then(() => {
        setCopied(true);
        toast.success("API key copied");
        setTimeout(() => setCopied(false), 2000);
      });
      return;
    }
    reveal.mutate(undefined, {
      onSuccess: (data) => {
        void navigator.clipboard.writeText(data.api_key).then(() => {
          setCopied(true);
          toast.success("API key copied");
          setTimeout(() => setCopied(false), 2000);
        });
      },
      onError: (err) => toast.error((err as Error).message),
    });
  }

  return (
    <div style={card}>
      <p style={sectionHeader}>API Key</p>
      <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "0 0 12px" }}>
        Use this key for external integrations (scripts, other *arr apps, Overseerr).
      </p>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <code
          style={{
            flex: 1,
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "8px 12px",
            fontSize: 13,
            fontFamily: "var(--font-family-mono)",
            color: "var(--color-text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            userSelect: show ? "all" : "none",
          }}
        >
          {show ? fullKey : maskedKey}
        </code>
        <button
          onClick={handleToggle}
          style={{
            background: "none",
            border: "none",
            cursor: "pointer",
            fontSize: 12,
            color: "var(--color-text-muted)",
            padding: "4px 6px",
            whiteSpace: "nowrap",
          }}
        >
          {show ? "hide" : "show"}
        </button>
        <button
          onClick={handleCopy}
          title="Copy to clipboard"
          style={{
            display: "flex",
            alignItems: "center",
            gap: 4,
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "6px 12px",
            fontSize: 12,
            color: copied ? "var(--color-success)" : "var(--color-text-secondary)",
            cursor: "pointer",
            whiteSpace: "nowrap",
          }}
          onMouseEnter={(e) => {
            if (!copied) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-subtle)";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)";
          }}
        >
          <Copy size={13} strokeWidth={2} />
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
    </div>
  );
}

// ── Section 6: Backup & Restore ───────────────────────────────────────────────

function BackupSection() {
  const [restoreMsg, setRestoreMsg] = useState<string | null>(null);
  const [restoreError, setRestoreError] = useState<string | null>(null);

  function handleDownload() {
    // Navigate directly to the backup endpoint — the server sets
    // Content-Disposition: attachment so the browser downloads the file
    // without needing a blob: URL (which CSP can block).
    window.location.href = "/api/v1/system/backup";
  }

  async function handleRestore(file: File) {
    setRestoreMsg(null);
    setRestoreError(null);
    try {
      const res = await fetch("/api/v1/system/restore", {
        method: "POST",
        headers: { "Content-Type": "application/octet-stream" },
        body: file,
      });
      if (!res.ok) throw new Error(`Server returned ${res.status}`);
      setRestoreMsg("Restore staged — restart Luminarr to apply the backup.");
    } catch (e) {
      setRestoreError((e as Error).message);
    }
  }

  const btnStyle: React.CSSProperties = {
    background: "var(--color-bg-elevated)",
    border: "1px solid var(--color-border-default)",
    borderRadius: 6,
    padding: "7px 14px",
    fontSize: 13,
    color: "var(--color-text-secondary)",
    cursor: "pointer",
    whiteSpace: "nowrap",
    flexShrink: 0,
  };

  return (
    <div style={card}>
      <p style={sectionHeader}>Backup & Restore</p>
      <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
        {/* Download */}
        <div style={{ display: "flex", alignItems: "flex-start", gap: 16, flexWrap: "wrap" }}>
          <div style={{ flex: 1 }}>
            <span
              style={{
                display: "block",
                fontSize: 13,
                fontWeight: 500,
                color: "var(--color-text-primary)",
                marginBottom: 4,
              }}
            >
              Download Backup
            </span>
            <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
              Downloads a consistent snapshot of the database.
            </span>
          </div>
          <button
            onClick={handleDownload}
            style={btnStyle}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-bg-subtle)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)";
            }}
          >
            Download Backup
          </button>
        </div>

        {/* Restore */}
        <div style={{ display: "flex", alignItems: "flex-start", gap: 16, flexWrap: "wrap" }}>
          <div style={{ flex: 1 }}>
            <span
              style={{
                display: "block",
                fontSize: 13,
                fontWeight: 500,
                color: "var(--color-text-primary)",
                marginBottom: 4,
              }}
            >
              Restore from Backup
            </span>
            <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
              Select a .db backup file. Changes take effect after restart.
            </span>
          </div>
          <label
            style={{ ...btnStyle, cursor: "pointer", display: "inline-block" }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLLabelElement).style.background = "var(--color-bg-subtle)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLLabelElement).style.background = "var(--color-bg-elevated)";
            }}
          >
            Choose File
            <input
              type="file"
              accept=".db"
              style={{ display: "none" }}
              onChange={(e) => {
                const file = e.currentTarget.files?.[0];
                if (file) void handleRestore(file);
                e.currentTarget.value = "";
              }}
            />
          </label>
        </div>

        {restoreMsg && (
          <p style={{ margin: 0, fontSize: 13, color: "var(--color-success)" }}>{restoreMsg}</p>
        )}
        {restoreError && (
          <p style={{ margin: 0, fontSize: 13, color: "var(--color-danger)" }}>{restoreError}</p>
        )}
      </div>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function AppSettingsPage() {
  return (
    <div style={{ padding: 24, maxWidth: 860 }}>
      <div style={{ marginBottom: 24 }}>
        <h1
          style={{
            fontSize: 20,
            fontWeight: 600,
            color: "var(--color-text-primary)",
            margin: 0,
            marginBottom: 4,
            letterSpacing: "-0.01em",
          }}
        >
          App Settings
        </h1>
        <p style={{ fontSize: 13, color: "var(--color-text-secondary)", margin: 0 }}>
          Appearance, preferences, and application-level configuration.
        </p>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
        <AppearanceSection />
        <UIPreferencesSection />
        <ConfigSection />
        <APIKeySection />
        <BackupSection />
      </div>
    </div>
  );
}
