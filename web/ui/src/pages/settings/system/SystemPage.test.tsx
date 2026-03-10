import { describe, it, expect, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { systemStatusFixture, healthyReport, degradedReport } from "@/test/fixtures";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import SystemPage from "./SystemPage";

function renderPage() {
  return renderWithProviders(createElement(SystemPage));
}

// ── StatsStrip ───────────────────────────────────────────────────────────────

describe("StatsStrip", () => {
  it("renders movie count and queue count", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json({ movies: [], total: 42, page: 1, per_page: 1 })
      ),
      http.get("/api/v1/queue", () =>
        HttpResponse.json([{ id: "q-1" }, { id: "q-2" }])
      )
    );

    renderPage();
    await waitFor(() => expect(screen.getByText("42")).toBeInTheDocument());
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("Total Movies")).toBeInTheDocument();
    expect(screen.getByText("Downloading")).toBeInTheDocument();
  });
});

// ── StatusSection ────────────────────────────────────────────────────────────

describe("StatusSection", () => {
  it("shows skeletons while loading", () => {
    // Never resolve — sections stay in loading state
    server.use(
      http.get("/api/v1/system/status", () => new Promise(() => {})),
      http.get("/api/v1/system/health", () => new Promise(() => {})),
      http.get("/api/v1/tasks", () => new Promise(() => {})),
      http.get("/api/v1/system/logs", () => new Promise(() => {})),
      http.get("/api/v1/movies", () => new Promise(() => {})),
      http.get("/api/v1/queue", () => new Promise(() => {}))
    );

    const { container } = renderPage();
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders system info when loaded", async () => {
    server.use(
      http.get("/api/v1/system/status", () =>
        HttpResponse.json(systemStatusFixture)
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/Luminarr 0\.0\.0-test/)).toBeInTheDocument()
    );

    expect(screen.getByText("go1.23.0")).toBeInTheDocument();
    expect(screen.getByText("sqlite3")).toBeInTheDocument();
    expect(screen.getByText(":memory:")).toBeInTheDocument();
    expect(screen.getByText("1h 0m")).toBeInTheDocument();
    expect(screen.getByText("Disabled")).toBeInTheDocument(); // ai_enabled: false
    expect(screen.getByText("Configured")).toBeInTheDocument(); // tmdb_enabled: true
  });

  it("shows 'Check for updates' button", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
  });

  it("shows 'Up to date' when no update available", async () => {
    server.use(
      http.get("/api/v1/system/updates", () =>
        HttpResponse.json({
          update_available: false,
          current_version: "0.0.0-test",
          latest_version: "0.0.0-test",
        })
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Check for updates"));

    await waitFor(() =>
      expect(screen.getByText(/Up to date/)).toBeInTheDocument()
    );
  });

  it("opens update modal when update is available", async () => {
    server.use(
      http.get("/api/v1/system/updates", () =>
        HttpResponse.json({
          update_available: true,
          current_version: "0.0.0-test",
          latest_version: "1.0.0",
          release_url: "https://github.com/luminarr/luminarr/releases/tag/v1.0.0",
          release_notes: "## What's New\n- First release",
          published_at: "2025-06-01T00:00:00Z",
        })
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Check for updates"));

    await waitFor(() =>
      expect(screen.getByText("Update Available")).toBeInTheDocument()
    );
    expect(screen.getByText("0.0.0-test")).toBeInTheDocument();
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    expect(screen.getByText("View on GitHub →")).toBeInTheDocument();
    expect(screen.getByText("Docker Compose")).toBeInTheDocument();
    expect(screen.getByText("Docker Pull")).toBeInTheDocument();
  });

  it("closes update modal on Close button", async () => {
    server.use(
      http.get("/api/v1/system/updates", () =>
        HttpResponse.json({
          update_available: true,
          current_version: "0.0.0-test",
          latest_version: "1.0.0",
        })
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Check for updates"));

    await waitFor(() =>
      expect(screen.getByText("Update Available")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Close"));

    await waitFor(() =>
      expect(screen.queryByText("Update Available")).not.toBeInTheDocument()
    );
  });
});

// ── HealthSection ────────────────────────────────────────────────────────────

describe("HealthSection", () => {
  it("renders 'All systems healthy' for healthy status", async () => {
    server.use(
      http.get("/api/v1/system/health", () => HttpResponse.json(healthyReport))
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("All systems healthy")).toBeInTheDocument()
    );
  });

  it("renders degraded status with check details", async () => {
    server.use(
      http.get("/api/v1/system/health", () =>
        HttpResponse.json(degradedReport)
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Degraded")).toBeInTheDocument()
    );
    expect(screen.getByText("indexer")).toBeInTheDocument();
    expect(screen.getByText("1 indexer offline")).toBeInTheDocument();
  });

  it("shows error message on fetch failure", async () => {
    server.use(
      http.get("/api/v1/system/health", () =>
        HttpResponse.json({ error: "Internal error" }, { status: 500 })
      )
    );

    renderPage();
    await waitFor(() =>
      expect(
        screen.getByText("Failed to load health data.")
      ).toBeInTheDocument()
    );
  });
});

// ── TasksSection ─────────────────────────────────────────────────────────────

describe("TasksSection", () => {
  it("shows 'No tasks registered' for empty list", async () => {
    server.use(
      http.get("/api/v1/tasks", () => HttpResponse.json([]))
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("No tasks registered.")).toBeInTheDocument()
    );
  });

  it("renders task table with names and intervals", async () => {
    server.use(
      http.get("/api/v1/tasks", () =>
        HttpResponse.json([
          { name: "refresh-metadata", interval: "24h" },
          { name: "check-downloads", interval: "5m" },
        ])
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("refresh-metadata")).toBeInTheDocument()
    );
    expect(screen.getByText("24h")).toBeInTheDocument();
    expect(screen.getByText("check-downloads")).toBeInTheDocument();
    expect(screen.getByText("5m")).toBeInTheDocument();
  });

  it("shows Run Now buttons for each task", async () => {
    server.use(
      http.get("/api/v1/tasks", () =>
        HttpResponse.json([
          { name: "refresh-metadata", interval: "24h" },
        ])
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Run Now")).toBeInTheDocument()
    );
  });

  it("triggers task and shows 'Triggered' confirmation", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    server.use(
      http.get("/api/v1/tasks", () =>
        HttpResponse.json([
          { name: "refresh-metadata", interval: "24h" },
        ])
      ),
      http.post("/api/v1/tasks/refresh-metadata/run", () =>
        new HttpResponse(null, { status: 202 })
      )
    );

    renderPage();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    await waitFor(() =>
      expect(screen.getByText("Run Now")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Run Now"));

    await waitFor(() =>
      expect(screen.getByText(/Triggered/)).toBeInTheDocument()
    );

    vi.useRealTimers();
  });
});

// ── LogsSection ──────────────────────────────────────────────────────────────

describe("LogsSection", () => {
  it("shows 'No log entries' when empty", async () => {
    server.use(
      http.get("/api/v1/system/logs", () => HttpResponse.json([]))
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("No log entries.")).toBeInTheDocument()
    );
  });

  it("renders log entries with time, level, and message", async () => {
    const logs = [
      {
        time: "2025-01-01T12:34:56Z",
        level: "INFO",
        message: "Server started",
      },
      {
        time: "2025-01-01T12:35:00Z",
        level: "ERROR",
        message: "Database connection failed",
        fields: { db: "sqlite3", error: "locked" },
      },
    ];
    server.use(
      http.get("/api/v1/system/logs", () => HttpResponse.json(logs))
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Server started")).toBeInTheDocument()
    );
    // "INFO" also appears in the level filter dropdown, so use getAllByText
    expect(screen.getAllByText("INFO").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("ERROR").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Database connection failed")).toBeInTheDocument();
  });

  it("shows error message on fetch failure", async () => {
    server.use(
      http.get("/api/v1/system/logs", () =>
        HttpResponse.json({ error: "fail" }, { status: 500 })
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Failed to load logs.")).toBeInTheDocument()
    );
  });

  it("has level filter dropdown", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Level:")).toBeInTheDocument()
    );

    const select = screen.getByRole("combobox");
    expect(select).toBeInTheDocument();

    // Check options
    const options = within(select).getAllByRole("option");
    expect(options).toHaveLength(5);
    expect(options.map((o) => o.textContent)).toEqual([
      "All",
      "DEBUG",
      "INFO",
      "WARN",
      "ERROR",
    ]);
  });

  it("expands log entry fields on click", async () => {
    const logs = [
      {
        time: "2025-01-01T12:00:00Z",
        level: "ERROR",
        message: "Something broke",
        fields: { component: "scheduler", error: "timeout" },
      },
    ];
    server.use(
      http.get("/api/v1/system/logs", () => HttpResponse.json(logs))
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("Something broke")).toBeInTheDocument()
    );

    // Click the row to expand fields
    await user.click(screen.getByText("Something broke"));

    await waitFor(() =>
      expect(screen.getByText(/"component": "scheduler"/)).toBeInTheDocument()
    );
  });
});

// ── Page-level ───────────────────────────────────────────────────────────────

describe("SystemPage", () => {
  it("renders heading and description", async () => {
    renderPage();
    expect(screen.getByText("System")).toBeInTheDocument();
    expect(
      screen.getByText("Runtime status, health checks, and configuration.")
    ).toBeInTheDocument();
  });

  it("renders all four section headers", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Status")).toBeInTheDocument()
    );
    expect(screen.getByText("Health")).toBeInTheDocument();
    expect(screen.getByText("Tasks")).toBeInTheDocument();
    expect(screen.getByText("Logs")).toBeInTheDocument();
  });
});
