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
  teal: AboutPhoto[];
  red: AboutPhoto[];
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
}

export function shuffle<T>(items: T[]): T[] {
  const result = [...items];
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]];
  }
  return result;
}
