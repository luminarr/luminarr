import { describe, it, expect } from "vitest";
import { cn, formatBytes, formatDuration } from "./utils";

describe("cn", () => {
  it("joins class names", () => {
    expect(cn("a", "b", "c")).toBe("a b c");
  });

  it("filters falsy values", () => {
    expect(cn("a", false, null, undefined, "b")).toBe("a b");
  });

  it("returns empty string for no inputs", () => {
    expect(cn()).toBe("");
  });
});

describe("formatBytes", () => {
  it("formats 0 bytes", () => {
    expect(formatBytes(0)).toBe("0 B");
  });

  it("formats bytes", () => {
    expect(formatBytes(500)).toBe("500 B");
  });

  it("formats kilobytes", () => {
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
  });

  it("formats megabytes", () => {
    expect(formatBytes(1_048_576)).toBe("1 MB");
  });

  it("formats gigabytes", () => {
    expect(formatBytes(1_073_741_824)).toBe("1 GB");
    expect(formatBytes(8_589_934_592)).toBe("8 GB");
  });

  it("formats terabytes", () => {
    expect(formatBytes(1_099_511_627_776)).toBe("1 TB");
  });
});

describe("formatDuration", () => {
  it("formats minutes only", () => {
    expect(formatDuration(120)).toBe("2m");
    expect(formatDuration(0)).toBe("0m");
    expect(formatDuration(59)).toBe("0m");
  });

  it("formats hours and minutes", () => {
    expect(formatDuration(3600)).toBe("1h 0m");
    expect(formatDuration(5400)).toBe("1h 30m");
  });

  it("formats days and hours", () => {
    expect(formatDuration(86400)).toBe("1d 0h");
    expect(formatDuration(90000)).toBe("1d 1h");
  });
});
