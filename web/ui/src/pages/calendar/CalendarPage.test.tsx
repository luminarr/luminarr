import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { movieFixture } from "@/test/fixtures";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import CalendarPage from "./CalendarPage";

function renderPage() {
  return renderWithProviders(createElement(CalendarPage));
}

describe("CalendarPage", () => {
  it("renders heading and Today button", () => {
    renderPage();
    expect(screen.getByText("Calendar")).toBeInTheDocument();
    expect(screen.getByText("Today")).toBeInTheDocument();
  });

  it("shows day-of-week headers", async () => {
    renderPage();
    expect(screen.getByText("Sun")).toBeInTheDocument();
    expect(screen.getByText("Mon")).toBeInTheDocument();
    expect(screen.getByText("Sat")).toBeInTheDocument();
  });

  it("shows current month and year", () => {
    renderPage();
    const now = new Date();
    const months = [
      "January", "February", "March", "April", "May", "June",
      "July", "August", "September", "October", "November", "December",
    ];
    expect(
      screen.getByText(new RegExp(`${months[now.getMonth()]} ${now.getFullYear()}`))
    ).toBeInTheDocument();
  });

  it("has navigation arrows", () => {
    renderPage();
    // Previous and next month buttons
    const buttons = screen.getAllByRole("button");
    // At least Today + prev + next
    expect(buttons.length).toBeGreaterThanOrEqual(3);
  });

  it("shows movie on its release date", async () => {
    // Set a movie releasing today
    const today = new Date();
    const releaseDate = today.toISOString().slice(0, 10);
    const todayMovie = {
      ...movieFixture,
      release_date: releaseDate,
      title: "Calendar Test Movie",
    };
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json({ movies: [todayMovie], total: 1, page: 1, per_page: 1000 })
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Calendar Test Movie")).toBeInTheDocument()
    );
  });
});
