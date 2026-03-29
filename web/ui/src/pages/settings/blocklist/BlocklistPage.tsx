import { useState } from "react";
import { toast } from "sonner";
import { Trash2 } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import { useBlocklist, useDeleteBlocklistEntry, useClearBlocklist } from "@/api/blocklist";
import { formatBytes, formatDate } from "@/lib/utils";

export default function BlocklistPage() {
  const [page, setPage] = useState(1);
  const perPage = 50;

  const { data, isLoading, error } = useBlocklist(page, perPage);
  const deleteMutation = useDeleteBlocklistEntry();
  const clearMutation = useClearBlocklist();

  async function handleDelete(id: string, title: string) {
    try {
      await deleteMutation.mutateAsync(id);
      toast.success(`Removed "${title}" from blocklist`);
    } catch {
      toast.error("Failed to remove entry");
    }
  }

  async function handleClear() {
    if (!confirm("Remove all entries from the blocklist?")) return;
    try {
      await clearMutation.mutateAsync();
      setPage(1);
      toast.success("Blocklist cleared");
    } catch {
      toast.error("Failed to clear blocklist");
    }
  }

  const totalPages = data ? Math.ceil(data.total / perPage) : 1;

  return (
    <div style={{ padding: "32px", maxWidth: 1100 }}>
      <PageHeader
        title="Blocklist"
        description="Releases that will be skipped during automatic and manual searches."
        docsUrl={DOCS_URLS.blocklist}
        action={
          data && data.total > 0 ? (
            <button
              onClick={handleClear}
              disabled={clearMutation.isPending}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 6,
                background: "var(--color-danger-muted)",
                border: "1px solid var(--color-danger)",
                borderRadius: 6,
                padding: "7px 14px",
                fontSize: 13,
                fontWeight: 500,
                color: "var(--color-danger)",
                cursor: "pointer",
                whiteSpace: "nowrap",
              }}
            >
              <Trash2 size={14} />
              {clearMutation.isPending ? "Clearing…" : "Clear All"}
            </button>
          ) : undefined
        }
      />

      {/* Loading skeleton */}
      {isLoading && (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className="skeleton"
              style={{ height: 48, borderRadius: 6 }}
            />
          ))}
        </div>
      )}

      {/* Error */}
      {error && (
        <div
          style={{
            background: "var(--color-danger-muted, rgba(239,68,68,0.08))",
            border: "1px solid var(--color-danger)",
            borderRadius: 8,
            padding: "16px 20px",
            color: "var(--color-danger)",
            fontSize: 13,
          }}
        >
          Failed to load blocklist: {(error as Error).message}
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && data && data.items.length === 0 && (
        <div
          style={{
            textAlign: "center",
            padding: "64px 32px",
            color: "var(--color-text-muted)",
          }}
        >
          <div style={{ fontSize: 40, marginBottom: 12 }}>✓</div>
          <div style={{ fontSize: 15, fontWeight: 500, marginBottom: 4 }}>
            Blocklist is empty
          </div>
          <div style={{ fontSize: 13 }}>
            Releases are automatically added here when a grab fails.
          </div>
        </div>
      )}

      {/* Table */}
      {!isLoading && !error && data && data.items.length > 0 && (
        <>
          <div
            style={{
              background: "var(--color-bg-surface)",
              border: "1px solid var(--color-border-subtle)",
              borderRadius: 8,
              overflow: "hidden",
            }}
          >
            {/* Table header */}
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "2fr 1fr 100px 80px 160px 40px",
                gap: "0 12px",
                padding: "10px 16px",
                borderBottom: "1px solid var(--color-border-subtle)",
                fontSize: 11,
                fontWeight: 600,
                color: "var(--color-text-muted)",
                textTransform: "uppercase",
                letterSpacing: "0.06em",
              }}
            >
              <span>Release</span>
              <span>Movie</span>
              <span>Protocol</span>
              <span style={{ textAlign: "right" }}>Size</span>
              <span>Added</span>
              <span />
            </div>

            {/* Rows */}
            {data.items.map((entry) => (
              <div
                key={entry.id}
                style={{
                  display: "grid",
                  gridTemplateColumns: "2fr 1fr 100px 80px 160px 40px",
                  gap: "0 12px",
                  padding: "12px 16px",
                  borderBottom: "1px solid var(--color-border-subtle)",
                  alignItems: "center",
                }}
              >
                {/* Release title + notes */}
                <div style={{ minWidth: 0 }}>
                  <div
                    style={{
                      fontSize: 13,
                      color: "var(--color-text-primary)",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                    title={entry.release_title}
                  >
                    {entry.release_title}
                  </div>
                  {entry.notes && (
                    <div
                      style={{
                        fontSize: 11,
                        color: "var(--color-text-muted)",
                        marginTop: 2,
                      }}
                    >
                      {entry.notes}
                    </div>
                  )}
                </div>

                {/* Movie */}
                <div
                  style={{
                    fontSize: 13,
                    color: "var(--color-text-secondary)",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                  title={entry.movie_title}
                >
                  {entry.movie_title}
                </div>

                {/* Protocol */}
                <div style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
                  {entry.protocol || "—"}
                </div>

                {/* Size */}
                <div
                  style={{
                    fontSize: 12,
                    color: "var(--color-text-muted)",
                    textAlign: "right",
                  }}
                >
                  {entry.size > 0 ? formatBytes(entry.size) : "—"}
                </div>

                {/* Added at */}
                <div style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
                  {formatDate(entry.added_at, true)}
                </div>

                {/* Delete */}
                <div style={{ display: "flex", justifyContent: "flex-end" }}>
                  <button
                    onClick={() => handleDelete(entry.id, entry.release_title)}
                    disabled={deleteMutation.isPending}
                    title="Remove from blocklist"
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
                    }}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLButtonElement).style.color =
                        "var(--color-danger)";
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLButtonElement).style.color =
                        "var(--color-text-muted)";
                    }}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                marginTop: 16,
                fontSize: 13,
                color: "var(--color-text-muted)",
              }}
            >
              <span>
                {data.total} entr{data.total === 1 ? "y" : "ies"}
              </span>
              <div style={{ display: "flex", gap: 8 }}>
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1}
                  style={{
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-default)",
                    borderRadius: 5,
                    padding: "4px 12px",
                    fontSize: 12,
                    color: "var(--color-text-secondary)",
                    cursor: page === 1 ? "not-allowed" : "pointer",
                    opacity: page === 1 ? 0.4 : 1,
                  }}
                >
                  Previous
                </button>
                <span style={{ padding: "4px 8px", fontSize: 12 }}>
                  Page {page} of {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                  style={{
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-default)",
                    borderRadius: 5,
                    padding: "4px 12px",
                    fontSize: 12,
                    color: "var(--color-text-secondary)",
                    cursor: page === totalPages ? "not-allowed" : "pointer",
                    opacity: page === totalPages ? 0.4 : 1,
                  }}
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
