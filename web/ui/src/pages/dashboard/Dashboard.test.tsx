import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { movieFixture } from "@/test/fixtures";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import Dashboard from "./Dashboard";

function renderPage() {
  return renderWithProviders(createElement(Dashboard));
}

function moviesResponse(movies: typeof movieFixture[]) {
  return { movies, total: movies.length, page: 1, per_page: 2000 };
}

describe("Dashboard", () => {
  it("renders heading", async () => {
    renderPage();
    expect(screen.getByText("Library")).toBeInTheDocument();
  });

  it("shows loading skeletons", () => {
    server.use(
      http.get("/api/v1/movies", () => new Promise(() => {}))
    );

    const { container } = renderPage();
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("shows empty state when no movies", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json(moviesResponse([]))
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("No movies in your library")).toBeInTheDocument()
    );
    expect(
      screen.getByText("Search TMDB to add your first movie.")
    ).toBeInTheDocument();
  });

  it("shows error state on fetch failure", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json({ error: "fail" }, { status: 500 })
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Failed to load movies.")).toBeInTheDocument()
    );
  });

  it("renders movie count and movie cards", async () => {
    const movie2 = {
      ...movieFixture,
      id: "movie-2",
      title: "The Matrix",
      year: 1999,
      poster_url: "",
    };
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json(moviesResponse([movieFixture, movie2]))
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("2 movies")).toBeInTheDocument()
    );
    expect(screen.getByAltText("Fight Club")).toBeInTheDocument();
    // Movie without poster shows title as text (may appear multiple times)
    expect(screen.getAllByText("The Matrix").length).toBeGreaterThanOrEqual(1);
  });

  it("has search input", async () => {
    renderPage();
    expect(
      screen.getByPlaceholderText("Search by title…")
    ).toBeInTheDocument();
  });

  it("has filter dropdowns", async () => {
    renderPage();
    const combos = screen.getAllByRole("combobox");
    // At least monitored, status, on disk, sort field
    expect(combos.length).toBeGreaterThanOrEqual(3);
  });

  it("has + Add Movie button", async () => {
    renderPage();
    expect(screen.getByText("+ Add Movie")).toBeInTheDocument();
  });

  it("has view mode toggle buttons", async () => {
    renderPage();
    expect(screen.getByTitle("Grid view")).toBeInTheDocument();
    expect(screen.getByTitle("List view")).toBeInTheDocument();
  });

  it("filters movies by search", async () => {
    const movies = [
      movieFixture,
      { ...movieFixture, id: "m2", title: "Inception", original_title: "Inception", poster_url: "" },
    ];
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json(moviesResponse(movies))
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("2 movies")).toBeInTheDocument()
    );

    await user.type(screen.getByPlaceholderText("Search by title…"), "inception");
    await waitFor(() =>
      expect(screen.getByText(/1 of 2 movies/)).toBeInTheDocument()
    );
  });

  it("shows 'No movies match' when filters exclude all", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json(moviesResponse([movieFixture]))
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("1 movie")).toBeInTheDocument()
    );

    await user.type(
      screen.getByPlaceholderText("Search by title…"),
      "nonexistent"
    );
    await waitFor(() =>
      expect(
        screen.getByText("No movies match the current filters.")
      ).toBeInTheDocument()
    );
    expect(screen.getByText("Clear filters")).toBeInTheDocument();
  });
});
