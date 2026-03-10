import type {
  Movie,
  QueueItem,
  Library,
  QualityProfile,
  Quality,
  HealthReport,
  SystemStatus,
  IndexerConfig,
  DownloadClientConfig,
  Release,
} from "@/types";

// ── Qualities ──────────────────────────────────────────────────────────────

export const quality1080pBluray = {
  resolution: "1080p",
  source: "bluray",
  codec: "x264",
  hdr: "",
  name: "Bluray-1080p",
} satisfies Quality;

export const quality2160pWebdl = {
  resolution: "2160p",
  source: "webdl",
  codec: "x265",
  hdr: "HDR10",
  name: "WEBDL-2160p",
} satisfies Quality;

// ── Movies ─────────────────────────────────────────────────────────────────

export const movieFixture = {
  id: "movie-1",
  tmdb_id: 550,
  imdb_id: "tt0137523",
  title: "Fight Club",
  original_title: "Fight Club",
  year: 1999,
  overview: "An insomniac office worker and a devil-may-care soap maker form an underground fight club.",
  runtime_minutes: 139,
  genres: ["Drama", "Thriller"],
  poster_url: "https://image.tmdb.org/t/p/w500/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg",
  status: "released",
  monitored: true,
  library_id: "lib-1",
  quality_profile_id: "qp-1",
  minimum_availability: "released",
  release_date: "1999-10-15",
  path: "/movies/Fight Club (1999)",
  added_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
} satisfies Movie;

// ── Queue ──────────────────────────────────────────────────────────────────

export const queueItemFixture = {
  id: "q-1",
  movie_id: "movie-1",
  release_title: "Fight.Club.1999.1080p.BluRay.x264-GROUP",
  protocol: "torrent",
  size: 8_589_934_592,
  downloaded_bytes: 4_294_967_296,
  status: "downloading",
  download_client_id: "dc-1",
  client_item_id: "abc123",
  grabbed_at: "2025-01-01T12:00:00Z",
} satisfies QueueItem;

// ── Libraries ──────────────────────────────────────────────────────────────

export const libraryFixture = {
  id: "lib-1",
  name: "Movies",
  root_path: "/movies",
  default_quality_profile_id: "qp-1",
  min_free_space_gb: 10,
  tags: [],
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
} satisfies Library;

// ── Quality Profiles ───────────────────────────────────────────────────────

export const qualityProfileFixture = {
  id: "qp-1",
  name: "HD-1080p",
  cutoff: quality1080pBluray,
  qualities: [quality1080pBluray],
  upgrade_allowed: true,
  upgrade_until: quality1080pBluray,
} satisfies QualityProfile;

// ── System ─────────────────────────────────────────────────────────────────

export const systemStatusFixture = {
  app_name: "Luminarr",
  version: "0.0.0-test",
  build_time: "2025-01-01T00:00:00Z",
  go_version: "go1.23.0",
  db_type: "sqlite3",
  db_path: ":memory:",
  uptime_seconds: 3600,
  start_time: "2025-01-01T00:00:00Z",
  ai_enabled: false,
  tmdb_enabled: true,
} satisfies SystemStatus;

export const healthyReport = {
  status: "healthy",
  checks: [],
} satisfies HealthReport;

export const degradedReport = {
  status: "degraded",
  checks: [
    { name: "indexer", status: "degraded", message: "1 indexer offline" },
  ],
} satisfies HealthReport;

// ── Indexers ───────────────────────────────────────────────────────────────

export const indexerFixture = {
  id: "idx-1",
  name: "Test Indexer",
  kind: "torznab",
  enabled: true,
  priority: 25,
  settings: { url: "http://localhost:9696", api_key: "abc123" },
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
} satisfies IndexerConfig;

// ── Download Clients ───────────────────────────────────────────────────────

export const downloadClientFixture = {
  id: "dc-1",
  name: "qBittorrent",
  kind: "qbittorrent",
  enabled: true,
  priority: 1,
  settings: { url: "http://localhost:8080", username: "admin", password: "" },
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
} satisfies DownloadClientConfig;

// ── Releases ───────────────────────────────────────────────────────────────

export const releaseFixture = {
  guid: "release-1",
  title: "Fight.Club.1999.1080p.BluRay.x264-GROUP",
  indexer: "Test Indexer",
  protocol: "torrent",
  download_url: "https://example.com/download/1",
  size: 8_589_934_592,
  seeds: 42,
  peers: 10,
  age_days: 365,
  quality: quality1080pBluray,
  quality_score: 850,
} satisfies Release;
