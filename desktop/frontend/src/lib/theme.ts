/**
 * Theme utilities for accent color and brightness control.
 * These functions dynamically set CSS variables on :root based on
 * user preferences, overriding theme defaults.
 */

/* ─── Per-theme baseline HSL values for background surfaces ─── */
export const THEME_BASELINES: Record<string, Record<string, string>> = {
  dark: {
    "--background": "230 20% 5%",
    "--foreground": "210 40% 96%",
    "--card": "230 15% 9%",
    "--popover": "230 15% 9%",
    "--secondary": "230 12% 14%",
    "--muted": "230 12% 14%",
    "--accent": "230 12% 18%",
    "--border": "230 10% 18%",
    "--input": "230 10% 18%",
    "--scrollbar-thumb": "230 10% 22%",
    "--scrollbar-track": "230 20% 4%",
  },
  catppuccin: {
    "--background": "240 21% 15%",
    "--foreground": "226 64% 88%",
    "--card": "240 21% 12%",
    "--popover": "240 21% 12%",
    "--secondary": "240 16% 18%",
    "--muted": "240 16% 18%",
    "--accent": "240 16% 23%",
    "--border": "240 16% 21%",
    "--input": "240 16% 21%",
    "--scrollbar-thumb": "240 16% 25%",
    "--scrollbar-track": "240 21% 12%",
  },
  tokyonight: {
    "--background": "235 18% 13%",
    "--foreground": "234 61% 86%",
    "--card": "237 16% 18%",
    "--popover": "237 16% 18%",
    "--secondary": "235 18% 18%",
    "--muted": "235 18% 18%",
    "--accent": "235 18% 23%",
    "--border": "234 14% 25%",
    "--input": "234 14% 25%",
    "--scrollbar-thumb": "234 14% 30%",
    "--scrollbar-track": "235 18% 13%",
  },
  light: {
    "--background": "0 0% 100%",
    "--foreground": "222 47% 11%",
    "--card": "0 0% 100%",
    "--popover": "0 0% 100%",
    "--secondary": "210 40% 96%",
    "--muted": "210 40% 96%",
    "--accent": "210 40% 92%",
    "--border": "214 32% 91%",
    "--input": "214 32% 91%",
    "--scrollbar-thumb": "214 32% 80%",
    "--scrollbar-track": "210 40% 96%",
  },
  dracula: {
    "--background": "231 15% 18%",
    "--foreground": "60 30% 96%",
    "--card": "232 14% 25%",
    "--popover": "232 14% 25%",
    "--secondary": "231 15% 22%",
    "--muted": "231 15% 22%",
    "--accent": "231 14% 28%",
    "--border": "231 14% 28%",
    "--input": "231 14% 28%",
    "--scrollbar-thumb": "231 14% 32%",
    "--scrollbar-track": "231 15% 18%",
  },
};

/* ─── Preset accent colours ─── */
export const ACCENT_PRESETS = [
  { name: "Green", hex: "#22C55E" },
  { name: "Emerald", hex: "#10B981" },
  { name: "Blue", hex: "#3B82F6" },
  { name: "Indigo", hex: "#6366F1" },
  { name: "Purple", hex: "#8B5CF6" },
  { name: "Pink", hex: "#EC4899" },
  { name: "Rose", hex: "#F43F5E" },
  { name: "Red", hex: "#EF4444" },
  { name: "Orange", hex: "#F97316" },
  { name: "Yellow", hex: "#EAB308" },
  { name: "Cyan", hex: "#06B6D4" },
  { name: "Teal", hex: "#14B8A6" },
];

/* ─── Public API ─── */

/** Convert a hex colour string (e.g. "#22C55E") to HSL values. */
export function hexToHSL(hex: string): { h: number; s: number; l: number } | null {
  const clean = hex.replace("#", "");
  if (!/^[0-9a-fA-F]{6}$/.test(clean)) return null;

  const r = parseInt(clean.slice(0, 2), 16) / 255;
  const g = parseInt(clean.slice(2, 4), 16) / 255;
  const b = parseInt(clean.slice(4, 6), 16) / 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h = 0;
  let s = 0;
  const l = (max + min) / 2;

  if (max !== min) {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
    switch (max) {
      case r: h = ((g - b) / d + (g < b ? 6 : 0)) / 6; break;
      case g: h = ((b - r) / d + 2) / 6; break;
      case b: h = ((r - g) / d + 4) / 6; break;
    }
  }

  return { h: Math.round(h * 360), s: Math.round(s * 100), l: Math.round(l * 100) };
}

/** Validate a hex colour string. */
export function isValidHex(hex: string): boolean {
  return /^#[0-9a-fA-F]{6}$/.test(hex);
}

/** Set CSS variables on :root to override the theme's primary colour. */
export function applyAccentColor(hex: string): void {
  const root = document.documentElement;
  // Clear override if no accent is set
  if (!hex) {
    for (const v of ["--primary", "--ring", "--primary-foreground"]) {
      root.style.removeProperty(v);
    }
    return;
  }

  const hsl = hexToHSL(hex);
  if (!hsl) return;

  const { h, s, l } = hsl;
  const hslStr = `${h} ${s}% ${l}%`;

  // Determine foreground colour (white for dark accents, dark for light)
  const fgL = l > 50 ? 222 : 0;
  const fgPct = l > 50 ? 47 : 0;
  const fgStr = `${fgL} ${fgPct}% ${fgL > 50 ? 11 : 100}%`;

  root.style.setProperty("--primary", hslStr);
  root.style.setProperty("--ring", hslStr);
  root.style.setProperty("--primary-foreground", fgStr);
}

/** Compute and apply brightness-adjusted background HSL values. */
export function applyBrightness(brightness: number, theme: string): void {
  const root = document.documentElement;

  if (brightness < 0 || brightness > 100) return;

  // brightness: 0–100, 50 = default (no change)
  // offset: -8 to +8 percentage points on the L component
  const offset = ((brightness - 50) / 50) * 8;

  const baselines = THEME_BASELINES[theme];
  if (!baselines) return;

  for (const [varName, hslVal] of Object.entries(baselines)) {
    const parts = hslVal.trim().split(/\s+/);
    if (parts.length < 3) continue;

    const h = parts[0];
    const s = parts[1];
    const lVal = parseFloat(parts[2]);
    if (isNaN(lVal)) continue;

    const adjustedL = Math.max(0, Math.min(100, lVal + offset));
    root.style.setProperty(varName, `${h} ${s} ${adjustedL}%`);
  }
}
