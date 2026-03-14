import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

// Default handlers — tests override these with server.use() as needed.
export const handlers = [
  // System
  http.get("/api/v1/system/status", () =>
    HttpResponse.json({
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
    })
  ),
  http.get("/api/v1/system/health", () =>
    HttpResponse.json({
      status: "healthy",
      checks: [],
    })
  ),
  http.get("/api/v1/tasks", () => HttpResponse.json([])),
  http.get("/api/v1/system/logs", () => HttpResponse.json([])),

  // Movies
  http.get("/api/v1/movies", () =>
    HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
  ),

  // Queue
  http.get("/api/v1/queue", () => HttpResponse.json([])),

  // History
  http.get("/api/v1/history", () => HttpResponse.json([])),

  // Wanted
  http.get("/api/v1/wanted/missing", () =>
    HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
  ),
  http.get("/api/v1/wanted/cutoff", () =>
    HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
  ),

  // Libraries
  http.get("/api/v1/libraries", () => HttpResponse.json([])),

  // Quality profiles
  http.get("/api/v1/quality-profiles", () => HttpResponse.json([])),

  // Quality definitions
  http.get("/api/v1/quality-definitions", () => HttpResponse.json([])),
];

export const server = setupServer(...handlers);
