import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Eye, EyeOff, ArrowLeft, CheckCircle, XCircle, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { useRadarrPreview, useRadarrImport } from "@/api/import";
import type { RadarrPreviewResult, RadarrImportOptions, RadarrImportResult, CategoryResult } from "@/types";

// ── localStorage helpers ───────────────────────────────────────────────────────

const LS_URL = "luminarr.radarr.url";
const LS_KEY = "luminarr.radarr.api_key";

function loadSaved(): { url: string; apiKey: string } | null {
  const url = localStorage.getItem(LS_URL);
  const key = localStorage.getItem(LS_KEY);
  return url && key ? { url, apiKey: key } : null;
}

function saveCreds(url: string, apiKey: string) {
  localStorage.setItem(LS_URL, url);
  localStorage.setItem(LS_KEY, apiKey);
}

function clearCreds() {
  localStorage.removeItem(LS_URL);
  localStorage.removeItem(LS_KEY);
}

// ── Shared styles ─────────────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  width: "100%",
  padding: "8px 12px",
  background: "var(--color-bg-elevated)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  color: "var(--color-text-primary)",
  fontSize: 13,
  outline: "none",
  boxSizing: "border-box",
};

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 12,
  fontWeight: 500,
  color: "var(--color-text-secondary)",
  marginBottom: 6,
  letterSpacing: "0.02em",
};

const btnPrimary: React.CSSProperties = {
  padding: "8px 20px",
  background: "var(--color-accent)",
  color: "#fff",
  border: "none",
  borderRadius: 6,
  fontSize: 13,
  fontWeight: 500,
  cursor: "pointer",
  display: "inline-flex",
  alignItems: "center",
  gap: 8,
};

const btnSecondary: React.CSSProperties = {
  padding: "8px 20px",
  background: "transparent",
  color: "var(--color-text-secondary)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  fontSize: 13,
  fontWeight: 500,
  cursor: "pointer",
};

const card: React.CSSProperties = {
  background: "var(--color-bg-surface)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 8,
  padding: 24,
};

const sectionTitle: React.CSSProperties = {
  fontSize: 13,
  fontWeight: 600,
  color: "var(--color-text-primary)",
  marginBottom: 12,
};

// ── Quick Sync card (shown when saved credentials exist) ──────────────────────

function QuickSyncCard({
  saved,
  onSync,
  onChangeSetting,
  onForget,
}: {
  saved: { url: string; apiKey: string };
  onSync: (result: RadarrPreviewResult) => void;
  onChangeSetting: () => void;
  onForget: () => void;
}) {
  const preview = useRadarrPreview();

  function handleSync() {
    preview.mutate(
      { url: saved.url, api_key: saved.apiKey },
      { onSuccess: (result) => onSync(result) },
    );
  }

  return (
    <div
      style={{
        ...card,
        display: "flex",
        flexDirection: "column",
        gap: 14,
        marginBottom: 28,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <RefreshCw size={15} color="var(--color-accent)" />
        <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>
          Saved connection
        </span>
      </div>

      <div
        style={{
          fontSize: 12,
          color: "var(--color-text-secondary)",
          fontFamily: "monospace",
          background: "var(--color-bg-elevated)",
          padding: "6px 10px",
          borderRadius: 4,
        }}
      >
        {saved.url}
      </div>

      {preview.error && (
        <div
          style={{
            padding: "8px 12px",
            background: "var(--color-danger-muted, rgba(239,68,68,0.08))",
            border: "1px solid var(--color-danger, #ef4444)",
            borderRadius: 6,
            fontSize: 12,
            color: "var(--color-danger, #ef4444)",
          }}
        >
          {preview.error.message}
        </div>
      )}

      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button style={btnPrimary} onClick={handleSync} disabled={preview.isPending}>
          {preview.isPending && (
            <Loader2 size={14} style={{ animation: "spin 1s linear infinite" }} />
          )}
          {preview.isPending ? "Connecting…" : "Sync"}
        </button>
        <button
          style={{ ...btnSecondary, padding: "8px 14px" }}
          onClick={onChangeSetting}
          disabled={preview.isPending}
        >
          Change settings
        </button>
        <button
          onClick={onForget}
          disabled={preview.isPending}
          style={{
            background: "none",
            border: "none",
            cursor: "pointer",
            fontSize: 12,
            color: "var(--color-text-muted)",
            padding: 0,
            marginLeft: "auto",
          }}
        >
          Forget
        </button>
      </div>
    </div>
  );
}

// ── Step 1: Connect ───────────────────────────────────────────────────────────

function ConnectStep({
  initialUrl,
  initialApiKey,
  onPreview,
}: {
  initialUrl?: string;
  initialApiKey?: string;
  onPreview: (url: string, apiKey: string, result: RadarrPreviewResult, remember: boolean) => void;
}) {
  const [url, setUrl] = useState(initialUrl ?? "http://localhost:7878");
  const [apiKey, setApiKey] = useState(initialApiKey ?? "");
  const [showKey, setShowKey] = useState(false);
  const [remember, setRemember] = useState(Boolean(initialUrl));
  const preview = useRadarrPreview();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    preview.mutate(
      { url: url.trim(), api_key: apiKey.trim() },
      { onSuccess: (result) => onPreview(url.trim(), apiKey.trim(), result, remember) },
    );
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <div>
        <label style={labelStyle}>Radarr URL</label>
        <input
          style={inputStyle}
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="http://localhost:7878"
          required
        />
        <p style={{ margin: "6px 0 0", fontSize: 12, color: "var(--color-text-muted)" }}>
          The base URL of your Radarr instance. Include protocol and port.
        </p>
      </div>

      <div>
        <label style={labelStyle}>API Key</label>
        <div style={{ position: "relative" }}>
          <input
            style={{ ...inputStyle, paddingRight: 36 }}
            type={showKey ? "text" : "password"}
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="Radarr API key"
            required
          />
          <button
            type="button"
            onClick={() => setShowKey((v) => !v)}
            style={{
              position: "absolute",
              right: 10,
              top: "50%",
              transform: "translateY(-50%)",
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              padding: 0,
              display: "flex",
            }}
          >
            {showKey ? <EyeOff size={15} /> : <Eye size={15} />}
          </button>
        </div>
        <p style={{ margin: "6px 0 0", fontSize: 12, color: "var(--color-text-muted)" }}>
          Found in Radarr → Settings → General → Security.
        </p>
      </div>

      {preview.error && (
        <div
          style={{
            padding: "10px 14px",
            background: "var(--color-danger-muted, rgba(239,68,68,0.08))",
            border: "1px solid var(--color-danger, #ef4444)",
            borderRadius: 6,
            fontSize: 13,
            color: "var(--color-danger, #ef4444)",
          }}
        >
          {preview.error.message}
        </div>
      )}

      <label
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          cursor: "pointer",
          fontSize: 12,
          color: "var(--color-text-secondary)",
        }}
      >
        <input
          type="checkbox"
          checked={remember}
          onChange={(e) => setRemember(e.target.checked)}
          style={{ accentColor: "var(--color-accent)" }}
        />
        Remember this connection
        <span style={{ color: "var(--color-text-muted)" }}>(stored locally in your browser)</span>
      </label>

      <div>
        <button
          type="submit"
          style={btnPrimary}
          disabled={preview.isPending}
        >
          {preview.isPending && <Loader2 size={14} style={{ animation: "spin 1s linear infinite" }} />}
          {preview.isPending ? "Connecting…" : "Connect & Preview"}
        </button>
      </div>
    </form>
  );
}

// ── Step 2: Preview + select ──────────────────────────────────────────────────

function CheckboxRow({
  label,
  checked,
  onChange,
  detail,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  detail?: string;
}) {
  return (
    <label
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: 10,
        cursor: "pointer",
        padding: "10px 0",
        borderBottom: "1px solid var(--color-border-subtle)",
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        style={{ marginTop: 2, accentColor: "var(--color-accent)" }}
      />
      <div>
        <div style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
          {label}
        </div>
        {detail && (
          <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 2 }}>
            {detail}
          </div>
        )}
      </div>
    </label>
  );
}

function PreviewStep({
  url,
  apiKey,
  preview,
  onBack,
  onDone,
}: {
  url: string;
  apiKey: string;
  preview: RadarrPreviewResult;
  onBack: () => void;
  onDone: (result: RadarrImportResult) => void;
}) {
  const [opts, setOpts] = useState<RadarrImportOptions>({
    quality_profiles: preview.quality_profiles.length > 0,
    libraries: preview.root_folders.length > 0,
    indexers: preview.indexers.length > 0,
    download_clients: preview.download_clients.length > 0,
    movies: preview.movie_count > 0,
  });

  const importMutation = useRadarrImport();

  function toggle(key: keyof RadarrImportOptions) {
    setOpts((o) => ({ ...o, [key]: !o[key] }));
  }

  function handleImport() {
    importMutation.mutate(
      { url, api_key: apiKey, options: opts },
      { onSuccess: (result) => onDone(result) },
    );
  }

  const supportedIndexers = preview.indexers.filter((i) => i.kind !== "");
  const skippedIndexers = preview.indexers.filter((i) => i.kind === "");
  const supportedClients = preview.download_clients.filter((c) => c.kind !== "");
  const skippedClients = preview.download_clients.filter((c) => c.kind === "");

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      {/* Summary banner */}
      <div
        style={{
          padding: "14px 18px",
          background: "var(--color-accent-muted)",
          border: "1px solid var(--color-accent)",
          borderRadius: 8,
          fontSize: 13,
          color: "var(--color-text-primary)",
        }}
      >
        Connected to Radarr <strong>{preview.version}</strong>. Found{" "}
        <strong>{preview.movie_count}</strong> movies,{" "}
        <strong>{preview.quality_profiles.length}</strong> quality profiles,{" "}
        <strong>{supportedIndexers.length}</strong> supported indexers,{" "}
        <strong>{supportedClients.length}</strong> supported download clients.
      </div>

      {/* Category selection */}
      <div style={card}>
        <div style={sectionTitle}>What to import</div>

        <CheckboxRow
          label={`Quality Profiles (${preview.quality_profiles.length})`}
          checked={opts.quality_profiles}
          onChange={() => toggle("quality_profiles")}
          detail={preview.quality_profiles.map((p) => p.name).join(", ") || "None"}
        />

        <CheckboxRow
          label={`Libraries from root folders (${preview.root_folders.length})`}
          checked={opts.libraries}
          onChange={() => toggle("libraries")}
          detail={preview.root_folders.map((f) => f.path).join(", ") || "None"}
        />

        <CheckboxRow
          label={`Indexers (${supportedIndexers.length} supported${skippedIndexers.length > 0 ? `, ${skippedIndexers.length} unsupported` : ""})`}
          checked={opts.indexers}
          onChange={() => toggle("indexers")}
          detail={
            supportedIndexers.length > 0
              ? supportedIndexers.map((i) => `${i.name} (${i.kind})`).join(", ")
              : "None supported (only Torznab/Newznab)"
          }
        />

        <CheckboxRow
          label={`Download Clients (${supportedClients.length} supported${skippedClients.length > 0 ? `, ${skippedClients.length} unsupported` : ""})`}
          checked={opts.download_clients}
          onChange={() => toggle("download_clients")}
          detail={
            supportedClients.length > 0
              ? supportedClients.map((c) => `${c.name} (${c.kind})`).join(", ")
              : "None supported (only qBittorrent/Deluge)"
          }
        />

        <CheckboxRow
          label={`Movies (${preview.movie_count})`}
          checked={opts.movies}
          onChange={() => toggle("movies")}
          detail="Movies already in Luminarr (by TMDB ID) will be skipped."
        />
      </div>

      {importMutation.error && (
        <div
          style={{
            padding: "10px 14px",
            background: "var(--color-danger-muted, rgba(239,68,68,0.08))",
            border: "1px solid var(--color-danger, #ef4444)",
            borderRadius: 6,
            fontSize: 13,
            color: "var(--color-danger, #ef4444)",
          }}
        >
          {importMutation.error.message}
        </div>
      )}

      <div style={{ display: "flex", gap: 12 }}>
        <button style={btnSecondary} onClick={onBack} disabled={importMutation.isPending}>
          <span style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <ArrowLeft size={14} /> Back
          </span>
        </button>
        <button style={btnPrimary} onClick={handleImport} disabled={importMutation.isPending}>
          {importMutation.isPending && (
            <Loader2 size={14} style={{ animation: "spin 1s linear infinite" }} />
          )}
          {importMutation.isPending
            ? `Importing ${preview.movie_count > 0 && opts.movies ? `${preview.movie_count} movies…` : "…"}`
            : "Import"}
        </button>
      </div>
    </div>
  );
}

// ── Step 3: Done ──────────────────────────────────────────────────────────────

function CategoryRow({ label, result }: { label: string; result: CategoryResult }) {
  const hasIssues = result.failed > 0;
  const allSkipped = result.imported === 0 && result.skipped > 0 && result.failed === 0;

  return (
    <tr>
      <td
        style={{
          padding: "10px 12px",
          borderBottom: "1px solid var(--color-border-subtle)",
          fontSize: 13,
          color: "var(--color-text-primary)",
          fontWeight: 500,
        }}
      >
        {label}
      </td>
      <td
        style={{
          padding: "10px 12px",
          borderBottom: "1px solid var(--color-border-subtle)",
          fontSize: 13,
          color: "var(--color-success, #22c55e)",
          textAlign: "center",
        }}
      >
        {result.imported}
      </td>
      <td
        style={{
          padding: "10px 12px",
          borderBottom: "1px solid var(--color-border-subtle)",
          fontSize: 13,
          color: allSkipped ? "var(--color-text-muted)" : "var(--color-warning, #f59e0b)",
          textAlign: "center",
        }}
      >
        {result.skipped}
      </td>
      <td
        style={{
          padding: "10px 12px",
          borderBottom: "1px solid var(--color-border-subtle)",
          fontSize: 13,
          color: hasIssues ? "var(--color-danger, #ef4444)" : "var(--color-text-muted)",
          textAlign: "center",
        }}
      >
        {result.failed}
      </td>
    </tr>
  );
}

function DoneStep({ result }: { result: RadarrImportResult }) {
  const navigate = useNavigate();
  const [showErrors, setShowErrors] = useState(false);

  const totalImported =
    result.quality_profiles.imported +
    result.libraries.imported +
    result.indexers.imported +
    result.download_clients.imported +
    result.movies.imported;

  const hasErrors = result.errors.length > 0;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      {/* Status banner */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          padding: "14px 18px",
          background: hasErrors
            ? "var(--color-warning-muted, rgba(245,158,11,0.08))"
            : "var(--color-success-muted, rgba(34,197,94,0.08))",
          border: `1px solid ${hasErrors ? "var(--color-warning, #f59e0b)" : "var(--color-success, #22c55e)"}`,
          borderRadius: 8,
        }}
      >
        {hasErrors ? (
          <AlertCircle size={20} color="var(--color-warning, #f59e0b)" />
        ) : (
          <CheckCircle size={20} color="var(--color-success, #22c55e)" />
        )}
        <div>
          <div style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Import {hasErrors ? "completed with warnings" : "successful"}
          </div>
          <div style={{ fontSize: 12, color: "var(--color-text-secondary)", marginTop: 2 }}>
            {totalImported} records imported
            {hasErrors ? `, ${result.errors.length} warnings` : ""}
          </div>
        </div>
      </div>

      {/* Results table */}
      <div style={card}>
        <div style={sectionTitle}>Import summary</div>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr>
              <th
                style={{
                  padding: "8px 12px",
                  fontSize: 11,
                  fontWeight: 600,
                  color: "var(--color-text-muted)",
                  letterSpacing: "0.06em",
                  textTransform: "uppercase",
                  textAlign: "left",
                  borderBottom: "1px solid var(--color-border-default)",
                }}
              >
                Category
              </th>
              <th
                style={{
                  padding: "8px 12px",
                  fontSize: 11,
                  fontWeight: 600,
                  color: "var(--color-text-muted)",
                  letterSpacing: "0.06em",
                  textTransform: "uppercase",
                  textAlign: "center",
                  borderBottom: "1px solid var(--color-border-default)",
                }}
              >
                Imported
              </th>
              <th
                style={{
                  padding: "8px 12px",
                  fontSize: 11,
                  fontWeight: 600,
                  color: "var(--color-text-muted)",
                  letterSpacing: "0.06em",
                  textTransform: "uppercase",
                  textAlign: "center",
                  borderBottom: "1px solid var(--color-border-default)",
                }}
              >
                Skipped
              </th>
              <th
                style={{
                  padding: "8px 12px",
                  fontSize: 11,
                  fontWeight: 600,
                  color: "var(--color-text-muted)",
                  letterSpacing: "0.06em",
                  textTransform: "uppercase",
                  textAlign: "center",
                  borderBottom: "1px solid var(--color-border-default)",
                }}
              >
                Failed
              </th>
            </tr>
          </thead>
          <tbody>
            <CategoryRow label="Quality Profiles" result={result.quality_profiles} />
            <CategoryRow label="Libraries" result={result.libraries} />
            <CategoryRow label="Indexers" result={result.indexers} />
            <CategoryRow label="Download Clients" result={result.download_clients} />
            <CategoryRow label="Movies" result={result.movies} />
          </tbody>
        </table>
      </div>

      {/* Error list */}
      {hasErrors && (
        <div style={card}>
          <button
            onClick={() => setShowErrors((v) => !v)}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              background: "none",
              border: "none",
              cursor: "pointer",
              padding: 0,
              fontSize: 13,
              fontWeight: 600,
              color: "var(--color-warning, #f59e0b)",
            }}
          >
            <XCircle size={15} />
            {showErrors ? "Hide" : "Show"} {result.errors.length} warnings
          </button>
          {showErrors && (
            <ul
              style={{
                margin: "12px 0 0",
                padding: "0 0 0 4px",
                listStyle: "none",
                display: "flex",
                flexDirection: "column",
                gap: 6,
              }}
            >
              {result.errors.map((err, i) => (
                <li
                  key={i}
                  style={{
                    fontSize: 12,
                    color: "var(--color-text-secondary)",
                    fontFamily: "monospace",
                    padding: "6px 10px",
                    background: "var(--color-bg-elevated)",
                    borderRadius: 4,
                  }}
                >
                  {err}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <div>
        <button style={btnPrimary} onClick={() => navigate("/")}>
          Go to Library
        </button>
      </div>
    </div>
  );
}

// ── Main page ─────────────────────────────────────────────────────────────────

type Step = "connect" | "preview" | "done";

export default function ImportPage() {
  const [step, setStep] = useState<Step>("connect");
  const [url, setUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [preview, setPreview] = useState<RadarrPreviewResult | null>(null);
  const [importResult, setImportResult] = useState<RadarrImportResult | null>(null);
  // Saved credentials from localStorage — loaded once on mount.
  const [saved, setSaved] = useState(loadSaved);
  // When true, show the manual form even if saved credentials exist.
  const [showForm, setShowForm] = useState(false);

  function handlePreview(u: string, k: string, result: RadarrPreviewResult, remember: boolean) {
    if (remember) {
      saveCreds(u, k);
      setSaved({ url: u, apiKey: k });
    } else {
      clearCreds();
      setSaved(null);
    }
    setUrl(u);
    setApiKey(k);
    setPreview(result);
    setStep("preview");
  }

  function handleQuickSync(result: RadarrPreviewResult) {
    if (!saved) return;
    setUrl(saved.url);
    setApiKey(saved.apiKey);
    setPreview(result);
    setStep("preview");
  }

  function handleForget() {
    clearCreds();
    setSaved(null);
    setShowForm(false);
  }

  function handleDone(result: RadarrImportResult) {
    setImportResult(result);
    setStep("done");
  }

  const stepLabels: { key: Step; label: string }[] = [
    { key: "connect", label: "1. Connect" },
    { key: "preview", label: "2. Select" },
    { key: "done", label: "3. Done" },
  ];

  // Whether to render the QuickSync card instead of the form.
  const showQuickSync = step === "connect" && saved !== null && !showForm;

  return (
    <div style={{ padding: 24, maxWidth: 640 }}>
      {/* Page header */}
      <div style={{ marginBottom: 24 }}>
        <h1
          style={{
            margin: 0,
            fontSize: 20,
            fontWeight: 600,
            color: "var(--color-text-primary)",
            letterSpacing: "-0.01em",
          }}
        >
          Import from Radarr
        </h1>
        <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
          Import your Radarr configuration into Luminarr. Requires a running Radarr instance.
        </p>
      </div>

      {/* Step indicator */}
      <div style={{ display: "flex", gap: 0, marginBottom: 28 }}>
        {stepLabels.map((s, i) => {
          const isActive = s.key === step;
          const isDone =
            (step === "preview" && s.key === "connect") ||
            (step === "done" && (s.key === "connect" || s.key === "preview"));
          return (
            <div
              key={s.key}
              style={{
                flex: 1,
                padding: "8px 0",
                textAlign: "center",
                fontSize: 12,
                fontWeight: 600,
                color: isActive
                  ? "var(--color-accent)"
                  : isDone
                    ? "var(--color-text-secondary)"
                    : "var(--color-text-muted)",
                borderBottom: isActive
                  ? "2px solid var(--color-accent)"
                  : "2px solid var(--color-border-subtle)",
                letterSpacing: "0.02em",
                transition: "color 150ms ease, border-color 150ms ease",
                ...(i > 0 ? { marginLeft: -1 } : {}),
              }}
            >
              {s.label}
            </div>
          );
        })}
      </div>

      {/* Step content */}
      {step === "connect" && showQuickSync && saved && (
        <>
          <QuickSyncCard
            saved={saved}
            onSync={handleQuickSync}
            onChangeSetting={() => setShowForm(true)}
            onForget={handleForget}
          />
        </>
      )}

      {step === "connect" && !showQuickSync && (
        <>
          {showForm && saved && (
            <button
              onClick={() => setShowForm(false)}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 6,
                background: "none",
                border: "none",
                cursor: "pointer",
                fontSize: 12,
                color: "var(--color-text-muted)",
                padding: "0 0 16px",
              }}
            >
              <ArrowLeft size={12} /> Back to saved connection
            </button>
          )}
          <ConnectStep
            initialUrl={saved?.url}
            initialApiKey={saved?.apiKey}
            onPreview={handlePreview}
          />
        </>
      )}

      {step === "preview" && preview && (
        <PreviewStep
          url={url}
          apiKey={apiKey}
          preview={preview}
          onBack={() => setStep("connect")}
          onDone={handleDone}
        />
      )}
      {step === "done" && importResult && <DoneStep result={importResult} />}
    </div>
  );
}
