import { useState, useEffect } from "react";
import { useMediaManagement, useUpdateMediaManagement } from "@/api/media-management";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import type { MediaManagement } from "@/types";

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
  fontFamily: "var(--font-family-mono)",
};

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 12,
  fontWeight: 500,
  color: "var(--color-text-secondary)",
  marginBottom: 6,
};

// ── Preview helper ────────────────────────────────────────────────────────────

function previewFormat(format: string): string {
  return format
    .replace("{Movie Title}", "Batman Begins")
    .replace("{Movie CleanTitle}", "Batman Begins")
    .replace("{Original Title}", "Batman Begins")
    .replace("{Release Year}", "2005")
    .replace("{Quality Full}", "Bluray-1080p")
    .replace("{MediaInfo VideoCodec}", "x264");
}

// ── Toggle row ────────────────────────────────────────────────────────────────

interface ToggleRowProps {
  label: string;
  description: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}

function ToggleRow({ label, description, checked, onChange }: ToggleRowProps) {
  return (
    <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 16 }}>
      <div>
        <div style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>{label}</div>
        <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 2 }}>{description}</div>
      </div>
      <button
        type="button"
        onClick={() => onChange(!checked)}
        style={{
          flexShrink: 0,
          width: 44,
          height: 24,
          borderRadius: 12,
          border: "none",
          cursor: "pointer",
          background: checked ? "var(--color-accent)" : "var(--color-border-default)",
          position: "relative",
          transition: "background 0.15s",
        }}
      >
        <span style={{
          position: "absolute",
          top: 3,
          left: checked ? 23 : 3,
          width: 18,
          height: 18,
          borderRadius: "50%",
          background: "white",
          transition: "left 0.15s",
        }} />
      </button>
    </div>
  );
}

// ── Section card ─────────────────────────────────────────────────────────────

function SectionCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{
      background: "var(--color-bg-surface)",
      border: "1px solid var(--color-border-subtle)",
      borderRadius: 10,
      overflow: "hidden",
    }}>
      <div style={{
        padding: "14px 20px",
        borderBottom: "1px solid var(--color-border-subtle)",
        fontSize: 13,
        fontWeight: 600,
        color: "var(--color-text-primary)",
        letterSpacing: "0.01em",
      }}>
        {title}
      </div>
      <div style={{ padding: "20px", display: "flex", flexDirection: "column", gap: 20 }}>
        {children}
      </div>
    </div>
  );
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function MediaManagementPage() {
  const { data, isLoading } = useMediaManagement();
  const update = useUpdateMediaManagement();

  const [form, setForm] = useState<MediaManagement | null>(null);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (data && !dirty) {
      setForm(data);
    }
  }, [data, dirty]);

  function set<K extends keyof MediaManagement>(key: K, value: MediaManagement[K]) {
    setForm((f) => f ? { ...f, [key]: value } : f);
    setDirty(true);
  }

  function handleSave() {
    if (!form) return;
    update.mutate(form, {
      onSuccess: () => setDirty(false),
    });
  }

  function onInputFocus(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
  }
  function onInputBlur(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-border-default)";
  }

  if (isLoading) {
    return (
      <div style={{ padding: "24px 32px", display: "flex", flexDirection: "column", gap: 16 }}>
        {[1, 2, 3].map((i) => (
          <div key={i} className="skeleton" style={{ height: 120, borderRadius: 10 }} />
        ))}
      </div>
    );
  }

  if (!form) return null;

  const dimmed = !form.rename_movies;
  const formatFieldStyle: React.CSSProperties = dimmed
    ? { opacity: 0.4, pointerEvents: "none" }
    : {};

  return (
    <div style={{ padding: "24px 32px", display: "flex", flexDirection: "column", gap: 20, maxWidth: 720 }}>
      <PageHeader
        title="Media Management"
        description="Control how movies are named and organized on disk."
        docsUrl={DOCS_URLS.mediaManagement}
        action={
          <button
            type="button"
            onClick={handleSave}
            disabled={!dirty || update.isPending}
            style={{
              padding: "8px 18px",
              background: dirty ? "var(--color-accent)" : "var(--color-bg-elevated)",
              color: dirty ? "white" : "var(--color-text-muted)",
              border: "none",
              borderRadius: 6,
              fontSize: 13,
              fontWeight: 500,
              cursor: dirty ? "pointer" : "default",
              transition: "all 0.15s",
            }}
          >
            {update.isPending ? "Saving…" : "Save Changes"}
          </button>
        }
      />

      {/* Movie Naming */}
      <SectionCard title="Movie Naming">
        <ToggleRow
          label="Rename Movies"
          description="Rename imported movie files using the format template below"
          checked={form.rename_movies}
          onChange={(v) => set("rename_movies", v)}
        />

        <div style={formatFieldStyle}>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            <label style={labelStyle}>Standard Movie Format</label>
            <input
              style={inputStyle}
              value={form.standard_movie_format}
              onChange={(e) => set("standard_movie_format", e.currentTarget.value)}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
              placeholder="{Movie Title} ({Release Year}) {Quality Full}"
            />
            <div style={{ fontSize: 11, color: "var(--color-text-muted)", fontFamily: "var(--font-family-mono)" }}>
              Preview: {previewFormat(form.standard_movie_format)}
            </div>
          </div>
        </div>

        <div style={formatFieldStyle}>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            <label style={labelStyle}>Movie Folder Format</label>
            <input
              style={inputStyle}
              value={form.movie_folder_format}
              onChange={(e) => set("movie_folder_format", e.currentTarget.value)}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
              placeholder="{Movie Title} ({Release Year})"
            />
            <div style={{ fontSize: 11, color: "var(--color-text-muted)", fontFamily: "var(--font-family-mono)" }}>
              Preview: {previewFormat(form.movie_folder_format)}
            </div>
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <label style={labelStyle}>Colon Replacement</label>
          <select
            style={{ ...inputStyle, fontFamily: "inherit" }}
            value={form.colon_replacement}
            onChange={(e) => set("colon_replacement", e.currentTarget.value as MediaManagement["colon_replacement"])}
            onFocus={onInputFocus}
            onBlur={onInputBlur}
          >
            <option value="delete">Delete — "Batman: Begins" → "Batman Begins"</option>
            <option value="dash">Dash — "Batman: Begins" → "Batman- Begins"</option>
            <option value="space-dash">Space Dash — "Batman: Begins" → "Batman - Begins"</option>
            <option value="smart">Smart — context-aware space dash</option>
          </select>
        </div>
      </SectionCard>

      {/* Importing */}
      <SectionCard title="Importing">
        <ToggleRow
          label="Import Extra Files"
          description="Copy subtitle and metadata files alongside the video when importing"
          checked={form.import_extra_files}
          onChange={(v) => set("import_extra_files", v)}
        />

        <div style={!form.import_extra_files ? { opacity: 0.4, pointerEvents: "none" } : {}}>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            <label style={labelStyle}>Extra File Extensions</label>
            <input
              style={inputStyle}
              value={form.extra_file_extensions}
              onChange={(e) => set("extra_file_extensions", e.currentTarget.value)}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
              placeholder="srt,nfo,sub,idx"
            />
            <div style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
              Comma-separated list of extensions to import alongside the video file
            </div>
          </div>
        </div>
      </SectionCard>

      {/* File Management */}
      <SectionCard title="File Management">
        <ToggleRow
          label="Unmonitor Deleted Movies"
          description="When a movie file is deleted from disk, stop monitoring that movie"
          checked={form.unmonitor_deleted_movies}
          onChange={(v) => set("unmonitor_deleted_movies", v)}
        />
      </SectionCard>

      {/* Token reference */}
      <div style={{
        background: "var(--color-bg-surface)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 10,
        padding: "16px 20px",
      }}>
        <div style={{ fontSize: 12, fontWeight: 600, color: "var(--color-text-secondary)", marginBottom: 10 }}>
          Available Tokens
        </div>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "4px 24px" }}>
          {[
            ["{Movie Title}", "Movie title as-is"],
            ["{Movie CleanTitle}", "Filesystem-safe title"],
            ["{Original Title}", "Original language title"],
            ["{Release Year}", "Release year"],
            ["{Quality Full}", "e.g. Bluray-1080p"],
            ["{MediaInfo VideoCodec}", "e.g. x264, x265"],
          ].map(([token, desc]) => (
            <div key={token} style={{ display: "flex", gap: 8, alignItems: "baseline" }}>
              <code style={{
                fontSize: 11,
                fontFamily: "var(--font-family-mono)",
                color: "var(--color-accent)",
                background: "color-mix(in srgb, var(--color-accent) 10%, transparent)",
                borderRadius: 3,
                padding: "1px 4px",
                whiteSpace: "nowrap",
              }}>
                {token}
              </code>
              <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{desc}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
