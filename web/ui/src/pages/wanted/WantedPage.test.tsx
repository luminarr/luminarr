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
});
