import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { queueItemFixture } from "@/test/fixtures";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import Queue from "./Queue";

function renderPage() {
  return renderWithProviders(createElement(Queue));
}

describe("Queue", () => {
  it("renders heading", () => {
    renderPage();
    expect(screen.getByText("Queue")).toBeInTheDocument();
  });

  it("shows loading skeletons", () => {
    server.use(http.get("/api/v1/queue", () => new Promise(() => {})));
    const { container } = renderPage();
    expect(container.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });

  it("shows empty state when queue is empty", async () => {
    server.use(http.get("/api/v1/queue", () => HttpResponse.json([])));
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Queue is empty")).toBeInTheDocument()
    );
    expect(screen.getByText("No downloads in progress.")).toBeInTheDocument();
  });

  it("renders queue items with release titles", async () => {
    server.use(
      http.get("/api/v1/queue", () =>
        HttpResponse.json([queueItemFixture])
      )
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/Fight\.Club/)).toBeInTheDocument()
    );
    expect(screen.getByText("1 item — 1 active")).toBeInTheDocument();
  });

  it("shows Downloading indicator for active items", async () => {
    server.use(
      http.get("/api/v1/queue", () =>
        HttpResponse.json([queueItemFixture])
      )
    );
    renderPage();
    // "Downloading" appears as both the status badge and the indicator label
    await waitFor(() =>
      expect(screen.getAllByText("Downloading").length).toBeGreaterThanOrEqual(1)
    );
  });
});
