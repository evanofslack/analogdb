import { smallImage } from "@lib/images";

export type CatalogKind = "films" | "cameras";

export type CatalogCover = {
  id: number;
  title: string;
  url: string;
  phoneUrl?: string;
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
};

export type CatalogImage = {
  resolution?: string;
  url?: string;
  width?: number;
  height?: number;
};

export type CatalogPost = {
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

export function pickDistinctCovers(
  lists: CatalogPost[][]
): (CatalogCover | null)[] {
  const used = new Set<number>();
  return lists.map((topPosts) => {
    const unused = coverCandidates(topPosts).filter(
      (cover) => !used.has(cover.id)
    );
    const cover = unused.find(isGoodCover) ?? unused[0] ?? pickCover(topPosts);
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
