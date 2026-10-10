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
  rainbow: RainbowGroup[];
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
