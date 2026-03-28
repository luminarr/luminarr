import type { ReactNode } from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { CommandPalette } from "./CommandPalette";

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={qc}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  };
}

/** Check if palette item is the active/highlighted one. */
function isHighlighted(el: HTMLElement): boolean {
  return el.getAttribute("aria-selected") === "true";
}

describe("CommandPalette", () => {
  it("renders the input and ESC hint", () => {
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });
    expect(screen.getByTestId("command-palette-input")).toBeInTheDocument();
    expect(screen.getByText("ESC")).toBeInTheDocument();
  });

  it("shows navigation items by default", () => {
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });
    expect(screen.getByText("Pages")).toBeInTheDocument();
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Queue")).toBeInTheDocument();
  });

  it("shows action items by default", () => {
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });
    expect(screen.getByText("Actions")).toBeInTheDocument();
    expect(screen.getByText("Run RSS Sync")).toBeInTheDocument();
  });

  it("filters items on typing", async () => {
    const user = userEvent.setup();
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });

    const input = screen.getByTestId("command-palette-input");
    await user.type(input, "queue");

    // Queue should be visible, Dashboard should not
    expect(screen.getByText("Queue")).toBeInTheDocument();
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument();
  });

  it("calls onClose when Escape is pressed", async () => {
    const onClose = vi.fn();
    render(<CommandPalette onClose={onClose} />, { wrapper: createWrapper() });

    const input = screen.getByTestId("command-palette-input");
    fireEvent.keyDown(input, { key: "Escape" });

    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when backdrop is clicked", () => {
    const onClose = vi.fn();
    render(<CommandPalette onClose={onClose} />, { wrapper: createWrapper() });

    fireEvent.click(screen.getByTestId("command-palette-backdrop"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not close when palette panel is clicked", () => {
    const onClose = vi.fn();
    render(<CommandPalette onClose={onClose} />, { wrapper: createWrapper() });

    // Click on the dialog panel itself (not the backdrop)
    fireEvent.click(screen.getByRole("dialog"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("navigates through items with arrow keys", () => {
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });

    const input = screen.getByTestId("command-palette-input");

    // First item should be active (index 0 = Dashboard)
    expect(isHighlighted(screen.getByTestId("palette-item-nav:dashboard"))).toBe(true);

    // ArrowDown moves to next item (Activity is now second after Dashboard)
    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(isHighlighted(screen.getByTestId("palette-item-nav:activity"))).toBe(true);
    // First item should no longer be highlighted
    expect(isHighlighted(screen.getByTestId("palette-item-nav:dashboard"))).toBe(false);

    // ArrowUp moves back
    fireEvent.keyDown(input, { key: "ArrowUp" });
    expect(isHighlighted(screen.getByTestId("palette-item-nav:dashboard"))).toBe(true);
  });

  it("ArrowUp on first item stays at first item", () => {
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });

    const input = screen.getByTestId("command-palette-input");
    fireEvent.keyDown(input, { key: "ArrowUp" });

    const firstItem = screen.getByTestId("palette-item-nav:dashboard");
    expect(isHighlighted(firstItem)).toBe(true);
  });

  it("shows 'No results' when nothing matches", async () => {
    const user = userEvent.setup();
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });

    const input = screen.getByTestId("command-palette-input");
    // "9" matches no label or keyword; single char avoids movie search
    await user.type(input, "9");

    expect(screen.getByText("No results")).toBeInTheDocument();
  });

  it("searches movies when typing 2+ characters", async () => {
    server.use(
      http.post("/api/v1/movies/lookup", () =>
        HttpResponse.json([
          {
            tmdb_id: 550,
            title: "Fight Club",
            original_title: "Fight Club",
            overview: "",
            release_date: "1999-10-15",
            year: 1999,
            poster_path: null,
            backdrop_path: null,
            popularity: 50,
          },
        ])
      )
    );

    const user = userEvent.setup();
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });

    const input = screen.getByTestId("command-palette-input");
    await user.type(input, "fight");

    // Wait for debounce + API response
    await waitFor(
      () => {
        // "Fight Club" appears in both the poster placeholder and the row label
        expect(screen.getAllByText("Fight Club").length).toBeGreaterThanOrEqual(1);
      },
      { timeout: 2000 }
    );

    expect(screen.getByText("Movies")).toBeInTheDocument();
    expect(screen.getAllByText("1999").length).toBeGreaterThanOrEqual(1);
  });

  it("runs action on Enter and closes", async () => {
    server.use(
      http.post("/api/v1/tasks/rss_sync/run", () =>
        new HttpResponse(null, { status: 202 })
      )
    );

    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<CommandPalette onClose={onClose} />, { wrapper: createWrapper() });

    const input = screen.getByTestId("command-palette-input");
    await user.type(input, "rss sync");

    // The only visible item should be "Run RSS Sync"
    expect(screen.getByText("Run RSS Sync")).toBeInTheDocument();

    // Press Enter to select
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onClose).toHaveBeenCalledOnce();
  });

  it("locks body scroll when open", () => {
    const { unmount } = render(<CommandPalette onClose={() => {}} />, {
      wrapper: createWrapper(),
    });
    expect(document.body.style.overflow).toBe("hidden");

    unmount();
    expect(document.body.style.overflow).toBe("");
  });

  it("highlights item on mouse hover", () => {
    render(<CommandPalette onClose={() => {}} />, { wrapper: createWrapper() });

    const queueItem = screen.getByTestId("palette-item-nav:queue");
    fireEvent.mouseEnter(queueItem);

    expect(isHighlighted(queueItem)).toBe(true);

    // Dashboard should no longer be highlighted
    expect(isHighlighted(screen.getByTestId("palette-item-nav:dashboard"))).toBe(false);
  });
});
