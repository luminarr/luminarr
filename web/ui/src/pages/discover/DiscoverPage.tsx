import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Star } from "lucide-react";
import { toast } from "sonner";
import {
  useDiscover,
  useDiscoverByGenre,
  useGenreList,
  type DiscoverResult,
} from "@/api/discover";
import { useAddMovie } from "@/api/movies";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import { Poster } from "@/components/Poster";
import Modal from "@/components/Modal";

// ── Types ────────────────────────────────────────────────────────────────────

type Tab = "trending" | "popular" | "top-rated" | "upcoming" | "genre";

const TABS: { value: Tab; label: string }[] = [
  { value: "trending", label: "Trending" },
  { value: "popular", label: "Popular" },
  { value: "top-rated", label: "Top Rated" },
  { value: "upcoming", label: "Upcoming" },
  { value: "genre", label: "By Genre" },
];

// ── Add Movie Modal ─────────────────────────────────────────────────────────

function QuickAddModal({
  movie,
  onClose,
}: {
  movie: DiscoverResult;
  onClose: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const addMovie = useAddMovie();

  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");

  // Auto-select first library/profile
  const lib = libraryId || libraries?.[0]?.id || "";
  const prof = profileId || profiles?.[0]?.id || "";

  function handleAdd() {
    if (!lib || !prof) return;
    addMovie.mutate(
      {
        tmdb_id: movie.tmdb_id,
        library_id: lib,
        quality_profile_id: prof,
        monitored: true,
      },
      {
        onSuccess: () => {
          toast.success(`Added ${movie.title}`);
          onClose();
        },
        onError: (err) => toast.error((err as Error).message),
      }
    );
  }

  const inputStyle: React.CSSProperties = {
    width: "100%",
    padding: "8px 10px",
    borderRadius: 6,
    border: "1px solid var(--color-border-default)",
    background: "var(--color-bg-elevated)",
    color: "var(--color-text-primary)",
    fontSize: 13,
  };

  return (
    <Modal onClose={onClose} width={400}>
      <div style={{ padding: 24 }}>
        <h3
          style={{
            margin: "0 0 4px",
            fontSize: 16,
            fontWeight: 600,
            color: "var(--color-text-primary)",
          }}
        >
          Add {movie.title}
        </h3>
        <p
          style={{
            margin: "0 0 20px",
            fontSize: 12,
            color: "var(--color-text-muted)",
          }}
        >
          {movie.year > 0 && movie.year}
        </p>

        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div>
            <label
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--color-text-secondary)",
                marginBottom: 4,
              }}
            >
              Library
            </label>
            <select
              value={lib}
              onChange={(e) => setLibraryId(e.target.value)}
              style={inputStyle}
              data-testid="add-library-select"
            >
              {libraries?.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--color-text-secondary)",
                marginBottom: 4,
              }}
            >
              Quality Profile
            </label>
            <select
              value={prof}
              onChange={(e) => setProfileId(e.target.value)}
              style={inputStyle}
              data-testid="add-profile-select"
            >
              {profiles?.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div
          style={{
            display: "flex",
            justifyContent: "flex-end",
            gap: 8,
            marginTop: 20,
          }}
        >
          <button
            onClick={onClose}
            style={{
              padding: "7px 14px",
              borderRadius: 6,
              border: "1px solid var(--color-border-default)",
              background: "transparent",
              color: "var(--color-text-secondary)",
              fontSize: 13,
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleAdd}
            disabled={!lib || !prof || addMovie.isPending}
            data-testid="add-movie-btn"
            style={{
              padding: "7px 16px",
              borderRadius: 6,
              border: "none",
              background:
                !lib || !prof || addMovie.isPending
                  ? "var(--color-bg-subtle)"
                  : "var(--color-accent)",
              color:
                !lib || !prof || addMovie.isPending
                  ? "var(--color-text-muted)"
                  : "var(--color-accent-fg)",
              fontSize: 13,
              fontWeight: 500,
              cursor:
                !lib || !prof || addMovie.isPending
                  ? "not-allowed"
                  : "pointer",
            }}
          >
            {addMovie.isPending ? "Adding..." : "Add Movie"}
          </button>
        </div>
      </div>
    </Modal>
  );
}

// ── Movie Card ──────────────────────────────────────────────────────────────

function DiscoverCard({
  movie,
  onAdd,
}: {
  movie: DiscoverResult;
  onAdd: (m: DiscoverResult) => void;
}) {
  const navigate = useNavigate();

  return (
    <div style={{ width: "100%" }}>
      <div style={{ position: "relative" }}>
        <Poster
          src={
            movie.poster_path
              ? `https://image.tmdb.org/t/p/w342${movie.poster_path}`
              : undefined
          }
          title={movie.title}
          year={movie.year}
        />

        {/* Rating badge */}
        {movie.rating > 0 && (
          <div
            style={{
              position: "absolute",
              top: 6,
              right: 6,
              display: "flex",
              alignItems: "center",
              gap: 3,
              padding: "2px 6px",
              borderRadius: 4,
              background: "rgba(0,0,0,0.7)",
              backdropFilter: "blur(4px)",
              fontSize: 11,
              fontWeight: 600,
              color: "#fbbf24",
            }}
          >
            <Star size={10} fill="#fbbf24" stroke="none" />
            {movie.rating.toFixed(1)}
          </div>
        )}
      </div>

      <div style={{ marginTop: 8 }}>
        <span
          style={{
            display: "block",
            fontSize: 12,
            fontWeight: 500,
            color: "var(--color-text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {movie.title}
        </span>
        <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
          {movie.year > 0 && movie.year}
        </span>
      </div>

      {/* Action badge */}
      <div style={{ marginTop: 6 }}>
        {movie.in_library ? (
          <button
            onClick={() =>
              movie.library_movie_id &&
              navigate(`/movies/${movie.library_movie_id}`)
            }
            data-testid={`in-library-${movie.tmdb_id}`}
            style={{
              fontSize: 10,
              padding: "3px 8px",
              borderRadius: 4,
              border: "none",
              background:
                "color-mix(in srgb, var(--color-success) 15%, transparent)",
              color: "var(--color-success)",
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            In Library
          </button>
        ) : movie.excluded ? (
          <span
            style={{
              fontSize: 10,
              padding: "3px 8px",
              borderRadius: 4,
              background: "var(--color-bg-subtle)",
              color: "var(--color-text-muted)",
              fontWeight: 500,
            }}
          >
            Excluded
          </span>
        ) : (
          <button
            onClick={() => onAdd(movie)}
            data-testid={`add-${movie.tmdb_id}`}
            style={{
              fontSize: 10,
              padding: "3px 8px",
              borderRadius: 4,
              border: "none",
              background:
                "color-mix(in srgb, var(--color-accent) 15%, transparent)",
              color: "var(--color-accent)",
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            + Add
          </button>
        )}
      </div>
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function DiscoverPage() {
  const [tab, setTab] = useState<Tab>("trending");
  const [page, setPage] = useState(1);
  const [genreId, setGenreId] = useState(0);
  const [addingMovie, setAddingMovie] = useState<DiscoverResult | null>(null);

  // Reset page when switching tabs
  function switchTab(t: Tab) {
    setTab(t);
    setPage(1);
  }

  const category = tab !== "genre" ? tab : undefined;
  const listQuery = useDiscover(category ?? "trending", category ? page : 0);
  const genreQuery = useDiscoverByGenre(
    tab === "genre" ? genreId : 0,
    tab === "genre" ? page : 0
  );
  const { data: genres } = useGenreList();

  const activeQuery = tab === "genre" ? genreQuery : listQuery;
  const data = activeQuery.data;
  const isLoading = activeQuery.isLoading;
  const isError = activeQuery.isError;

  return (
    <div style={{ padding: 24, maxWidth: 1200 }}>
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
          Discover
        </h1>
        <p
          style={{
            fontSize: 13,
            color: "var(--color-text-secondary)",
            margin: 0,
          }}
        >
          Find movies to add to your library
        </p>
      </div>

      {/* Tab pills */}
      <div
        style={{
          display: "flex",
          gap: 6,
          marginBottom: 20,
          flexWrap: "wrap",
          alignItems: "center",
        }}
      >
        {TABS.map((t) => {
          const active = tab === t.value;
          return (
            <button
              key={t.value}
              onClick={() => switchTab(t.value)}
              data-testid={`tab-${t.value}`}
              style={{
                padding: "5px 12px",
                borderRadius: 6,
                border: active
                  ? "1px solid var(--color-accent)"
                  : "1px solid var(--color-border-default)",
                background: active
                  ? "var(--color-accent-muted)"
                  : "transparent",
                color: active
                  ? "var(--color-accent-hover)"
                  : "var(--color-text-secondary)",
                fontSize: 12,
                fontWeight: 500,
                cursor: "pointer",
              }}
            >
              {t.label}
            </button>
          );
        })}

        {/* Genre dropdown */}
        {tab === "genre" && genres && (
          <select
            value={genreId}
            onChange={(e) => {
              setGenreId(Number(e.target.value));
              setPage(1);
            }}
            data-testid="genre-select"
            style={{
              padding: "5px 10px",
              borderRadius: 6,
              border: "1px solid var(--color-border-default)",
              background: "var(--color-bg-elevated)",
              color: "var(--color-text-primary)",
              fontSize: 12,
            }}
          >
            <option value={0}>Select genre...</option>
            {genres.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </select>
        )}
      </div>

      {/* Loading */}
      {isLoading && (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))",
            gap: 20,
          }}
        >
          {Array.from({ length: 20 }).map((_, i) => (
            <div key={i}>
              <div
                className="skeleton"
                style={{ paddingBottom: "150%", borderRadius: 8 }}
              />
              <div
                className="skeleton"
                style={{ height: 14, width: "80%", marginTop: 8, borderRadius: 4 }}
              />
            </div>
          ))}
        </div>
      )}

      {/* Error */}
      {isError && (
        <p
          style={{
            fontSize: 13,
            color: "var(--color-danger)",
            textAlign: "center",
            padding: "48px 0",
          }}
        >
          Failed to load movies. Check that TMDB is configured.
        </p>
      )}

      {/* Empty */}
      {data && data.results.length === 0 && (
        <p
          style={{
            fontSize: 13,
            color: "var(--color-text-muted)",
            textAlign: "center",
            padding: "48px 0",
          }}
          data-testid="empty-state"
        >
          {tab === "genre" && genreId === 0
            ? "Select a genre to browse"
            : "No movies found"}
        </p>
      )}

      {/* Grid */}
      {data && data.results.length > 0 && (
        <>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))",
              gap: 20,
            }}
            data-testid="discover-grid"
          >
            {data.results.map((movie) => (
              <DiscoverCard
                key={movie.tmdb_id}
                movie={movie}
                onAdd={setAddingMovie}
              />
            ))}
          </div>

          {/* Pagination */}
          {data.total_pages > 1 && (
            <div
              style={{
                display: "flex",
                justifyContent: "center",
                gap: 12,
                marginTop: 32,
              }}
            >
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                style={{
                  padding: "6px 14px",
                  borderRadius: 6,
                  border: "1px solid var(--color-border-default)",
                  background: "transparent",
                  color:
                    page <= 1
                      ? "var(--color-text-muted)"
                      : "var(--color-text-secondary)",
                  fontSize: 12,
                  cursor: page <= 1 ? "not-allowed" : "pointer",
                }}
              >
                Previous
              </button>
              <span
                style={{
                  fontSize: 12,
                  color: "var(--color-text-muted)",
                  alignSelf: "center",
                }}
              >
                Page {data.page} of {data.total_pages}
              </span>
              <button
                onClick={() =>
                  setPage((p) => Math.min(data.total_pages, p + 1))
                }
                disabled={page >= data.total_pages}
                data-testid="next-page"
                style={{
                  padding: "6px 14px",
                  borderRadius: 6,
                  border: "1px solid var(--color-border-default)",
                  background: "transparent",
                  color:
                    page >= data.total_pages
                      ? "var(--color-text-muted)"
                      : "var(--color-text-secondary)",
                  fontSize: 12,
                  cursor:
                    page >= data.total_pages ? "not-allowed" : "pointer",
                }}
              >
                Next
              </button>
            </div>
          )}
        </>
      )}

      {/* Add modal */}
      {addingMovie && (
        <QuickAddModal
          movie={addingMovie}
          onClose={() => setAddingMovie(null)}
        />
      )}
    </div>
  );
}
