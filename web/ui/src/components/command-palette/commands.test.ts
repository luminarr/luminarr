import { describe, it, expect } from "vitest";
import {
  NAV_COMMANDS,
  ACTION_COMMANDS,
  filterCommands,
  fuzzyScore,
} from "./commands";

describe("NAV_COMMANDS", () => {
  it("has unique IDs", () => {
    const ids = NAV_COMMANDS.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("all have navigation category", () => {
    for (const cmd of NAV_COMMANDS) {
      expect(cmd.category).toBe("navigation");
    }
  });

  it("covers expected routes", () => {
    const ids = NAV_COMMANDS.map((c) => c.id);
    expect(ids).toContain("nav:dashboard");
    expect(ids).toContain("nav:queue");
    expect(ids).toContain("nav:system");
    expect(ids).toContain("nav:indexers");
  });
});

describe("ACTION_COMMANDS", () => {
  it("has unique IDs", () => {
    const ids = ACTION_COMMANDS.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("all have a taskName", () => {
    for (const cmd of ACTION_COMMANDS) {
      expect(cmd.taskName).toBeTruthy();
    }
  });
});

describe("fuzzyScore", () => {
  it("returns 0 for empty query", () => {
    expect(fuzzyScore("", "anything")).toBe(0);
  });

  it("returns -1 when query is longer than text", () => {
    expect(fuzzyScore("longquery", "short")).toBe(-1);
  });

  it("returns -1 when characters are not all present in order", () => {
    expect(fuzzyScore("xyz", "hello")).toBe(-1);
  });

  it("matches exact text", () => {
    expect(fuzzyScore("queue", "Queue")).toBeGreaterThan(0);
  });

  it("matches substring", () => {
    expect(fuzzyScore("settings", "App Settings")).toBeGreaterThan(0);
  });

  it("matches fuzzy (non-contiguous characters)", () => {
    expect(fuzzyScore("qp", "Quality Profiles")).toBeGreaterThan(0);
    expect(fuzzyScore("dwcl", "Download Clients")).toBeGreaterThan(0);
  });

  it("scores word-boundary matches higher", () => {
    // "qp" matches Q(uality) P(rofiles) at boundaries vs scattered chars
    const boundaryScore = fuzzyScore("qp", "Quality Profiles");
    const scatteredScore = fuzzyScore("qp", "qxxxxxxxpx");
    expect(boundaryScore).toBeGreaterThan(scatteredScore);
  });

  it("scores consecutive matches higher", () => {
    const consecutive = fuzzyScore("set", "Settings");
    const scattered = fuzzyScore("set", "sxexxt");
    expect(consecutive).toBeGreaterThan(scattered);
  });
});

describe("filterCommands", () => {
  it("returns all commands when query is empty", () => {
    expect(filterCommands(NAV_COMMANDS, "")).toEqual(NAV_COMMANDS);
  });

  it("matches by label (case-insensitive)", () => {
    const result = filterCommands(NAV_COMMANDS, "dash");
    expect(result.some((c) => c.id === "nav:dashboard")).toBe(true);
  });

  it("matches by keyword", () => {
    const result = filterCommands(NAV_COMMANDS, "torznab");
    expect(result.some((c) => c.id === "nav:indexers")).toBe(true);
  });

  it("returns empty array when nothing matches", () => {
    expect(filterCommands(NAV_COMMANDS, "zzzznothing")).toEqual([]);
  });

  it("matches partial keyword", () => {
    const result = filterCommands(ACTION_COMMANDS, "rss");
    expect(result.some((c) => c.id === "action:rss-sync")).toBe(true);
  });

  it("is case-insensitive for keywords", () => {
    const result = filterCommands(NAV_COMMANDS, "QUEUE");
    expect(result.some((c) => c.id === "nav:queue")).toBe(true);
  });

  it("fuzzy-matches 'settings' to 'App Settings'", () => {
    const result = filterCommands(NAV_COMMANDS, "settings");
    expect(result.some((c) => c.id === "nav:app-settings")).toBe(true);
  });

  it("ranks label matches above keyword-only matches", () => {
    const result = filterCommands(NAV_COMMANDS, "settings");
    // "App Settings" has "settings" in the label — should rank above items
    // that only match via the "settings" keyword
    const appIdx = result.findIndex((c) => c.id === "nav:app-settings");
    const libIdx = result.findIndex((c) => c.id === "nav:libraries");
    expect(appIdx).toBeLessThan(libIdx);
  });

  it("fuzzy-matches abbreviations across word boundaries", () => {
    const result = filterCommands(NAV_COMMANDS, "qp");
    expect(result.some((c) => c.id === "nav:quality-profiles")).toBe(true);
  });

  it("sorts results by match quality", () => {
    // "queue" should be an exact label match and rank first
    const result = filterCommands(NAV_COMMANDS, "queue");
    expect(result[0].id).toBe("nav:queue");
  });
});
