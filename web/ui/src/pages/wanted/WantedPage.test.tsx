import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import WantedPage from "./WantedPage";

function renderPage() {
  return renderWithProviders(createElement(WantedPage));
}

describe("WantedPage", () => {
  it("renders heading and tabs", () => {
    server.use(
      http.get("/api/v1/wanted/missing", () =>
        HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
      )
    );
    renderPage();
    expect(screen.getByText("Wanted")).toBeInTheDocument();
    expect(screen.getByText("Missing")).toBeInTheDocument();
    expect(screen.getByText("Cutoff Unmet")).toBeInTheDocument();
  });

  it("shows missing tab by default", async () => {
    server.use(
      http.get("/api/v1/wanted/missing", () =>
        HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("All caught up!")).toBeInTheDocument()
    );
    expect(
      screen.getByText("No monitored movies are missing a file.")
    ).toBeInTheDocument();
  });

  it("can switch to cutoff tab", async () => {
    server.use(
      http.get("/api/v1/wanted/missing", () =>
        HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
      ),
      http.get("/api/v1/wanted/cutoff", () =>
        HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
      )
    );
    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("All caught up!")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Cutoff Unmet"));

    await waitFor(() =>
      expect(screen.getByText("All at cutoff!")).toBeInTheDocument()
    );
  });

  // ── Upgrades tab ─────────────────────────────────────────────────────────

  it("shows Upgrades tab button", () => {
    server.use(
      http.get("/api/v1/wanted/missing", () =>
        HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
      )
    );
    renderPage();
    expect(screen.getByText("Upgrades")).toBeInTheDocument();
  });

  it("Upgrades tab shows tier cards when recommendations exist", async () => {
    server.use(
      http.get("/api/v1/wanted/missing", () =>
        HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
      ),
      http.get("/api/v1/wanted/upgrades", () =>
        HttpResponse.json({
          total: 3,
          tiers: [
            {
              label: "720p → 1080p",
              from_quality: "720p HDTV",
              to_quality: "1080p Bluray",
              count: 2,
              movie_ids: ["movie-1", "movie-2"],
            },
            {
              label: "1080p → 2160p",
              from_quality: "1080p Bluray",
              to_quality: "2160p Remux",
              count: 1,
              movie_ids: ["movie-3"],
            },
          ],
        })
      )
    );

    renderPage();
    const user = userEvent.setup();
    await user.click(screen.getByText("Upgrades"));

    await waitFor(() => {
      expect(screen.getByText("720p → 1080p")).toBeInTheDocument();
    });
    expect(screen.getByText("1080p → 2160p")).toBeInTheDocument();
    // Upgrade count summary line
    expect(screen.getByText("3 movies can be upgraded")).toBeInTheDocument();
    // From/to quality labels rendered inside tier cards
    expect(screen.getByText("720p HDTV → 1080p Bluray")).toBeInTheDocument();
    expect(screen.getByText("1080p Bluray → 2160p Remux")).toBeInTheDocument();
  });

  it("shows empty state when no upgrade recommendations", async () => {
    server.use(
      http.get("/api/v1/wanted/missing", () =>
        HttpResponse.json({ movies: [], total: 0, page: 1, per_page: 50 })
      ),
      http.get("/api/v1/wanted/upgrades", () =>
        HttpResponse.json({ total: 0, tiers: [] })
      )
    );

    renderPage();
    const user = userEvent.setup();
    await user.click(screen.getByText("Upgrades"));

    await waitFor(() => {
      expect(screen.getByText("Nothing to upgrade")).toBeInTheDocument();
    });
    expect(
      screen.getByText("All movies are at their best available quality.")
    ).toBeInTheDocument();
  });
});
