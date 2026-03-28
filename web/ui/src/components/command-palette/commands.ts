import type { ElementType } from "react";
import type { NavigateFunction } from "react-router-dom";
import {
  Activity,
  LayoutDashboard,
  CalendarDays,
  Bookmark,
  RefreshCw,
  BarChart2,
  Download,
  History,
  Library,
  Film,
  ScanLine,
  SlidersHorizontal,
  Gauge,
  Search,
  Settings2,
  Bell,
  MonitorPlay,
  Ban,
  ArrowDownToLine,
  Server,
  Paintbrush,
  Layers,
  Rss,
  HardDrive,
  RotateCw,
} from "lucide-react";

// ── Types ────────────────────────────────────────────────────────────────────

export type CommandCategory = "navigation" | "movie" | "action";

export interface Command {
  id: string;
  category: CommandCategory;
  label: string;
  keywords: string[];
  icon: ElementType;
  onSelect: (navigate: NavigateFunction) => void;
}

// ── Navigation commands ──────────────────────────────────────────────────────

export const NAV_COMMANDS: Command[] = [
  { id: "nav:dashboard",        category: "navigation", label: "Dashboard",            keywords: ["home", "overview"],       icon: LayoutDashboard, onSelect: (n) => n("/") },
  { id: "nav:activity",         category: "navigation", label: "Activity",             keywords: ["events", "timeline", "log"], icon: Activity,     onSelect: (n) => n("/activity") },
  { id: "nav:discover",         category: "navigation", label: "Discover",             keywords: ["browse", "trending", "popular", "genre"], icon: Search, onSelect: (n) => n("/discover") },
  { id: "nav:calendar",         category: "navigation", label: "Calendar",             keywords: ["schedule", "upcoming"],   icon: CalendarDays,    onSelect: (n) => n("/calendar") },
  { id: "nav:wanted",           category: "navigation", label: "Wanted",               keywords: ["missing", "cutoff"],      icon: Bookmark,        onSelect: (n) => n("/wanted") },
  { id: "nav:library-sync",     category: "navigation", label: "Library Sync",         keywords: ["plex", "sync"],           icon: RefreshCw,       onSelect: (n) => n("/library-sync") },
  { id: "nav:stats",            category: "navigation", label: "Statistics",            keywords: ["graphs", "charts"],       icon: BarChart2,       onSelect: (n) => n("/stats") },
  { id: "nav:queue",            category: "navigation", label: "Queue",                keywords: ["downloads", "progress"],  icon: Download,        onSelect: (n) => n("/queue") },
  { id: "nav:history",          category: "navigation", label: "History",              keywords: ["grabs", "past"],          icon: History,         onSelect: (n) => n("/history") },
  { id: "nav:collections",      category: "navigation", label: "Collections",          keywords: ["director", "actor"],      icon: Layers,          onSelect: (n) => n("/collections") },

  // Settings
  { id: "nav:libraries",         category: "navigation", label: "Libraries",            keywords: ["settings", "root", "path"],         icon: Library,            onSelect: (n) => n("/settings/libraries") },
  { id: "nav:media-management",  category: "navigation", label: "Media Management",     keywords: ["settings", "rename", "format"],      icon: Film,               onSelect: (n) => n("/settings/media-management") },
  { id: "nav:media-scanning",    category: "navigation", label: "Media Scanning",       keywords: ["settings", "scan"],                  icon: ScanLine,           onSelect: (n) => n("/settings/media-scanning") },
  { id: "nav:quality-profiles",  category: "navigation", label: "Quality Profiles",     keywords: ["settings", "quality"],               icon: SlidersHorizontal,  onSelect: (n) => n("/settings/quality-profiles") },
  { id: "nav:quality-defs",      category: "navigation", label: "Quality Definitions",  keywords: ["settings", "size", "limits"],        icon: Gauge,              onSelect: (n) => n("/settings/quality-definitions") },
  { id: "nav:indexers",          category: "navigation", label: "Indexers",             keywords: ["settings", "torznab", "newznab"],    icon: Search,             onSelect: (n) => n("/settings/indexers") },
  { id: "nav:download-clients",  category: "navigation", label: "Download Clients",    keywords: ["settings", "qbittorrent", "deluge"], icon: Settings2,          onSelect: (n) => n("/settings/download-clients") },
  { id: "nav:notifications",     category: "navigation", label: "Notifications",        keywords: ["settings", "discord", "webhook"],    icon: Bell,               onSelect: (n) => n("/settings/notifications") },
  { id: "nav:media-servers",     category: "navigation", label: "Media Servers",        keywords: ["settings", "plex", "jellyfin"],      icon: MonitorPlay,        onSelect: (n) => n("/settings/media-servers") },
  { id: "nav:blocklist",         category: "navigation", label: "Blocklist",            keywords: ["settings", "block", "ban"],          icon: Ban,                onSelect: (n) => n("/settings/blocklist") },
  { id: "nav:import",            category: "navigation", label: "Import",               keywords: ["settings", "radarr"],                icon: ArrowDownToLine,    onSelect: (n) => n("/settings/import") },
  { id: "nav:system",            category: "navigation", label: "System",               keywords: ["settings", "health", "tasks"],       icon: Server,             onSelect: (n) => n("/settings/system") },
  { id: "nav:app-settings",      category: "navigation", label: "App Settings",         keywords: ["settings", "theme", "api key"],      icon: Paintbrush,         onSelect: (n) => n("/settings/app") },
];

// ── Action commands ──────────────────────────────────────────────────────────
// NOTE: actions need their taskName resolved at runtime via useRunTask.
// The onSelect here is a no-op; the component maps action IDs to mutations.

export interface ActionCommand extends Command {
  taskName: string;
}

export const ACTION_COMMANDS: ActionCommand[] = [
  { id: "action:rss-sync",         category: "action", label: "Run RSS Sync",           keywords: ["rss", "feed", "check"],    icon: Rss,        taskName: "rss_sync",         onSelect: () => {} },
  { id: "action:library-scan",     category: "action", label: "Scan All Libraries",     keywords: ["scan", "disk", "files"],   icon: HardDrive,  taskName: "library_scan",     onSelect: () => {} },
  { id: "action:refresh-metadata", category: "action", label: "Refresh All Metadata",   keywords: ["tmdb", "refresh", "sync"], icon: RotateCw,   taskName: "refresh_metadata", onSelect: () => {} },
];

// ── Fuzzy matching ──────────────────────────────────────────────────────────

/**
 * Fuzzy match `query` against `text`. All query characters must appear in
 * order within text. Returns -1 for no match, otherwise a non-negative score
 * (higher = better). Bonuses for consecutive runs and word-boundary matches.
 */
export function fuzzyScore(query: string, text: string): number {
  const q = query.toLowerCase();
  const t = text.toLowerCase();

  if (!q) return 0;
  if (q.length > t.length) return -1;

  let qi = 0;
  let score = 0;
  let consecutive = 0;

  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) {
      qi++;
      consecutive++;
      score += consecutive;
      if (ti === 0 || /[\s\-_]/.test(t[ti - 1])) score += 10;
    } else {
      consecutive = 0;
    }
  }

  return qi === q.length ? score : -1;
}

// ── Filter helper ────────────────────────────────────────────────────────────

export function filterCommands<T extends Command>(commands: T[], query: string): T[] {
  if (!query) return commands;

  const scored: Array<{ cmd: T; score: number }> = [];

  for (const cmd of commands) {
    const labelScore = fuzzyScore(query, cmd.label);
    // Label matches get a +50 bonus so they rank above keyword-only matches
    const best = labelScore >= 0 ? labelScore + 50 : -1;

    let kwBest = -1;
    for (const kw of cmd.keywords) {
      const s = fuzzyScore(query, kw);
      if (s > kwBest) kwBest = s;
    }

    const score = Math.max(best, kwBest);
    if (score >= 0) scored.push({ cmd, score });
  }

  scored.sort((a, b) => b.score - a.score);
  return scored.map((s) => s.cmd);
}
