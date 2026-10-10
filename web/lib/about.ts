export interface AboutPhoto {
  id: number;
  url: string;
  smallUrl: string;
  width?: number;
  height?: number;
  alt: string;
  caption?: string;
  camera?: string;
  film?: string;
  lens?: string;
  keywords: string[];
  colors: string[];
}

export interface ColorData {
  red: AboutPhoto[];
  navy: AboutPhoto[];
  olive: AboutPhoto[];
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

export interface ApiSample {
  query: string;
  full: string;
  short: string;
}

export interface AboutData {
  numPosts: number;
  numAuthors: number;
  numCameras: number;
  numFilms: number;
  colorData: ColorData;
  allSimilarityData: SimilarityData[];
  films: FilmSet[];
  searches: SearchDemo[];
  apiSample: ApiSample | null;
}

export function shuffle<T>(items: T[]): T[] {
  const result = [...items];
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]];
  }
  return result;
}
