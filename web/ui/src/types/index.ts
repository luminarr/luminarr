// ── System ─────────────────────────────────────────────────────────────────

export interface SystemStatus {
  app_name: string;
  version: string;
  build_time: string;
  go_version: string;
  db_type: string;
  db_path: string;
  uptime_seconds: number;
  start_time: string;
  ai_enabled: boolean;
  tmdb_enabled: boolean;
}

export type HealthStatus = "healthy" | "degraded" | "unhealthy";

export interface HealthItem {
  name: string;
  status: HealthStatus;
  message: string;
}

export interface HealthReport {
  status: HealthStatus;
  checks: HealthItem[];
}

export interface Task {
  name: string;
  interval: string;
}

export interface LogEntry {
  time: string;
  level: string;
  message: string;
  fields: Record<string, unknown>;
}

export interface PluginList {
  indexers: string[];
  downloaders: string[];
  notifications: string[];
}

// ── Movies ─────────────────────────────────────────────────────────────────

export interface Movie {
  id: string;
  tmdb_id: number;
  imdb_id?: string;
  title: string;
  original_title: string;
  year: number;
  overview: string;
  runtime_minutes: number;
  genres: string[];
  poster_url?: string;
  fanart_url?: string;
  status: string;
  monitored: boolean;
  library_id: string;
  quality_profile_id: string;
  path?: string;
  added_at: string;
  updated_at: string;
  metadata_refreshed_at?: string;
}

export interface MovieListResponse {
  movies: Movie[];
  total: number;
  page: number;
  per_page: number;
}

export interface TMDBResult {
  tmdb_id: number;
  title: string;
  original_title: string;
  overview: string;
  release_date: string;
  year: number;
  poster_path?: string;
  backdrop_path?: string;
  popularity: number;
}

export interface Release {
  guid: string;
  title: string;
  indexer: string;
  protocol: string;
  download_url: string;
  info_url?: string;
  size: number;
  seeds?: number;
  peers?: number;
  age_days?: number;
  quality: Quality;
  quality_score: number;
}

export interface GrabHistory {
  id: string;
  movie_id: string;
  indexer_id?: string;
  release_guid: string;
  release_title: string;
  release_source?: string;
  release_resolution?: string;
  protocol: string;
  size: number;
  download_client_id?: string;
  client_item_id?: string;
  download_status: string;
  grabbed_at: string;
}

// ── Queue ──────────────────────────────────────────────────────────────────

export interface QueueItem {
  id: string;
  movie_id: string;
  release_title: string;
  protocol: string;
  size: number;
  downloaded_bytes: number;
  status: string;
  download_client_id?: string;
  client_item_id?: string;
  grabbed_at: string;
}

// ── Libraries ──────────────────────────────────────────────────────────────

export interface Library {
  id: string;
  name: string;
  root_path: string;
  default_quality_profile_id?: string;
  min_free_space_gb: number;
  naming_format?: string;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface LibraryStats {
  movie_count: number;
  total_size_bytes: number;
  free_space_bytes: number;
  health_ok: boolean;
  health_message: string;
}

export interface LibraryRequest {
  name: string;
  root_path: string;
  default_quality_profile_id?: string;
  naming_format?: string;
  min_free_space_gb?: number;
  tags?: string[];
}

// ── Library disk import ────────────────────────────────────────────────────

export interface DiskFile {
  path: string;
  size_bytes: number;
  parsed_title: string;
  parsed_year: number;
}

// ── Quality Profiles ───────────────────────────────────────────────────────

export interface Quality {
  resolution: string;
  source: string;
  codec: string;
  hdr: string;
  name: string;
}

export interface QualityProfile {
  id: string;
  name: string;
  cutoff: Quality;
  qualities: Quality[];
  upgrade_allowed: boolean;
  upgrade_until?: Quality;
}

export interface QualityProfileRequest {
  name: string;
  cutoff: Quality;
  qualities: Quality[];
  upgrade_allowed: boolean;
  upgrade_until?: Quality;
}

// ── Indexers ───────────────────────────────────────────────────────────────

export interface IndexerConfig {
  id: string;
  name: string;
  kind: string;
  enabled: boolean;
  priority: number;
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface IndexerRequest {
  name: string;
  kind: string;
  enabled?: boolean;
  priority?: number;
  settings: Record<string, unknown>;
}

// ── Download Clients ───────────────────────────────────────────────────────

export interface DownloadClientConfig {
  id: string;
  name: string;
  kind: string;
  enabled: boolean;
  priority: number;
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface DownloadClientRequest {
  name: string;
  kind: string;
  enabled?: boolean;
  priority?: number;
  settings: Record<string, unknown>;
}

// ── Notifications ──────────────────────────────────────────────────────────

export interface NotificationConfig {
  id: string;
  name: string;
  kind: string;
  enabled: boolean;
  on_events: string[];
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface NotificationRequest {
  name: string;
  kind: string;
  enabled: boolean;
  settings: Record<string, unknown>;
  on_events?: string[];
}

// ── Test result ────────────────────────────────────────────────────────────

export interface TestResult {
  ok: boolean;
  message?: string;
}

// ── Radarr import ──────────────────────────────────────────────────────────

export interface RadarrPreviewResult {
  version: string;
  movie_count: number;
  quality_profiles: { id: number; name: string }[];
  root_folders: { path: string; free_space_gb: number }[];
  indexers: { id: number; name: string; kind: string }[];
  download_clients: { id: number; name: string; kind: string }[];
}

export interface RadarrImportOptions {
  quality_profiles: boolean;
  libraries: boolean;
  indexers: boolean;
  download_clients: boolean;
  movies: boolean;
}

export interface CategoryResult {
  imported: number;
  skipped: number;
  failed: number;
}

export interface RadarrImportResult {
  quality_profiles: CategoryResult;
  libraries: CategoryResult;
  indexers: CategoryResult;
  download_clients: CategoryResult;
  movies: CategoryResult;
  errors: string[];
}
