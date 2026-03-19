import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { releaseFixture } from "@/test/fixtures";
import type { Release } from "@/types";
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
    fireEvent.keyDown(document, { key: "Escape" });
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

  // ── Sorting tests ─────────────────────────────────────────────────────────

  function makeRelease(overrides: Partial<Release> & { guid: string; title: string }): Release {
    return { ...releaseFixture, ...overrides };
  }

  const sortableReleases: Release[] = [
    makeRelease({ guid: "r-low", title: "Low.Score", quality_score: 200, size: 5_000_000_000, seeds: 100, age_days: 10 }),
    makeRelease({ guid: "r-high", title: "High.Score", quality_score: 900, size: 1_000_000_000, seeds: 5, age_days: 500 }),
    makeRelease({ guid: "r-mid", title: "Mid.Score", quality_score: 500, size: 10_000_000_000, seeds: 50, age_days: 100 }),
  ];

  function getReleaseOrder(): string[] {
    // Each release title is in a monospace-font div — grab them in DOM order
    return Array.from(document.querySelectorAll("[style*='font-family: var(--font-family-mono)']"))
      .map((el) => el.textContent ?? "");
  }

  it("shows sort buttons when releases are loaded", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json(sortableReleases))
    );
    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => expect(screen.getByText("3 releases found")).toBeInTheDocument());

    expect(screen.getByLabelText("Sort by Seeds")).toBeInTheDocument();
    expect(screen.getByLabelText("Sort by Size")).toBeInTheDocument();
    expect(screen.getByLabelText("Sort by Age")).toBeInTheDocument();
  });

  it("sorts by seeds descending by default", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json(sortableReleases))
    );
    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => expect(screen.getByText("3 releases found")).toBeInTheDocument());

    const titles = getReleaseOrder();
    // Descending seeds: Low.Score (100) → Mid.Score (50) → High.Score (5)
    expect(titles).toEqual(["Low.Score", "Mid.Score", "High.Score"]);
  });

  it("toggles sort direction when clicking active sort field", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json(sortableReleases))
    );
    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => expect(screen.getByText("3 releases found")).toBeInTheDocument());

    // Click Seeds again to flip to ascending
    fireEvent.click(screen.getByLabelText("Sort by Seeds"));
    const titles = getReleaseOrder();
    // Ascending seeds: High.Score (5) → Mid.Score (50) → Low.Score (100)
    expect(titles).toEqual(["High.Score", "Mid.Score", "Low.Score"]);
  });

  it("sorts by size when Size button is clicked", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json(sortableReleases))
    );
    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => expect(screen.getByText("3 releases found")).toBeInTheDocument());

    fireEvent.click(screen.getByLabelText("Sort by Size"));
    const titles = getReleaseOrder();
    // Descending size: Mid (10GB) → Low (5GB) → High (1GB)
    expect(titles).toEqual(["Mid.Score", "Low.Score", "High.Score"]);
  });

  it("sorts by age when Age button is clicked", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json(sortableReleases))
    );
    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => expect(screen.getByText("3 releases found")).toBeInTheDocument());

    fireEvent.click(screen.getByLabelText("Sort by Age"));
    const titles = getReleaseOrder();
    // Descending age: High (500d) → Mid (100d) → Low (10d)
    expect(titles).toEqual(["High.Score", "Mid.Score", "Low.Score"]);
  });

  // ── Conflict pills ────────────────────────────────────────────────────────

  it("shows conflict warning pills when release has conflicts", async () => {
    const conflictRelease: Release = {
      ...releaseFixture,
      conflicts: [
        {
          dimension: "codec",
          severity: "warning",
          current: "x265",
          candidate: "x264",
          summary: "Codec downgrade",
        },
      ],
    };
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () =>
        HttpResponse.json([conflictRelease])
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("Codec downgrade")).toBeInTheDocument();
    });
  });

  it("does not show conflict section when no conflicts", async () => {
    const cleanRelease: Release = {
      ...releaseFixture,
      conflicts: [],
    };
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () =>
        HttpResponse.json([cleanRelease])
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("1 release found")).toBeInTheDocument();
    });
    // No conflict summary text should be present
    expect(screen.queryByText("Codec downgrade")).not.toBeInTheDocument();
    expect(screen.queryByText("downgrade")).not.toBeInTheDocument();
  });

  it("warning severity pill renders the TriangleAlert icon", async () => {
    const conflictRelease: Release = {
      ...releaseFixture,
      conflicts: [
        {
          dimension: "source",
          severity: "warning",
          current: "remux",
          candidate: "webdl",
          summary: "Source downgrade",
        },
      ],
    };
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () =>
        HttpResponse.json([conflictRelease])
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("Source downgrade")).toBeInTheDocument();
    });

    // The warning pill wraps both the icon SVG and the summary text inside the
    // same <span>. The lucide TriangleAlert renders as an <svg> element.
    const pill = screen.getByText("Source downgrade").closest("span")!;
    expect(pill.querySelector("svg")).not.toBeNull();
  });

  it("caution severity pill does not render the TriangleAlert icon", async () => {
    const conflictRelease: Release = {
      ...releaseFixture,
      conflicts: [
        {
          dimension: "hdr",
          severity: "caution",
          current: "HDR10",
          candidate: "none",
          summary: "HDR difference",
        },
      ],
    };
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () =>
        HttpResponse.json([conflictRelease])
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText("HDR difference")).toBeInTheDocument();
    });

    // Caution pills omit the icon — no SVG inside the pill span
    const pill = screen.getByText("HDR difference").closest("span")!;
    expect(pill.querySelector("svg")).toBeNull();
  });

  // ── Explain panel (Why? button) ───────────────────────────────────────────

  it("Why button is visible in the modal header", () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([]))
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    expect(screen.getByText("Why?")).toBeInTheDocument();
  });

  it("clicking Why? shows the decision panel with profile name", async () => {
    const explainData = {
      movie_id: "movie-1",
      profile_name: "HD-1080p",
      current_file: {
        resolution: "720p",
        source: "webdl",
        codec: "x264",
        hdr: "",
        name: "WEBDL-720p",
      },
      decisions: [
        {
          title: "Fight.Club.1999.1080p.BluRay.x264-GROUP",
          guid: "release-1",
          outcome: "grabbed" as const,
          reason: "score_above_cutoff",
          explanation: "Score 850 exceeds cutoff of 500",
          quality_score: 850,
          cf_score: 0,
        },
      ],
    };

    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([])),
      http.get("/api/v1/movies/movie-1/releases/explain", () =>
        HttpResponse.json(explainData)
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    fireEvent.click(screen.getByText("Why?"));

    await waitFor(() => {
      expect(screen.getByText(/Profile: HD-1080p/)).toBeInTheDocument();
    });
    // Decision count summary is shown in the collapsed header
    expect(screen.getByText("1 decision")).toBeInTheDocument();
  });

  it("expanding the decision panel shows individual decision rows", async () => {
    const explainData = {
      movie_id: "movie-1",
      profile_name: "HD-1080p",
      decisions: [
        {
          title: "Fight.Club.1999.1080p.BluRay.x264-GROUP",
          guid: "release-1",
          outcome: "grabbed" as const,
          reason: "score_above_cutoff",
          explanation: "Score 850 exceeds cutoff of 500",
          quality_score: 850,
          cf_score: 0,
        },
        {
          title: "Fight.Club.1999.720p.HDTV-GROUP",
          guid: "release-2",
          outcome: "skipped" as const,
          reason: "below_cutoff",
          explanation: "Score 200 is below cutoff",
          quality_score: 200,
          cf_score: 0,
        },
      ],
    };

    server.use(
      http.get("/api/v1/movies/movie-1/releases", () => HttpResponse.json([])),
      http.get("/api/v1/movies/movie-1/releases/explain", () =>
        HttpResponse.json(explainData)
      )
    );

    renderWithProviders(<ManualSearchModal {...defaultProps} />);
    fireEvent.click(screen.getByText("Why?"));

    await waitFor(() => {
      expect(screen.getByText("2 decisions")).toBeInTheDocument();
    });

    // Click the collapsible header to expand
    fireEvent.click(screen.getByText("2 decisions").closest("button")!);

    await waitFor(() => {
      expect(screen.getByText("Score 850 exceeds cutoff of 500")).toBeInTheDocument();
    });
    expect(screen.getByText("Score 200 is below cutoff")).toBeInTheDocument();
    // Both outcome badges
    expect(screen.getByText("grabbed")).toBeInTheDocument();
    expect(screen.getByText("skipped")).toBeInTheDocument();
  });
});
