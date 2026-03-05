import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  useCollections,
  useCreateCollection,
  useDeleteCollection,
  useSearchAll,
} from "@/api/collections";
import type { EntitySearchResult } from "@/types";

const TMDB_IMG_BASE = "https://image.tmdb.org/t/p/w45";

// ── Add Collection Modal ───────────────────────────────────────────────────

function AddCollectionModal({ onClose }: { onClose: () => void }) {
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  // For person results: track whether each person is being added as director or actor.
  const [selectedType, setSelectedType] = useState<Record<number, "director" | "actor">>({});
  const createCollection = useCreateCollection();

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 400);
    return () => clearTimeout(t);
  }, [query]);

  const { data: results, isFetching } = useSearchAll(debouncedQuery);

  function handleAdd(result: EntitySearchResult, overrideType?: string) {
    const personType =
      result.result_type === "franchise"
        ? "franchise"
        : (overrideType ?? selectedType[result.id] ?? "director");
    createCollection.mutate(
      { person_id: result.id, person_type: personType },
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
          width: 520,
          maxHeight: "80vh",
          overflow: "hidden",
          display: "flex",
          flexDirection: "column",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div
          style={{
            padding: "16px 20px",
            borderBottom: "1px solid var(--color-border-subtle)",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <span style={{ fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Add Collection
          </span>
          <button
            onClick={onClose}
            style={{ background: "none", border: "none", color: "var(--color-text-muted)", cursor: "pointer", fontSize: 18, lineHeight: 1, padding: 2 }}
          >
            ×
          </button>
        </div>

        {/* Search */}
        <div style={{ padding: "14px 20px", borderBottom: "1px solid var(--color-border-subtle)" }}>
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search for a director, actor, or franchise (e.g. Alien)…"
            style={{
              width: "100%",
              padding: "8px 12px",
              background: "var(--color-bg-surface)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              color: "var(--color-text-primary)",
              fontSize: 13,
              boxSizing: "border-box",
              outline: "none",
            }}
          />
        </div>

        {/* Results */}
        <div style={{ overflowY: "auto", flex: 1, padding: "8px 0" }}>
          {isFetching && (
            <div style={{ padding: "12px 20px", fontSize: 12, color: "var(--color-text-muted)" }}>
              Searching…
            </div>
          )}
          {!isFetching && results && results.length === 0 && debouncedQuery.length >= 2 && (
            <div style={{ padding: "12px 20px", fontSize: 12, color: "var(--color-text-muted)" }}>
              No results for "{debouncedQuery}"
            </div>
          )}
          {results?.map((result) => {
            const isFranchise = result.result_type === "franchise";
            const imageSrc = result.image_path ? `${TMDB_IMG_BASE}${result.image_path}` : null;
            const personType = selectedType[result.id] ?? "director";

            return (
              <div
                key={`${result.result_type}-${result.id}`}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 12,
                  padding: "10px 20px",
                  borderBottom: "1px solid var(--color-border-subtle)",
                }}
              >
                {/* Thumbnail */}
                {imageSrc ? (
                  <img
                    src={imageSrc}
                    alt={result.name}
                    style={{
                      width: 36,
                      height: 36,
                      borderRadius: isFranchise ? 4 : "50%",
                      objectFit: "cover",
                      flexShrink: 0,
                    }}
                  />
                ) : (
                  <div
                    style={{
                      width: 36,
                      height: 36,
                      borderRadius: isFranchise ? 4 : "50%",
                      background: "var(--color-bg-surface)",
                      flexShrink: 0,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      fontSize: 16,
                    }}
                  >
                    {isFranchise ? "🎬" : "👤"}
                  </div>
                )}

                {/* Name + subtitle */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {result.name}
                  </div>
                  <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                    <span
                      style={{
                        fontSize: 10,
                        fontWeight: 500,
                        padding: "1px 5px",
                        borderRadius: 3,
                        background: isFranchise ? "rgba(139,92,246,0.15)" : "rgba(59,130,246,0.12)",
                        color: isFranchise ? "#8b5cf6" : "#60a5fa",
                      }}
                    >
                      {isFranchise ? "Franchise" : "Person"}
                    </span>
                    <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
                      {result.subtitle}
                    </span>
                  </div>
                </div>

                {/* Director/Actor selector — only for persons */}
                {!isFranchise && (
                  <select
                    value={personType}
                    onChange={(e) =>
                      setSelectedType((prev) => ({
                        ...prev,
                        [result.id]: e.target.value as "director" | "actor",
                      }))
                    }
                    style={{
                      background: "var(--color-bg-surface)",
                      border: "1px solid var(--color-border-default)",
                      borderRadius: 4,
                      color: "var(--color-text-secondary)",
                      fontSize: 11,
                      padding: "3px 6px",
                      cursor: "pointer",
                    }}
                  >
                    <option value="director">Director</option>
                    <option value="actor">Actor</option>
                  </select>
                )}

                <button
                  onClick={() => handleAdd(result, isFranchise ? "franchise" : personType)}
                  disabled={createCollection.isPending}
                  style={{
                    background: "var(--color-accent)",
                    color: "var(--color-accent-fg)",
                    border: "none",
                    borderRadius: 5,
                    padding: "5px 12px",
                    fontSize: 12,
                    fontWeight: 500,
                    cursor: createCollection.isPending ? "not-allowed" : "pointer",
                    whiteSpace: "nowrap",
                  }}
                >
                  Add
                </button>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ── Collection Card ────────────────────────────────────────────────────────

function CollectionCard({
  id,
  name,
  personType,
  total,
  inLibrary,
  missing,
}: {
  id: string;
  name: string;
  personType: string;
  total: number;
  inLibrary: number;
  missing: number;
}) {
  const navigate = useNavigate();
  const deleteCollection = useDeleteCollection();
  const [hovered, setHovered] = useState(false);

  const pct = total > 0 ? Math.round((inLibrary / total) * 100) : 0;
  const roleLabel = personType === "franchise" ? "Franchise" : personType === "director" ? "Director" : "Actor";

  return (
    <div
      onClick={() => navigate(`/collections/${id}`)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: hovered ? "var(--color-bg-elevated)" : "var(--color-bg-surface)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        padding: "16px 20px",
        cursor: "pointer",
        display: "flex",
        flexDirection: "column",
        gap: 10,
        transition: "background 100ms ease",
      }}
    >
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 8 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: 14,
              fontWeight: 600,
              color: "var(--color-text-primary)",
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
          >
            {name}
          </div>
          <span
            style={{
              display: "inline-block",
              marginTop: 4,
              fontSize: 11,
              fontWeight: 500,
              color: "var(--color-text-muted)",
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-subtle)",
              borderRadius: 3,
              padding: "1px 6px",
            }}
          >
            {roleLabel}
          </span>
        </div>
        <button
          onClick={(e) => {
            e.stopPropagation();
            deleteCollection.mutate(id);
          }}
          style={{
            background: "none",
            border: "none",
            color: "var(--color-text-muted)",
            cursor: "pointer",
            fontSize: 14,
            padding: 2,
            flexShrink: 0,
          }}
        >
          ✕
        </button>
      </div>

      {/* Progress */}
      <div>
        <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 5 }}>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
            {inLibrary} / {total} in library
          </span>
          <span style={{ fontSize: 11, color: missing > 0 ? "var(--color-text-muted)" : "var(--color-success)" }}>
            {missing > 0 ? `${missing} missing` : "complete"}
          </span>
        </div>
        <div
          style={{
            height: 4,
            background: "var(--color-bg-elevated)",
            borderRadius: 2,
            overflow: "hidden",
          }}
        >
          <div
            style={{
              height: "100%",
              width: `${pct}%`,
              background: pct === 100 ? "var(--color-success)" : "var(--color-accent)",
              borderRadius: 2,
              transition: "width 300ms ease",
            }}
          />
        </div>
      </div>
    </div>
  );
}

// ── Collections Page ───────────────────────────────────────────────────────

export default function CollectionsPage() {
  const { data: collections, isLoading } = useCollections();
  const [showAdd, setShowAdd] = useState(false);

  return (
    <div style={{ maxWidth: 900, margin: "0 auto", padding: "24px 32px" }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 20,
        }}
      >
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600, color: "var(--color-text-primary)" }}>
          Collections
        </h2>
        <button
          onClick={() => setShowAdd(true)}
          style={{
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "7px 16px",
            fontSize: 13,
            fontWeight: 500,
            cursor: "pointer",
          }}
        >
          + Add Collection
        </button>
      </div>

      {isLoading ? (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
            gap: 12,
          }}
        >
          {[...Array(4)].map((_, i) => (
            <div
              key={i}
              className="skeleton"
              style={{ height: 110, borderRadius: 8 }}
            />
          ))}
        </div>
      ) : collections && collections.length > 0 ? (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
            gap: 12,
          }}
        >
          {collections.map((c) => (
            <CollectionCard
              key={c.id}
              id={c.id}
              name={c.name}
              personType={c.person_type}
              total={c.total}
              inLibrary={c.in_library}
              missing={c.missing}
            />
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
          No collections yet. Add a director or actor to get started.
        </div>
      )}

      {showAdd && <AddCollectionModal onClose={() => setShowAdd(false)} />}
    </div>
  );
}
