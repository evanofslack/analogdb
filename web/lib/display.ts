// Gallery display preferences, stored in a cookie so the server renders the
// chosen layout and nothing moves on hydration

export const displayCookie = "adb_display";

export const spacingOptions = ["none", "tight", "normal", "loose"] as const;
export const gearOptions = ["off", "hover", "always"] as const;

export type Spacing = (typeof spacingOptions)[number];
export type Gear = (typeof gearOptions)[number];
export type Columns = "auto" | number;

export interface Display {
  columns: Columns;
  spacing: Spacing;
  gear: Gear;
}

export const defaultDisplay: Display = {
  columns: "auto",
  spacing: "normal",
  gear: "off",
};

export const spacingPx: Record<Spacing, number> = {
  none: 0,
  tight: 4,
  normal: 15,
  loose: 32,
};

// Height of the gear line under each image when it is always shown
export const gearPx = 24;

export type WidthClass = "mobile" | "tablet" | "desktop";

// Same breakpoints as the grid has always used
export const widthQueries = {
  mobile: "(max-width: 720px)",
  tablet: "(max-width: 1440px)",
};

const columnChoices: Record<WidthClass, number[]> = {
  mobile: [1, 2, 3],
  tablet: [2, 3, 4, 5],
  desktop: [2, 3, 4, 5, 6],
};

const autoColumns: Record<WidthClass, number> = {
  mobile: 2,
  tablet: 3,
  desktop: 4,
};

// Rough content width per class, used to turn pixel gaps into image ratios
// for masonry balancing. Fixed so server and client place images the same
const classWidth: Record<WidthClass, number> = {
  mobile: 360,
  tablet: 1060,
  desktop: 1780,
};

export function columnOptions(width: WidthClass): number[] {
  return columnChoices[width];
}

export function resolveColumns(display: Display, width: WidthClass): number {
  const { columns } = display;
  if (columns !== "auto" && columnChoices[width].includes(columns)) {
    return columns;
  }
  return autoColumns[width];
}

export function deviceClass(device: { type?: string }): WidthClass {
  if (device.type === "mobile") return "mobile";
  if (device.type === "tablet") return "tablet";
  return "desktop";
}

export function columnWidth(width: WidthClass, columns: number): number {
  return classWidth[width] / columns;
}

export function isDefault(display: Display): boolean {
  return (
    display.columns === defaultDisplay.columns &&
    display.spacing === defaultDisplay.spacing &&
    display.gear === defaultDisplay.gear
  );
}

// Stored as columns.spacing.gear, e.g. "4.tight.hover"
export function serializeDisplay(display: Display): string {
  return `${display.columns}.${display.spacing}.${display.gear}`;
}

export function parseDisplay(raw: string | null | undefined): Display {
  const [columns, spacing, gear] = (raw ?? "").split(".");
  const count = Number(columns);
  return {
    columns:
      Number.isInteger(count) && count >= 1 && count <= 6
        ? count
        : defaultDisplay.columns,
    spacing: spacingOptions.includes(spacing as Spacing)
      ? (spacing as Spacing)
      : defaultDisplay.spacing,
    gear: gearOptions.includes(gear as Gear)
      ? (gear as Gear)
      : defaultDisplay.gear,
  };
}
