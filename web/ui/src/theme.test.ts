import { describe, it, expect, beforeEach } from "vitest";
import {
  getStoredMode,
  getStoredPreset,
  findPreset,
  resolveMode,
  setThemeMode,
  setThemePreset,
  getTooltipsEnabled,
  setTooltipsEnabled,
  THEME_PRESETS,
  DEFAULT_DARK_PRESET,
  DEFAULT_LIGHT_PRESET,
} from "./theme";

beforeEach(() => {
  localStorage.clear();
  // Reset inline styles on documentElement
  document.documentElement.style.cssText = "";
});

describe("getStoredMode", () => {
  it("defaults to dark when nothing stored", () => {
    expect(getStoredMode()).toBe("dark");
  });

  it("reads dark from localStorage", () => {
    localStorage.setItem("luminarr-theme-mode", "dark");
    expect(getStoredMode()).toBe("dark");
  });

  it("reads light from localStorage", () => {
    localStorage.setItem("luminarr-theme-mode", "light");
    expect(getStoredMode()).toBe("light");
  });

  it("reads system from localStorage", () => {
    localStorage.setItem("luminarr-theme-mode", "system");
    expect(getStoredMode()).toBe("system");
  });

  it("defaults to dark for invalid values", () => {
    localStorage.setItem("luminarr-theme-mode", "invalid");
    expect(getStoredMode()).toBe("dark");
  });
});

describe("getStoredPreset", () => {
  it("defaults to luminarr for dark mode", () => {
    expect(getStoredPreset("dark")).toBe(DEFAULT_DARK_PRESET);
  });

  it("defaults to catppuccin-latte for light mode", () => {
    expect(getStoredPreset("light")).toBe(DEFAULT_LIGHT_PRESET);
  });

  it("reads stored dark preset", () => {
    localStorage.setItem("luminarr-theme-dark", "dracula");
    expect(getStoredPreset("dark")).toBe("dracula");
  });

  it("reads stored light preset", () => {
    localStorage.setItem("luminarr-theme-light", "gruvbox-light");
    expect(getStoredPreset("light")).toBe("gruvbox-light");
  });
});

describe("findPreset", () => {
  it("finds an existing preset", () => {
    const preset = findPreset("dracula");
    expect(preset.id).toBe("dracula");
    expect(preset.label).toBe("Dracula");
  });

  it("falls back to first preset for unknown ID", () => {
    const preset = findPreset("nonexistent");
    expect(preset.id).toBe(THEME_PRESETS[0].id);
  });
});

describe("resolveMode", () => {
  it("returns dark for dark mode", () => {
    expect(resolveMode("dark")).toBe("dark");
  });

  it("returns light for light mode", () => {
    expect(resolveMode("light")).toBe("light");
  });

  // system mode depends on matchMedia which varies in test env
});

describe("setThemeMode", () => {
  it("persists mode to localStorage and applies theme", () => {
    setThemeMode("light");
    expect(localStorage.getItem("luminarr-theme-mode")).toBe("light");
    // Should have set CSS vars on documentElement
    const bgBase = document.documentElement.style.getPropertyValue("--color-bg-base");
    expect(bgBase).toBeTruthy();
  });
});

describe("setThemePreset", () => {
  it("persists preset for dark mode", () => {
    // First set mode to dark so the preset gets applied
    localStorage.setItem("luminarr-theme-mode", "dark");
    setThemePreset("dark", "nord");
    expect(localStorage.getItem("luminarr-theme-dark")).toBe("nord");
  });

  it("persists preset for light mode", () => {
    localStorage.setItem("luminarr-theme-mode", "light");
    setThemePreset("light", "gruvbox-light");
    expect(localStorage.getItem("luminarr-theme-light")).toBe("gruvbox-light");
  });

  it("applies CSS vars when preset matches active mode", () => {
    localStorage.setItem("luminarr-theme-mode", "dark");
    setThemePreset("dark", "dracula");
    const bgBase = document.documentElement.style.getPropertyValue("--color-bg-base");
    expect(bgBase).toBe("#1e1f29");
  });
});

describe("THEME_PRESETS", () => {
  it("has both dark and light presets", () => {
    const darkCount = THEME_PRESETS.filter((p) => p.mode === "dark").length;
    const lightCount = THEME_PRESETS.filter((p) => p.mode === "light").length;
    expect(darkCount).toBeGreaterThan(0);
    expect(lightCount).toBeGreaterThan(0);
  });

  it("all presets have unique IDs", () => {
    const ids = THEME_PRESETS.map((p) => p.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("all presets have all required CSS vars", () => {
    const requiredKeys = [
      "bg-base", "bg-surface", "bg-elevated", "bg-subtle",
      "border-subtle", "border-default", "border-strong",
      "accent", "accent-hover", "accent-muted", "accent-fg",
      "text-primary", "text-secondary", "text-muted",
      "success", "warning", "danger",
    ] as const;

    for (const preset of THEME_PRESETS) {
      for (const key of requiredKeys) {
        expect(
          preset.vars[key],
          `${preset.id} missing var ${key}`
        ).toBeTruthy();
      }
    }
  });
});

describe("tooltips", () => {
  it("defaults to enabled", () => {
    expect(getTooltipsEnabled()).toBe(true);
  });

  it("can be disabled", () => {
    setTooltipsEnabled(false);
    expect(getTooltipsEnabled()).toBe(false);
  });

  it("can be re-enabled", () => {
    setTooltipsEnabled(false);
    setTooltipsEnabled(true);
    expect(getTooltipsEnabled()).toBe(true);
  });
});
