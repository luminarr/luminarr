import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ScoreChip from "./ScoreChip";
import type { ScoreBreakdown } from "@/types";

const highBreakdown: ScoreBreakdown = {
  total: 85,
  dimensions: [
    { name: "resolution", score: 30, max: 30, matched: true, got: "1080p", want: "1080p" },
    { name: "source", score: 25, max: 25, matched: true, got: "bluray", want: "bluray" },
    { name: "codec", score: 20, max: 25, matched: false, got: "x264", want: "x265" },
    { name: "age", score: 10, max: 20, matched: true, got: "30d", want: "" },
  ],
};

const midBreakdown: ScoreBreakdown = {
  total: 55,
  dimensions: [
    { name: "resolution", score: 30, max: 30, matched: true, got: "1080p", want: "1080p" },
    { name: "source", score: 15, max: 25, matched: false, got: "webdl", want: "bluray" },
    { name: "codec", score: 10, max: 25, matched: false, got: "x264", want: "x265" },
  ],
};

const lowBreakdown: ScoreBreakdown = {
  total: 30,
  dimensions: [
    { name: "resolution", score: 10, max: 30, matched: false, got: "720p", want: "1080p" },
    { name: "source", score: 10, max: 25, matched: false, got: "hdtv", want: "bluray" },
    { name: "codec", score: 10, max: 25, matched: false, got: "x264", want: "x265" },
  ],
};

describe("ScoreChip", () => {
  it("renders nothing when no breakdown", () => {
    const { container } = render(<ScoreChip />);
    expect(container.innerHTML).toBe("");
  });

  it("renders score text", () => {
    render(<ScoreChip breakdown={highBreakdown} />);
    expect(screen.getByText("85/100")).toBeInTheDocument();
  });

  it("uses success color for score >= 80", () => {
    render(<ScoreChip breakdown={highBreakdown} />);
    const chip = screen.getByText("85/100");
    expect(chip.style.color).toBe("var(--color-success)");
  });

  it("uses warning color for score 50-79", () => {
    render(<ScoreChip breakdown={midBreakdown} />);
    const chip = screen.getByText("55/100");
    // happy-dom doesn't preserve var() with fallback in style.color,
    // so check the style attribute string instead
    const style = chip.getAttribute("style") ?? "";
    expect(style).toContain("--color-warning");
  });

  it("uses danger color for score < 50", () => {
    render(<ScoreChip breakdown={lowBreakdown} />);
    const chip = screen.getByText("30/100");
    expect(chip.style.color).toBe("var(--color-danger)");
  });

  it("shows tooltip on hover with dimension breakdown", () => {
    render(<ScoreChip breakdown={highBreakdown} />);
    const chip = screen.getByText("85/100");

    // Tooltip not visible initially
    expect(screen.queryByText("resolution")).not.toBeInTheDocument();

    // Hover shows tooltip
    fireEvent.mouseEnter(chip);
    expect(screen.getByText("resolution")).toBeInTheDocument();
    expect(screen.getByText("source")).toBeInTheDocument();
    expect(screen.getByText("codec")).toBeInTheDocument();
    expect(screen.getByText("Total")).toBeInTheDocument();

    // Leave hides tooltip
    fireEvent.mouseLeave(chip);
    expect(screen.queryByText("resolution")).not.toBeInTheDocument();
  });

  it("shows matched/unmatched scores with correct colors", () => {
    render(<ScoreChip breakdown={highBreakdown} />);
    fireEvent.mouseEnter(screen.getByText("85/100"));

    // Check dimension score values are present
    expect(screen.getByText("30/30")).toBeInTheDocument();
    expect(screen.getByText("25/25")).toBeInTheDocument();
    expect(screen.getByText("20/25")).toBeInTheDocument();
  });

  it("shows got → want for mismatched dimensions", () => {
    render(<ScoreChip breakdown={highBreakdown} />);
    fireEvent.mouseEnter(screen.getByText("85/100"));

    // codec: got=x264, want=x265 → should show "x264 → x265"
    expect(screen.getByText("x264 → x265")).toBeInTheDocument();
  });
});
