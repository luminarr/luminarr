import { describe, it, expect } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import StatsPage from "./StatsPage";

// ── Shared fixtures ───────────────────────────────────────────────────────────

const collectionStats = {
  total_movies: 50,
  monitored: 45,
  with_file: 38,
  missing: 10,
  needs_upgrade: 3,
  edition_mismatches: 0,
  recently_added: 2,
};

// Two buckets: one 1080p Bluray and one 2160p WebDL.
// This gives the QualityCard data to render with total > 0 so the toggle appears.
const qualityBuckets = [
  { resolution: "1080p", source: "Bluray", codec: "x264", hdr: "none", count: 30 },
  { resolution: "2160p", source: "WebDL", codec: "x265", hdr: "HDR10", count: 10 },
];

const qualityTiers = [
  { resolution: "1080p", source: "Bluray", count: 30 },
  { resolution: "2160p", source: "WebDL", count: 10 },
];

const storageStats = {
  total_bytes: 2_000_000_000_000,
  file_count: 40,
  trend: [],
};

const grabStats = {
  total_grabs: 60,
  successful: 55,
  failed: 5,
  success_rate: 0.917,
  top_indexers: [],
};

const decadeStats = [
  { decade: "1990s", count: 5 },
  { decade: "2000s", count: 20 },
];

const growthStats = [
  { month: "2025-01", added: 10, cumulative: 10 },
];

const genreStats = [
  { genre: "Action", count: 20 },
  { genre: "Drama", count: 15 },
];

// Register all stats endpoints for a full page render.
function useFullStatsHandlers() {
  server.use(
    http.get("/api/v1/stats/collection", () => HttpResponse.json(collectionStats)),
    http.get("/api/v1/stats/quality", () => HttpResponse.json(qualityBuckets)),
    http.get("/api/v1/stats/quality/tiers", () => HttpResponse.json(qualityTiers)),
    http.get("/api/v1/stats/storage", () => HttpResponse.json(storageStats)),
    http.get("/api/v1/stats/grabs", () => HttpResponse.json(grabStats)),
    http.get("/api/v1/stats/decades", () => HttpResponse.json(decadeStats)),
    http.get("/api/v1/stats/growth", () => HttpResponse.json(growthStats)),
    http.get("/api/v1/stats/genres", () => HttpResponse.json(genreStats))
  );
}

function renderPage() {
  return renderWithProviders(createElement(StatsPage));
}

describe("StatsPage", () => {
  it("renders the Statistics heading", () => {
    useFullStatsHandlers();
    renderPage();
    expect(screen.getByText("Statistics")).toBeInTheDocument();
  });

  it("shows skeleton cards while data is loading", () => {
    useFullStatsHandlers();
    const { container } = renderPage();
    // At least one skeleton should be present before data arrives
    expect(container.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });

  it("renders collection stats after loading", async () => {
    useFullStatsHandlers();
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("50")).toBeInTheDocument();
    });
    expect(screen.getByText("Total Movies")).toBeInTheDocument();
    expect(screen.getByText("38")).toBeInTheDocument();
    expect(screen.getByText("Have File")).toBeInTheDocument();
  });

  // ── Quality tier toggle ───────────────────────────────────────────────────

  it("shows By Dimension and By Tier toggle buttons", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Dimension")).toBeInTheDocument();
    });
    expect(screen.getByText("By Tier")).toBeInTheDocument();
  });

  it("By Dimension view is active by default", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Dimension")).toBeInTheDocument();
    });

    // Dimension sub-charts are rendered in the default view
    expect(screen.getByText("Resolution")).toBeInTheDocument();
    expect(screen.getByText("Source")).toBeInTheDocument();
    expect(screen.getByText("Codec")).toBeInTheDocument();
    expect(screen.getByText("HDR")).toBeInTheDocument();

    // The tier-specific prompt should not be visible yet
    expect(
      screen.queryByText("Click a tier to filter the movie library.")
    ).not.toBeInTheDocument();
  });

  it("switching to By Tier shows tier chart heading and filter prompt", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Tier")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("By Tier"));

    await waitFor(() => {
      expect(
        screen.getByText("Click a tier to filter the movie library.")
      ).toBeInTheDocument();
    });

    // The combined resolution+source chart heading
    expect(screen.getByText("Resolution + Source")).toBeInTheDocument();
  });

  it("switching back to By Dimension restores dimension charts", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Tier")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("By Tier"));

    await waitFor(() => {
      expect(screen.getByText("Resolution + Source")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("By Dimension"));

    await waitFor(() => {
      expect(screen.getByText("Resolution")).toBeInTheDocument();
    });
    expect(
      screen.queryByText("Click a tier to filter the movie library.")
    ).not.toBeInTheDocument();
  });

  it("shows empty Quality Distribution when there are no files", async () => {
    server.use(
      http.get("/api/v1/stats/collection", () => HttpResponse.json(collectionStats)),
      http.get("/api/v1/stats/quality", () => HttpResponse.json([])),
      http.get("/api/v1/stats/quality/tiers", () => HttpResponse.json([])),
      http.get("/api/v1/stats/storage", () => HttpResponse.json(storageStats)),
      http.get("/api/v1/stats/grabs", () => HttpResponse.json(grabStats)),
      http.get("/api/v1/stats/decades", () => HttpResponse.json(decadeStats)),
      http.get("/api/v1/stats/growth", () => HttpResponse.json(growthStats)),
      http.get("/api/v1/stats/genres", () => HttpResponse.json(genreStats))
    );
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("No movie files yet.")).toBeInTheDocument();
    });

    // Toggle should not be rendered when there are no files
    expect(screen.queryByText("By Dimension")).not.toBeInTheDocument();
    expect(screen.queryByText("By Tier")).not.toBeInTheDocument();
  });
});
