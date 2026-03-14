import type { ReactNode } from "react";
import { describe, it, expect } from "vitest";
import { render, screen, waitFor, within, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import type { Quality, QualityProfile, QualityDefinition } from "@/types";
import QualityProfileList from "./QualityProfileList";

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

// ── Fixtures ──────────────────────────────────────────────────────────────────

// Popular definitions (shown in simple mode)
const def1080pBluray: QualityDefinition = { id: "1080p-bluray-x265-none", name: "1080p Bluray", resolution: "1080p", source: "bluray", codec: "x265", hdr: "none", min_size: 4, max_size: 95, preferred_size: 95, sort_order: 70 };
const def1080pWebdl: QualityDefinition = { id: "1080p-webdl-x264-none", name: "1080p WEBDL", resolution: "1080p", source: "webdl", codec: "x264", hdr: "none", min_size: 4, max_size: 40, preferred_size: 40, sort_order: 80 };
const def720pBluray: QualityDefinition = { id: "720p-bluray-x264-none", name: "720p Bluray", resolution: "720p", source: "bluray", codec: "x264", hdr: "none", min_size: 2, max_size: 30, preferred_size: 30, sort_order: 110 };
const def1080pRemux: QualityDefinition = { id: "1080p-remux-x265-none", name: "1080p Remux", resolution: "1080p", source: "remux", codec: "x265", hdr: "none", min_size: 15, max_size: 200, preferred_size: 200, sort_order: 60 };
const def2160pRemux: QualityDefinition = { id: "2160p-remux-x265-hdr10", name: "2160p Remux HDR", resolution: "2160p", source: "remux", codec: "x265", hdr: "hdr10", min_size: 35, max_size: 800, preferred_size: 800, sort_order: 10 };

// Non-popular definition (only visible in advanced mode)
const defSdHdtv: QualityDefinition = { id: "sd-hdtv-x264-none", name: "SD HDTV", resolution: "sd", source: "hdtv", codec: "x264", hdr: "none", min_size: 0.5, max_size: 2, preferred_size: 2, sort_order: 210 };

const popularDefs = [def1080pBluray, def1080pWebdl, def720pBluray, def1080pRemux, def2160pRemux];
const allDefs = [...popularDefs, defSdHdtv];

// Profile with qualities matching real DB data: codec/resolution/source often "unknown",
// name is the reliable identifier (e.g. "Bluray-1080p" matches def "1080p Bluray").
const q1080pBluray: Quality = { resolution: "1080p", source: "bluray", codec: "unknown", hdr: "none", name: "Bluray-1080p" };
const q720pBluray: Quality = { resolution: "720p", source: "bluray", codec: "unknown", hdr: "none", name: "Bluray-720p" };
// Remux stored with resolution/source = "unknown" — must match by name
const qRemux1080p: Quality = { resolution: "unknown", source: "unknown", codec: "unknown", hdr: "none", name: "Remux-1080p" };

const profileFixture: QualityProfile = {
  id: "qp-1",
  name: "HD Standard",
  cutoff: q1080pBluray,
  qualities: [q1080pBluray, q720pBluray, qRemux1080p],
  upgrade_allowed: true,
  upgrade_until: q1080pBluray,
};

// Profile with a non-popular quality (should auto-flip to advanced)
const qSdHdtv: Quality = { resolution: "sd", source: "hdtv", codec: "unknown", hdr: "none", name: "SDTV" };
const profileWithNonPopular: QualityProfile = {
  id: "qp-2",
  name: "Everything",
  cutoff: q1080pBluray,
  qualities: [q1080pBluray, qSdHdtv],
  upgrade_allowed: false,
};

describe("QualityProfileList", () => {
  it("shows empty state when no profiles exist", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });
    expect(screen.getByText(/Add a profile to control/)).toBeInTheDocument();
  });

  it("renders profiles from API", async () => {
    server.use(
      http.get("/api/v1/quality-profiles", () => HttpResponse.json([profileFixture])),
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("HD Standard")).toBeInTheDocument();
    });
    expect(screen.getByText("3")).toBeInTheDocument(); // qualities count
  });

  it("shows skeletons while loading", () => {
    server.use(
      http.get("/api/v1/quality-profiles", () => new Promise(() => {})),
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    const { container } = render(<QualityProfileList />, { wrapper: createWrapper() });

    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBe(3);
  });

  it("pre-checks existing qualities when editing (resilient to codec/hdr drift)", async () => {
    server.use(
      http.get("/api/v1/quality-profiles", () => HttpResponse.json([profileFixture])),
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("HD Standard")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Edit"));

    await waitFor(() => {
      expect(screen.getByText("Edit Profile")).toBeInTheDocument();
    });

    // Get the modal's quality table (second table on page)
    const tables = screen.getAllByRole("table");
    const modalTable = tables[tables.length - 1];
    const qualityCheckboxes = within(modalTable).getAllByRole("checkbox");
    expect(qualityCheckboxes).toHaveLength(5);

    // Fixture order: 1080p-bluray(70), 1080p-webdl(80), 720p-bluray(110), 1080p-remux(60), 2160p-remux(10)
    // Profile has: 1080p-bluray (field match), 720p-bluray (field match), Remux-1080p (name match)
    expect(qualityCheckboxes[0]).toBeChecked();       // 1080p Bluray — in profile (field match)
    expect(qualityCheckboxes[1]).not.toBeChecked();   // 1080p WEBDL — not in profile
    expect(qualityCheckboxes[2]).toBeChecked();       // 720p Bluray — in profile (field match)
    expect(qualityCheckboxes[3]).toBeChecked();       // 1080p Remux — in profile (name match: "Remux-1080p")
    expect(qualityCheckboxes[4]).not.toBeChecked();   // 2160p Remux HDR — not in profile
  });

  it("shows only popular qualities in simple mode (default)", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(allDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });

    await user.click(screen.getByText("+ Add Profile"));

    await waitFor(() => {
      expect(screen.getByText("Add Quality Profile")).toBeInTheDocument();
    });

    // Simple mode: should show 4 popular definitions, not 5
    const table = screen.getByRole("table");
    const checkboxes = within(table).getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(5);

    // SD HDTV should not be visible
    expect(screen.queryByText("SD HDTV")).not.toBeInTheDocument();

    // Codec/HDR columns should not be visible in simple mode
    expect(screen.queryByText("Codec")).not.toBeInTheDocument();
    expect(screen.queryByText("HDR")).not.toBeInTheDocument();
  });

  it("shows all qualities and extra columns in advanced mode", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(allDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });

    await user.click(screen.getByText("+ Add Profile"));

    await waitFor(() => {
      expect(screen.getByText("Add Quality Profile")).toBeInTheDocument();
    });

    // Toggle to advanced
    fireEvent.click(screen.getByRole("switch"));

    // All 6 definitions should be visible (5 popular + 1 non-popular)
    await waitFor(() => {
      const table = screen.getByRole("table");
      const checkboxes = within(table).getAllByRole("checkbox");
      expect(checkboxes).toHaveLength(6);
    });

    // SD HDTV should now be visible
    expect(screen.getByText("SD HDTV")).toBeInTheDocument();

    // Codec and HDR columns should be visible
    expect(screen.getByText("Codec")).toBeInTheDocument();
    expect(screen.getByText("HDR")).toBeInTheDocument();
  });

  it("auto-flips to advanced when editing a profile with non-popular qualities", async () => {
    server.use(
      http.get("/api/v1/quality-profiles", () => HttpResponse.json([profileWithNonPopular])),
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(allDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Everything")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Edit"));

    await waitFor(() => {
      expect(screen.getByText("Edit Profile")).toBeInTheDocument();
    });

    // Should auto-flip to advanced because profile contains SD HDTV
    // "SD HDTV" appears in both the quality table and the cutoff dropdown, so use getAllByText
    expect(screen.getAllByText("SD HDTV").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Codec")).toBeInTheDocument();
  });

  it("does not pre-check any qualities when creating a new profile", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });

    await user.click(screen.getByText("+ Add Profile"));

    await waitFor(() => {
      expect(screen.getByText("Add Quality Profile")).toBeInTheDocument();
    });

    const table = screen.getByRole("table");
    const checkboxes = within(table).getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(5);
    for (const cb of checkboxes) {
      expect(cb).not.toBeChecked();
    }
  });

  it("shows validation error when submitting without required fields", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });

    await user.click(screen.getByText("+ Add Profile"));

    await waitFor(() => {
      expect(screen.getByText("Add Quality Profile")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Add Profile"));

    expect(screen.getByText("Name is required.")).toBeInTheDocument();
  });

  it("toggles quality checkboxes and shows cutoff dropdown", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });

    await user.click(screen.getByText("+ Add Profile"));

    await waitFor(() => {
      expect(screen.getByText("Add Quality Profile")).toBeInTheDocument();
    });

    const table = screen.getByRole("table");
    const checkboxes = within(table).getAllByRole("checkbox");

    await user.click(checkboxes[0]);
    expect(checkboxes[0]).toBeChecked();

    await waitFor(() => {
      expect(screen.getByText("Cutoff (minimum) *")).toBeInTheDocument();
    });

    await user.click(checkboxes[0]);
    expect(checkboxes[0]).not.toBeChecked();
  });

  it("closes modal when Cancel is clicked", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });

    await user.click(screen.getByText("+ Add Profile"));

    await waitFor(() => {
      expect(screen.getByText("Add Quality Profile")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Cancel"));

    await waitFor(() => {
      expect(screen.queryByText("Add Quality Profile")).not.toBeInTheDocument();
    });
  });

  it("closes modal when Escape key is pressed", async () => {
    server.use(
      http.get("/api/v1/quality-definitions", () => HttpResponse.json(popularDefs)),
    );

    const user = userEvent.setup();
    render(<QualityProfileList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No quality profiles")).toBeInTheDocument();
    });

    await user.click(screen.getByText("+ Add Profile"));

    await waitFor(() => {
      expect(screen.getByText("Add Quality Profile")).toBeInTheDocument();
    });

    await user.keyboard("{Escape}");

    await waitFor(() => {
      expect(screen.queryByText("Add Quality Profile")).not.toBeInTheDocument();
    });
  });
});
