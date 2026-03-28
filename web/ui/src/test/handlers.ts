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

  // Activity
  http.get("/api/v1/activity", () =>
    HttpResponse.json({ activities: [], total: 0 })
  ),

  // Movie credits
  http.get("/api/v1/movies/:id/credits", () =>
    HttpResponse.json({ cast: [], crew: [], recommendations: [] })
  ),

  // Watch
  http.post("/api/v1/watch-sync/run", () =>
    HttpResponse.json({ status: "ok" })
  ),
  http.get("/api/v1/stats/watch", () =>
    HttpResponse.json({ watched_count: 0, total_count: 0, percentage: 0 })
  ),

  // Discover
  http.get("/api/v1/discover/trending", () =>
    HttpResponse.json({ results: [], page: 1, total_pages: 0 })
  ),
  http.get("/api/v1/discover/popular", () =>
    HttpResponse.json({ results: [], page: 1, total_pages: 0 })
  ),
  http.get("/api/v1/discover/top-rated", () =>
    HttpResponse.json({ results: [], page: 1, total_pages: 0 })
  ),
  http.get("/api/v1/discover/upcoming", () =>
    HttpResponse.json({ results: [], page: 1, total_pages: 0 })
  ),
  http.get("/api/v1/discover/genre/:id", () =>
    HttpResponse.json({ results: [], page: 1, total_pages: 0 })
  ),
  http.get("/api/v1/discover/genres", () =>
    HttpResponse.json([
      { id: 28, name: "Action" },
      { id: 35, name: "Comedy" },
      { id: 18, name: "Drama" },
      { id: 878, name: "Science Fiction" },
      { id: 27, name: "Horror" },
    ])
  ),

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
