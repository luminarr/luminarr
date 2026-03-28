import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Poster, PosterPlaceholder, placeholderHue } from "./Poster";

describe("placeholderHue", () => {
  it("returns a number between 0 and 359", () => {
    const hue = placeholderHue("Alien");
    expect(hue).toBeGreaterThanOrEqual(0);
    expect(hue).toBeLessThan(360);
  });

  it("is deterministic — same title always produces the same hue", () => {
    const a = placeholderHue("Inception");
    const b = placeholderHue("Inception");
    expect(a).toBe(b);
  });

  it("produces different hues for different titles", () => {
    const titles = [
      "Alien", "Inception", "The Matrix", "Dune", "Interstellar",
      "Blade Runner", "Avatar", "Jaws", "Rocky", "Titanic",
    ];
    const hues = new Set(titles.map(placeholderHue));
    expect(hues.size).toBeGreaterThanOrEqual(5);
  });

  it("handles empty string without throwing", () => {
    const hue = placeholderHue("");
    expect(hue).toBeGreaterThanOrEqual(0);
    expect(hue).toBeLessThan(360);
  });
});

describe("PosterPlaceholder", () => {
  it("renders movie title", () => {
    render(<PosterPlaceholder title="Alien" />);
    expect(screen.getByText("Alien")).toBeInTheDocument();
  });

  it("renders year when provided", () => {
    render(<PosterPlaceholder title="Alien" year={1979} />);
    expect(screen.getByText("1979")).toBeInTheDocument();
  });

  it("omits year when not provided", () => {
    render(<PosterPlaceholder title="Alien" />);
    expect(screen.queryByText(/\d{4}/)).not.toBeInTheDocument();
  });

  it("omits year when zero", () => {
    render(<PosterPlaceholder title="Alien" year={0} />);
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });

  it("has poster-placeholder test id", () => {
    render(<PosterPlaceholder title="Alien" />);
    expect(screen.getByTestId("poster-placeholder")).toBeInTheDocument();
  });

  it("sets aria-label to title", () => {
    render(<PosterPlaceholder title="Alien" />);
    expect(screen.getByLabelText("Alien")).toBeInTheDocument();
  });
});

describe("Poster", () => {
  it("renders img when src is provided", () => {
    render(<Poster src="https://example.com/poster.jpg" title="Alien" />);
    expect(screen.getByTestId("poster-img")).toBeInTheDocument();
    expect(screen.getByTestId("poster-img")).toHaveAttribute(
      "src",
      "https://example.com/poster.jpg"
    );
  });

  it("renders placeholder when src is undefined", () => {
    render(<Poster src={undefined} title="Alien" year={1979} />);
    expect(screen.getByTestId("poster-placeholder")).toBeInTheDocument();
    expect(screen.getByText("Alien")).toBeInTheDocument();
    expect(screen.getByText("1979")).toBeInTheDocument();
  });

  it("renders placeholder when src is null", () => {
    render(<Poster src={null} title="Alien" />);
    expect(screen.getByTestId("poster-placeholder")).toBeInTheDocument();
  });

  it("renders placeholder when src is empty string", () => {
    render(<Poster src="" title="Alien" />);
    expect(screen.getByTestId("poster-placeholder")).toBeInTheDocument();
  });

  it("renders placeholder when img fires onError", () => {
    render(<Poster src="https://example.com/broken.jpg" title="Alien" year={1979} />);
    // Initially renders img
    expect(screen.getByTestId("poster-img")).toBeInTheDocument();

    // Simulate load error
    fireEvent.error(screen.getByTestId("poster-img"));

    // Now should show placeholder
    expect(screen.getByTestId("poster-placeholder")).toBeInTheDocument();
    expect(screen.getByText("Alien")).toBeInTheDocument();
  });

  it("sets alt attribute to title on img", () => {
    render(<Poster src="https://example.com/poster.jpg" title="Alien" />);
    expect(screen.getByTestId("poster-img")).toHaveAttribute("alt", "Alien");
  });

  it("passes loading prop to img", () => {
    render(
      <Poster src="https://example.com/poster.jpg" title="Alien" loading="eager" />
    );
    expect(screen.getByTestId("poster-img")).toHaveAttribute("loading", "eager");
  });

  it("defaults to lazy loading", () => {
    render(<Poster src="https://example.com/poster.jpg" title="Alien" />);
    expect(screen.getByTestId("poster-img")).toHaveAttribute("loading", "lazy");
  });
});
