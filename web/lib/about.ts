export interface AboutPhoto {
  id: number;
  url: string;
  smallUrl: string;
  width?: number;
  height?: number;
  alt: string;
  camera?: string;
  film?: string;
  lens?: string;
  keywords: string[];
  colors: string[];
}

export interface ColorData {
  red: AboutPhoto[];
  green: AboutPhoto[];
  navy: AboutPhoto[];
}

export interface RainbowPhoto {
  photo: AboutPhoto;
  hue: number;
}

// one color's pool for the phone rainbow, count is how many to show
export interface RainbowGroup {
  color: string;
  count: number;
  photos: RainbowPhoto[];
}

export interface SimilarityData {
  centerPost: AboutPhoto;
  similarPosts: AboutPhoto[];
}

export interface FilmSet {
  slug: string;
  label: string;
  postCount: number;
  photos: AboutPhoto[];
}

export interface SearchDemo {
  query: string;
  photos: AboutPhoto[];
  keywords: string[];
}

// post is one photo, grid a row of results, similar the source photo then its matches
export type ApiDemoLayout = "post" | "grid" | "similar";

// one real request for the api demo, short is the response trimmed to its key fields
export interface ApiDemo {
  path: string;
  photos: AboutPhoto[];
  short: string;
  response: unknown;
}

// an endpoint and the requests the demo rotates through for it
export interface ApiDemoGroup {
  name: string;
  layout: ApiDemoLayout;
  demos: ApiDemo[];
}

export interface RepoLanguage {
  name: string;
  percent: number;
  color: string;
}

// the github repo as the open source card shows it
export interface RepoData {
  description: string;
  stars: number;
  license: string | null;
  pushedAt: string;
  languages: RepoLanguage[];
}

export interface AboutData {
  numPosts: number;
  numAuthors: number;
  numCameras: number;
  numFilms: number;
  colorData: ColorData;
  rainbow: RainbowGroup[];
  allSimilarityData: SimilarityData[];
  films: FilmSet[];
  searches: SearchDemo[];
  apiDemos: ApiDemoGroup[];
  repo: RepoData | null;
}

export function shuffle<T>(items: T[]): T[] {
  const result = [...items];
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]];
  }
  return result;
}

function hexToRgb(hex: string): [number, number, number] | null {
  const match = /^#?([0-9a-f]{6})$/i.exec(hex);
  if (!match) return null;
  const value = parseInt(match[1], 16);
  return [(value >> 16) & 255, (value >> 8) & 255, value & 255];
}

function colorDistance(a: string, b: string): number {
  const x = hexToRgb(a);
  const y = hexToRgb(b);
  if (!x || !y) return 0;
  return Math.hypot(x[0] - y[0], x[1] - y[1], x[2] - y[2]);
}

// pick clusters whose sources look unlike each other, judged by dominant color
export function pickSimilarRows(
  clusters: SimilarityData[],
  count: number,
  matches: number
): SimilarityData[] {
  const full = clusters.filter((c) => c.similarPosts.length >= matches);
  const pool = shuffle(full.length >= count ? full : clusters);
  const color = (c: SimilarityData) => c.centerPost.colors[0] ?? "";
  const picked = pool.splice(0, 1);
  while (picked.length < count && pool.length) {
    let best = 0;
    let bestDistance = -1;
    pool.forEach((candidate, index) => {
      const distance = Math.min(
        ...picked.map((p) => colorDistance(color(p), color(candidate)))
      );
      if (distance > bestDistance) {
        best = index;
        bestDistance = distance;
      }
    });
    picked.push(...pool.splice(best, 1));
  }
  return picked.map((c) => ({
    ...c,
    similarPosts: c.similarPosts.slice(0, matches),
  }));
}
