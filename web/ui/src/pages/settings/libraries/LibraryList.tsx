import { useState, useEffect, useRef } from "react";
import {
  useLibraries,
  useCreateLibrary,
  useUpdateLibrary,
  useDeleteLibrary,
  useScanLibrary,
  useCandidates,
  useRescanDisk,
  useImportFile,
} from "@/api/libraries";
import { useLookupMovies } from "@/api/movies";
import { useQualityProfiles } from "@/api/quality-profiles";
import { useMediaManagement } from "@/api/media-management";
import { DirPicker } from "@/components/DirPicker";
import type { Library, LibraryRequest, DiskFile, DiskFileTMDBMatch, TMDBResult } from "@/types";

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

const fieldStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 0,
};

// ── Form state ────────────────────────────────────────────────────────────────

interface FormState {
  name: string;
  root_path: string;
  default_quality_profile_id: string;
  min_free_space_gb: string; // string for controlled input, parsed on submit
  naming_format: string;
  folder_format: string;
}

function emptyForm(): FormState {
  return { name: "", root_path: "", default_quality_profile_id: "", min_free_space_gb: "0", naming_format: "", folder_format: "" };
}

function libraryToForm(lib: Library): FormState {
  return {
    name: lib.name,
    root_path: lib.root_path,
    default_quality_profile_id: lib.default_quality_profile_id ?? "",
    min_free_space_gb: String(lib.min_free_space_gb),
    naming_format: lib.naming_format ?? "",
    folder_format: lib.folder_format ?? "",
  };
}

function formToRequest(f: FormState): LibraryRequest {
  return {
    name: f.name.trim(),
    root_path: f.root_path.trim(),
    default_quality_profile_id: f.default_quality_profile_id || undefined,
    min_free_space_gb: parseInt(f.min_free_space_gb, 10) || 0,
    naming_format: f.naming_format.trim() || undefined,
    folder_format: f.folder_format.trim() || undefined,
  };
}

// ── Library edit modal ────────────────────────────────────────────────────────

interface LibraryModalProps {
  editing: Library | null;
  onClose: () => void;
}

function LibraryModal({ editing, onClose }: LibraryModalProps) {
  const [form, setForm] = useState<FormState>(
    editing ? libraryToForm(editing) : emptyForm()
  );
  const [error, setError] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);

  const { data: profiles } = useQualityProfiles();
  const { data: mm } = useMediaManagement();
  const createLib = useCreateLibrary();
  const updateLib = useUpdateLibrary();

  const isPending = createLib.isPending || updateLib.isPending;

  function set(field: keyof FormState, value: string) {
    setForm((f) => ({ ...f, [field]: value }));
    setError(null);
  }

  function handleSubmit() {
    if (!form.name.trim()) { setError("Name is required."); return; }
    if (!form.root_path.trim()) { setError("Root path is required."); return; }

    const body = formToRequest(form);

    if (editing) {
      updateLib.mutate({ id: editing.id, ...body }, { onSuccess: onClose, onError: (e) => setError(e.message) });
    } else {
      createLib.mutate(body, { onSuccess: onClose, onError: (e) => setError(e.message) });
    }
  }

  function onInputFocus(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
  }
  function onInputBlur(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-border-default)";
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
            {editing ? "Edit Library" : "Add Library"}
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
          <div style={fieldStyle}>
            <label style={labelStyle}>Name *</label>
            <input
              style={inputStyle}
              value={form.name}
              onChange={(e) => set("name", e.currentTarget.value)}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
              placeholder="e.g. Movies"
              autoFocus
            />
          </div>

          <div style={fieldStyle}>
            <label style={labelStyle}>Root Path *</label>
            <div style={{ display: "flex", gap: 6 }}>
              <input
                style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12, flex: 1 }}
                value={form.root_path}
                onChange={(e) => set("root_path", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
                placeholder="/data/movies"
              />
              <button
                type="button"
                onClick={() => setPickerOpen(true)}
                title="Browse for folder"
                style={{
                  flexShrink: 0,
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "0 12px",
                  fontSize: 13,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                  whiteSpace: "nowrap",
                }}
                onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-accent)"; (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
                onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-border-default)"; (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)"; }}
              >
                Browse…
              </button>
            </div>
          </div>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
            <div style={fieldStyle}>
              <label style={labelStyle}>Quality Profile</label>
              <select
                style={{ ...inputStyle, cursor: "pointer" }}
                value={form.default_quality_profile_id}
                onChange={(e) => set("default_quality_profile_id", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
              >
                <option value="">None</option>
                {profiles?.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>

            <div style={fieldStyle}>
              <label style={labelStyle}>Min Free Space (GB)</label>
              <input
                style={inputStyle}
                type="number"
                min="0"
                value={form.min_free_space_gb}
                onChange={(e) => set("min_free_space_gb", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
              />
            </div>

            <div style={fieldStyle}>
              <label style={labelStyle}>File Naming Format Override</label>
              <input
                style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12 }}
                value={form.naming_format}
                onChange={(e) => set("naming_format", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
                placeholder={mm?.standard_movie_format ?? "Leave blank to use global default"}
              />
            </div>

            <div style={fieldStyle}>
              <label style={labelStyle}>Folder Format Override</label>
              <input
                style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12 }}
                value={form.folder_format}
                onChange={(e) => set("folder_format", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
                placeholder={mm?.movie_folder_format ?? "Leave blank to use global default"}
              />
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
            {isPending ? "Saving…" : editing ? "Save Changes" : "Add Library"}
          </button>
        </div>
      </div>

      <DirPicker
        key={pickerOpen ? "open" : "closed"}
        open={pickerOpen}
        value={form.root_path}
        onSelect={(path) => set("root_path", path)}
        onClose={() => setPickerOpen(false)}
      />
    </div>
  );
}

// ── Disk scan modal ───────────────────────────────────────────────────────────

interface FileRowState {
  file: DiskFile;
  selected: boolean;
  match: TMDBResult | null;
  searchQuery: string;
  searchOpen: boolean;
  searchResults: TMDBResult[];
  importing: boolean;
  imported: boolean;
  autoMatchLoading: boolean;
  autoMatched: boolean;
}

function formatBytes(b: number): string {
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1024 * 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(1)} MB`;
  return `${(b / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function basename(path: string): string {
  return path.split("/").pop() ?? path;
}

// ── Auto-match helpers ────────────────────────────────────────────────────────

/** Converts a pre-computed DB match into the TMDBResult shape used by the UI. */
function dbMatchToResult(m: DiskFileTMDBMatch | undefined): TMDBResult | null {
  if (!m) return null;
  return {
    tmdb_id: m.tmdb_id,
    title: m.title,
    original_title: m.original_title,
    year: m.year,
    overview: "",
    release_date: "",
    poster_path: "",
    backdrop_path: "",
    popularity: 0,
  };
}

function normalizeTitle(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9\s]/g, "").replace(/\s+/g, " ").trim();
}

/**
 * Returns the first TMDB result whose (normalized) title matches parsedTitle
 * AND whose year matches parsedYear. Returns null if no confident match is found
 * or if parsedYear is 0 (too ambiguous without a year).
 */
function pickBestMatch(
  results: TMDBResult[],
  parsedTitle: string,
  parsedYear: number,
): TMDBResult | null {
  if (parsedYear === 0 || results.length === 0) return null;
  const norm = normalizeTitle(parsedTitle);
  for (const r of results) {
    if (
      (normalizeTitle(r.title) === norm || normalizeTitle(r.original_title) === norm) &&
      r.year === parsedYear
    ) {
      return r;
    }
  }
  return null;
}

interface DiskScanModalProps {
  library: Library;
  onClose: () => void;
}

function DiskScanModal({ library, onClose }: DiskScanModalProps) {
  const { data: diskFiles, isLoading, error: scanError } = useCandidates(library.id);
  const rescan = useRescanDisk();
  const lookupMovies = useLookupMovies();
  const importFile = useImportFile();
  const autoMatchRef = useRef(false);

  const [rows, setRows] = useState<Map<string, FileRowState>>(new Map());
  const [showUnmatched, setShowUnmatched] = useState(true);
  const [isImporting, setIsImporting] = useState(false);
  const [importDone, setImportDone] = useState(0);
  const [importTotal, setImportTotal] = useState(0);

  // Populate rows when disk scan results arrive.
  useEffect(() => {
    if (!diskFiles) return;
    setRows((prev) => {
      const next = new Map<string, FileRowState>();

      // Always keep imported rows — they should remain visible with "✓ Imported".
      for (const [path, row] of prev) {
        if (row.imported) next.set(path, row);
      }

      // Sync with the current candidate list: add new files, preserve existing
      // row state (user's match selections etc.), drop rows no longer present.
      for (const f of diskFiles) {
        if (next.has(f.path)) continue; // already kept as imported
        const existing = prev.get(f.path);
        if (existing) {
          next.set(f.path, existing); // preserve match/selected state
        } else {
          const preMatch = dbMatchToResult(f.tmdb_match);
          next.set(f.path, {
            file: f,
            selected: false,
            match: preMatch,
            searchQuery: f.parsed_title + (f.parsed_year ? ` ${f.parsed_year}` : ""),
            searchOpen: false,
            searchResults: [],
            importing: false,
            imported: false,
            autoMatchLoading: false,
            autoMatched: !!preMatch,
          });
        }
      }

      return next;
    });
  }, [diskFiles]);

  // Auto-match effect: fires once when rows are first populated.
  // Searches TMDB sequentially for each file with a parsed year, and
  // pre-selects the match only when title + year both agree.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (rows.size === 0 || autoMatchRef.current) return;
    autoMatchRef.current = true;

    let cancelled = false;

    async function runAutoMatch() {
      for (const [path, row] of rows.entries()) {
        if (cancelled) break;
        if (row.match) continue; // Already pre-matched from DB — skip live lookup
        if (!row.file.parsed_title.trim() || row.file.parsed_year === 0) continue;

        updateRow(path, { autoMatchLoading: true });
        try {
          const results = await lookupMovies.mutateAsync({
            query: row.file.parsed_title,
            year: row.file.parsed_year,
          });
          if (cancelled) break;
          const best = pickBestMatch(results, row.file.parsed_title, row.file.parsed_year);
          if (best) {
            updateRow(path, { match: best, autoMatched: true, autoMatchLoading: false });
          } else {
            updateRow(path, { autoMatchLoading: false });
          }
        } catch {
          if (!cancelled) updateRow(path, { autoMatchLoading: false });
        }
      }
    }

    void runAutoMatch();
    return () => { cancelled = true; };
  }, [rows.size]);

  function updateRow(path: string, patch: Partial<FileRowState>) {
    setRows((prev) => {
      const next = new Map(prev);
      const r = next.get(path);
      if (r) next.set(path, { ...r, ...patch });
      return next;
    });
  }

  function toggleSelect(path: string) {
    updateRow(path, { selected: !rows.get(path)?.selected });
  }

  function toggleSelectAll() {
    const allSelectable = [...rows.values()].filter((r) => !r.imported);
    const allSelected = allSelectable.every((r) => r.selected);
    setRows((prev) => {
      const next = new Map(prev);
      for (const [k, r] of next) {
        if (!r.imported) {
          next.set(k, { ...r, selected: !allSelected });
        }
      }
      return next;
    });
  }

  function openSearch(path: string) {
    // Close all others, open this one.
    setRows((prev) => {
      const next = new Map(prev);
      for (const [k, r] of next) {
        next.set(k, { ...r, searchOpen: k === path });
      }
      return next;
    });
  }

  function closeSearch(path: string) {
    updateRow(path, { searchOpen: false });
  }

  function setSearchQuery(path: string, q: string) {
    updateRow(path, { searchQuery: q });
  }

  async function runSearch(path: string) {
    const row = rows.get(path);
    if (!row || !row.searchQuery.trim()) return;
    const results = await lookupMovies.mutateAsync({ query: row.searchQuery.trim() });
    updateRow(path, { searchResults: results.slice(0, 6) });
  }

  function selectMatch(path: string, result: TMDBResult) {
    updateRow(path, {
      match: result,
      searchOpen: false,
      selected: true,
    });
  }

  function clearMatch(path: string) {
    updateRow(path, { match: null, selected: false });
  }

  const allRows = [...rows.values()];
  const displayRows = showUnmatched ? allRows : allRows.filter((r) => r.match || r.imported);
  const matched = allRows.filter((r) => r.match && !r.imported);
  const selected = allRows.filter((r) => r.selected && !r.imported);
  const unmatchedCount = allRows.filter((r) => !r.match && !r.imported).length;

  async function handleImport() {
    if (selected.length === 0) return;
    setIsImporting(true);
    setImportDone(0);
    setImportTotal(selected.length);

    for (const row of selected) {
      updateRow(row.file.path, { importing: true });
      try {
        await importFile.mutateAsync({
          libraryId: library.id,
          file_path: row.file.path,
          tmdb_id: row.match?.tmdb_id ?? 0,
        });
        updateRow(row.file.path, { importing: false, imported: true, selected: false });
      } catch {
        updateRow(row.file.path, { importing: false });
      }
      setImportDone((n) => n + 1);
    }

    setIsImporting(false);
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.7)",
        backdropFilter: "blur(3px)",
        display: "flex",
        alignItems: "stretch",
        justifyContent: "center",
        zIndex: 200,
        padding: "32px 24px",
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: "var(--color-bg-surface)",
          border: "1px solid var(--color-border-subtle)",
          borderRadius: 12,
          boxShadow: "var(--shadow-modal)",
          display: "flex",
          flexDirection: "column",
          width: "100%",
          maxWidth: 980,
          overflow: "hidden",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* ── Header ── */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "16px 24px",
            borderBottom: "1px solid var(--color-border-subtle)",
            flexShrink: 0,
          }}
        >
          <div>
            <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
              Import files — {library.name}
            </h2>
            <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-muted)" }}>
              {library.root_path}
            </p>
          </div>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              fontSize: 20,
              lineHeight: 1,
              padding: "4px 8px",
              borderRadius: 4,
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; }}
          >
            ✕
          </button>
        </div>

        {/* ── Controls bar ── */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            padding: "12px 24px",
            borderBottom: "1px solid var(--color-border-subtle)",
            flexShrink: 0,
            flexWrap: "wrap",
          }}
        >
          {/* Stats */}
          <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>
            {isLoading ? "Loading…" : rescan.isPending ? "Scanning disk…" : `${allRows.length} file${allRows.length !== 1 ? "s" : ""} found`}
            {matched.length > 0 && ` · ${matched.length} matched`}
            {selected.length > 0 && ` · ${selected.length} selected`}
          </span>

          <div style={{ flex: 1 }} />

          {/* Rescan button */}
          <button
            onClick={() => rescan.mutate(library.id)}
            disabled={rescan.isPending || isImporting}
            style={smallBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
          >
            {rescan.isPending ? "Scanning…" : "Rescan disk"}
          </button>

          {/* Show unmatched toggle */}
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: 6,
              fontSize: 12,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
              userSelect: "none",
            }}
          >
            <input
              type="checkbox"
              checked={showUnmatched}
              onChange={(e) => setShowUnmatched(e.currentTarget.checked)}
              style={{ cursor: "pointer" }}
            />
            Show unmatched ({unmatchedCount})
          </label>

          {/* Select all / deselect all */}
          {allRows.filter((r) => !r.imported).length > 0 && (
            <button
              onClick={toggleSelectAll}
              style={smallBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
            >
              {allRows.filter((r) => !r.imported).every((r) => r.selected) ? "Deselect all" : "Select all"}
            </button>
          )}
        </div>

        {/* ── File table ── */}
        <div style={{ flex: 1, overflow: "auto" }}>
          {isLoading ? (
            <div style={{ padding: 32, display: "flex", flexDirection: "column", gap: 10 }}>
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="skeleton" style={{ height: 40, borderRadius: 4 }} />
              ))}
            </div>
          ) : scanError ? (
            <div style={{ padding: 32, fontSize: 13, color: "var(--color-danger)" }}>
              Failed to scan library. Make sure the root path is accessible.
            </div>
          ) : displayRows.length === 0 ? (
            <div style={{ padding: 48, textAlign: "center" }}>
              <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
                {allRows.length === 0 ? "No candidates found" : "No matched files to show"}
              </p>
              <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
                {allRows.length === 0
                  ? "Click \"Rescan disk\" to discover untracked video files."
                  : "Use the toggle above to show unmatched files."}
              </p>
            </div>
          ) : (
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                  {["", "Filename", "Size", "Guessed Title", "TMDB Match"].map((h, i) => (
                    <th
                      key={i}
                      style={{
                        textAlign: "left",
                        padding: "10px 16px",
                        fontSize: 11,
                        fontWeight: 600,
                        letterSpacing: "0.08em",
                        textTransform: "uppercase",
                        color: "var(--color-text-muted)",
                        whiteSpace: "nowrap",
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
                {displayRows.map((row) => (
                  <FileTableRow
                    key={row.file.path}
                    row={row}
                    onToggleSelect={() => toggleSelect(row.file.path)}
                    onOpenSearch={() => openSearch(row.file.path)}
                    onCloseSearch={() => closeSearch(row.file.path)}
                    onSearchQueryChange={(q) => setSearchQuery(row.file.path, q)}
                    onRunSearch={() => runSearch(row.file.path)}
                    onSelectMatch={(r) => selectMatch(row.file.path, r)}
                    onClearMatch={() => clearMatch(row.file.path)}
                  />
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* ── Legend + footer ── */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "12px 24px",
            borderTop: "1px solid var(--color-border-subtle)",
            flexShrink: 0,
            gap: 16,
            flexWrap: "wrap",
          }}
        >
          {/* Legend */}
          <div style={{ display: "flex", gap: 16, fontSize: 12, color: "var(--color-text-muted)" }}>
            <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <div style={{ width: 10, height: 10, borderRadius: 2, background: "var(--color-success)" }} />
              <span>Matched to TMDB</span>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <div style={{ width: 10, height: 10, borderRadius: 2, background: "var(--color-warning)" }} />
              <span>Unmatched — click "Match" to search</span>
            </div>
          </div>

          {/* Import progress + button */}
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            {isImporting && (
              <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>
                Importing {importDone}/{importTotal}…
              </span>
            )}
            <button
              onClick={handleImport}
              disabled={selected.length === 0 || isImporting}
              style={{
                background: selected.length > 0 && !isImporting ? "var(--color-accent)" : "var(--color-bg-subtle)",
                color: selected.length > 0 && !isImporting ? "var(--color-accent-fg)" : "var(--color-text-muted)",
                border: "none",
                borderRadius: 6,
                padding: "8px 20px",
                fontSize: 13,
                fontWeight: 500,
                cursor: selected.length > 0 && !isImporting ? "pointer" : "not-allowed",
              }}
              onMouseEnter={(e) => {
                if (selected.length > 0 && !isImporting)
                  (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)";
              }}
              onMouseLeave={(e) => {
                if (selected.length > 0 && !isImporting)
                  (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
              }}
            >
              {isImporting
                ? "Importing…"
                : selected.length === 0
                ? "No files selected"
                : `Import ${selected.length} file${selected.length !== 1 ? "s" : ""}`}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── File table row ────────────────────────────────────────────────────────────

interface FileTableRowProps {
  row: FileRowState;
  onToggleSelect: () => void;
  onOpenSearch: () => void;
  onCloseSearch: () => void;
  onSearchQueryChange: (q: string) => void;
  onRunSearch: () => void;
  onSelectMatch: (r: TMDBResult) => void;
  onClearMatch: () => void;
}

function FileTableRow({
  row,
  onToggleSelect,
  onOpenSearch,
  onCloseSearch,
  onSearchQueryChange,
  onRunSearch,
  onSelectMatch,
  onClearMatch,
}: FileTableRowProps) {
  const { file, selected, match, searchOpen, searchQuery, searchResults, importing, imported, autoMatchLoading, autoMatched } = row;

  // Row highlight colour.
  let rowBg = "transparent";
  if (imported) rowBg = "color-mix(in srgb, var(--color-success) 6%, transparent)";
  else if (match) rowBg = "color-mix(in srgb, var(--color-success) 8%, transparent)";
  else rowBg = "color-mix(in srgb, var(--color-warning) 5%, transparent)";

  return (
    <tr
      style={{
        borderBottom: "1px solid var(--color-border-subtle)",
        background: rowBg,
        opacity: imported ? 0.6 : 1,
      }}
    >
      {/* Checkbox */}
      <td style={{ padding: "0 8px 0 16px", width: 32 }}>
        <input
          type="checkbox"
          checked={selected}
          disabled={imported || importing}
          onChange={onToggleSelect}
          style={{ cursor: !imported && !importing ? "pointer" : "default" }}
        />
      </td>

      {/* Filename */}
      <td style={{ padding: "10px 16px", maxWidth: 280 }}>
        <span
          title={file.path}
          style={{
            display: "block",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            fontFamily: "var(--font-family-mono)",
            fontSize: 11,
            color: imported ? "var(--color-text-muted)" : "var(--color-text-primary)",
          }}
        >
          {basename(file.path)}
        </span>
      </td>

      {/* Size */}
      <td style={{ padding: "10px 16px", whiteSpace: "nowrap", color: "var(--color-text-muted)", fontSize: 12 }}>
        {formatBytes(file.size_bytes)}
      </td>

      {/* Guessed title */}
      <td style={{ padding: "10px 16px", color: "var(--color-text-secondary)", fontSize: 12 }}>
        {file.parsed_title || <span style={{ color: "var(--color-text-muted)" }}>—</span>}
        {file.parsed_year > 0 && (
          <span style={{ color: "var(--color-text-muted)", marginLeft: 6 }}>({file.parsed_year})</span>
        )}
      </td>

      {/* TMDB match column */}
      <td style={{ padding: "10px 16px", minWidth: 220 }}>
        {imported ? (
          <span style={{ fontSize: 12, color: "var(--color-success)" }}>✓ Imported</span>
        ) : importing ? (
          <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>Importing…</span>
        ) : match ? (
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span
              style={{
                fontSize: 12,
                color: "var(--color-success)",
                fontWeight: 500,
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
                maxWidth: 160,
                cursor: autoMatched ? "pointer" : "default",
              }}
              title={`${match.title} (${match.year})${autoMatched ? " — click to change" : ""}`}
              onClick={autoMatched ? onOpenSearch : undefined}
            >
              {match.title} {match.year > 0 && `(${match.year})`}
            </span>
            {autoMatched && (
              <span style={{ fontSize: 10, color: "var(--color-text-muted)", fontWeight: 400, flexShrink: 0 }}>
                auto
              </span>
            )}
            <button
              onClick={onClearMatch}
              title="Clear match"
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                color: "var(--color-text-muted)",
                fontSize: 11,
                padding: "1px 4px",
                borderRadius: 3,
                flexShrink: 0,
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-danger)"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; }}
            >
              ✕
            </button>
          </div>
        ) : searchOpen ? (
          <InlineSearch
            query={searchQuery}
            results={searchResults}
            onQueryChange={onSearchQueryChange}
            onSearch={onRunSearch}
            onSelect={onSelectMatch}
            onClose={onCloseSearch}
          />
        ) : autoMatchLoading ? (
          <span style={{ fontSize: 12, color: "var(--color-text-muted)", fontStyle: "italic" }}>
            Searching…
          </span>
        ) : (
          <button
            onClick={onOpenSearch}
            style={smallBtn("var(--color-accent)", "color-mix(in srgb, var(--color-accent) 12%, transparent)")}
          >
            Match
          </button>
        )}
      </td>
    </tr>
  );
}

// ── Inline TMDB search ────────────────────────────────────────────────────────

interface InlineSearchProps {
  query: string;
  results: TMDBResult[];
  onQueryChange: (q: string) => void;
  onSearch: () => void;
  onSelect: (r: TMDBResult) => void;
  onClose: () => void;
}

function InlineSearch({ query, results, onQueryChange, onSearch, onSelect, onClose }: InlineSearchProps) {
  function handleKey(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") { e.preventDefault(); onSearch(); }
    if (e.key === "Escape") { e.preventDefault(); onClose(); }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      <div style={{ display: "flex", gap: 4 }}>
        <input
          autoFocus
          value={query}
          onChange={(e) => onQueryChange(e.currentTarget.value)}
          onKeyDown={handleKey}
          placeholder="Search TMDB…"
          style={{
            flex: 1,
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-accent)",
            borderRadius: 4,
            padding: "4px 8px",
            fontSize: 12,
            color: "var(--color-text-primary)",
            outline: "none",
            minWidth: 0,
          }}
        />
        <button
          onClick={onSearch}
          style={smallBtn("var(--color-accent-fg)", "var(--color-accent)")}
        >
          Go
        </button>
        <button
          onClick={onClose}
          style={smallBtn("var(--color-text-muted)", "var(--color-bg-elevated)")}
        >
          ✕
        </button>
      </div>
      {results.length > 0 && (
        <div
          style={{
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 4,
            overflow: "hidden",
          }}
        >
          {results.map((r) => (
            <button
              key={r.tmdb_id}
              onClick={() => onSelect(r)}
              style={{
                display: "block",
                width: "100%",
                textAlign: "left",
                padding: "6px 10px",
                background: "none",
                border: "none",
                borderBottom: "1px solid var(--color-border-subtle)",
                cursor: "pointer",
                fontSize: 12,
                color: "var(--color-text-primary)",
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-subtle)";
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLButtonElement).style.background = "none";
              }}
            >
              <span style={{ fontWeight: 500 }}>{r.title}</span>
              {r.year > 0 && (
                <span style={{ color: "var(--color-text-muted)", marginLeft: 6 }}>({r.year})</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ── Row actions ───────────────────────────────────────────────────────────────

interface RowActionsProps {
  library: Library;
  onEdit: () => void;
  onImport: () => void;
}

function RowActions({ library, onEdit, onImport }: RowActionsProps) {
  const [confirming, setConfirming] = useState(false);
  const [scanned, setScanned] = useState(false);
  const deleteLib = useDeleteLibrary();
  const scanLib = useScanLibrary();

  function handleScan() {
    scanLib.mutate(library.id, {
      onSuccess: () => {
        setScanned(true);
        setTimeout(() => setScanned(false), 2000);
      },
    });
  }

  if (confirming) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
        <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>Delete?</span>
        <button
          onClick={() => deleteLib.mutate(library.id, { onSuccess: () => setConfirming(false) })}
          disabled={deleteLib.isPending}
          style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 15%, transparent)")}
        >
          {deleteLib.isPending ? "…" : "Yes"}
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

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
      {scanned ? (
        <span style={{ fontSize: 12, color: "var(--color-success)" }}>Scanning ✓</span>
      ) : (
        <button
          onClick={handleScan}
          disabled={scanLib.isPending}
          style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
        >
          {scanLib.isPending ? "…" : "Scan"}
        </button>
      )}
      <button
        onClick={onImport}
        style={actionBtn("var(--color-accent)", "color-mix(in srgb, var(--color-accent) 12%, transparent)")}
      >
        Import
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

function actionBtn(color: string, bg: string): React.CSSProperties {
  return {
    background: bg,
    border: "1px solid var(--color-border-default)",
    borderRadius: 5,
    padding: "3px 10px",
    fontSize: 12,
    color,
    cursor: "pointer",
  };
}

function smallBtn(color: string, bg: string): React.CSSProperties {
  return {
    background: bg,
    border: "none",
    borderRadius: 4,
    padding: "3px 8px",
    fontSize: 12,
    color,
    cursor: "pointer",
    whiteSpace: "nowrap" as const,
  };
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function LibraryList() {
  const { data, isLoading, error } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const [modal, setModal] = useState<{ open: boolean; editing: Library | null }>({
    open: false,
    editing: null,
  });
  const [importLibrary, setImportLibrary] = useState<Library | null>(null);

  const profileMap = Object.fromEntries((profiles ?? []).map((p) => [p.id, p.name]));

  function openCreate() { setModal({ open: true, editing: null }); }
  function openEdit(lib: Library) { setModal({ open: true, editing: lib }); }
  function closeModal() { setModal({ open: false, editing: null }); }

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 24 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", letterSpacing: "-0.01em" }}>
            Libraries
          </h1>
          <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
            Media root folders scanned for movie files.
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
          + Add Library
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
            Failed to load libraries.
          </div>
        ) : !data?.length ? (
          <div style={{ padding: 48, textAlign: "center" }}>
            <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
              No libraries configured
            </p>
            <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
              Add a library to start tracking movies.
            </p>
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {["Name", "Root Path", "Quality Profile", "Min Free Space", ""].map((h) => (
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
              {data.map((lib, i) => (
                <tr
                  key={lib.id}
                  style={{
                    borderBottom: i < data.length - 1 ? "1px solid var(--color-border-subtle)" : "none",
                  }}
                >
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-primary)", fontWeight: 500 }}>
                    {lib.name}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, maxWidth: 260 }}>
                    <span
                      style={{
                        display: "block",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                        fontFamily: "var(--font-family-mono)",
                        fontSize: 12,
                        color: "var(--color-text-secondary)",
                      }}
                      title={lib.root_path}
                    >
                      {lib.root_path}
                    </span>
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)" }}>
                    {lib.default_quality_profile_id
                      ? (profileMap[lib.default_quality_profile_id] ?? "—")
                      : <span style={{ color: "var(--color-text-muted)" }}>None</span>}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)", whiteSpace: "nowrap" }}>
                    {lib.min_free_space_gb > 0 ? `${lib.min_free_space_gb} GB` : <span style={{ color: "var(--color-text-muted)" }}>None</span>}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, width: 1 }}>
                    <RowActions
                      library={lib}
                      onEdit={() => openEdit(lib)}
                      onImport={() => setImportLibrary(lib)}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Edit/create modal */}
      {modal.open && (
        <LibraryModal editing={modal.editing} onClose={closeModal} />
      )}

      {/* Disk import modal */}
      {importLibrary && (
        <DiskScanModal library={importLibrary} onClose={() => setImportLibrary(null)} />
      )}
    </div>
  );
}
