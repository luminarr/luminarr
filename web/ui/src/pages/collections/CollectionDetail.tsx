import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useCollection, useAddMissing, useAddSelected } from "@/api/collections";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import type { CollectionItem } from "@/types";

const TMDB_POSTER_BASE = "https://image.tmdb.org/t/p/w185";

// ── Shared dropdowns component ─────────────────────────────────────────────

function AddDropdowns({
  libraryId,
  setLibraryId,
  profileId,
  setProfileId,
  minAvail,
  setMinAvail,
}: {
  libraryId: string;
  setLibraryId: (v: string) => void;
  profileId: string;
  setProfileId: (v: string) => void;
  minAvail: string;
  setMinAvail: (v: string) => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();

  useEffect(() => {
    if (libraries && libraries.length > 0 && !libraryId) setLibraryId(libraries[0].id);
  }, [libraries, libraryId, setLibraryId]);

  useEffect(() => {
    if (profiles && profiles.length > 0 && !profileId) setProfileId(profiles[0].id);
  }, [profiles, profileId, setProfileId]);

  const selectStyle = {
    display: "block",
    width: "100%",
    marginTop: 4,
    padding: "7px 10px",
    background: "var(--color-bg-surface)",
    border: "1px solid var(--color-border-default)",
    borderRadius: 6,
    color: "var(--color-text-primary)",
    fontSize: 13,
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
        Library
        <select value={libraryId} onChange={(e) => setLibraryId(e.target.value)} style={selectStyle}>
          {(libraries ?? []).map((l) => (
            <option key={l.id} value={l.id}>{l.name}</option>
          ))}
        </select>
      </label>
      <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
        Quality Profile
        <select value={profileId} onChange={(e) => setProfileId(e.target.value)} style={selectStyle}>
          {(profiles ?? []).map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
      </label>
      <label style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
        Minimum Availability
        <select value={minAvail} onChange={(e) => setMinAvail(e.target.value)} style={selectStyle}>
          <option value="announced">Announced</option>
          <option value="in_cinemas">In Cinemas</option>
          <option value="released">Released</option>
        </select>
      </label>
    </div>
  );
}

// ── Modal shell ────────────────────────────────────────────────────────────

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
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
          {title}
        </h3>
        {children}
      </div>
    </div>
  );
}

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
  const addMissing = useAddMissing(collectionId);
  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");
  const [minAvail, setMinAvail] = useState("released");
  const [error, setError] = useState<string | null>(null);

  function handleAdd() {
    if (!libraryId || !profileId) { setError("Select a library and quality profile."); return; }
    addMissing.mutate(
      { library_id: libraryId, quality_profile_id: profileId, minimum_availability: minAvail },
      { onSuccess: onClose }
    );
  }

  return (
    <Modal title={`Add ${missingCount} Missing Film${missingCount === 1 ? "" : "s"}`} onClose={onClose}>
      <AddDropdowns
        libraryId={libraryId} setLibraryId={setLibraryId}
        profileId={profileId} setProfileId={setProfileId}
        minAvail={minAvail} setMinAvail={setMinAvail}
      />
      {error && <p style={{ margin: "10px 0 0", fontSize: 12, color: "var(--color-danger)" }}>{error}</p>}
      <div style={{ display: "flex", gap: 10, marginTop: 20, justifyContent: "flex-end" }}>
        <button onClick={onClose} style={{ background: "none", border: "1px solid var(--color-border-default)", borderRadius: 6, padding: "7px 16px", fontSize: 13, color: "var(--color-text-secondary)", cursor: "pointer" }}>
          Cancel
        </button>
        <button
          onClick={handleAdd}
          disabled={addMissing.isPending}
          style={{ background: "var(--color-accent)", color: "var(--color-accent-fg)", border: "none", borderRadius: 6, padding: "7px 16px", fontSize: 13, fontWeight: 500, cursor: addMissing.isPending ? "not-allowed" : "pointer" }}
        >
          {addMissing.isPending ? "Adding…" : `Add ${missingCount} Film${missingCount === 1 ? "" : "s"}`}
        </button>
      </div>
    </Modal>
  );
}

// ── Add Selected Modal ─────────────────────────────────────────────────────

function AddSelectedModal({
  collectionId,
  selectedCount,
  tmdbIds,
  onClose,
}: {
  collectionId: string;
  selectedCount: number;
  tmdbIds: number[];
  onClose: () => void;
}) {
  const addSelected = useAddSelected(collectionId);
  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");
  const [minAvail, setMinAvail] = useState("released");
  const [error, setError] = useState<string | null>(null);

  function handleAdd() {
    if (!libraryId || !profileId) { setError("Select a library and quality profile."); return; }
    addSelected.mutate(
      { tmdb_ids: tmdbIds, library_id: libraryId, quality_profile_id: profileId, minimum_availability: minAvail },
      { onSuccess: onClose }
    );
  }

  return (
    <Modal title={`Add ${selectedCount} Selected Film${selectedCount === 1 ? "" : "s"}`} onClose={onClose}>
      <AddDropdowns
        libraryId={libraryId} setLibraryId={setLibraryId}
        profileId={profileId} setProfileId={setProfileId}
        minAvail={minAvail} setMinAvail={setMinAvail}
      />
      {error && <p style={{ margin: "10px 0 0", fontSize: 12, color: "var(--color-danger)" }}>{error}</p>}
      <div style={{ display: "flex", gap: 10, marginTop: 20, justifyContent: "flex-end" }}>
        <button onClick={onClose} style={{ background: "none", border: "1px solid var(--color-border-default)", borderRadius: 6, padding: "7px 16px", fontSize: 13, color: "var(--color-text-secondary)", cursor: "pointer" }}>
          Cancel
        </button>
        <button
          onClick={handleAdd}
          disabled={addSelected.isPending}
          style={{ background: "var(--color-accent)", color: "var(--color-accent-fg)", border: "none", borderRadius: 6, padding: "7px 16px", fontSize: 13, fontWeight: 500, cursor: addSelected.isPending ? "not-allowed" : "pointer" }}
        >
          {addSelected.isPending ? "Adding…" : `Add ${selectedCount} Film${selectedCount === 1 ? "" : "s"}`}
        </button>
      </div>
    </Modal>
  );
}

// ── Film Card ──────────────────────────────────────────────────────────────

// Border meanings:
//   green  = in library AND has a physical file on disk
//   amber  = in library but no file yet (monitored, downloading)
//   default = not in library (can be selected)

function FilmCard({
  item,
  selected,
  onToggleSelect,
}: {
  item: CollectionItem;
  selected: boolean;
  onToggleSelect: (item: CollectionItem) => void;
}) {
  const [hovered, setHovered] = useState(false);
  const posterSrc = item.poster_path ? `${TMDB_POSTER_BASE}${item.poster_path}` : null;

  // Determine border color based on 3 states.
  let borderColor = "var(--color-border-subtle)";
  if (item.in_library) {
    borderColor = item.has_file ? "var(--color-success)" : "#d97706"; // amber for monitored-no-file
  }
  if (selected) {
    borderColor = "var(--color-accent)";
  }

  const card = (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={!item.in_library ? () => onToggleSelect(item) : undefined}
      style={{
        position: "relative",
        borderRadius: 6,
        overflow: "hidden",
        border: `2px solid ${borderColor}`,
        background: selected ? "var(--color-accent-muted)" : "var(--color-bg-surface)",
        cursor: item.in_library ? "default" : "pointer",
        transition: "border-color 120ms ease",
      }}
    >
      {/* Poster */}
      <div style={{ aspectRatio: "2/3", background: "var(--color-bg-elevated)" }}>
        {posterSrc ? (
          <img src={posterSrc} alt={item.title} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
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

      {/* Status badge (top-right) */}
      {item.in_library && (
        <div
          style={{
            position: "absolute",
            top: 6,
            right: 6,
            background: item.has_file ? "var(--color-success)" : "#d97706",
            color: "#fff",
            borderRadius: 3,
            fontSize: 10,
            fontWeight: 700,
            padding: "2px 5px",
          }}
          title={item.has_file ? "On disk" : "Monitored — no file yet"}
        >
          {item.has_file ? "✓" : "↓"}
        </div>
      )}

      {/* Selected checkmark (top-left) */}
      {selected && !item.in_library && (
        <div
          style={{
            position: "absolute",
            top: 6,
            left: 6,
            width: 18,
            height: 18,
            background: "var(--color-accent)",
            borderRadius: 3,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: 11,
            color: "#fff",
            fontWeight: 700,
          }}
        >
          ✓
        </div>
      )}

      {/* Hover indicator — empty checkbox in top-left, same position as selected checkmark */}
      {!item.in_library && !selected && hovered && (
        <div
          style={{
            position: "absolute",
            top: 6,
            left: 6,
            width: 18,
            height: 18,
            border: "2px solid rgba(255,255,255,0.85)",
            borderRadius: 3,
            background: "rgba(0,0,0,0.25)",
            pointerEvents: "none",
          }}
        />
      )}
    </div>
  );

  // In-library items navigate to the movie; missing items are click-to-select.
  if (item.in_library && item.movie_id) {
    return (
      <Link to={`/movies/${item.movie_id}`} style={{ textDecoration: "none" }}>
        {card}
        <FilmLabel item={item} />
      </Link>
    );
  }

  return (
    <div>
      {card}
      <FilmLabel item={item} />
    </div>
  );
}

function FilmLabel({ item }: { item: CollectionItem }) {
  return (
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
  );
}

// ── Collection Detail Page ─────────────────────────────────────────────────

export default function CollectionDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: coll, isLoading, error } = useCollection(id ?? "");

  // Multi-select: set of TMDB IDs for selected missing films.
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [showAddMissing, setShowAddMissing] = useState(false);
  const [showAddSelected, setShowAddSelected] = useState(false);

  // Clear selection when collection changes.
  useEffect(() => { setSelected(new Set()); }, [id]);

  function toggleSelect(item: CollectionItem) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(item.tmdb_id)) { next.delete(item.tmdb_id); } else { next.add(item.tmdb_id); }
      return next;
    });
  }

  function selectAllMissing() {
    const missing = (coll?.items ?? []).filter((i) => !i.in_library).map((i) => i.tmdb_id);
    setSelected(new Set(missing));
  }

  function clearSelection() {
    setSelected(new Set());
  }

  if (isLoading) {
    return (
      <div style={{ maxWidth: 1100, margin: "0 auto", padding: "24px 32px" }}>
        <div className="skeleton" style={{ height: 28, width: 200, borderRadius: 4, marginBottom: 8 }} />
        <div className="skeleton" style={{ height: 16, width: 120, borderRadius: 4, marginBottom: 24 }} />
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(120px, 1fr))", gap: 12 }}>
          {[...Array(12)].map((_, i) => (
            <div key={i} className="skeleton" style={{ aspectRatio: "2/3", borderRadius: 6 }} />
          ))}
        </div>
      </div>
    );
  }

  if (error || !coll) {
    return (
      <div style={{ maxWidth: 1100, margin: "0 auto", padding: "24px 32px", color: "var(--color-danger)", fontSize: 13 }}>
        Failed to load collection.
      </div>
    );
  }

  const roleLabel = coll.person_type === "franchise" ? "Franchise" : coll.person_type === "director" ? "Director" : "Actor";
  const missingCount = coll.missing ?? 0;
  const selectedCount = selected.size;
  const selectedIds = Array.from(selected);

  return (
    <div style={{ maxWidth: 1100, margin: "0 auto", padding: "24px 32px" }}>
      {/* Breadcrumb */}
      <button
        onClick={() => navigate("/collections")}
        style={{ background: "none", border: "none", color: "var(--color-text-muted)", cursor: "pointer", fontSize: 12, padding: 0, marginBottom: 12 }}
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
          marginBottom: 16,
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

        {/* Action buttons */}
        <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
          {missingCount > 0 && (
            <button
              onClick={selectAllMissing}
              style={{
                background: "none",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                padding: "7px 14px",
                fontSize: 13,
                color: "var(--color-text-secondary)",
                cursor: "pointer",
                whiteSpace: "nowrap",
              }}
            >
              Select All Missing
            </button>
          )}
          {selectedCount > 0 && (
            <>
              <button
                onClick={clearSelection}
                style={{
                  background: "none",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "7px 14px",
                  fontSize: 13,
                  color: "var(--color-text-muted)",
                  cursor: "pointer",
                  whiteSpace: "nowrap",
                }}
              >
                Clear
              </button>
              <button
                onClick={() => setShowAddSelected(true)}
                style={{
                  background: "var(--color-accent)",
                  color: "var(--color-accent-fg)",
                  border: "none",
                  borderRadius: 6,
                  padding: "7px 16px",
                  fontSize: 13,
                  fontWeight: 500,
                  cursor: "pointer",
                  whiteSpace: "nowrap",
                }}
              >
                Add {selectedCount} Selected Film{selectedCount === 1 ? "" : "s"}
              </button>
            </>
          )}
          {selectedCount === 0 && missingCount > 0 && (
            <button
              onClick={() => setShowAddMissing(true)}
              style={{
                background: "var(--color-accent)",
                color: "var(--color-accent-fg)",
                border: "none",
                borderRadius: 6,
                padding: "7px 16px",
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
      </div>

      {/* Legend */}
      {coll.items && coll.items.length > 0 && (
        <div style={{ display: "flex", gap: 16, marginBottom: 16, fontSize: 11, color: "var(--color-text-muted)" }}>
          <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <span style={{ width: 10, height: 10, borderRadius: 2, background: "var(--color-success)", display: "inline-block" }} />
            On disk
          </span>
          <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <span style={{ width: 10, height: 10, borderRadius: 2, background: "#d97706", display: "inline-block" }} />
            Monitored
          </span>
          <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <span style={{ width: 10, height: 10, borderRadius: 2, background: "var(--color-border-subtle)", border: "1px solid var(--color-border-subtle)", display: "inline-block" }} />
            Missing — click to select
          </span>
        </div>
      )}

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
            <FilmCard
              key={item.tmdb_id}
              item={item}
              selected={selected.has(item.tmdb_id)}
              onToggleSelect={toggleSelect}
            />
          ))}
        </div>
      ) : (
        <div style={{ textAlign: "center", padding: "60px 20px", color: "var(--color-text-muted)", fontSize: 14 }}>
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

      {showAddSelected && (
        <AddSelectedModal
          collectionId={coll.id}
          selectedCount={selectedCount}
          tmdbIds={selectedIds}
          onClose={() => { setShowAddSelected(false); setSelected(new Set()); }}
        />
      )}
    </div>
  );
}
