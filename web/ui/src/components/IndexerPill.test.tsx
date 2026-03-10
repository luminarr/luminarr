import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import IndexerPill from "./IndexerPill";

describe("IndexerPill", () => {
  it("renders the indexer name", () => {
    render(<IndexerPill name="NZBgeek" />);
    expect(screen.getByText("NZBgeek")).toBeInTheDocument();
  });

  it("produces consistent hue for the same name", () => {
    const { container: c1 } = render(<IndexerPill name="Prowlarr" />);
    const { container: c2 } = render(<IndexerPill name="Prowlarr" />);

    const bg1 = (c1.firstChild as HTMLElement).style.background;
    const bg2 = (c2.firstChild as HTMLElement).style.background;
    expect(bg1).toBe(bg2);
  });

  it("produces different hues for different names", () => {
    const { container: c1 } = render(<IndexerPill name="NZBgeek" />);
    const { container: c2 } = render(<IndexerPill name="Jackett" />);

    const bg1 = (c1.firstChild as HTMLElement).style.background;
    const bg2 = (c2.firstChild as HTMLElement).style.background;
    expect(bg1).not.toBe(bg2);
  });

  it("applies inline styles for pill appearance", () => {
    render(<IndexerPill name="TorrentLeech" />);
    const el = screen.getByText("TorrentLeech");
    expect(el.style.display).toBe("inline-block");
    expect(el.style.borderRadius).toBe("4px");
    expect(el.style.fontSize).toBe("10px");
  });
});
