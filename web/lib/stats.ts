import type {
  AnalogdbImage,
  AnalogdbStats,
  AnalogdbStatsFilm,
  AnalogdbStatsPost,
  AnalogdbStatsTrendItem,
} from "analogdb-generated";
import { parseAsString, parseAsStringLiteral } from "nuqs/server";

export type Stats = AnalogdbStats;
export type StatsFilm = AnalogdbStatsFilm;
export type StatsPost = AnalogdbStatsPost;
export type TrendItem = AnalogdbStatsTrendItem;

// page state lives in the url, so a shared link keeps it
export const statsParsers = {
  nsfw: parseAsStringLiteral(["include", "exclude"] as const)
    .withDefault("exclude")
    .withOptions({ shallow: true, scroll: false }),
  film: parseAsString.withOptions({ shallow: true, scroll: false }),
};

export const lookSectionId = "look";

export const familyColors: Record<string, string> = {
  landscape: "var(--mantine-color-teal-6)",
  portrait: "var(--mantine-color-orange-6)",
  consumer: "var(--mantine-color-yellow-6)",
  night: "var(--mantine-color-indigo-5)",
  bw: "var(--mantine-color-gray-6)",
};

export const lineColors = [
  "var(--mantine-color-blue-6)",
  "var(--mantine-color-orange-6)",
  "var(--mantine-color-teal-6)",
  "var(--mantine-color-grape-6)",
  "var(--mantine-color-yellow-7)",
  "var(--mantine-color-red-6)",
  "var(--mantine-color-cyan-7)",
  "var(--mantine-color-gray-6)",
];

export const styleLabels: {
  key: keyof NonNullable<StatsFilm["styles"]>;
  label: string;
}[] = [
  { key: "vivid", label: "vivid" },
  { key: "tealOrange", label: "teal and orange" },
  { key: "redNight", label: "red night light" },
  { key: "grain", label: "grain" },
  { key: "crisp", label: "crisp" },
];

export function filmName(film: { make?: string; type?: string }): string {
  return `${film.make ?? ""} ${film.type ?? ""}`.trim().toUpperCase();
}

export function isColorFilm(film: StatsFilm): boolean {
  return film.colorType !== "bw";
}

// galleryHref links to the gallery filtered by a film and optionally a tag.
// B&W photos are hidden by default there, so B&W films turn them on.
export function galleryHref(film: StatsFilm, text?: string): string {
  const params = new URLSearchParams();
  if (film.make) params.set("film_make", film.make);
  if (film.type) params.set("film_type", film.type);
  if (text) params.set("text", text);
  if (!isColorFilm(film)) params.set("bw", "include");
  return `/?${params.toString()}`;
}

export function hueColor(hue: number, lightness = 0.7, chroma = 0.12): string {
  return `oklch(${lightness} ${chroma} ${hue})`;
}

export function pct(value: number | undefined, digits = 1): string {
  return `${(value ?? 0).toFixed(digits)}%`;
}

export function share(value: number | undefined): string {
  return `${Math.round((value ?? 0) * 100)}%`;
}

export function popularity(value: number | undefined): string {
  return (value ?? 0).toFixed(0);
}

export function lowImage(post: StatsPost): AnalogdbImage | undefined {
  const images = post.images ?? [];
  return images.find((image) => image.resolution === "low") ?? images[0];
}

const months = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

// monthLabel turns "2024-07" into "Jul 2024"
export function monthLabel(month: string | undefined): string {
  if (!month) return "";
  const [year, m] = month.split("-");
  return `${months[Number(m) - 1] ?? ""} ${year}`.trim();
}

// generated is a Date from the client, or a string once it went through the data cache
export function generatedDate(stats: Stats): Date | null {
  const value = stats.generated as Date | string | undefined;
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function windowStart(stats: Stats): Date | null {
  const from = stats.window?.from;
  if (!from) return null;
  const date = new Date(`${from}T00:00:00Z`);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function formatMonthYear(date: Date | null): string {
  if (!date) return "";
  return date.toLocaleDateString("en-US", {
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  });
}
