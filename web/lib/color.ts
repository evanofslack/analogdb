import { AnalogdbColor } from "analogdb-generated";

export const colorOptions = [
  {
    id: "red",
    label: "red",
    bg: "#f03e3e",
    tooltipColor: "red.7",
    checkedColor: "red.8",
    colorProp: "red.8",
  },
  {
    id: "orange",
    label: "orange",
    bg: "#fd7e14",
    tooltipColor: "orange.6",
    checkedColor: "orange.7",
    colorProp: "orange.7",
  },
  {
    id: "tan",
    label: "beige",
    bg: "#ffdcb0",
    tooltipColor: "brown.1",
    checkedColor: "brown.2",
    colorProp: "brown.2",
  },
  {
    id: "yellow",
    label: "yellow",
    bg: "#ffd43b",
    tooltipColor: "yellow.4",
    checkedColor: "yellow.5",
    colorProp: "yellow.5",
  },
  {
    id: "green",
    label: "green",
    bg: "#2f9e44",
    tooltipColor: "green.8",
    checkedColor: "green.9",
    colorProp: "green.9",
  },
  {
    id: "olive",
    label: "olive",
    bg: "#4c4d00",
    tooltipColor: "olive.8",
    checkedColor: "olive.9",
    colorProp: "olive.9",
  },
  {
    id: "teal",
    label: "teal",
    bg: "#22b8cf",
    tooltipColor: "cyan.5",
    checkedColor: "cyan.6",
    colorProp: "cyan.6",
  },
  {
    id: "navy",
    label: "blue",
    bg: "#1c7ed6",
    tooltipColor: "blue.7",
    checkedColor: "blue.8",
    colorProp: "blue.8",
  },
  {
    id: "purple",
    label: "purple",
    bg: "#9c36b5",
    tooltipColor: "grape.8",
    checkedColor: "grape.9",
    colorProp: "grape.9",
  },
  {
    id: "pink",
    label: "pink",
    bg: "#f06595",
    tooltipColor: "pink.5",
    checkedColor: "pink.6",
    colorProp: "pink.6",
  },
  {
    id: "gray",
    label: "gray",
    bg: "#868e96",
    tooltipColor: "dark.2",
    checkedColor: "dark.3",
    colorProp: "dark.3",
  },
  {
    id: "brown",
    label: "brown",
    bg: "#7d4500",
    tooltipColor: "brown.7",
    checkedColor: "brown.8",
    colorProp: "brown.8",
  },
  {
    id: "black",
    label: "black",
    bg: "#141517",
    outline: "var(--adb-swatch-black-outline)",
    tooltipColor: "dark.8",
    checkedColor: "dark.9",
    colorProp: "dark.9",
  },
  {
    id: "white",
    label: "white",
    bg: "#e9ecef",
    outline: "var(--adb-swatch-white-outline)",
    tooltipColor: "gray.2",
    checkedColor: "gray.3",
    colorProp: "gray.3",
  },
];

const colorOptionById = new Map(
  colorOptions.map((option) => [option.id, option])
);

// a color query value is a bucket name or a 6 digit hex, # optional
export function colorSwatch(value: string): { bg: string; outline?: string } {
  const option = colorOptionById.get(value.toLowerCase());
  if (option) return { bg: option.bg, outline: option.outline };
  return { bg: `#${bareHex(value)}` };
}

export function colorLabel(value: string): string {
  const option = colorOptionById.get(value.toLowerCase());
  return option ? option.label : `#${bareHex(value)}`;
}

export function isColorName(value: string): boolean {
  return colorOptionById.has(value.toLowerCase());
}

export function bareHex(hex: string): string {
  return hex.replace(/^#/, "").toLowerCase();
}

export type Lab = { l: number; a: number; b: number };

const NEUTRAL_CHROMA = 0.04;
const DISTINCT_DELTA_E = 0.09;
const PALETTE_SIZE = 3;
const MIN_CHROMATIC = 2;

function toLinear(c: number): number {
  const v = c / 255;
  return v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
}

export function hexToLab(hex: string): Lab | null {
  const value = bareHex(hex);
  if (!/^[0-9a-f]{6}$/.test(value)) return null;
  const n = parseInt(value, 16);
  const r = toLinear((n >> 16) & 255);
  const g = toLinear((n >> 8) & 255);
  const b = toLinear(n & 255);
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
  return {
    l: 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    a: 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s,
    b: 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s,
  };
}

export function deltaE(x: Lab, y: Lab): number {
  return Math.hypot(x.l - y.l, x.a - y.a, x.b - y.b);
}

export function chroma(lab: Lab): number {
  return Math.hypot(lab.a, lab.b);
}

// OKLCH hue in degrees from 0 to 360
export function hue(lab: Lab): number {
  return ((Math.atan2(lab.b, lab.a) * 180) / Math.PI + 360) % 360;
}

type Swatch = { hex: string; percent: number; lab: Lab };

function swatches(colors: AnalogdbColor[] | undefined): Swatch[] {
  return (colors ?? [])
    .map((color) => ({
      hex: bareHex(color.hex ?? ""),
      percent: color.percent ?? 0,
      lab: hexToLab(color.hex ?? ""),
    }))
    .filter((s): s is Swatch => s.lab !== null)
    .sort((a, b) => b.percent - a.percent);
}

// up to three distinct hexes of a palette, largest first. Chromatic colors
// come first and neutrals only fill in when fewer than two are chromatic
export function similarPalette(colors: AnalogdbColor[] | undefined): string[] {
  const all = swatches(colors);
  const chromatic = all.filter((s) => chroma(s.lab) >= NEUTRAL_CHROMA);
  const neutral = all.filter((s) => chroma(s.lab) < NEUTRAL_CHROMA);

  const picked: Swatch[] = [];
  const pick = (pool: Swatch[], limit: number) => {
    for (const s of pool) {
      if (picked.length >= limit) return;
      if (picked.every((p) => deltaE(p.lab, s.lab) >= DISTINCT_DELTA_E)) {
        picked.push(s);
      }
    }
  };
  pick(chromatic, PALETTE_SIZE);
  if (picked.length < MIN_CHROMATIC) pick(neutral, PALETTE_SIZE);
  return picked.map((s) => s.hex);
}

// hue of the most colorful large cluster, null for a grayscale palette
export function representativeHue(
  colors: AnalogdbColor[] | undefined
): number | null {
  let best: Swatch | null = null;
  for (const s of swatches(colors)) {
    const c = chroma(s.lab);
    if (c < NEUTRAL_CHROMA) continue;
    if (!best || s.percent * c > best.percent * chroma(best.lab)) best = s;
  }
  return best ? hue(best.lab) : null;
}
