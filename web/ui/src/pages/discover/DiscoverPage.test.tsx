import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import DiscoverPage from "./DiscoverPage";

function renderPage() {
  return renderWithProviders(createElement(DiscoverPage));
}

const fixtures = {
  results: [
    {
      tmdb_id: 550,
      title: "Fight Club",
      year: 1999,
      overview: "An insomniac office worker...",
      poster_path: "/poster1.jpg",
      rating: 8.4,
      in_library: false,
      excluded: false,
    },
    {
      tmdb_id: 155,
      title: "The Dark Knight",
      year: 2008,
      overview: "Batman raises the stakes...",
      poster_path: "/poster2.jpg",
      rating: 8.5,
      in_library: true,
      excluded: false,
      library_movie_id: "movie-1",
    },
    {
      tmdb_id: 999,
      title: "Excluded Movie",
      year: 2020,
      overview: "...",
      poster_path: "/poster3.jpg",
      rating: 5.0,
      in_library: false,
      excluded: true,
    },
  ],
  page: 1,
  total_pages: 5,
};

describe("DiscoverPage", () => {
  it("renders heading", () => {
    renderPage();
    expect(screen.getByText("Discover")).toBeInTheDocument();
  });

  it("renders all tab pills", () => {
    renderPage();
    expect(screen.getByTestId("tab-trending")).toBeInTheDocument();
    expect(screen.getByTestId("tab-popular")).toBeInTheDocument();
    expect(screen.getByTestId("tab-top-rated")).toBeInTheDocument();
    expect(screen.getByTestId("tab-upcoming")).toBeInTheDocument();
    expect(screen.getByTestId("tab-genre")).toBeInTheDocument();
  });

  it("shows loading skeletons while fetching", () => {
    server.use(
      http.get("/api/v1/discover/trending", () => new Promise(() => {}))
    );
    const { container } = renderPage();
    expect(container.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });

  it("renders movie grid from API data", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Fight Club")).toBeInTheDocument()
    );
    expect(screen.getByText("The Dark Knight")).toBeInTheDocument();
    expect(screen.getByText("Excluded Movie")).toBeInTheDocument();
  });

  it("shows In Library badge for movies in library", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId("in-library-155")).toBeInTheDocument()
    );
  });

  it("shows + Add button for movies not in library", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId("add-550")).toBeInTheDocument()
    );
  });

  it("shows Excluded badge for excluded movies", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Excluded")).toBeInTheDocument()
    );
  });

  it("shows TMDB rating", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("8.4")).toBeInTheDocument()
    );
  });

  it("shows pagination when total_pages > 1", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Page 1 of 5")).toBeInTheDocument()
    );
    expect(screen.getByTestId("next-page")).toBeInTheDocument();
  });

  it("switches tabs and fetches correct endpoint", async () => {
    const user = userEvent.setup();
    let lastPath = "";

    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      ),
      http.get("/api/v1/discover/popular", ({ request }) => {
        lastPath = new URL(request.url).pathname;
        return HttpResponse.json({
          ...fixtures,
          results: [
            {
              ...fixtures.results[0],
              tmdb_id: 111,
              title: "Popular Movie",
            },
          ],
        });
      })
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Fight Club")).toBeInTheDocument()
    );

    await user.click(screen.getByTestId("tab-popular"));
    await waitFor(() => expect(lastPath).toBe("/api/v1/discover/popular"));
    await waitFor(() =>
      expect(screen.getByText("Popular Movie")).toBeInTheDocument()
    );
  });

  it("shows genre dropdown when By Genre tab is selected", async () => {
    const user = userEvent.setup();
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Fight Club")).toBeInTheDocument()
    );

    await user.click(screen.getByTestId("tab-genre"));
    await waitFor(() =>
      expect(screen.getByTestId("genre-select")).toBeInTheDocument()
    );
    // Genre options from MSW handler
    expect(screen.getByText("Action")).toBeInTheDocument();
    expect(screen.getByText("Comedy")).toBeInTheDocument();
    expect(screen.getByText("Science Fiction")).toBeInTheDocument();
  });

  it("shows empty state when no results", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json({ results: [], page: 1, total_pages: 0 })
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("No movies found")).toBeInTheDocument()
    );
  });

  it("shows error state on API failure", async () => {
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(
          { title: "Internal Server Error" },
          { status: 500 }
        )
      )
    );
    renderPage();
    await waitFor(() =>
      expect(
        screen.getByText(/Failed to load movies/)
      ).toBeInTheDocument()
    );
  });

  it("opens add modal when clicking + Add", async () => {
    const user = userEvent.setup();
    server.use(
      http.get("/api/v1/discover/trending", () =>
        HttpResponse.json(fixtures)
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId("add-550")).toBeInTheDocument()
    );

    await user.click(screen.getByTestId("add-550"));
    await waitFor(() =>
      expect(screen.getByText("Add Fight Club")).toBeInTheDocument()
    );
    expect(screen.getByTestId("add-movie-btn")).toBeInTheDocument();
  });
});
