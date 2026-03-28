import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { useMovies } from "@/api/movies";
import { Poster } from "@/components/Poster";
import type { Movie } from "@/types";

// ── Helpers ───────────────────────────────────────────────────────────────────

function movieBorderColor(movie: Movie): string {
  if (!movie.monitored) return "var(--color-border-default)";
  if (movie.status === "downloaded") return "var(--color-success)";
  return "var(--color-warning)";
}

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate();
}

const MONTH_NAMES = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];
const DAY_NAMES = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

// ── Movie chip ────────────────────────────────────────────────────────────────

function MovieChip({ movie, onClick }: { movie: Movie; onClick: () => void }) {
  const border = movieBorderColor(movie);

  return (
    <button
      onClick={onClick}
      title={movie.title}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 4,
        width: "100%",
        background: "var(--color-bg-surface)",
        border: `1px solid ${border}`,
        borderLeft: `3px solid ${border}`,
        borderRadius: 4,
        padding: "2px 4px",
        cursor: "pointer",
        textAlign: "left",
        minWidth: 0,
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)";
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-surface)";
      }}
    >
      <Poster
        src={movie.poster_url}
        title={movie.title}
        style={{ width: 14, height: 20, borderRadius: 2, flexShrink: 0, padding: 0, fontSize: 0 }}
        imgStyle={{ width: 14, height: 20, borderRadius: 2 }}
      />
      <span
        style={{
          fontSize: 10,
          color: "var(--color-text-primary)",
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
          lineHeight: 1.3,
        }}
      >
        {movie.title}
      </span>
    </button>
  );
}

// ── Day cell ──────────────────────────────────────────────────────────────────

function DayCell({
  date,
  movies,
  isToday,
  isCurrentMonth,
  onMovieClick,
}: {
  date: Date;
  movies: Movie[];
  isToday: boolean;
  isCurrentMonth: boolean;
  onMovieClick: (id: string) => void;
}) {
  return (
    <div
      style={{
        minHeight: 90,
        background: isCurrentMonth ? "var(--color-bg-surface)" : "transparent",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 4,
        padding: "4px 5px",
        display: "flex",
        flexDirection: "column",
        gap: 3,
      }}
    >
      {/* Day number */}
      <div
        style={{
          fontSize: 11,
          fontWeight: isToday ? 700 : 400,
          color: isToday
            ? "var(--color-accent)"
            : isCurrentMonth
            ? "var(--color-text-secondary)"
            : "var(--color-text-muted)",
          lineHeight: 1,
          ...(isToday && {
            display: "inline-flex",
            alignItems: "center",
            justifyContent: "center",
            width: 18,
            height: 18,
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            borderRadius: "50%",
          }),
        }}
      >
        {date.getDate()}
      </div>

      {/* Movie chips */}
      {movies.slice(0, 4).map((m) => (
        <MovieChip key={m.id} movie={m} onClick={() => onMovieClick(m.id)} />
      ))}
      {movies.length > 4 && (
        <span style={{ fontSize: 9, color: "var(--color-text-muted)", paddingLeft: 2 }}>
          +{movies.length - 4} more
        </span>
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function CalendarPage() {
  const navigate = useNavigate();

  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth()); // 0-indexed

  // Fetch all movies for client-side filtering by release date.
  const { data, isLoading } = useMovies({ per_page: 10000, page: 1 });
  const allMovies = data?.movies ?? [];
  const totalMovies = data?.total ?? 0;
  const truncated = allMovies.length < totalMovies;

  // Build a map from "YYYY-MM-DD" string → movies releasing that day.
  const moviesByDate = useMemo(() => {
    const map = new Map<string, Movie[]>();
    for (const m of allMovies) {
      if (!m.release_date) continue;
      const key = m.release_date.slice(0, 10); // YYYY-MM-DD
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(m);
    }
    return map;
  }, [allMovies]);

  // Calendar grid: first day of the month, padding to Sunday start.
  const firstDay = new Date(year, month, 1);
  const lastDay = new Date(year, month + 1, 0);
  const startPad = firstDay.getDay(); // 0=Sun ... 6=Sat
  const totalCells = startPad + lastDay.getDate();
  const rows = Math.ceil(totalCells / 7);

  const today = new Date();

  function prevMonth() {
    if (month === 0) { setYear(y => y - 1); setMonth(11); }
    else setMonth(m => m - 1);
  }

  function nextMonth() {
    if (month === 11) { setYear(y => y + 1); setMonth(0); }
    else setMonth(m => m + 1);
  }

  function goToday() {
    setYear(now.getFullYear());
    setMonth(now.getMonth());
  }

  return (
    <div style={{ padding: 24, maxWidth: 1100 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 20 }}>
        <h1 style={{ margin: 0, fontSize: 20, fontWeight: 700, color: "var(--color-text-primary)", letterSpacing: "-0.02em", flex: 1 }}>
          Calendar
        </h1>

        <button
          onClick={goToday}
          style={{
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "5px 12px",
            fontSize: 12,
            color: "var(--color-text-secondary)",
            cursor: "pointer",
          }}
        >
          Today
        </button>

        <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
          <button
            onClick={prevMonth}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "5px 10px",
              fontSize: 14,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
              lineHeight: 1,
            }}
          >
            ‹
          </button>

          <span style={{ fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)", minWidth: 140, textAlign: "center" }}>
            {MONTH_NAMES[month]} {year}
          </span>

          <button
            onClick={nextMonth}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "5px 10px",
              fontSize: 14,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
              lineHeight: 1,
            }}
          >
            ›
          </button>
        </div>
      </div>

      {/* Legend */}
      <div style={{ display: "flex", gap: 16, marginBottom: 14, alignItems: "center" }}>
        {[
          { color: "var(--color-success)", label: "Downloaded" },
          { color: "var(--color-warning)", label: "Monitored" },
          { color: "var(--color-border-default)", label: "Unmonitored" },
        ].map(({ color, label }) => (
          <div key={label} style={{ display: "flex", alignItems: "center", gap: 5 }}>
            <div style={{ width: 12, height: 12, borderRadius: 2, background: color, flexShrink: 0 }} />
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{label}</span>
          </div>
        ))}
        {isLoading && (
          <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginLeft: "auto" }}>Loading…</span>
        )}
        {truncated && (
          <span style={{ fontSize: 11, color: "var(--color-warning)", marginLeft: "auto" }}>
            Showing {allMovies.length} of {totalMovies} movies
          </span>
        )}
      </div>

      {/* Day-of-week headers */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(7, 1fr)", gap: 4, marginBottom: 4 }}>
        {DAY_NAMES.map((d) => (
          <div
            key={d}
            style={{ fontSize: 11, fontWeight: 600, color: "var(--color-text-muted)", textAlign: "center", padding: "4px 0", textTransform: "uppercase", letterSpacing: "0.05em" }}
          >
            {d}
          </div>
        ))}
      </div>

      {/* Calendar grid */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(7, 1fr)", gap: 4 }}>
        {Array.from({ length: rows * 7 }, (_, i) => {
          const dayNum = i - startPad + 1;
          const isCurrentMonth = dayNum >= 1 && dayNum <= lastDay.getDate();
          const cellDate = new Date(year, month, dayNum);
          const dateKey = `${cellDate.getFullYear()}-${String(cellDate.getMonth() + 1).padStart(2, "0")}-${String(cellDate.getDate()).padStart(2, "0")}`;
          const cellMovies = isCurrentMonth ? (moviesByDate.get(dateKey) ?? []) : [];
          const isToday = isCurrentMonth && isSameDay(cellDate, today);

          return (
            <DayCell
              key={i}
              date={cellDate}
              movies={cellMovies}
              isToday={isToday}
              isCurrentMonth={isCurrentMonth}
              onMovieClick={(id) => navigate(`/movies/${id}`)}
            />
          );
        })}
      </div>
    </div>
  );
}
