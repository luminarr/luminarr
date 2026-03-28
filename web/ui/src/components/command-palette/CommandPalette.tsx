import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { Poster } from "@/components/Poster";
import { toast } from "sonner";
import { useLookupMovies } from "@/api/movies";
import { useRunTask, useSystemStatus } from "@/api/system";
import { useSendAICommand, useConfirmAIAction, type AICommandResponse } from "@/api/ai";
import {
  NAV_COMMANDS,
  ACTION_COMMANDS,
  filterCommands,
  type Command,
  type ActionCommand,
} from "./commands";
import type { Movie, MovieListResponse, TMDBResult } from "@/types";
import { Film, Sparkles, Settings, Check, X } from "lucide-react";

// ── Types ────────────────────────────────────────────────────────────────────

interface PaletteItem {
  id: string;
  category: "navigation" | "movie" | "action" | "ai";
  label: string;
  subtitle?: string;
  icon?: React.ElementType;
  posterUrl?: string;
  inLibrary?: boolean;
  onSelect: () => void;
}

interface CommandPaletteProps {
  onClose: () => void;
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function getCachedMovies(qc: ReturnType<typeof useQueryClient>): Map<number, Movie> {
  const map = new Map<number, Movie>();
  const cache = qc.getQueriesData<MovieListResponse>({ queryKey: ["movies"] });
  for (const [, data] of cache) {
    if (data?.movies) {
      for (const m of data.movies) {
        map.set(m.tmdb_id, m);
      }
    }
  }
  return map;
}

// ── Component ────────────────────────────────────────────────────────────────

export function CommandPalette({ onClose }: CommandPaletteProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const lookup = useLookupMovies();
  const runTask = useRunTask();
  const aiCommand = useSendAICommand();
  const confirmAction = useConfirmAIAction();
  const { data: status } = useSystemStatus();

  const aiEnabled = status?.ai_enabled ?? false;

  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [aiResponse, setAiResponse] = useState<AICommandResponse | null>(null);
  const [confirmResult, setConfirmResult] = useState<AICommandResponse | null>(null);

  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const previousFocus = useRef<Element | null>(null);

  // Capture previous focus and lock body scroll
  useEffect(() => {
    previousFocus.current = document.activeElement;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = "";
      if (previousFocus.current instanceof HTMLElement) {
        previousFocus.current.focus();
      }
    };
  }, []);

  // Debounce query for movie search
  useEffect(() => {
    if (query.length < 2) {
      setDebouncedQuery("");
      return;
    }
    const timer = setTimeout(() => setDebouncedQuery(query), 300);
    return () => clearTimeout(timer);
  }, [query]);

  // Fire movie lookup when debounced query changes
  useEffect(() => {
    if (debouncedQuery) {
      lookup.mutate({ query: debouncedQuery });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedQuery]);

  // Reset state when query changes
  useEffect(() => {
    setActiveIndex(0);
    setAiResponse(null);
    setConfirmResult(null);
  }, [query]);

  // ── AI command handler ─────────────────────────────────────────────────────

  const handleAICommand = useCallback(() => {
    if (!query.trim()) return;

    aiCommand.mutate(query.trim(), {
      onSuccess: (resp) => {
        setAiResponse(resp);

        // If the action requires confirmation, show the confirm UI — don't execute yet.
        if (resp.requires_confirmation) return;

        switch (resp.action) {
          case "navigate":
            if (resp.params?.path) {
              navigate(resp.params.path as string);
              onClose();
            }
            break;
          case "search_movie":
            if (resp.params?.query) {
              lookup.mutate({ query: resp.params.query as string });
            }
            break;
          // query_library, explain, fallback — show inline response
        }
      },
      onError: (err) => {
        toast.error((err as Error).message);
      },
    });
  }, [query, aiCommand, navigate, onClose, lookup]);

  const handleConfirm = useCallback(() => {
    if (!aiResponse?.pending_action_id) return;

    confirmAction.mutate(aiResponse.pending_action_id, {
      onSuccess: (resp) => {
        setConfirmResult(resp);
        toast.success(resp.explanation);
      },
      onError: (err) => {
        toast.error((err as Error).message);
      },
    });
  }, [aiResponse, confirmAction]);

  const handleDismiss = useCallback(() => {
    setAiResponse(null);
    setConfirmResult(null);
  }, []);

  // ── Build flat item list ───────────────────────────────────────────────────

  const handleAction = useCallback(
    (cmd: ActionCommand) => {
      runTask.mutate(cmd.taskName, {
        onSuccess: () => toast.success(`${cmd.label} triggered`),
      });
      onClose();
    },
    [runTask, onClose],
  );

  const handleNav = useCallback(
    (cmd: Command) => {
      cmd.onSelect(navigate);
      onClose();
    },
    [navigate, onClose],
  );

  const handleMovie = useCallback(
    (_movie: TMDBResult, libraryMovie: Movie | undefined) => {
      if (libraryMovie) {
        navigate(`/movies/${libraryMovie.id}`);
      } else {
        navigate("/");
      }
      onClose();
    },
    [navigate, onClose],
  );

  // Build items
  const filteredNav = filterCommands(NAV_COMMANDS, query);
  const filteredActions = filterCommands(ACTION_COMMANDS, query);
  const cachedMovies = getCachedMovies(queryClient);
  const movieResults: TMDBResult[] =
    query.length >= 2 && lookup.data ? lookup.data : [];

  const items: PaletteItem[] = [];

  // Navigation
  for (const cmd of filteredNav) {
    items.push({
      id: cmd.id,
      category: "navigation",
      label: cmd.label,
      icon: cmd.icon,
      onSelect: () => handleNav(cmd),
    });
  }

  // Movies
  for (const movie of movieResults) {
    const libraryMovie = cachedMovies.get(movie.tmdb_id);
    items.push({
      id: `movie:${movie.tmdb_id}`,
      category: "movie",
      label: movie.title,
      subtitle: movie.year ? String(movie.year) : undefined,
      posterUrl: movie.poster_path
        ? `https://image.tmdb.org/t/p/w92${movie.poster_path}`
        : undefined,
      inLibrary: !!libraryMovie,
      onSelect: () => handleMovie(movie, libraryMovie),
    });
  }

  // Actions
  for (const cmd of filteredActions) {
    items.push({
      id: cmd.id,
      category: "action",
      label: cmd.label,
      icon: cmd.icon,
      onSelect: () => handleAction(cmd),
    });
  }

  // "Ask AI" option — shown when there's a query and AI is enabled
  if (query.length >= 2 && aiEnabled) {
    items.push({
      id: "ai:ask",
      category: "ai",
      label: `Ask AI: "${query}"`,
      icon: Sparkles,
      onSelect: handleAICommand,
    });
  }

  // "Set up AI" option — shown when there's a query but AI is not enabled
  if (query.length >= 2 && !aiEnabled) {
    items.push({
      id: "ai:setup",
      category: "ai",
      label: "Set up AI in Settings > App",
      icon: Settings,
      onSelect: () => {
        navigate("/settings/app");
        onClose();
      },
    });
  }

  // ── Keyboard handling ──────────────────────────────────────────────────────

  function onKeyDown(e: React.KeyboardEvent) {
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, items.length - 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
        break;
      case "Enter":
        e.preventDefault();
        if (items[activeIndex]) {
          items[activeIndex].onSelect();
        }
        break;
      case "Escape":
        e.preventDefault();
        onClose();
        break;
    }
  }

  // Scroll active item into view
  useEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const active = list.querySelector(`[data-index="${activeIndex}"]`);
    if (active) {
      active.scrollIntoView({ block: "nearest" });
    }
  }, [activeIndex]);

  // ── Grouped rendering ─────────────────────────────────────────────────────

  const navItems = items.filter((i) => i.category === "navigation");
  const movieItems = items.filter((i) => i.category === "movie");
  const actionItems = items.filter((i) => i.category === "action");
  const aiItems = items.filter((i) => i.category === "ai");

  // Track global index for each item
  let globalIndex = 0;
  function nextIndex() {
    return globalIndex++;
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.5)",
        backdropFilter: "blur(2px)",
        zIndex: 300,
        display: "flex",
        alignItems: "flex-start",
        justifyContent: "center",
        paddingTop: "20vh",
      }}
      onClick={onClose}
      data-testid="command-palette-backdrop"
    >
      <div
        style={{
          background: "var(--color-bg-surface)",
          border: "1px solid var(--color-border-default)",
          borderRadius: 12,
          width: 560,
          maxWidth: "calc(100vw - 32px)",
          maxHeight: "min(480px, 60vh)",
          boxShadow: "var(--shadow-modal)",
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label="Command palette"
      >
        {/* Input */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            borderBottom: "1px solid var(--color-border-subtle)",
          }}
        >
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Type a command or search..."
            autoFocus
            style={{
              flex: 1,
              padding: "14px 16px",
              background: "transparent",
              border: "none",
              outline: "none",
              fontSize: 15,
              color: "var(--color-text-primary)",
            }}
            data-testid="command-palette-input"
          />
          <kbd
            style={{
              marginRight: 12,
              fontSize: 10,
              padding: "2px 6px",
              borderRadius: 4,
              background: "var(--color-bg-elevated)",
              color: "var(--color-text-muted)",
              border: "1px solid var(--color-border-subtle)",
              flexShrink: 0,
            }}
          >
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div
          ref={listRef}
          style={{
            flex: 1,
            overflowY: "auto",
            padding: "8px 0",
          }}
          data-testid="command-palette-list"
        >
          {items.length === 0 && (
            <div
              style={{
                textAlign: "center",
                padding: "32px 16px",
                color: "var(--color-text-muted)",
                fontSize: 13,
              }}
            >
              {query.length >= 2 && lookup.isPending
                ? "Searching..."
                : "No results"}
            </div>
          )}

          {navItems.length > 0 && (
            <PaletteGroup label="Pages">
              {navItems.map((item) => {
                const idx = nextIndex();
                return (
                  <PaletteRow
                    key={item.id}
                    item={item}
                    index={idx}
                    isActive={idx === activeIndex}
                    onHover={setActiveIndex}
                  />
                );
              })}
            </PaletteGroup>
          )}

          {movieItems.length > 0 && (
            <PaletteGroup label="Movies">
              {movieItems.map((item) => {
                const idx = nextIndex();
                return (
                  <PaletteRow
                    key={item.id}
                    item={item}
                    index={idx}
                    isActive={idx === activeIndex}
                    onHover={setActiveIndex}
                  />
                );
              })}
            </PaletteGroup>
          )}

          {query.length >= 2 && lookup.isPending && movieItems.length === 0 && (
            <PaletteGroup label="Movies">
              {[1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="skeleton"
                  style={{ height: 36, margin: "0 8px 4px", borderRadius: 6 }}
                />
              ))}
            </PaletteGroup>
          )}

          {actionItems.length > 0 && (
            <PaletteGroup label="Actions">
              {actionItems.map((item) => {
                const idx = nextIndex();
                return (
                  <PaletteRow
                    key={item.id}
                    item={item}
                    index={idx}
                    isActive={idx === activeIndex}
                    onHover={setActiveIndex}
                  />
                );
              })}
            </PaletteGroup>
          )}

          {aiItems.length > 0 && (
            <PaletteGroup label="AI">
              {aiItems.map((item) => {
                const idx = nextIndex();
                return (
                  <PaletteRow
                    key={item.id}
                    item={item}
                    index={idx}
                    isActive={idx === activeIndex}
                    onHover={setActiveIndex}
                  />
                );
              })}
            </PaletteGroup>
          )}

          {/* AI loading state */}
          {aiCommand.isPending && (
            <PaletteGroup label="AI">
              <div
                style={{
                  padding: "12px 16px",
                  fontSize: 13,
                  color: "var(--color-text-muted)",
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                }}
              >
                <Sparkles size={14} style={{ animation: "pulse 1.5s ease-in-out infinite" }} />
                Thinking...
              </div>
            </PaletteGroup>
          )}

          {/* AI confirmation prompt */}
          {aiResponse?.requires_confirmation && !confirmResult && !confirmAction.isPending && (
            <AIConfirmPrompt
              response={aiResponse}
              onConfirm={handleConfirm}
              onDismiss={handleDismiss}
            />
          )}

          {/* AI confirm loading */}
          {confirmAction.isPending && (
            <PaletteGroup label="AI">
              <div
                style={{
                  padding: "12px 16px",
                  fontSize: 13,
                  color: "var(--color-text-muted)",
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                }}
              >
                <Sparkles size={14} style={{ animation: "pulse 1.5s ease-in-out infinite" }} />
                Executing...
              </div>
            </PaletteGroup>
          )}

          {/* AI confirm result */}
          {confirmResult && (
            <AIResponseDisplay response={confirmResult} />
          )}

          {/* AI response display (non-confirmation) */}
          {aiResponse && !aiResponse.requires_confirmation && !aiCommand.isPending && (
            <AIResponseDisplay response={aiResponse} />
          )}
        </div>
      </div>
    </div>
  );
}

// ── Sub-components ───────────────────────────────────────────────────────────

function AIConfirmPrompt({
  response,
  onConfirm,
  onDismiss,
}: {
  response: AICommandResponse;
  onConfirm: () => void;
  onDismiss: () => void;
}) {
  const message = response.confirmation_message || response.explanation;

  return (
    <PaletteGroup label="Confirm Action">
      <div
        style={{
          padding: "10px 16px",
          fontSize: 13,
          color: "var(--color-text-primary)",
          lineHeight: 1.5,
        }}
      >
        {message}
      </div>
      <div
        style={{
          display: "flex",
          gap: 8,
          padding: "4px 16px 12px",
        }}
      >
        <button
          onClick={onConfirm}
          data-testid="ai-confirm-btn"
          style={{
            display: "flex",
            alignItems: "center",
            gap: 6,
            padding: "6px 14px",
            borderRadius: 6,
            border: "none",
            background: "var(--color-accent)",
            color: "var(--color-accent-fg)",
            fontSize: 13,
            fontWeight: 500,
            cursor: "pointer",
          }}
        >
          <Check size={14} />
          Confirm
        </button>
        <button
          onClick={onDismiss}
          data-testid="ai-dismiss-btn"
          style={{
            display: "flex",
            alignItems: "center",
            gap: 6,
            padding: "6px 14px",
            borderRadius: 6,
            border: "1px solid var(--color-border-default)",
            background: "transparent",
            color: "var(--color-text-secondary)",
            fontSize: 13,
            cursor: "pointer",
          }}
        >
          <X size={14} />
          Cancel
        </button>
      </div>
    </PaletteGroup>
  );
}

function AIResponseDisplay({ response }: { response: AICommandResponse }) {
  // Don't show inline for navigate/search_movie — those actions execute immediately
  if (response.action === "navigate" || response.action === "search_movie") {
    return null;
  }

  const answer =
    (response.action === "query_library" && response.result?.answer)
      ? String(response.result.answer)
      : response.explanation;

  return (
    <PaletteGroup label="AI Response">
      <div
        style={{
          padding: "10px 16px",
          fontSize: 13,
          color: "var(--color-text-primary)",
          lineHeight: 1.5,
          whiteSpace: "pre-wrap",
        }}
        data-testid="ai-response"
      >
        {answer}
      </div>
    </PaletteGroup>
  );
}

function PaletteGroup({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div
        style={{
          padding: "8px 16px 4px",
          fontSize: 11,
          fontWeight: 600,
          letterSpacing: "0.08em",
          textTransform: "uppercase",
          color: "var(--color-text-muted)",
        }}
      >
        {label}
      </div>
      {children}
    </div>
  );
}

function PaletteRow({
  item,
  index,
  isActive,
  onHover,
}: {
  item: PaletteItem;
  index: number;
  isActive: boolean;
  onHover: (index: number) => void;
}) {
  const Icon = item.icon ?? Film;

  return (
    <button
      data-index={index}
      data-testid={`palette-item-${item.id}`}
      aria-selected={isActive}
      onClick={item.onSelect}
      onMouseEnter={() => onHover(index)}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 16px",
        cursor: "pointer",
        background: isActive ? "var(--color-bg-elevated)" : "transparent",
        border: "none",
        width: "100%",
        textAlign: "left",
        fontSize: 13,
        color: isActive
          ? "var(--color-text-primary)"
          : "var(--color-text-secondary)",
        borderRadius: 0,
      }}
    >
      {item.category === "movie" ? (
        <Poster
          src={item.posterUrl}
          title={item.label}
          style={{ width: 24, height: 36, borderRadius: 3, flexShrink: 0, padding: 0, fontSize: 0 }}
          imgStyle={{ width: 24, height: 36, borderRadius: 3 }}
        />
      ) : (
        <Icon size={16} strokeWidth={1.5} style={{ flexShrink: 0 }} />
      )}
      <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
        {item.label}
      </span>
      {item.subtitle && (
        <span style={{ fontSize: 11, color: "var(--color-text-muted)", flexShrink: 0 }}>
          {item.subtitle}
        </span>
      )}
      {item.inLibrary && (
        <span
          style={{
            fontSize: 10,
            padding: "1px 6px",
            borderRadius: 4,
            background: "color-mix(in srgb, var(--color-success) 15%, transparent)",
            color: "var(--color-success)",
            fontWeight: 600,
            flexShrink: 0,
          }}
        >
          In Library
        </span>
      )}
    </button>
  );
}
