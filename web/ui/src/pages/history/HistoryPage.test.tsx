import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import HistoryPage from "./HistoryPage";

function renderPage() {
  return renderWithProviders(createElement(HistoryPage));
}

describe("HistoryPage", () => {
  it("renders heading", () => {
    renderPage();
    expect(screen.getByText("History")).toBeInTheDocument();
  });

  it("shows loading skeletons", () => {
    server.use(http.get("/api/v1/history", () => new Promise(() => {})));
    const { container } = renderPage();
    expect(container.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });

  it("shows empty state when no history", async () => {
    server.use(http.get("/api/v1/history", () => HttpResponse.json([])));
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("No history yet")).toBeInTheDocument()
    );
  });

  it("renders history items", async () => {
    const items = [
      {
        id: "h-1",
        movie_id: "movie-1",
        release_guid: "r1",
        release_title: "Movie.2024.1080p.BluRay",
        protocol: "torrent",
        size: 1_000_000_000,
        download_status: "completed",
        grabbed_at: "2025-01-01T12:00:00Z",
        release_source: "bluray",
        release_resolution: "1080p",
      },
    ];
    server.use(http.get("/api/v1/history", () => HttpResponse.json(items)));
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Movie.2024.1080p.BluRay")).toBeInTheDocument()
    );
    expect(screen.getByText("1 grab")).toBeInTheDocument();
    // QualityBadge renders "resolution source" joined
    expect(screen.getByText("1080p bluray")).toBeInTheDocument();
  });

  it("shows error state on fetch failure", async () => {
    server.use(
      http.get("/api/v1/history", () =>
        HttpResponse.json({ error: "fail" }, { status: 500 })
      )
    );
    renderPage();
    // Both the subtitle and the table area show failure text
    await waitFor(() =>
      expect(screen.getAllByText(/Failed to load history/).length).toBeGreaterThanOrEqual(1)
    );
  });

  it("has status and protocol filter dropdowns", () => {
    renderPage();
    const combos = screen.getAllByRole("combobox");
    expect(combos.length).toBeGreaterThanOrEqual(2);
  });
});
