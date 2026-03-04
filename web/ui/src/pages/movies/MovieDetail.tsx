import { useState, useCallback, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  useMovie,
  useMovieReleases,
  useGrabRelease,
  useDeleteMovie,
  useUpdateMovie,
  useRefreshMovie,
  useMovieFiles,
  useDeleteMovieFile,
  useMovieHistory,
  useMatchMovie,
  useLookupMovies,
  useMovieSuggestions,
  type GrabReleaseRequest,
} from "@/api/movies";
import { ManualSearchModal } from "@/components/ManualSearchModal";
import type { Release, TMDBResult } from "@/types";
import { formatBytes } from "@/lib/utils";

// ── Helpers ────────────────────────────────────────────────────────────────────

function formatRuntime(minutes: number): string {
  if (!minutes) return "—";
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

function actionBtn(color: string, bg: string): React.CSSProperties {
  return {
    background: bg,
    border: "1px solid var(--color-border-default)",
    borderRadius: 5,
    padding: "5px 12px",
    fontSize: 12,
    color,
    cursor: "pointer",
    whiteSpace: "nowrap",
  };
}

// ── Release quality badge ──────────────────────────────────────────────────────

function QualityBadge({ quality }: { quality: Release["quality"] }) {
  const label = [quality.resolution, quality.source].filter(Boolean).join(" ");
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 6px",
        borderRadius: 4,
        fontSize: 10,
        fontWeight: 600,
        textTransform: "uppercase",
        letterSpacing: "0.05em",
        background: "color-mix(in srgb, var(--color-accent) 12%, transparent)",
        color: "var(--color-accent)",
      }}
    >
      {label || "Unknown"}
    </span>
  );
}

// ── Releases tab ───────────────────────────────────────────────────────────────

interface ReleasesTabProps {
  movieId: string;
}

function ReleasesTab({ movieId }: ReleasesTabProps) {
  const { data, isLoading, error, refetch } = useMovieReleases(movieId);
  const grab = useGrabRelease();
  const [grabbedGuids, setGrabbedGuids] = useState<Set<string>>(new Set());
  const [pendingGuids, setPendingGuids] = useState<Set<string>>(new Set());
  const [grabErrors, setGrabErrors] = useState<Record<string, string>>({});

  const handleGrab = useCallback((release: Release) => {
    const body: GrabReleaseRequest & { movieId: string } = {
      movieId,
      guid: release.guid,
      title: release.title,
      protocol: release.protocol,
      download_url: release.download_url,
      size: release.size,
    };
    setPendingGuids((prev) => new Set([...prev, release.guid]));
    grab.mutate(body, {
      onSuccess: () => {
        setPendingGuids((prev) => { const n = new Set(prev); n.delete(release.guid); return n; });
        setGrabbedGuids((prev) => new Set([...prev, release.guid]));
      },
      onError: (e) => {
        setPendingGuids((prev) => { const n = new Set(prev); n.delete(release.guid); return n; });
        setGrabErrors((prev) => ({ ...prev, [release.guid]: e.message }));
        setTimeout(() => setGrabErrors((prev) => { const n = { ...prev }; delete n[release.guid]; return n; }), 5000);
      },
    });
  }, [movieId, grab]);

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 12, padding: "8px 0" }}>
        {[1, 2, 3].map((i) => (
          <div key={i} className="skeleton" style={{ height: 56, borderRadius: 6 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: "24px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 13, color: "var(--color-text-muted)" }}>
          Failed to search indexers: {error.message}
        </p>
        <button
          onClick={() => refetch()}
          style={{
            marginTop: 12,
            ...actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)"),
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  if (!data?.length) {
    return (
      <div style={{ padding: "48px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
          No releases found
        </p>
        <p style={{ margin: "6px 0 16px", fontSize: 13, color: "var(--color-text-muted)" }}>
          No results from any configured indexer.
        </p>
        <button onClick={() => refetch()} style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}>
          Search Again
        </button>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 1 }}>
      <p style={{ margin: "0 0 12px", fontSize: 12, color: "var(--color-text-muted)" }}>
        {data.length} release{data.length !== 1 ? "s" : ""} found across all indexers.
      </p>
      {data.map((release) => (
        <ReleaseRow
          key={release.guid}
          release={release}
          grabbed={grabbedGuids.has(release.guid)}
          grabError={grabErrors[release.guid]}
          onGrab={() => handleGrab(release)}
          isPending={pendingGuids.has(release.guid)}
        />
      ))}
    </div>
  );
}

interface ReleaseRowProps {
  release: Release;
  grabbed: boolean;
  grabError?: string;
  onGrab: () => void;
  isPending: boolean;
}

function ReleaseRow({ release, grabbed, grabError, onGrab, isPending }: ReleaseRowProps) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 14px",
        background: "var(--color-bg-elevated)",
        borderRadius: 6,
        border: "1px solid var(--color-border-subtle)",
      }}
    >
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 13,
            color: "var(--color-text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            fontFamily: "var(--font-family-mono)",
          }}
          title={release.title}
        >
          {release.title}
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 10, marginTop: 4, flexWrap: "wrap" }}>
          <QualityBadge quality={release.quality} />
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
            {formatBytes(release.size)}
          </span>
          {release.seeds !== undefined && (
            <span style={{ fontSize: 11, color: "var(--color-success)" }}>
              ↑{release.seeds}
            </span>
          )}
          {release.peers !== undefined && (
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
              ↓{release.peers}
            </span>
          )}
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
            {release.indexer}
          </span>
          {release.age_days !== undefined && release.age_days > 0 && (
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
              {Math.round(release.age_days)}d old
            </span>
          )}
        </div>
        {grabError && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-danger)" }}>{grabError}</p>
        )}
      </div>

      {grabbed ? (
        <span style={{ fontSize: 12, color: "var(--color-success)", flexShrink: 0 }}>Grabbed ✓</span>
      ) : (
        <button
          onClick={onGrab}
          disabled={isPending}
          style={{
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "5px 14px",
            fontSize: 12,
            fontWeight: 500,
            cursor: isPending ? "not-allowed" : "pointer",
            flexShrink: 0,
          }}
          onMouseEnter={(e) => {
            if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)";
          }}
          onMouseLeave={(e) => {
            if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
          }}
        >
          Grab
        </button>
      )}
    </div>
  );
}

// ── Files tab ──────────────────────────────────────────────────────────────────

function FilesTab({ movieId }: { movieId: string }) {
  const { data: files, isLoading, error } = useMovieFiles(movieId);
  const deleteFile = useDeleteMovieFile(movieId);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  async function handleDelete(fileId: string, deleteFromDisk: boolean) {
    setDeletingId(fileId);
    try {
      await deleteFile.mutateAsync({ fileId, deleteFromDisk });
      toast.success(deleteFromDisk ? "File deleted from disk" : "File record removed");
    } catch {
      toast.error("Failed to delete file");
    } finally {
      setDeletingId(null);
    }
  }

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: "8px 0" }}>
        {[1, 2].map((i) => (
          <div key={i} className="skeleton" style={{ height: 56, borderRadius: 6 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p style={{ fontSize: 13, color: "var(--color-danger)", margin: 0 }}>
        Failed to load files: {(error as Error).message}
      </p>
    );
  }

  if (!files?.length) {
    return (
      <div style={{ padding: "32px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
          No files
        </p>
        <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
          No files are associated with this movie yet.
        </p>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      {files.map((file) => (
        <div
          key={file.id}
          style={{
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-subtle)",
            borderRadius: 6,
            padding: "12px 14px",
          }}
        >
          {/* Path */}
          <div
            style={{
              fontSize: 12,
              fontFamily: "var(--font-family-mono)",
              color: "var(--color-text-secondary)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
              marginBottom: 6,
            }}
            title={file.path}
          >
            {file.path}
          </div>

          {/* Meta row */}
          <div style={{ display: "flex", alignItems: "center", gap: 16, flexWrap: "wrap" }}>
            <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
              {formatBytes(file.size_bytes)}
            </span>
            {file.quality?.resolution && (
              <span
                style={{
                  display: "inline-block",
                  padding: "1px 6px",
                  borderRadius: 4,
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: "uppercase",
                  letterSpacing: "0.05em",
                  background: "color-mix(in srgb, var(--color-accent) 12%, transparent)",
                  color: "var(--color-accent)",
                }}
              >
                {[file.quality.resolution, file.quality.source].filter(Boolean).join(" ")}
              </span>
            )}
            {file.edition && (
              <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>{file.edition}</span>
            )}
            <span style={{ fontSize: 12, color: "var(--color-text-muted)", marginLeft: "auto" }}>
              Imported {new Date(file.imported_at).toLocaleDateString()}
            </span>

            {/* Delete */}
            <button
              onClick={() => {
                const fromDisk = confirm(
                  "Also delete the file from disk?\n\nOK = delete from disk\nCancel = remove record only"
                );
                handleDelete(file.id, fromDisk);
              }}
              disabled={deletingId === file.id}
              title="Delete file"
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                color: "var(--color-text-muted)",
                display: "flex",
                alignItems: "center",
                padding: 4,
                borderRadius: 4,
                transition: "color 150ms ease",
                flexShrink: 0,
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLButtonElement).style.color = "var(--color-danger, #ef4444)";
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)";
              }}
            >
              <Trash2 size={14} />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── History tab ────────────────────────────────────────────────────────────────

function statusColor(status: string): string {
  if (status === "completed") return "var(--color-success)";
  if (status === "failed" || status === "removed") return "var(--color-danger)";
  return "var(--color-text-muted)";
}

function HistoryTab({ movieId }: { movieId: string }) {
  const { data: items, isLoading, error } = useMovieHistory(movieId);

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: "8px 0" }}>
        {[1, 2, 3].map((i) => (
          <div key={i} className="skeleton" style={{ height: 48, borderRadius: 6 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p style={{ fontSize: 13, color: "var(--color-danger)", margin: 0 }}>
        Failed to load history: {(error as Error).message}
      </p>
    );
  }

  if (!items?.length) {
    return (
      <div style={{ padding: "32px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
          No history
        </p>
        <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
          No grabs have been recorded for this movie yet.
        </p>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 1 }}>
      {items.map((item) => (
        <div
          key={item.id}
          style={{
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-subtle)",
            borderRadius: 6,
            padding: "10px 14px",
            display: "flex",
            alignItems: "center",
            gap: 12,
          }}
        >
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 12,
                fontFamily: "var(--font-family-mono)",
                color: "var(--color-text-primary)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
              title={item.release_title}
            >
              {item.release_title}
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 12, marginTop: 4, flexWrap: "wrap" }}>
              {item.release_resolution && (
                <span
                  style={{
                    display: "inline-block",
                    padding: "1px 6px",
                    borderRadius: 4,
                    fontSize: 10,
                    fontWeight: 600,
                    textTransform: "uppercase",
                    letterSpacing: "0.05em",
                    background: "color-mix(in srgb, var(--color-accent) 12%, transparent)",
                    color: "var(--color-accent)",
                  }}
                >
                  {item.release_resolution}
                </span>
              )}
              <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
                {item.protocol}
              </span>
              {item.size > 0 && (
                <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
                  {formatBytes(item.size)}
                </span>
              )}
            </div>
          </div>
          <div style={{ flexShrink: 0, textAlign: "right" }}>
            <div style={{ fontSize: 11, fontWeight: 600, color: statusColor(item.download_status), textTransform: "capitalize" }}>
              {item.download_status}
            </div>
            <div style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 2 }}>
              {new Date(item.grabbed_at).toLocaleDateString()}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Match to TMDB banner ────────────────────────────────────────────────────────

function MatchTMDBBanner({ movieId }: { movieId: string }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [manualResults, setManualResults] = useState<TMDBResult[] | null>(null);

  const suggestions = useMovieSuggestions(movieId, open);
  const lookup = useLookupMovies();
  const match = useMatchMovie();

  // When the panel opens, pre-fill the query with the parsed title.
  useEffect(() => {
    if (open && suggestions.data) {
      setQuery(suggestions.data.parsed_title);
    }
  }, [open, suggestions.data]);

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    const r = await lookup.mutateAsync({ query: query.trim() });
    setManualResults(r.slice(0, 8));
  }

  function handleClose() {
    setOpen(false);
    setManualResults(null);
    setQuery("");
  }

  // Show manual results if the user has searched; otherwise show auto-suggestions.
  const displayResults: TMDBResult[] = manualResults ?? suggestions.data?.results ?? [];

  function handleSelect(r: TMDBResult) {
    match.mutate({ id: movieId, tmdb_id: r.tmdb_id });
  }

  return (
    <div
      style={{
        background: "color-mix(in srgb, var(--color-warning) 8%, var(--color-bg-surface))",
        border: "1px solid color-mix(in srgb, var(--color-warning) 35%, var(--color-border-subtle))",
        borderRadius: 8,
        padding: "12px 16px",
        marginBottom: 20,
        display: "flex",
        flexDirection: "column",
        gap: 10,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
        <span style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500 }}>
          This movie has no TMDB match. Artwork and metadata won't be available until it's matched.
        </span>
        {!open && (
          <button
            onClick={() => setOpen(true)}
            style={{
              flexShrink: 0,
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 5,
              padding: "5px 14px",
              fontSize: 12,
              fontWeight: 500,
              cursor: "pointer",
              whiteSpace: "nowrap",
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
          >
            Match to TMDB
          </button>
        )}
      </div>

      {open && (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {suggestions.data && (
            <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>
              Parsed from filename: <strong>{suggestions.data.parsed_title}</strong>
              {suggestions.data.parsed_year > 0 && <> ({suggestions.data.parsed_year})</>}
            </p>
          )}

          <form onSubmit={handleSearch} style={{ display: "flex", gap: 6 }}>
            <input
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.currentTarget.value)}
              placeholder="Search TMDB by title…"
              style={{
                flex: 1,
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 5,
                padding: "6px 10px",
                fontSize: 13,
                color: "var(--color-text-primary)",
                outline: "none",
              }}
              onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = "var(--color-accent)"; }}
              onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = "var(--color-border-default)"; }}
            />
            <button
              type="submit"
              disabled={lookup.isPending}
              style={actionBtn("var(--color-accent-fg)", "var(--color-accent)")}
            >
              {lookup.isPending ? "…" : "Search"}
            </button>
            <button
              type="button"
              onClick={handleClose}
              style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
            >
              Cancel
            </button>
          </form>

          {suggestions.isLoading && (
            <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>Looking up suggestions…</p>
          )}

          {displayResults.length > 0 && (
            <div
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                overflow: "hidden",
              }}
            >
              {displayResults.map((r) => (
                <button
                  key={r.tmdb_id}
                  onClick={() => handleSelect(r)}
                  disabled={match.isPending}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    width: "100%",
                    textAlign: "left",
                    padding: "8px 12px",
                    background: "none",
                    border: "none",
                    borderBottom: "1px solid var(--color-border-subtle)",
                    cursor: match.isPending ? "not-allowed" : "pointer",
                  }}
                  onMouseEnter={(e) => {
                    if (!match.isPending)
                      (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-subtle)";
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.background = "none";
                  }}
                >
                  {r.poster_path && (
                    <img
                      src={`https://image.tmdb.org/t/p/w92${r.poster_path}`}
                      alt=""
                      style={{ width: 32, height: 48, objectFit: "cover", borderRadius: 3, flexShrink: 0 }}
                    />
                  )}
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
                      {r.title}
                    </div>
                    {r.year > 0 && (
                      <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 1 }}>
                        {r.year}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}

          {match.isPending && (
            <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>Matching…</p>
          )}
        </div>
      )}
    </div>
  );
}

// ── Delete confirm ─────────────────────────────────────────────────────────────

function DeleteConfirmBar({ movieId, onCancel }: { movieId: string; onCancel: () => void }) {
  const del = useDeleteMovie();
  const navigate = useNavigate();

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 16px",
        background: "color-mix(in srgb, var(--color-danger) 8%, var(--color-bg-surface))",
        border: "1px solid color-mix(in srgb, var(--color-danger) 25%, var(--color-border-subtle))",
        borderRadius: 8,
        marginBottom: 20,
      }}
    >
      <span style={{ fontSize: 13, color: "var(--color-text-primary)", flex: 1 }}>
        Remove this movie from Luminarr? (Files on disk are not deleted.)
      </span>
      <button
        onClick={() => del.mutate(movieId, { onSuccess: () => navigate("/") })}
        disabled={del.isPending}
        style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 15%, transparent)")}
      >
        {del.isPending ? "Deleting…" : "Yes, Remove"}
      </button>
      <button onClick={onCancel} style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}>
        Cancel
      </button>
    </div>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

type Tab = "overview" | "releases" | "files" | "history";

export default function MovieDetail() {
  const { id } = useParams<{ id: string }>();
  const { data: movie, isLoading, error } = useMovie(id ?? "");
  const updateMovie = useUpdateMovie();
  const refreshMovie = useRefreshMovie();

  const [tab, setTab] = useState<Tab>("overview");
  const [confirming, setConfirming] = useState(false);
  const [refreshed, setRefreshed] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);

  function handleMonitoredToggle() {
    if (!movie) return;
    updateMovie.mutate({
      id: movie.id,
      title: movie.title,
      monitored: !movie.monitored,
      library_id: movie.library_id,
      quality_profile_id: movie.quality_profile_id,
      minimum_availability: movie.minimum_availability,
    });
  }

  function handleMinimumAvailabilityChange(value: string) {
    if (!movie) return;
    updateMovie.mutate({
      id: movie.id,
      title: movie.title,
      monitored: movie.monitored,
      library_id: movie.library_id,
      quality_profile_id: movie.quality_profile_id,
      minimum_availability: value,
    });
  }

  function handleRefresh() {
    if (!movie) return;
    refreshMovie.mutate(movie.id, {
      onSuccess: () => {
        setRefreshed(true);
        setTimeout(() => setRefreshed(false), 2000);
      },
    });
  }

  if (isLoading) {
    return (
      <div style={{ padding: 24, display: "flex", flexDirection: "column", gap: 20 }}>
        <div className="skeleton" style={{ height: 24, width: 200, borderRadius: 4 }} />
        <div style={{ display: "flex", gap: 24 }}>
          <div className="skeleton" style={{ width: 200, height: 300, borderRadius: 8 }} />
          <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: 12 }}>
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="skeleton" style={{ height: 20, borderRadius: 4 }} />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error || !movie) {
    return (
      <div style={{ padding: 24 }}>
        <Link to="/" style={{ fontSize: 13, color: "var(--color-accent)", textDecoration: "none" }}>
          ← Movies
        </Link>
        <p style={{ marginTop: 24, fontSize: 13, color: "var(--color-text-muted)" }}>
          Movie not found or failed to load.
        </p>
      </div>
    );
  }

  const posterSrc = movie.poster_url || null;

  return (
    <div style={{ padding: 24, maxWidth: 1000 }}>
      {/* Back link */}
      <Link
        to="/"
        style={{ fontSize: 13, color: "var(--color-text-muted)", textDecoration: "none", display: "inline-block", marginBottom: 20 }}
        onMouseEnter={(e) => { (e.currentTarget as HTMLAnchorElement).style.color = "var(--color-text-primary)"; }}
        onMouseLeave={(e) => { (e.currentTarget as HTMLAnchorElement).style.color = "var(--color-text-muted)"; }}
      >
        ← Movies
      </Link>

      {/* Unmatched banner */}
      {movie.tmdb_id === 0 && <MatchTMDBBanner movieId={movie.id} />}

      {/* Delete confirmation bar */}
      {confirming && (
        <DeleteConfirmBar movieId={movie.id} onCancel={() => setConfirming(false)} />
      )}

      {/* Header row */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 24, gap: 16 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700, color: "var(--color-text-primary)", letterSpacing: "-0.02em" }}>
            {movie.title}
          </h1>
          {movie.year > 0 && (
            <p style={{ margin: "2px 0 0", fontSize: 14, color: "var(--color-text-muted)" }}>{movie.year}</p>
          )}
        </div>
        <div style={{ display: "flex", gap: 8, flexShrink: 0 }}>
          <button
            onClick={handleMonitoredToggle}
            disabled={updateMovie.isPending}
            style={{
              ...actionBtn(
                movie.monitored ? "var(--color-success)" : "var(--color-text-muted)",
                movie.monitored
                  ? "color-mix(in srgb, var(--color-success) 12%, transparent)"
                  : "var(--color-bg-elevated)"
              ),
              borderColor: movie.monitored ? "var(--color-success)" : "var(--color-border-default)",
            }}
          >
            {movie.monitored ? "Monitored" : "Unmonitored"}
          </button>
          {refreshed ? (
            <span style={{ fontSize: 12, color: "var(--color-success)", display: "flex", alignItems: "center", padding: "0 4px" }}>
              Queued ✓
            </span>
          ) : (
            <button
              onClick={handleRefresh}
              disabled={refreshMovie.isPending}
              style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
            >
              {refreshMovie.isPending ? "Refreshing…" : "Refresh Metadata"}
            </button>
          )}
          <button
            onClick={() => setSearchOpen(true)}
            style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
          >
            Search
          </button>
          <button
            onClick={() => setConfirming((v) => !v)}
            style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 10%, transparent)")}
          >
            Delete
          </button>
        </div>
      </div>

      {/* Main layout */}
      <div style={{ display: "flex", gap: 24, alignItems: "flex-start" }}>
        {/* Poster */}
        <div style={{ flexShrink: 0 }}>
          {posterSrc ? (
            <img
              src={posterSrc}
              alt={movie.title}
              style={{
                width: 180,
                borderRadius: 8,
                boxShadow: "var(--shadow-modal)",
                display: "block",
              }}
            />
          ) : (
            <div
              style={{
                width: 180,
                height: 270,
                borderRadius: 8,
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-subtle)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>No poster</span>
            </div>
          )}
        </div>

        {/* Content */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* Tabs */}
          <div style={{ display: "flex", gap: 0, borderBottom: "1px solid var(--color-border-subtle)", marginBottom: 20 }}>
            {(["overview", "releases", "files", "history"] as Tab[]).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                style={{
                  background: "none",
                  border: "none",
                  borderBottom: `2px solid ${tab === t ? "var(--color-accent)" : "transparent"}`,
                  padding: "10px 16px",
                  fontSize: 13,
                  fontWeight: tab === t ? 600 : 400,
                  color: tab === t ? "var(--color-accent)" : "var(--color-text-muted)",
                  cursor: "pointer",
                  textTransform: "capitalize",
                  marginBottom: -1,
                  transition: "color 0.1s, border-color 0.1s",
                }}
              >
                {t}
              </button>
            ))}
          </div>

          {/* Tab content */}
          {tab === "overview" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              {/* Quick facts */}
              <div style={{ display: "flex", gap: 16, flexWrap: "wrap", alignItems: "flex-start" }}>
                {[
                  { label: "Runtime", value: formatRuntime(movie.runtime_minutes) },
                  { label: "Status", value: movie.status },
                  { label: "TMDB", value: String(movie.tmdb_id) },
                  ...(movie.imdb_id ? [{ label: "IMDB", value: movie.imdb_id }] : []),
                ].map(({ label, value }) => (
                  <div
                    key={label}
                    style={{
                      background: "var(--color-bg-elevated)",
                      border: "1px solid var(--color-border-subtle)",
                      borderRadius: 6,
                      padding: "8px 14px",
                    }}
                  >
                    <div style={{ fontSize: 10, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 2 }}>
                      {label}
                    </div>
                    <div style={{ fontSize: 13, color: "var(--color-text-primary)" }}>{value}</div>
                  </div>
                ))}

                {/* Minimum availability selector */}
                <div
                  style={{
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-subtle)",
                    borderRadius: 6,
                    padding: "8px 14px",
                  }}
                >
                  <div style={{ fontSize: 10, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 4 }}>
                    Min. Availability
                  </div>
                  <select
                    value={movie.minimum_availability || "released"}
                    onChange={(e) => handleMinimumAvailabilityChange(e.target.value)}
                    disabled={updateMovie.isPending}
                    style={{
                      fontSize: 13,
                      color: "var(--color-text-primary)",
                      background: "transparent",
                      border: "none",
                      padding: 0,
                      cursor: "pointer",
                      outline: "none",
                    }}
                  >
                    <option value="announced">Announced</option>
                    <option value="in_cinemas">In Cinemas</option>
                    <option value="released">Released</option>
                    <option value="tba">TBA</option>
                  </select>
                </div>
              </div>

              {/* Genres */}
              {movie.genres?.length > 0 && (
                <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                  {movie.genres.map((g) => (
                    <span
                      key={g}
                      style={{
                        padding: "3px 10px",
                        borderRadius: 4,
                        fontSize: 12,
                        background: "var(--color-bg-elevated)",
                        border: "1px solid var(--color-border-subtle)",
                        color: "var(--color-text-secondary)",
                      }}
                    >
                      {g}
                    </span>
                  ))}
                </div>
              )}

              {/* Overview */}
              {movie.overview && (
                <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", lineHeight: 1.65 }}>
                  {movie.overview}
                </p>
              )}

              {/* File path */}
              {movie.path && (
                <div>
                  <div style={{ fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 4 }}>
                    File Path
                  </div>
                  <code
                    style={{
                      display: "block",
                      padding: "8px 12px",
                      borderRadius: 6,
                      background: "var(--color-bg-elevated)",
                      border: "1px solid var(--color-border-subtle)",
                      fontSize: 12,
                      fontFamily: "var(--font-family-mono)",
                      color: "var(--color-text-secondary)",
                      overflowX: "auto",
                    }}
                  >
                    {movie.path}
                  </code>
                </div>
              )}
            </div>
          )}

          {tab === "releases" && <ReleasesTab movieId={movie.id} />}
          {tab === "files" && <FilesTab movieId={movie.id} />}
          {tab === "history" && <HistoryTab movieId={movie.id} />}
        </div>
      </div>

      {searchOpen && (
        <ManualSearchModal
          movieId={movie.id}
          movieTitle={movie.title}
          onClose={() => setSearchOpen(false)}
        />
      )}
    </div>
  );
}
