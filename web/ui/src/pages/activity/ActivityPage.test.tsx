import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import ActivityPage from "./ActivityPage";

function renderPage() {
  return renderWithProviders(createElement(ActivityPage));
}

const fixtures = {
  activities: [
    {
      id: "a-1",
      type: "grab_started",
      category: "grab",
      movie_id: "movie-1",
      title: "Grabbed Alien.1979.DC.1080p.BluRay from The Pirate Bay",
      detail: { indexer: "The Pirate Bay", quality: "1080p bluray" },
      created_at: new Date(Date.now() - 300_000).toISOString(), // 5m ago
    },
    {
      id: "a-2",
      type: "task_finished",
      category: "task",
      title: "RSS Sync completed",
      created_at: new Date(Date.now() - 3600_000).toISOString(), // 1h ago
    },
    {
      id: "a-3",
      type: "import_complete",
      category: "import",
      movie_id: "movie-2",
      title: "Imported Inception (2010) — 2160p Bluray",
      created_at: new Date(Date.now() - 86400_000).toISOString(), // 1d ago
    },
    {
      id: "a-4",
      type: "health_issue",
      category: "health",
      title: "disk_space: path not accessible",
      created_at: new Date(Date.now() - 172800_000).toISOString(), // 2d ago
    },
    {
      id: "a-5",
      type: "movie_added",
      category: "movie",
      movie_id: "movie-3",
      title: "Added The Matrix (1999) to library",
      created_at: new Date(Date.now() - 259200_000).toISOString(), // 3d ago
    },
  ],
  total: 5,
};

describe("ActivityPage", () => {
  it("renders heading", () => {
    renderPage();
    expect(screen.getByText("Activity")).toBeInTheDocument();
  });

  it("shows loading skeletons while fetching", () => {
    server.use(http.get("/api/v1/activity", () => new Promise(() => {})));
    const { container } = renderPage();
    expect(container.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });

  it("shows empty state when no activities", async () => {
    server.use(
      http.get("/api/v1/activity", () =>
        HttpResponse.json({ activities: [], total: 0 })
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId("empty-state")).toBeInTheDocument()
    );
    expect(screen.getByText("No recent activity")).toBeInTheDocument();
  });

  it("renders activity list from API data", async () => {
    server.use(
      http.get("/api/v1/activity", () => HttpResponse.json(fixtures))
    );
    renderPage();
    await waitFor(() =>
      expect(
        screen.getByText(
          "Grabbed Alien.1979.DC.1080p.BluRay from The Pirate Bay"
        )
      ).toBeInTheDocument()
    );
    expect(screen.getByText("RSS Sync completed")).toBeInTheDocument();
    expect(
      screen.getByText("Imported Inception (2010) — 2160p Bluray")
    ).toBeInTheDocument();
    expect(
      screen.getByText("disk_space: path not accessible")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Added The Matrix (1999) to library")
    ).toBeInTheDocument();
  });

  it("shows total count", async () => {
    server.use(
      http.get("/api/v1/activity", () => HttpResponse.json(fixtures))
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("5 events")).toBeInTheDocument()
    );
  });

  it("renders movie-linked activities as clickable links", async () => {
    server.use(
      http.get("/api/v1/activity", () => HttpResponse.json(fixtures))
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId("activity-link-a-1")).toBeInTheDocument()
    );
    const link = screen.getByTestId("activity-link-a-1");
    expect(link.tagName).toBe("A");
    expect(link.getAttribute("href")).toBe("/movies/movie-1");
  });

  it("renders non-movie activities as plain text", async () => {
    server.use(
      http.get("/api/v1/activity", () => HttpResponse.json(fixtures))
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId("activity-text-a-2")).toBeInTheDocument()
    );
    const el = screen.getByTestId("activity-text-a-2");
    expect(el.tagName).toBe("SPAN");
  });

  it("shows relative timestamps", async () => {
    server.use(
      http.get("/api/v1/activity", () => HttpResponse.json(fixtures))
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("5m ago")).toBeInTheDocument()
    );
    expect(screen.getByText("1h ago")).toBeInTheDocument();
    expect(screen.getByText("1d ago")).toBeInTheDocument();
  });

  it("filters by category when clicking a pill", async () => {
    const user = userEvent.setup();
    let lastCategory: string | null = null;

    server.use(
      http.get("/api/v1/activity", ({ request }) => {
        const url = new URL(request.url);
        lastCategory = url.searchParams.get("category");
        if (lastCategory === "grab") {
          return HttpResponse.json({
            activities: [fixtures.activities[0]],
            total: 1,
          });
        }
        return HttpResponse.json(fixtures);
      })
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("5 events")).toBeInTheDocument()
    );

    await user.click(screen.getByTestId("filter-grab"));
    await waitFor(() => expect(lastCategory).toBe("grab"));
    await waitFor(() =>
      expect(screen.getByText("1 events")).toBeInTheDocument()
    );
  });

  it("shows error state on API failure", async () => {
    server.use(
      http.get("/api/v1/activity", () =>
        HttpResponse.json({ title: "Internal Server Error" }, { status: 500 })
      )
    );
    renderPage();
    await waitFor(() =>
      expect(
        screen.getByText("Failed to load activity.")
      ).toBeInTheDocument()
    );
  });

  it("renders all category filter pills", () => {
    renderPage();
    expect(screen.getByTestId("filter-all")).toBeInTheDocument();
    expect(screen.getByTestId("filter-grab")).toBeInTheDocument();
    expect(screen.getByTestId("filter-import")).toBeInTheDocument();
    expect(screen.getByTestId("filter-task")).toBeInTheDocument();
    expect(screen.getByTestId("filter-health")).toBeInTheDocument();
    expect(screen.getByTestId("filter-movie")).toBeInTheDocument();
  });
});
