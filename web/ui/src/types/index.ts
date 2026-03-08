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

export interface UpdateCheck {
  update_available: boolean;
  current_version: string;
  latest_version: string;
  release_url?: string;
  release_notes?: string;
  published_at?: string;
}

export interface LogEntry {
  time: string;
  level: string;
  message: string;
  fields?: Record<string, unknown>;
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
  minimum_availability: string;
  release_date?: string;
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

export interface ScoreDimension {
  name: string;
  score: number;
  max: number;
  matched: boolean;
  got: string;
  want: string;
}

export interface ScoreBreakdown {
  total: number;
  dimensions: ScoreDimension[];
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
  score_breakdown?: ScoreBreakdown;
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
  score_breakdown?: ScoreBreakdown;
}

export interface RenamePreviewItem {
  file_id: string;
  old_path: string;
  new_path: string;
}

export interface RenameMovieResult {
  dry_run: boolean;
  renamed: RenamePreviewItem[];
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
  folder_format?: string;
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
  folder_format?: string;
  min_free_space_gb?: number;
  tags?: string[];
}

// ── Download Handling ───────────────────────────────────────────────────────

export interface DownloadHandling {
  enable_completed: boolean;
  check_interval_minutes: number;
  redownload_failed: boolean;
  redownload_failed_interactive: boolean;
}

export interface RemotePathMapping {
  id: string;
  host: string;
  remote_path: string;
  local_path: string;
}

export interface CreateRemotePathMappingRequest {
  host: string;
  remote_path: string;
  local_path: string;
}

// ── Media Management ────────────────────────────────────────────────────────

export interface MediaManagement {
  rename_movies: boolean;
  standard_movie_format: string;
  movie_folder_format: string;
  colon_replacement: "delete" | "dash" | "space-dash" | "smart";
  import_extra_files: boolean;
  extra_file_extensions: string;
  unmonitor_deleted_movies: boolean;
}

// ── Library disk import ────────────────────────────────────────────────────

export interface DiskFileTMDBMatch {
  tmdb_id: number;
  title: string;
  original_title: string;
  year: number;
}

export interface DiskFile {
  path: string;
  size_bytes: number;
  parsed_title: string;
  parsed_year: number;
  tmdb_match?: DiskFileTMDBMatch;
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

// ── Quality Definitions ────────────────────────────────────────────────────

export interface QualityDefinition {
  id: string;
  name: string;
  resolution: string;
  source: string;
  codec: string;
  hdr: string;
  min_size: number;       // MB per minute (0 = no minimum)
  max_size: number;       // MB per minute (0 = no limit)
  preferred_size: number; // MB per minute target within [min, max] (0 = same as max)
  sort_order: number;
}

export interface QualityDefinitionUpdate {
  id: string;
  min_size: number;
  max_size: number;
  preferred_size: number;
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

// ── Movie files ────────────────────────────────────────────────────────────

export interface MediaInfo {
  container?: string;
  duration_secs?: number;
  video_bitrate?: number;
  codec?: string;
  width?: number;
  height?: number;
  resolution?: string;
  color_space?: string;
  hdr_format?: string;
  bit_depth?: number;
  audio_codec?: string;
  audio_channels?: number;
}

export interface MovieFile {
  id: string;
  movie_id: string;
  path: string;
  size_bytes: number;
  quality: Quality;
  edition?: string;
  imported_at: string;
  mediainfo?: MediaInfo;
}

// ── Blocklist ──────────────────────────────────────────────────────────────

export interface BlocklistEntry {
  id: string;
  movie_id: string;
  movie_title: string;
  release_guid: string;
  release_title: string;
  indexer_id?: string;
  protocol: string;
  size: number;
  added_at: string;
  notes?: string;
}

export interface BlocklistPage {
  items: BlocklistEntry[];
  total: number;
  page: number;
  per_page: number;
}

// ── Filesystem browser ─────────────────────────────────────────────────────

export interface FsDirEntry {
  name: string;
  path: string;
}

export interface FsBrowseResult {
  path: string;
  parent: string | null;
  dirs: FsDirEntry[];
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

// ── Collections ────────────────────────────────────────────────────────────

export interface PersonSearchResult {
  person_id: number;
  name: string;
  profile_path: string;
  known_for_department: string;
}

export interface EntitySearchResult {
  id: number;
  name: string;
  image_path: string;
  subtitle: string;
  result_type: "person" | "franchise";
}

export interface CollectionItem {
  tmdb_id: number;
  title: string;
  year: number;
  poster_path: string;
  in_library: boolean;
  has_file?: boolean;
  movie_id: string;
  monitored: boolean;
}

export interface Collection {
  id: string;
  name: string;
  person_id: number;
  person_type: string;
  created_at: string;
  items?: CollectionItem[];
  total: number;
  in_library: number;
  missing: number;
}

// ── Media Servers ───────────────────────────────────────────────────────────

export interface MediaServerConfig {
  id: string;
  name: string;
  kind: string;
  enabled: boolean;
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface MediaServerRequest {
  name: string;
  kind: string;
  enabled: boolean;
  settings: Record<string, unknown>;
}

// ── Plex Sync ──────────────────────────────────────────────────────────────

export interface PlexSection {
  key: string;
  title: string;
  type: string;
}

export interface PlexSyncMovie {
  title: string;
  year: number;
  tmdb_id: number;
}

export interface LuminarrSyncMovie {
  id: string;
  title: string;
  year: number;
  tmdb_id: number;
  status: string;
}

export interface PlexSyncPreviewResult {
  plex_total: number;
  in_plex_only: PlexSyncMovie[];
  in_luminarr_only: LuminarrSyncMovie[];
  already_synced: number;
  unmatched: number;
}

export interface PlexSyncImportOptions {
  tmdb_ids: number[];
  library_id: string;
  quality_profile_id: string;
  monitored: boolean;
}

export interface PlexSyncImportResult {
  imported: number;
  skipped: number;
  failed: number;
  errors: string[];
}
