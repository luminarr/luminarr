import { useState, useCallback, useMemo } from "react";
import {
  useSystemStatus,
  useSystemHealth,
  useTasks,
  useRunTask,
  useCheckForUpdates,
  useSystemLogs,
} from "@/api/system";
import { useMovies } from "@/api/movies";
import { useQueue } from "@/api/queue";
import { marked } from "marked";
import DOMPurify from "dompurify";
import type { HealthStatus, UpdateCheck, LogEntry } from "@/types";
import { card, sectionHeader } from "@/lib/styles";
import Pill from "@/components/Pill";
import Modal from "@/components/Modal";

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(" ");
}

function healthColor(status: HealthStatus): string {
  if (status === "healthy") return "var(--color-success)";
  if (status === "degraded") return "var(--color-warning)";
  return "var(--color-danger)";
}

function SkeletonRow({ width = "100%", height = 16 }: { width?: string | number; height?: number }) {
  return (
    <div
      className="skeleton"
      style={{ width, height, borderRadius: 4, marginBottom: 8 }}
    />
  );
}

// ── Stats strip ───────────────────────────────────────────────────────────────

function StatsStrip() {
  const movies = useMovies({ per_page: 1 });
  const queue = useQueue();

  const stats: { label: string; value: string | number; loading: boolean }[] =
    [
      {
        label: "Total Movies",
        value: movies.data?.total ?? 0,
        loading: movies.isLoading,
      },
      {
        label: "Downloading",
        value: queue.data?.length ?? 0,
        loading: queue.isLoading,
      },
    ];

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))",
        gap: 12,
        marginBottom: 24,
      }}
    >
      {stats.map(({ label, value, loading }) => (
        <div
          key={label}
          style={{
            background: "var(--color-bg-surface)",
            border: "1px solid var(--color-border-subtle)",
            borderRadius: 8,
            padding: "16px 20px",
            boxShadow: "var(--shadow-card)",
          }}
        >
          <span
            style={{
              display: "block",
              fontSize: 11,
              fontWeight: 600,
              letterSpacing: "0.08em",
              textTransform: "uppercase",
              color: "var(--color-text-muted)",
              marginBottom: 6,
            }}
          >
            {label}
          </span>
          {loading ? (
            <div
              className="skeleton"
              style={{ height: 28, width: 60, borderRadius: 4 }}
            />
          ) : (
            <span
              style={{
                fontSize: 26,
                fontWeight: 700,
                color: "var(--color-text-primary)",
                letterSpacing: "-0.02em",
                lineHeight: 1,
              }}
            >
              {value}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

// ── Update Modal ──────────────────────────────────────────────────────────────

const DOCKER_IMAGE = "ghcr.io/luminarr/luminarr";

const composeCmd = `docker compose pull\ndocker compose up -d`;
const dockerPullCmd = `docker pull ${DOCKER_IMAGE}:latest\ndocker stop luminarr\ndocker rm luminarr\ndocker run -d --name luminarr \\\n  -p 8282:8282 \\\n  -v luminarr-config:/config \\\n  ${DOCKER_IMAGE}:latest`;

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  return (
    <button
      onClick={() => {
        navigator.clipboard.writeText(text).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 2000);
        });
      }}
      style={{
        background: "none",
        border: "none",
        padding: "2px 6px",
        fontSize: 11,
        color: copied ? "var(--color-success)" : "var(--color-text-muted)",
        cursor: "pointer",
        borderRadius: 4,
      }}
      onMouseEnter={(e) => {
        if (!copied) (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
      }}
      onMouseLeave={(e) => {
        if (!copied) (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)";
      }}
    >
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}

function UpdateModal({ data, onClose }: { data: UpdateCheck; onClose: () => void }) {
  const [method, setMethod] = useState<"compose" | "pull">("compose");

  const renderedNotes = useMemo(() => {
    if (!data.release_notes) return "";
    const raw = marked.parse(data.release_notes, { async: false }) as string;
    return DOMPurify.sanitize(raw);
  }, [data.release_notes]);

  return (
    <Modal onClose={onClose} width={600} maxHeight="85vh" innerStyle={{ padding: 28, overflowY: "auto" }}>
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 20 }}>
          <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Update Available
          </h2>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              fontSize: 18,
              color: "var(--color-text-muted)",
              cursor: "pointer",
              padding: "4px 8px",
              lineHeight: 1,
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; }}
          >
            ✕
          </button>
        </div>

        {/* Version badges */}
        <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 20 }}>
          <span style={{
            padding: "4px 10px",
            borderRadius: 6,
            fontSize: 13,
            fontWeight: 500,
            fontFamily: "var(--font-family-mono)",
            background: "var(--color-bg-subtle)",
            color: "var(--color-text-secondary)",
          }}>
            {data.current_version}
          </span>
          <span style={{ color: "var(--color-text-muted)", fontSize: 13 }}>→</span>
          <span style={{
            padding: "4px 10px",
            borderRadius: 6,
            fontSize: 13,
            fontWeight: 500,
            fontFamily: "var(--font-family-mono)",
            background: "color-mix(in srgb, var(--color-success) 12%, transparent)",
            color: "var(--color-success)",
          }}>
            {data.latest_version}
          </span>
          {data.published_at && (
            <span style={{ fontSize: 12, color: "var(--color-text-muted)", marginLeft: "auto" }}>
              {new Date(data.published_at).toLocaleDateString()}
            </span>
          )}
        </div>

        {/* Release notes */}
        {renderedNotes && (
          <div style={{ marginBottom: 20 }}>
            <p style={{ ...sectionHeader, marginBottom: 8 }}>Release Notes</p>
            <div
              className="release-notes"
              dangerouslySetInnerHTML={{ __html: renderedNotes }}
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-subtle)",
                borderRadius: 6,
                padding: 14,
                fontSize: 13,
                color: "var(--color-text-secondary)",
                maxHeight: 240,
                overflow: "auto",
              }}
            />
          </div>
        )}

        {/* How to update */}
        <div style={{ marginBottom: 20 }}>
          <p style={{ ...sectionHeader, marginBottom: 10 }}>How to Update</p>

          {/* Method tabs */}
          <div style={{ display: "flex", gap: 0, marginBottom: 12, borderBottom: "1px solid var(--color-border-subtle)" }}>
            {(["compose", "pull"] as const).map((m) => (
              <button
                key={m}
                onClick={() => setMethod(m)}
                style={{
                  background: "none",
                  border: "none",
                  borderBottom: method === m ? "2px solid var(--color-accent)" : "2px solid transparent",
                  padding: "6px 14px",
                  fontSize: 12,
                  fontWeight: 500,
                  color: method === m ? "var(--color-text-primary)" : "var(--color-text-muted)",
                  cursor: "pointer",
                  marginBottom: -1,
                }}
              >
                {m === "compose" ? "Docker Compose" : "Docker Pull"}
              </button>
            ))}
          </div>

          {/* Command block */}
          <div style={{ position: "relative" }}>
            <pre style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-subtle)",
              borderRadius: 6,
              padding: 14,
              fontSize: 12,
              fontFamily: "var(--font-family-mono)",
              color: "var(--color-text-primary)",
              whiteSpace: "pre-wrap",
              margin: 0,
            }}>
              {method === "compose" ? composeCmd : dockerPullCmd}
            </pre>
            <div style={{ position: "absolute", top: 6, right: 6 }}>
              <CopyButton text={method === "compose" ? composeCmd : dockerPullCmd} />
            </div>
          </div>

          {method === "pull" && (
            <p style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 6, marginBottom: 0 }}>
              Adjust the port, volume, and container name to match your setup.
            </p>
          )}
        </div>

        {/* Footer actions */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          {data.release_url && /^https?:\/\//i.test(data.release_url) && (
            <a
              href={data.release_url}
              target="_blank"
              rel="noreferrer"
              style={{ fontSize: 13, color: "var(--color-accent)", textDecoration: "none" }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLAnchorElement).style.textDecoration = "underline"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLAnchorElement).style.textDecoration = "none"; }}
            >
              View on GitHub →
            </a>
          )}
          <button
            onClick={onClose}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "6px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)";
            }}
          >
            Close
          </button>
        </div>
    </Modal>
  );
}

// ── Section 1: Status ─────────────────────────────────────────────────────────

function StatusSection() {
  const { data, isLoading } = useSystemStatus();
  const checkUpdates = useCheckForUpdates();
  const [updateResult, setUpdateResult] = useState<UpdateCheck | null>(null);
  const [showModal, setShowModal] = useState(false);

  const handleCheck = useCallback(() => {
    checkUpdates.mutate(undefined, {
      onSuccess: (result) => {
        setUpdateResult(result);
        if (result.update_available) {
          setShowModal(true);
        }
      },
    });
  }, [checkUpdates]);

  return (
    <>
      {showModal && updateResult?.update_available && (
        <UpdateModal data={updateResult} onClose={() => setShowModal(false)} />
      )}
      <div style={card}>
        <p style={sectionHeader}>Status</p>
        {isLoading ? (
          <div>
            <SkeletonRow width="60%" />
            <SkeletonRow width="40%" />
            <SkeletonRow width="50%" />
            <SkeletonRow width="45%" />
          </div>
        ) : data ? (
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "160px 1fr",
              rowGap: 10,
              fontSize: 13,
            }}
          >
            <span style={{ color: "var(--color-text-secondary)" }}>Application</span>
            <span style={{ color: "var(--color-text-primary)" }}>
              {data.app_name} {data.version}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>Updates</span>
            <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
              <button
                onClick={handleCheck}
                disabled={checkUpdates.isPending}
                style={{
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "3px 10px",
                  fontSize: 12,
                  color: checkUpdates.isPending ? "var(--color-text-muted)" : "var(--color-text-secondary)",
                  cursor: checkUpdates.isPending ? "not-allowed" : "pointer",
                }}
                onMouseEnter={(e) => {
                  if (!checkUpdates.isPending) {
                    (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
                  }
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLButtonElement).style.color =
                    checkUpdates.isPending ? "var(--color-text-muted)" : "var(--color-text-secondary)";
                }}
              >
                {checkUpdates.isPending ? "Checking…" : "Check for updates"}
              </button>
              {updateResult && !updateResult.update_available && (
                <span style={{ fontSize: 12, color: "var(--color-success)" }}>Up to date ✓</span>
              )}
              {updateResult?.update_available && (
                <button
                  onClick={() => setShowModal(true)}
                  style={{
                    background: "none",
                    border: "none",
                    padding: 0,
                    fontSize: 12,
                    color: "var(--color-accent)",
                    cursor: "pointer",
                  }}
                  onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.textDecoration = "underline"; }}
                  onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.textDecoration = "none"; }}
                >
                  {updateResult.latest_version} available →
                </button>
              )}
              {checkUpdates.isError && (
                <span style={{ fontSize: 12, color: "var(--color-danger)" }}>
                  {(checkUpdates.error as Error).message}
                </span>
              )}
            </div>

            <span style={{ color: "var(--color-text-secondary)" }}>Go Version</span>
            <span style={{ color: "var(--color-text-primary)" }}>{data.go_version}</span>

            <span style={{ color: "var(--color-text-secondary)" }}>Build Time</span>
            <span style={{ color: "var(--color-text-primary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
              {data.build_time}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>Database</span>
            <span style={{ color: "var(--color-text-primary)" }}>
              {data.db_type}
              {data.db_path && (
                <span style={{ display: "block", fontSize: 11, fontFamily: "var(--font-family-mono)", color: "var(--color-text-muted)", marginTop: 2 }}>
                  {data.db_path}
                </span>
              )}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>Uptime</span>
            <span style={{ color: "var(--color-text-primary)" }}>{formatUptime(data.uptime_seconds)}</span>

            <span style={{ color: "var(--color-text-secondary)" }}>Started</span>
            <span style={{ color: "var(--color-text-primary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
              {data.start_time}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>AI</span>
            <Pill ok={data.ai_enabled} labelTrue="Enabled" labelFalse="Disabled" />

            <span style={{ color: "var(--color-text-secondary)" }}>TMDB</span>
            <Pill ok={data.tmdb_enabled} labelTrue="Configured" labelFalse="Not configured" />
          </div>
        ) : null}
      </div>
    </>
  );
}

// ── Section: AI Command Palette ───────────────────────────────────────────────

function AISection() {
  const { data } = useSystemStatus();
  const enabled = data?.ai_enabled ?? false;

  const examples = [
    { cmd: "grab Dune in 4K", desc: "Add and search for a release" },
    { cmd: "how many movies am I missing?", desc: "Query library stats" },
    { cmd: "go to quality profiles", desc: "Navigate to a page" },
    { cmd: "scan my libraries", desc: "Run a scheduled task" },
    { cmd: "what is a custom format?", desc: "Explain a concept" },
  ];

  return (
    <div style={card}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
        <p style={{ ...sectionHeader, marginBottom: 0 }}>AI Command Palette</p>
        <Pill ok={enabled} labelTrue="Enabled" labelFalse="Not configured" />
      </div>

      <p style={{ fontSize: 13, color: "var(--color-text-secondary)", margin: "0 0 16px" }}>
        {enabled
          ? <>Open the command palette with <kbd style={{ padding: "1px 5px", borderRadius: 3, fontSize: 11, background: "var(--color-bg-elevated)", border: "1px solid var(--color-border-subtle)", color: "var(--color-text-muted)" }}>Cmd+K</kbd> and type a natural language command. State-modifying actions (grabs, task runs) always require confirmation.</>
          : <>Add a Claude API key in <a href="/settings/app" style={{ color: "var(--color-accent)", textDecoration: "none" }}>App Settings</a> to enable natural language commands in the command palette.</>
        }
      </p>

      {enabled && (
        <div style={{ display: "flex", flexDirection: "column", gap: 0 }}>
          <div style={{
            fontSize: 11,
            fontWeight: 600,
            letterSpacing: "0.08em",
            textTransform: "uppercase",
            color: "var(--color-text-muted)",
            marginBottom: 8,
          }}>
            Example Commands
          </div>
          {examples.map(({ cmd, desc }, i) => (
            <div
              key={cmd}
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                gap: 12,
                padding: "8px 0",
                borderBottom: i < examples.length - 1 ? "1px solid var(--color-border-subtle)" : "none",
              }}
            >
              <code style={{
                fontSize: 13,
                color: "var(--color-text-primary)",
                fontFamily: "var(--font-family-mono)",
              }}>
                {cmd}
              </code>
              <span style={{ fontSize: 12, color: "var(--color-text-muted)", flexShrink: 0 }}>
                {desc}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ── Section 2: Health ─────────────────────────────────────────────────────────

function HealthSection() {
  const { data, isLoading, error } = useSystemHealth();

  return (
    <div style={card}>
      <p style={sectionHeader}>Health</p>
      {isLoading ? (
        <div>
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
        </div>
      ) : error ? (
        <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
          Failed to load health data.
        </p>
      ) : data ? (
        <div>
          {/* Overall status */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              marginBottom: 16,
              fontSize: 13,
              fontWeight: 500,
              color: healthColor(data.status),
            }}
          >
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: "50%",
                background: "currentColor",
                flexShrink: 0,
              }}
            />
            {data.status === "healthy" && "All systems healthy"}
            {data.status === "degraded" && "Degraded"}
            {data.status === "unhealthy" && "Unhealthy"}
          </div>

          {/* Individual checks */}
          {data.checks.map((check, i) => (
            <div
              key={check.name}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                height: 40,
                borderBottom:
                  i < data.checks.length - 1
                    ? "1px solid var(--color-border-subtle)"
                    : "none",
              }}
            >
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: "50%",
                  background: healthColor(check.status),
                  flexShrink: 0,
                }}
              />
              <span style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500, minWidth: 140 }}>
                {check.name}
              </span>
              <span style={{ fontSize: 13, color: "var(--color-text-secondary)" }}>
                {check.message}
              </span>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

// ── Section 3: Tasks ──────────────────────────────────────────────────────────

function TasksSection() {
  const { data, isLoading } = useTasks();
  const runTask = useRunTask();
  const [triggered, setTriggered] = useState<string | null>(null);

  function handleRun(name: string) {
    runTask.mutate(name, {
      onSuccess: () => {
        setTriggered(name);
        setTimeout(() => setTriggered((prev) => (prev === name ? null : prev)), 2000);
      },
    });
  }

  return (
    <div style={card}>
      <p style={sectionHeader}>Tasks</p>
      {isLoading ? (
        <div>
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
        </div>
      ) : !data?.length ? (
        <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
          No tasks registered.
        </p>
      ) : (
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
          <thead>
            <tr>
              {["Task", "Interval", ""].map((h) => (
                <th
                  key={h}
                  style={{
                    textAlign: "left",
                    fontSize: 11,
                    fontWeight: 600,
                    letterSpacing: "0.08em",
                    textTransform: "uppercase",
                    color: "var(--color-text-muted)",
                    paddingBottom: 8,
                    borderBottom: "1px solid var(--color-border-subtle)",
                  }}
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.map((task, i) => {
              const isPending = runTask.isPending && runTask.variables === task.name;
              const wasTriggered = triggered === task.name;

              return (
                <tr key={task.name}>
                  <td
                    style={{
                      height: 44,
                      color: "var(--color-text-primary)",
                      fontWeight: 500,
                      borderBottom:
                        i < data.length - 1
                          ? "1px solid var(--color-border-subtle)"
                          : "none",
                      paddingRight: 16,
                    }}
                  >
                    {task.name}
                  </td>
                  <td
                    style={{
                      height: 44,
                      color: "var(--color-text-secondary)",
                      fontFamily: "var(--font-family-mono)",
                      fontSize: 12,
                      borderBottom:
                        i < data.length - 1
                          ? "1px solid var(--color-border-subtle)"
                          : "none",
                      paddingRight: 16,
                    }}
                  >
                    {task.interval}
                  </td>
                  <td
                    style={{
                      height: 44,
                      textAlign: "right",
                      borderBottom:
                        i < data.length - 1
                          ? "1px solid var(--color-border-subtle)"
                          : "none",
                    }}
                  >
                    {wasTriggered ? (
                      <span style={{ fontSize: 12, color: "var(--color-success)" }}>
                        Triggered ✓
                      </span>
                    ) : (
                      <button
                        disabled={isPending}
                        onClick={() => handleRun(task.name)}
                        style={{
                          background: "var(--color-bg-elevated)",
                          border: "1px solid var(--color-border-default)",
                          color: isPending ? "var(--color-text-muted)" : "var(--color-text-secondary)",
                          borderRadius: 6,
                          padding: "4px 12px",
                          fontSize: 12,
                          cursor: isPending ? "not-allowed" : "pointer",
                        }}
                        onMouseEnter={(e) => {
                          if (!isPending) {
                            (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-subtle)";
                            (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
                          }
                        }}
                        onMouseLeave={(e) => {
                          (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)";
                          (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)";
                        }}
                      >
                        {isPending ? "Running…" : "Run Now"}
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </div>
  );
}

// ── Section 4: Logs ──────────────────────────────────────────────────────────

const LOG_LEVELS = ["all", "debug", "info", "warn", "error"] as const;

function levelBadgeColor(level: string): { color: string; bg: string } {
  switch (level) {
    case "DEBUG":
      return { color: "var(--color-text-muted)", bg: "var(--color-bg-subtle)" };
    case "INFO":
      return { color: "var(--color-accent)", bg: "color-mix(in srgb, var(--color-accent) 12%, transparent)" };
    case "WARN":
      return { color: "var(--color-warning)", bg: "color-mix(in srgb, var(--color-warning) 12%, transparent)" };
    case "ERROR":
      return { color: "var(--color-danger)", bg: "color-mix(in srgb, var(--color-danger) 12%, transparent)" };
    default:
      return { color: "var(--color-text-secondary)", bg: "var(--color-bg-subtle)" };
  }
}

function formatLogTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
}

function LogRow({ entry, isLast }: { entry: LogEntry; isLast: boolean }) {
  const [expanded, setExpanded] = useState(false);
  const hasFields = entry.fields && Object.keys(entry.fields).length > 0;
  const badge = levelBadgeColor(entry.level);

  return (
    <>
      <tr
        onClick={hasFields ? () => setExpanded(!expanded) : undefined}
        style={{ cursor: hasFields ? "pointer" : "default" }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLTableRowElement).style.background = "var(--color-bg-subtle)";
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLTableRowElement).style.background = "transparent";
        }}
      >
        <td
          style={{
            height: 36,
            fontSize: 12,
            fontFamily: "var(--font-family-mono)",
            color: "var(--color-text-muted)",
            whiteSpace: "nowrap",
            paddingRight: 12,
            borderBottom: !expanded && !isLast ? "1px solid var(--color-border-subtle)" : "none",
            verticalAlign: "middle",
          }}
        >
          {formatLogTime(entry.time)}
        </td>
        <td
          style={{
            height: 36,
            paddingRight: 12,
            borderBottom: !expanded && !isLast ? "1px solid var(--color-border-subtle)" : "none",
            verticalAlign: "middle",
          }}
        >
          <span
            style={{
              display: "inline-block",
              padding: "1px 6px",
              borderRadius: 3,
              fontSize: 10,
              fontWeight: 600,
              letterSpacing: "0.04em",
              color: badge.color,
              background: badge.bg,
              minWidth: 42,
              textAlign: "center",
            }}
          >
            {entry.level}
          </span>
        </td>
        <td
          style={{
            height: 36,
            fontSize: 13,
            color: "var(--color-text-primary)",
            borderBottom: !expanded && !isLast ? "1px solid var(--color-border-subtle)" : "none",
            verticalAlign: "middle",
            maxWidth: 0,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {entry.message}
          {hasFields && (
            <span style={{ color: "var(--color-text-muted)", fontSize: 11, marginLeft: 6 }}>
              {expanded ? "▾" : "▸"}
            </span>
          )}
        </td>
      </tr>
      {expanded && hasFields && (
        <tr>
          <td colSpan={3} style={{ borderBottom: !isLast ? "1px solid var(--color-border-subtle)" : "none", padding: 0 }}>
            <pre
              style={{
                margin: 0,
                padding: "8px 12px 8px 40px",
                fontSize: 11,
                fontFamily: "var(--font-family-mono)",
                color: "var(--color-text-secondary)",
                background: "var(--color-bg-elevated)",
                borderRadius: 0,
                whiteSpace: "pre-wrap",
                wordBreak: "break-all",
              }}
            >
              {JSON.stringify(entry.fields, null, 2)}
            </pre>
          </td>
        </tr>
      )}
    </>
  );
}

function LogsSection() {
  const [level, setLevel] = useState<string>("all");
  const filterLevel = level === "all" ? undefined : level;
  const { data, isLoading, error } = useSystemLogs(filterLevel, 200);

  return (
    <div style={card}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
        <p style={{ ...sectionHeader, marginBottom: 0 }}>Logs</p>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>Level:</span>
          <select
            value={level}
            onChange={(e) => setLevel(e.target.value)}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 4,
              padding: "2px 6px",
              fontSize: 12,
              color: "var(--color-text-primary)",
              cursor: "pointer",
            }}
          >
            {LOG_LEVELS.map((l) => (
              <option key={l} value={l}>
                {l === "all" ? "All" : l.toUpperCase()}
              </option>
            ))}
          </select>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
            Auto-refresh 10s
          </span>
        </div>
      </div>
      {isLoading ? (
        <div>
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
        </div>
      ) : error ? (
        <p style={{ fontSize: 13, color: "var(--color-danger)", margin: 0 }}>
          Failed to load logs.
        </p>
      ) : !data?.length ? (
        <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
          No log entries{filterLevel ? ` at ${filterLevel.toUpperCase()} level` : ""}.
        </p>
      ) : (
        <div style={{ maxHeight: 420, overflowY: "auto" }}>
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr>
                {["Time", "Level", "Message"].map((h) => (
                  <th
                    key={h}
                    style={{
                      textAlign: "left",
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: "0.08em",
                      textTransform: "uppercase",
                      color: "var(--color-text-muted)",
                      paddingBottom: 8,
                      borderBottom: "1px solid var(--color-border-subtle)",
                      position: "sticky",
                      top: 0,
                      background: "var(--color-bg-surface)",
                      zIndex: 1,
                    }}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.map((entry, i) => (
                <LogRow key={`${entry.time}-${i}`} entry={entry} isLast={i === data.length - 1} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function SystemPage() {
  return (
    <div style={{ padding: 24, maxWidth: 800 }}>
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
          System
        </h1>
        <p style={{ fontSize: 13, color: "var(--color-text-secondary)", margin: 0 }}>
          Runtime status, health checks, and configuration.
        </p>
      </div>

      <StatsStrip />
      <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
        <StatusSection />
        <AISection />
        <HealthSection />
        <TasksSection />
        <LogsSection />
      </div>
    </div>
  );
}
