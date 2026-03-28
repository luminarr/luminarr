import { useState } from "react";
import { Link } from "react-router-dom";
import { useActivity, type Activity } from "@/api/activity";
import {
  Download,
  ArrowDownToLine,
  Clock,
  Heart,
  Film,
  AlertCircle,
} from "lucide-react";
import { card, sectionHeader } from "@/lib/styles";

const CATEGORIES = [
  { value: "", label: "All" },
  { value: "grab", label: "Grabs" },
  { value: "import", label: "Imports" },
  { value: "task", label: "Tasks" },
  { value: "health", label: "Health" },
  { value: "movie", label: "Movies" },
] as const;

function categoryIcon(category: string) {
  switch (category) {
    case "grab":
      return Download;
    case "import":
      return ArrowDownToLine;
    case "task":
      return Clock;
    case "health":
      return Heart;
    case "movie":
      return Film;
    default:
      return AlertCircle;
  }
}

function categoryColor(category: string): string {
  switch (category) {
    case "grab":
      return "var(--color-accent)";
    case "import":
      return "var(--color-success)";
    case "task":
      return "var(--color-text-muted)";
    case "health":
      return "var(--color-warning)";
    case "movie":
      return "var(--color-accent)";
    default:
      return "var(--color-text-muted)";
  }
}

function relativeTime(iso: string): string {
  const now = Date.now();
  const then = new Date(iso).getTime();
  const diffSec = Math.floor((now - then) / 1000);

  if (diffSec < 60) return "just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  if (diffSec < 604800) return `${Math.floor(diffSec / 86400)}d ago`;
  return new Date(iso).toLocaleDateString();
}

function ActivityRow({
  activity,
  isLast,
}: {
  activity: Activity;
  isLast: boolean;
}) {
  const Icon = categoryIcon(activity.category);
  const color = categoryColor(activity.category);

  return (
    <div
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: 12,
        padding: "12px 0",
        borderBottom: isLast
          ? "none"
          : "1px solid var(--color-border-subtle)",
      }}
      data-testid={`activity-row-${activity.id}`}
    >
      <div
        style={{
          width: 32,
          height: 32,
          borderRadius: 8,
          background: `color-mix(in srgb, ${color} 12%, transparent)`,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
          marginTop: 2,
        }}
      >
        <Icon size={15} strokeWidth={2} style={{ color }} />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 13,
            color: "var(--color-text-primary)",
            lineHeight: 1.4,
          }}
        >
          {activity.movie_id ? (
            <Link
              to={`/movies/${activity.movie_id}`}
              style={{
                color: "inherit",
                textDecoration: "none",
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLAnchorElement).style.color =
                  "var(--color-accent)";
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLAnchorElement).style.color = "inherit";
              }}
              data-testid={`activity-link-${activity.id}`}
            >
              {activity.title}
            </Link>
          ) : (
            <span data-testid={`activity-text-${activity.id}`}>
              {activity.title}
            </span>
          )}
        </div>
        <div
          style={{
            fontSize: 11,
            color: "var(--color-text-muted)",
            marginTop: 2,
          }}
        >
          {relativeTime(activity.created_at)}
        </div>
      </div>
    </div>
  );
}

export default function ActivityPage() {
  const [category, setCategory] = useState("");
  const { data, isLoading, error } = useActivity(
    category ? { category, limit: 100 } : { limit: 100 },
  );

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
          Activity
        </h1>
        <p
          style={{
            fontSize: 13,
            color: "var(--color-text-secondary)",
            margin: 0,
          }}
        >
          Recent events across grabs, imports, tasks, and library changes.
        </p>
      </div>

      {/* Category filter pills */}
      <div
        style={{
          display: "flex",
          gap: 6,
          marginBottom: 20,
          flexWrap: "wrap",
        }}
      >
        {CATEGORIES.map((cat) => {
          const active = category === cat.value;
          return (
            <button
              key={cat.value}
              onClick={() => setCategory(cat.value)}
              data-testid={`filter-${cat.value || "all"}`}
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
              {cat.label}
            </button>
          );
        })}
      </div>

      <div style={card}>
        <p style={{ ...sectionHeader, marginBottom: 12 }}>
          Timeline
          {data && (
            <span
              style={{
                fontWeight: 400,
                fontSize: 12,
                color: "var(--color-text-muted)",
                marginLeft: 8,
              }}
            >
              {data.total} events
            </span>
          )}
        </p>

        {isLoading && (
          <div>
            {[1, 2, 3, 4, 5].map((i) => (
              <div
                key={i}
                className="skeleton"
                style={{
                  height: 48,
                  borderRadius: 6,
                  marginBottom: 8,
                }}
              />
            ))}
          </div>
        )}

        {error && (
          <p
            style={{
              fontSize: 13,
              color: "var(--color-danger)",
              margin: 0,
            }}
          >
            Failed to load activity.
          </p>
        )}

        {data && data.activities.length === 0 && (
          <p
            style={{
              fontSize: 13,
              color: "var(--color-text-muted)",
              margin: 0,
              padding: "24px 0",
              textAlign: "center",
            }}
            data-testid="empty-state"
          >
            No recent activity
          </p>
        )}

        {data &&
          data.activities.map((a, i) => (
            <ActivityRow
              key={a.id}
              activity={a}
              isLast={i === data.activities.length - 1}
            />
          ))}
      </div>
    </div>
  );
}
