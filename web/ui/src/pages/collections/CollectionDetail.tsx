import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useCollection, useAddMissing } from "@/api/collections";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import { useAddMovie } from "@/api/movies";
import type { CollectionItem } from "@/types";

const TMDB_POSTER_BASE = "https://image.tmdb.org/t/p/w185";

// ── Add Missing Modal ──────────────────────────────────────────────────────

function AddMissingModal({
  collectionId,
  missingCount,
  onClose,
}: {
  collectionId: string;
  missingCount: number;
  onClose: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const addMissing = useAddMissing(collectionId);

  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");
  const [minAvail, setMinAvail] = useState("released");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (libraries && libraries.length > 0 && !libraryId) setLibraryId(libraries[0].id);
  }, [libraries, libraryId]);

  useEffect(() => {
    if (profiles && profiles.length > 0 && !profileId) setProfileId(profiles[0].id);
  }, [profiles, profileId]);

  function handleAdd() {
    if (!libraryId || !profileId) {
      setError("Select a library and quality profile.");
      return;
    }
    addMissing.mutate(
      { library_id: libraryId, quality_profile_id: profileId, minimum_availability: minAvail },
      { onSuccess: onClose }
    );
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.6)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: "var(--color-bg-elevated)",
          border: "1px solid var(--color-border-default)",
          borderRadius: 10,
          width: 400,
          padding: 24,
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h3 style={{ margin: "0 0 16px", fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)" }}>
          Add {missingCount} Missing Film{missingCount === 1 ? "" : "s"}
        </h3>

        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
            Library
            <select
              value={libraryId}
              onChange={(e) => setLibraryId(e.target.value)}
              style={{
                display: "block",
                width: "100%",
                marginTop: 4,
                padding: "7px 10px",
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                color: "var(--color-text-primary)",
                fontSize: 13,
              }}
            >
              {(libraries ?? []).map((l) => (
                <option key={l.id} value={l.id}>{l.name}</option>
              ))}
            </select>
          </label>

          <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
            Quality Profile
            <select
              value={profileId}
              onChange={(e) => setProfileId(e.target.value)}
              style={{
                display: "block",
                width: "100%",
                marginTop: 4,
                padding: "7px 10px",
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                color: "var(--color-text-primary)",
                fontSize: 13,
              }}
            >
              {(profiles ?? []).map((p) => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          </label>

          <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
            Minimum Availability
            <select
              value={minAvail}
              onChange={(e) => setMinAvail(e.target.value)}
              style={{
                display: "block",
                width: "100%",
                marginTop: 4,
                padding: "7px 10px",
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                color: "var(--color-text-primary)",
                fontSize: 13,
              }}
            >
              <option value="announced">Announced</option>
              <option value="in_cinemas">In Cinemas</option>
              <option value="released">Released</option>
            </select>
          </label>
        </div>

        {error && (
          <p style={{ margin: "10px 0 0", fontSize: 12, color: "var(--color-danger)" }}>{error}</p>
        )}

        <div style={{ display: "flex", gap: 10, marginTop: 20, justifyContent: "flex-end" }}>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "7px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleAdd}
            disabled={addMissing.isPending}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "7px 16px",
              fontSize: 13,
              fontWeight: 500,
              cursor: addMissing.isPending ? "not-allowed" : "pointer",
            }}
          >
            {addMissing.isPending ? "Adding…" : `Add ${missingCount} Film${missingCount === 1 ? "" : "s"}`}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Add Single Film Modal ──────────────────────────────────────────────────

function AddSingleModal({
  item,
  onClose,
}: {
  item: CollectionItem;
  onClose: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const addMovie = useAddMovie();

  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");
  const [minAvail, setMinAvail] = useState("released");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (libraries && libraries.length > 0 && !libraryId) setLibraryId(libraries[0].id);
  }, [libraries, libraryId]);

  useEffect(() => {
    if (profiles && profiles.length > 0 && !profileId) setProfileId(profiles[0].id);
  }, [profiles, profileId]);

  function handleAdd() {
    if (!libraryId || !profileId) {
      setError("Select a library and quality profile.");
      return;
    }
    addMovie.mutate(
      {
        tmdb_id: item.tmdb_id,
        library_id: libraryId,
        quality_profile_id: profileId,
        monitored: true,
        minimum_availability: minAvail,
      },
      {
        onSuccess: onClose,
        onError: (e) => setError((e as Error).message),
      }
    );
  }

  const posterSrc = item.poster_path ? `${TMDB_POSTER_BASE}${item.poster_path}` : null;

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.6)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: "var(--color-bg-elevated)",
          border: "1px solid var(--color-border-default)",
          borderRadius: 10,
          width: 400,
          padding: 24,
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: "flex", gap: 14, marginBottom: 16 }}>
          {posterSrc ? (
            <img
              src={posterSrc}
              alt={item.title}
              style={{ width: 56, height: 84, objectFit: "cover", borderRadius: 4, flexShrink: 0 }}
            />
          ) : (
            <div
              style={{
                width: 56,
                height: 84,
                background: "var(--color-bg-surface)",
                borderRadius: 4,
                flexShrink: 0,
              }}
            />
          )}
          <div>
            <h3 style={{ margin: "0 0 4px", fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)" }}>
              {item.title}
            </h3>
            {item.year > 0 && (
              <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>{item.year}</span>
            )}
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
            Library
            <select
              value={libraryId}
              onChange={(e) => setLibraryId(e.target.value)}
              style={{
                display: "block",
                width: "100%",
                marginTop: 4,
                padding: "7px 10px",
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                color: "var(--color-text-primary)",
                fontSize: 13,
              }}
            >
              {(libraries ?? []).map((l) => (
                <option key={l.id} value={l.id}>{l.name}</option>
              ))}
            </select>
          </label>
          <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
            Quality Profile
            <select
              value={profileId}
              onChange={(e) => setProfileId(e.target.value)}
              style={{
                display: "block",
                width: "100%",
                marginTop: 4,
                padding: "7px 10px",
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                color: "var(--color-text-primary)",
                fontSize: 13,
              }}
            >
              {(profiles ?? []).map((p) => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          </label>
          <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
            Minimum Availability
            <select
              value={minAvail}
              onChange={(e) => setMinAvail(e.target.value)}
              style={{
                display: "block",
                width: "100%",
                marginTop: 4,
                padding: "7px 10px",
                background: "var(--color-bg-surface)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                color: "var(--color-text-primary)",
                fontSize: 13,
              }}
            >
              <option value="announced">Announced</option>
              <option value="in_cinemas">In Cinemas</option>
              <option value="released">Released</option>
            </select>
          </label>
        </div>

        {error && (
          <p style={{ margin: "10px 0 0", fontSize: 12, color: "var(--color-danger)" }}>{error}</p>
        )}

        <div style={{ display: "flex", gap: 10, marginTop: 20, justifyContent: "flex-end" }}>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "7px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleAdd}
            disabled={addMovie.isPending}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "7px 16px",
              fontSize: 13,
              fontWeight: 500,
              cursor: addMovie.isPending ? "not-allowed" : "pointer",
            }}
          >
            {addMovie.isPending ? "Adding…" : "Add to Library"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Film Card ──────────────────────────────────────────────────────────────

function FilmCard({
  item,
  onAdd,
}: {
  item: CollectionItem;
  onAdd: (item: CollectionItem) => void;
}) {
  const [hovered, setHovered] = useState(false);
  const posterSrc = item.poster_path ? `${TMDB_POSTER_BASE}${item.poster_path}` : null;

  const card = (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        position: "relative",
        borderRadius: 6,
        overflow: "hidden",
        border: item.in_library
          ? "2px solid var(--color-success)"
          : "2px solid var(--color-border-subtle)",
        background: "var(--color-bg-surface)",
        cursor: item.in_library ? "pointer" : "default",
      }}
    >
      {/* Poster */}
      <div style={{ aspectRatio: "2/3", background: "var(--color-bg-elevated)" }}>
        {posterSrc ? (
          <img
            src={posterSrc}
            alt={item.title}
            style={{ width: "100%", height: "100%", objectFit: "cover" }}
          />
        ) : (
          <div
            style={{
              width: "100%",
              height: "100%",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 11,
              color: "var(--color-text-muted)",
              padding: 8,
              textAlign: "center",
            }}
          >
            {item.title}
          </div>
        )}
      </div>

      {/* In-library badge */}
      {item.in_library && (
        <div
          style={{
            position: "absolute",
            top: 6,
            right: 6,
            background: "var(--color-success)",
            color: "#fff",
            borderRadius: 3,
            fontSize: 10,
            fontWeight: 700,
            padding: "2px 5px",
          }}
        >
          ✓
        </div>
      )}

      {/* Overlay on hover for missing films */}
      {!item.in_library && hovered && (
        <div
          style={{
            position: "absolute",
            inset: 0,
            background: "rgba(0,0,0,0.55)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          <button
            onClick={() => onAdd(item)}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 5,
              padding: "6px 14px",
              fontSize: 12,
              fontWeight: 500,
              cursor: "pointer",
            }}
          >
            Add
          </button>
        </div>
      )}
    </div>
  );

  if (item.in_library && item.movie_id) {
    return (
      <Link to={`/movies/${item.movie_id}`} style={{ textDecoration: "none" }}>
        {card}
        <div style={{ marginTop: 6 }}>
          <div
            style={{
              fontSize: 11,
              fontWeight: 500,
              color: "var(--color-text-primary)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {item.title}
          </div>
          {item.year > 0 && (
            <div style={{ fontSize: 10, color: "var(--color-text-muted)" }}>{item.year}</div>
          )}
        </div>
      </Link>
    );
  }

  return (
    <div>
      {card}
      <div style={{ marginTop: 6 }}>
        <div
          style={{
            fontSize: 11,
            fontWeight: 500,
            color: "var(--color-text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {item.title}
        </div>
        {item.year > 0 && (
          <div style={{ fontSize: 10, color: "var(--color-text-muted)" }}>{item.year}</div>
        )}
      </div>
    </div>
  );
}

// ── Collection Detail Page ─────────────────────────────────────────────────

export default function CollectionDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: coll, isLoading, error } = useCollection(id ?? "");
  const [showAddMissing, setShowAddMissing] = useState(false);
  const [addSingleTarget, setAddSingleTarget] = useState<CollectionItem | null>(null);

  if (isLoading) {
    return (
      <div>
        <div className="skeleton" style={{ height: 28, width: 200, borderRadius: 4, marginBottom: 8 }} />
        <div className="skeleton" style={{ height: 16, width: 120, borderRadius: 4, marginBottom: 24 }} />
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(120px, 1fr))",
            gap: 12,
          }}
        >
          {[...Array(12)].map((_, i) => (
            <div key={i} className="skeleton" style={{ aspectRatio: "2/3", borderRadius: 6 }} />
          ))}
        </div>
      </div>
    );
  }

  if (error || !coll) {
    return (
      <div style={{ color: "var(--color-danger)", fontSize: 13 }}>
        Failed to load collection.
      </div>
    );
  }

  const roleLabel = coll.person_type === "director" ? "Director" : "Actor";
  const missingCount = coll.missing ?? 0;

  return (
    <div>
      {/* Breadcrumb */}
      <button
        onClick={() => navigate("/collections")}
        style={{
          background: "none",
          border: "none",
          color: "var(--color-text-muted)",
          cursor: "pointer",
          fontSize: 12,
          padding: 0,
          marginBottom: 12,
        }}
      >
        ← Collections
      </button>

      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          justifyContent: "space-between",
          gap: 12,
          marginBottom: 20,
          flexWrap: "wrap",
        }}
      >
        <div>
          <h2 style={{ margin: "0 0 4px", fontSize: 20, fontWeight: 700, color: "var(--color-text-primary)" }}>
            {coll.name}
          </h2>
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <span
              style={{
                fontSize: 11,
                fontWeight: 500,
                color: "var(--color-text-muted)",
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-subtle)",
                borderRadius: 3,
                padding: "1px 7px",
              }}
            >
              {roleLabel}
            </span>
            <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
              {coll.in_library} / {coll.total} in library
            </span>
          </div>
        </div>

        {missingCount > 0 && (
          <button
            onClick={() => setShowAddMissing(true)}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              fontWeight: 500,
              cursor: "pointer",
              whiteSpace: "nowrap",
            }}
          >
            Add {missingCount} Missing Film{missingCount === 1 ? "" : "s"}
          </button>
        )}
      </div>

      {/* Film grid */}
      {coll.items && coll.items.length > 0 ? (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(120px, 1fr))",
            gap: 12,
          }}
        >
          {coll.items.map((item) => (
            <FilmCard key={item.tmdb_id} item={item} onAdd={setAddSingleTarget} />
          ))}
        </div>
      ) : (
        <div
          style={{
            textAlign: "center",
            padding: "60px 20px",
            color: "var(--color-text-muted)",
            fontSize: 14,
          }}
        >
          No films found for this collection.
        </div>
      )}

      {showAddMissing && (
        <AddMissingModal
          collectionId={coll.id}
          missingCount={missingCount}
          onClose={() => setShowAddMissing(false)}
        />
      )}

      {addSingleTarget && (
        <AddSingleModal
          item={addSingleTarget}
          onClose={() => setAddSingleTarget(null)}
        />
      )}
    </div>
  );
}
