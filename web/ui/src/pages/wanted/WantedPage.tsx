import { useState } from "react";
import { Link } from "react-router-dom";
import { ChevronDown, ChevronUp } from "lucide-react";
import { toast } from "sonner";
import { useWantedMissing, useWantedCutoff, useUpgradeRecommendations } from "@/api/wanted";
import { useBulkAutoSearch } from "@/api/movies";
import { ManualSearchModal } from "@/components/ManualSearchModal";
import type { Movie, UpgradeTier } from "@/types";

// ── Shared helpers ─────────────────────────────────────────────────────────────

function statusBadge(status: string, monitored: boolean): React.CSSProperties {
  const base: React.CSSProperties = {
    display: "inline-block",
    padding: "1px 6px",
    borderRadius: 4,
    fontSize: 10,
    fontWeight: 600,
    textTransform: "uppercase",
    letterSpacing: "0.05em",
  };
  if (!monitored) return { ...base, background: "var(--color-bg-elevated)", color: "var(--color-text-muted)" };
  if (status === "downloaded") return { ...base, background: "color-mix(in srgb, var(--color-success) 15%, transparent)", color: "var(--color-success)" };
  return { ...base, background: "color-mix(in srgb, var(--color-warning) 15%, transparent)", color: "var(--color-warning)" };
}

// ── Movie row ─────────────────────────────────────────────────────────────────

function MovieRow({ movie, onSearch }: { movie: Movie; onSearch: () => void }) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 8,
        background: "var(--color-bg-elevated)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 6,
        overflow: "hidden",
      }}
    >
      <Link
        to={`/movies/${movie.id}`}
        style={{ textDecoration: "none", flex: 1, minWidth: 0 }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 12,
            padding: "10px 14px",
            transition: "background 120ms ease",
          }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLDivElement).style.background = "var(--color-bg-surface)"; }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLDivElement).style.background = "transparent"; }}
        >
          {/* Poster thumbnail */}
          {movie.poster_url ? (
            <img
              src={movie.poster_url}
              alt={movie.title}
              style={{ width: 36, height: 54, borderRadius: 4, objectFit: "cover", flexShrink: 0 }}
            />
          ) : (
            <div
              style={{
                width: 36,
                height: 54,
                borderRadius: 4,
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-subtle)",
                flexShrink: 0,
              }}
            />
          )}

          {/* Info */}
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 13,
                fontWeight: 500,
                color: "var(--color-text-primary)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
            >
              {movie.title}
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 3 }}>
              {movie.year > 0 && (
                <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{movie.year}</span>
              )}
              <span style={statusBadge(movie.status, movie.monitored)}>
                {movie.status}
              </span>
              {movie.minimum_availability && (
                <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
                  min: {movie.minimum_availability.replace("_", " ")}
                </span>
              )}
            </div>
          </div>
        </div>
      </Link>

      {/* Search button */}
      <button
        onClick={(e) => { e.stopPropagation(); onSearch(); }}
        title="Manual search"
        style={{
          background: "none",
          border: "none",
          borderLeft: "1px solid var(--color-border-subtle)",
          padding: "0 14px",
          height: "100%",
          alignSelf: "stretch",
          cursor: "pointer",
          fontSize: 12,
          color: "var(--color-text-muted)",
          whiteSpace: "nowrap",
          display: "flex",
          alignItems: "center",
        }}
        onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-accent)"; (e.currentTarget as HTMLButtonElement).style.background = "color-mix(in srgb, var(--color-accent) 8%, transparent)"; }}
        onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; (e.currentTarget as HTMLButtonElement).style.background = "none"; }}
      >
        Search
      </button>
    </div>
  );
}

// ── Missing tab ────────────────────────────────────────────────────────────────

const PER_PAGE = 50;

function MissingTab({ onSearch }: { onSearch: (m: Movie) => void }) {
  const [page, setPage] = useState(1);
  const { data, isLoading, error } = useWantedMissing(page, PER_PAGE);
  const bulkSearch = useBulkAutoSearch();

  function handleSearchAll() {
    const ids = (data?.movies ?? []).map((m) => m.id);
    if (ids.length === 0) return;
    bulkSearch.mutate(ids, {
      onSuccess: (res) => toast.info(`Searching ${res.total} movie${res.total !== 1 ? "s" : ""}… results via notification.`),
      onError: (err) => toast.error((err as Error).message),
    });
  }

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        {[...Array(8)].map((_, i) => (
          <div key={i} className="skeleton" style={{ height: 76, borderRadius: 6 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p style={{ margin: 0, fontSize: 13, color: "var(--color-danger)" }}>
        Failed to load: {(error as Error).message}
      </p>
    );
  }

  const movies = data?.movies ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / PER_PAGE);

  if (movies.length === 0) {
    return (
      <div style={{ padding: "48px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)" }}>
          All caught up!
        </p>
        <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
          No monitored movies are missing a file.
        </p>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", margin: "0 0 12px" }}>
        <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>
          {total} movie{total !== 1 ? "s" : ""} missing a file
        </p>
        <button
          onClick={handleSearchAll}
          disabled={bulkSearch.isPending}
          style={{
            background: "var(--color-accent)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 5,
            padding: "5px 12px",
            fontSize: 12,
            color: "var(--color-accent-fg)",
            cursor: bulkSearch.isPending ? "default" : "pointer",
            whiteSpace: "nowrap",
            opacity: bulkSearch.isPending ? 0.7 : 1,
          }}
        >
          {bulkSearch.isPending ? "Starting…" : "Search All Missing"}
        </button>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        {movies.map((m) => <MovieRow key={m.id} movie={m} onSearch={() => onSearch(m)} />)}
      </div>
      {totalPages > 1 && (
        <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: 8, marginTop: 20 }}>
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "6px 14px",
              fontSize: 12,
              color: page === 1 ? "var(--color-text-muted)" : "var(--color-text-primary)",
              cursor: page === 1 ? "default" : "pointer",
            }}
          >
            Previous
          </button>
          <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
            {page} / {totalPages}
          </span>
          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "6px 14px",
              fontSize: 12,
              color: page === totalPages ? "var(--color-text-muted)" : "var(--color-text-primary)",
              cursor: page === totalPages ? "default" : "pointer",
            }}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}

// ── Cutoff unmet tab ──────────────────────────────────────────────────────────

function CutoffTab({ onSearch }: { onSearch: (m: Movie) => void }) {
  const { data, isLoading, error } = useWantedCutoff();
  const bulkSearch = useBulkAutoSearch();

  function handleSearchAll() {
    const ids = (data?.movies ?? []).map((m) => m.id);
    if (ids.length === 0) return;
    bulkSearch.mutate(ids, {
      onSuccess: (res) => toast.info(`Searching ${res.total} movie${res.total !== 1 ? "s" : ""}… results via notification.`),
      onError: (err) => toast.error((err as Error).message),
    });
  }

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        {[...Array(6)].map((_, i) => (
          <div key={i} className="skeleton" style={{ height: 76, borderRadius: 6 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p style={{ margin: 0, fontSize: 13, color: "var(--color-danger)" }}>
        Failed to load: {(error as Error).message}
      </p>
    );
  }

  const movies = data?.movies ?? [];

  if (movies.length === 0) {
    return (
      <div style={{ padding: "48px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)" }}>
          All at cutoff!
        </p>
        <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
          All monitored movies meet their quality profile cutoff.
        </p>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", margin: "0 0 12px" }}>
        <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>
          {movies.length} movie{movies.length !== 1 ? "s" : ""} below cutoff quality
        </p>
        <button
          onClick={handleSearchAll}
          disabled={bulkSearch.isPending}
          style={{
            background: "var(--color-accent)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 5,
            padding: "5px 12px",
            fontSize: 12,
            color: "var(--color-accent-fg)",
            cursor: bulkSearch.isPending ? "default" : "pointer",
            whiteSpace: "nowrap",
            opacity: bulkSearch.isPending ? 0.7 : 1,
          }}
        >
          {bulkSearch.isPending ? "Starting…" : "Search All Cutoff Unmet"}
        </button>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        {movies.map((m) => <MovieRow key={m.id} movie={m} onSearch={() => onSearch(m)} />)}
      </div>
    </div>
  );
}

// ── Upgrades tab ──────────────────────────────────────────────────────────────

function UpgradeTierCard({ tier }: { tier: UpgradeTier }) {
  const [expanded, setExpanded] = useState(false);
  const bulkSearch = useBulkAutoSearch();

  function handleSearch() {
    bulkSearch.mutate(tier.movie_ids, {
      onSuccess: (res) =>
        toast.info(`Searching ${res.total} movie${res.total !== 1 ? "s" : ""}… results via notification.`),
      onError: (err) => toast.error((err as Error).message),
    });
  }

  return (
    <div
      style={{
        background: "var(--color-bg-elevated)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        overflow: "hidden",
      }}
    >
      {/* Tier header */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          padding: "12px 16px",
        }}
      >
        <button
          onClick={() => setExpanded((v) => !v)}
          style={{
            flex: 1,
            display: "flex",
            alignItems: "center",
            gap: 10,
            background: "none",
            border: "none",
            cursor: "pointer",
            textAlign: "left",
            padding: 0,
          }}
        >
          {expanded
            ? <ChevronUp size={14} style={{ color: "var(--color-text-muted)", flexShrink: 0 }} />
            : <ChevronDown size={14} style={{ color: "var(--color-text-muted)", flexShrink: 0 }} />
          }
          <div style={{ flex: 1, minWidth: 0 }}>
            <span style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
              {tier.label}
            </span>
            <span style={{ marginLeft: 8, fontSize: 12, color: "var(--color-text-muted)" }}>
              {tier.from_quality} → {tier.to_quality}
            </span>
          </div>
          <span
            style={{
              display: "inline-block",
              padding: "2px 8px",
              borderRadius: 4,
              fontSize: 11,
              fontWeight: 600,
              background: "color-mix(in srgb, var(--color-warning) 14%, transparent)",
              color: "var(--color-warning)",
              flexShrink: 0,
            }}
          >
            {tier.count}
          </span>
        </button>
        <button
          onClick={handleSearch}
          disabled={bulkSearch.isPending}
          style={{
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "5px 12px",
            fontSize: 12,
            fontWeight: 500,
            cursor: bulkSearch.isPending ? "not-allowed" : "pointer",
            flexShrink: 0,
            opacity: bulkSearch.isPending ? 0.7 : 1,
          }}
          onMouseEnter={(e) => {
            if (!bulkSearch.isPending)
              (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)";
          }}
          onMouseLeave={(e) => {
            if (!bulkSearch.isPending)
              (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
          }}
        >
          {bulkSearch.isPending ? "Starting…" : "Search"}
        </button>
      </div>

      {/* Expandable movie list */}
      {expanded && tier.movie_ids.length > 0 && (
        <div
          style={{
            borderTop: "1px solid var(--color-border-subtle)",
            padding: "8px 16px 12px",
            display: "flex",
            flexDirection: "column",
            gap: 4,
          }}
        >
          {tier.movie_ids.map((id) => (
            <Link
              key={id}
              to={`/movies/${id}`}
              style={{
                fontSize: 12,
                color: "var(--color-accent)",
                textDecoration: "none",
                fontFamily: "var(--font-family-mono)",
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLAnchorElement).style.textDecoration = "underline"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLAnchorElement).style.textDecoration = "none"; }}
            >
              {id}
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function UpgradesTab() {
  const { data, isLoading, error } = useUpgradeRecommendations();

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        {[...Array(4)].map((_, i) => (
          <div key={i} className="skeleton" style={{ height: 56, borderRadius: 8 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p style={{ margin: 0, fontSize: 13, color: "var(--color-danger)" }}>
        Failed to load: {(error as Error).message}
      </p>
    );
  }

  if (!data || data.total === 0) {
    return (
      <div style={{ padding: "48px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)" }}>
          Nothing to upgrade
        </p>
        <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
          All movies are at their best available quality.
        </p>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", margin: "0 0 16px" }}>
        <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>
          {data.total} movie{data.total !== 1 ? "s" : ""} can be upgraded
        </p>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        {(data.tiers ?? []).map((tier, i) => (
          <UpgradeTierCard key={i} tier={tier} />
        ))}
      </div>
    </div>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

type WantedTab = "missing" | "cutoff" | "upgrades";

export default function WantedPage() {
  const [tab, setTab] = useState<WantedTab>("missing");
  const [searchMovie, setSearchMovie] = useState<Movie | null>(null);

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      <h1 style={{ margin: "0 0 20px", fontSize: 20, fontWeight: 700, color: "var(--color-text-primary)", letterSpacing: "-0.02em" }}>
        Wanted
      </h1>

      {/* Tabs */}
      <div style={{ display: "flex", gap: 0, borderBottom: "1px solid var(--color-border-subtle)", marginBottom: 20 }}>
        {(["missing", "cutoff", "upgrades"] as WantedTab[]).map((t) => {
          const label = t === "missing" ? "Missing" : t === "cutoff" ? "Cutoff Unmet" : "Upgrades";
          return (
            <button
              key={t}
              onClick={() => setTab(t)}
              style={{
                background: "none",
                border: "none",
                borderBottom: `2px solid ${tab === t ? "var(--color-accent)" : "transparent"}`,
                padding: "10px 18px",
                fontSize: 13,
                fontWeight: tab === t ? 600 : 400,
                color: tab === t ? "var(--color-accent)" : "var(--color-text-muted)",
                cursor: "pointer",
                marginBottom: -1,
                transition: "color 0.1s, border-color 0.1s",
              }}
            >
              {label}
            </button>
          );
        })}
      </div>

      {tab === "missing" && <MissingTab onSearch={setSearchMovie} />}
      {tab === "cutoff" && <CutoffTab onSearch={setSearchMovie} />}
      {tab === "upgrades" && <UpgradesTab />}

      {searchMovie && (
        <ManualSearchModal
          movieId={searchMovie.id}
          movieTitle={searchMovie.title}
          onClose={() => setSearchMovie(null)}
        />
      )}
    </div>
  );
}
