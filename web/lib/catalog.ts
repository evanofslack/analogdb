import { smallImage } from "@lib/images";

export type CatalogKind = "films" | "cameras";

export type CatalogCover = {
  id: number;
  title: string;
  url: string;
  phoneUrl?: string;
  thumbUrl?: string;
  width: number;
  height: number;
};

export type CatalogEntry = {
  slug: string;
  make: string;
  name: string;
  postCount: number;
  description: string;
  speed?: number;
  colorType?: string;
  cover: CatalogCover | null;
  metaCover?: CatalogCover | null;
};

type CatalogImage = {
  resolution?: string;
  url?: string;
  width?: number;
  height?: number;
};

type CatalogPost = {
  id?: number;
  title?: string;
  score?: number;
  images?: CatalogImage[];
};

export const minPosts = 5;

const cardRatio = 4 / 5;
const ratioTolerance = 0.15;
const minCoverWidth = 600;

export function catalogName(entry: { make: string; name: string }): string {
  return `${entry.make} ${entry.name}`.toUpperCase();
}

export function catalogHref(kind: CatalogKind, entry: { slug: string }) {
  return `/${kind}/${entry.slug}`;
}

export function findBySlug(
  list: CatalogEntry[],
  slug: string
): CatalogEntry | null {
  return list.find((entry) => entry.slug === slug) ?? null;
}

export function findByName(
  list: CatalogEntry[],
  make?: string,
  name?: string
): CatalogEntry | null {
  if (!make || !name) return null;
  return (
    list.find((entry) => entry.make === make && entry.name === name) ?? null
  );
}

function toCover(post: CatalogPost): CatalogCover | null {
  const image = post.images?.find((img) => img.resolution === "medium");
  if (!image?.url || !image.width || !image.height) return null;
  return {
    id: post.id,
    title: post.title ?? "",
    url: image.url,
    phoneUrl: smallImage(post.images)?.url,
    thumbUrl: post.images?.find((img) => img.resolution === "low")?.url,
    width: image.width,
    height: image.height,
  };
}

function ratioDistance(cover: CatalogCover): number {
  return Math.abs(Math.log(cover.width / cover.height / cardRatio));
}

function coverCandidates(topPosts: CatalogPost[] = []): CatalogCover[] {
  return [...topPosts]
    .sort((a, b) => (b.score ?? 0) - (a.score ?? 0))
    .map(toCover)
    .filter(Boolean);
}

function isGoodCover(cover: CatalogCover): boolean {
  return (
    Math.abs(cover.width / cover.height / cardRatio - 1) <= ratioTolerance &&
    cover.width >= minCoverWidth
  );
}

export function pickCover(topPosts: CatalogPost[] = []): CatalogCover | null {
  const covers = coverCandidates(topPosts);

  const good = covers.find(isGoodCover);
  if (good) return good;

  let best: CatalogCover | null = null;
  for (const cover of covers) {
    if (!best || ratioDistance(cover) < ratioDistance(best)) best = cover;
  }
  return best;
}

export const coverPosts = 10;
const coverChoices = 5;
const hour = 60 * 60 * 1000;

export function hourlyRandom(): () => number {
  let seed = Math.floor(Date.now() / hour);
  return () => {
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function fitsThumb(cover: CatalogCover): boolean {
  return cover.width <= cover.height * 2 && cover.height <= cover.width * 2;
}

interface PickOptions {
  random?: () => number;
  fits?: (cover: CatalogCover) => boolean;
  used?: Set<number>;
}

export function pickCovers(
  lists: CatalogPost[][],
  { random, fits = isGoodCover, used = new Set<number>() }: PickOptions = {}
): (CatalogCover | null)[] {
  const choose = (covers: CatalogCover[]) => {
    const choices = covers.slice(0, random ? coverChoices : 1);
    if (choices.length === 0) return null;
    return random ? choices[Math.floor(random() * choices.length)] : choices[0];
  };
  return lists.map((topPosts) => {
    const covers = coverCandidates(topPosts);
    const unused = covers.filter((cover) => !used.has(cover.id));
    const cover =
      choose(unused.filter(fits)) ??
      choose(unused) ??
      choose(covers.filter(fits)) ??
      pickCover(topPosts);
    if (cover) used.add(cover.id);
    return cover;
  });
}

export function sameMake(
  list: CatalogEntry[],
  entry: CatalogEntry
): CatalogEntry[] {
  return list.filter(
    (other) => other.make === entry.make && other.slug !== entry.slug
  );
}

export type CatalogGroup = { label: string; entries: CatalogEntry[] };

const byCount = (a: CatalogEntry, b: CatalogEntry) => b.postCount - a.postCount;

export function groupFilms(list: CatalogEntry[]): CatalogGroup[] {
  const labels: Record<string, string> = {
    color: "Color",
    bw: "Black & white",
  };
  const groups = new Map<string, CatalogEntry[]>([
    ["color", []],
    ["bw", []],
  ]);
  for (const entry of list) {
    const key = entry.colorType || "other";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(entry);
  }
  return Array.from(groups.entries())
    .filter(([, entries]) => entries.length > 0)
    .map(([key, entries]) => ({
      label: labels[key] ?? key,
      entries: entries.sort(byCount),
    }));
}

export function groupCameras(list: CatalogEntry[]): CatalogGroup[] {
  const groups = new Map<string, CatalogEntry[]>();
  for (const entry of list) {
    if (!groups.has(entry.make)) groups.set(entry.make, []);
    groups.get(entry.make).push(entry);
  }
  const total = (entries: CatalogEntry[]) =>
    entries.reduce((sum, entry) => sum + entry.postCount, 0);
  return Array.from(groups.entries())
    .sort(([, a], [, b]) => total(b) - total(a))
    .map(([make, entries]) => ({
      label: make.toUpperCase(),
      entries: entries.sort(byCount),
    }));
}
