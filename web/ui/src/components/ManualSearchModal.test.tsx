import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { releaseFixture } from "@/test/fixtures";
import { ManualSearchModal } from "./ManualSearchModal";

const defaultProps = {
  movieId: "movie-1",
  movieTitle: "Fight Club",
  onClose: vi.fn(),
};

describe("ManualSearchModal", () => {
  it("shows loading skeletons while fetching", () => {
    // Delay the response so loading state is visible
    server.use(
      http.get("/api/v1/movies/movie-1/releases", async () => {
        await new Promise((r) => setTimeout(r, 200));
        return HttpResponse.json([]);
      })
    );

    const { container } = renderWithProviders(<ManualSearchModal {...defaultProps} />);
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBe(4);
  });

  it("shows movie title in header", () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([]))
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    expect(screen.getByText("Fight Club")).toBeInTheDocument();
    expect(screen.getByText("Manual Search")).toBeInTheDocument();
  });

  it("shows empty state when no releases found", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([]))
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("No releases found")).toBeInTheDocument();
    });
    expect(screen.getByText("Search Again")).toBeInTheDocument();
  });

  it("shows error state with retry button", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () =>
        HttpResponse.json({ title: "Search failed" }, { status: 500 })
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText(/Failed to search indexers/)).toBeInTheDocument();
    });
    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("renders release rows with details", async () => {
    const releases = [
      releaseFixture,
      { ...releaseFixture, guid: "release-2", title: "Fight.Club.1999.2160p.WEBDL", seeds: 10, peers: 5 },
    ];
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json(releases))
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("2 releases found")).toBeInTheDocument();
    });

    // Release titles displayed
    expect(screen.getByText(releaseFixture.title)).toBeInTheDocument();
    expect(screen.getByText("Fight.Club.1999.2160p.WEBDL")).toBeInTheDocument();

    // Grab buttons for both
    const grabButtons = screen.getAllByText("Grab");
    expect(grabButtons.length).toBe(2);
  });

  it("shows release count for single release", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([releaseFixture]))
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("1 release found")).toBeInTheDocument();
    });
  });

  it("calls onClose when backdrop is clicked", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([]))
    );

    const onClose = vi.fn();
    const { container } = renderWithProviders(<ManualSearchModal {...defaultProps} onClose={onClose} />);

    // The backdrop is the outermost fixed overlay div
    const backdrop = container.querySelector("[style*='position: fixed']")!;
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose when close button is clicked", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([]))
    );

    const onClose = vi.fn();
    renderWithProviders(<ManualSearchModal {...defaultProps} onClose={onClose} />);
    fireEvent.click(screen.getByText("✕"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose on Escape key", () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([]))
    );

    const onClose = vi.fn();
    renderWithProviders(<ManualSearchModal {...defaultProps} onClose={onClose} />);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("grabs a release and shows success state", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([releaseFixture])),
      http.post("/api/v1/movies/movie-1/releases/grab", () =>
        HttpResponse.json({ id: "grab-1", movie_id: "movie-1", grabbed_at: "2025-01-01T00:00:00Z" })
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("Grab")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Grab"));

    await waitFor(() => {
      expect(screen.getByText(/Grabbed/)).toBeInTheDocument();
    });
  });

  it("shows grab error message", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([releaseFixture])),
      http.post("/api/v1/movies/movie-1/releases/grab", () =>
        HttpResponse.json({ title: "No download client" }, { status: 500 })
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("Grab")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Grab"));

    await waitFor(() => {
      expect(screen.getByText("No download client")).toBeInTheDocument();
    });
  });
});
