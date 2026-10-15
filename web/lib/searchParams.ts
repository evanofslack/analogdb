import { HTTPQuery, PostsGetRequest, querystring } from "analogdb-generated";
import {
  inferParserType,
  parseAsArrayOf,
  parseAsFloat,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
} from "nuqs/server";
import { publicURL } from "./constants";
import { isValidSeed } from "./seed";

const sortOpts = ["time", "score", "random", "color"] as const;
const catalogSortOpts = ["time", "score", "random"] as const;
const orderOpts = ["desc", "asc"] as const;
const filterOpts = ["include", "exclude", "only"] as const;

export const postsLimits = {
  widthMin: 600,
  widthMax: 15000,
  heightMin: 400,
  heightMax: 12000,
  ratioMin: 0.3,
  ratioMax: 4.8,
};

// width / height ranges that touch end to end, so any run of neighbours is one range
export const ratioShapes = [
  { key: "portrait", min: postsLimits.ratioMin, max: 0.9 },
  { key: "square", min: 0.9, max: 1.1 },
  { key: "landscape", min: 1.1, max: 1.9 },
  { key: "panorama", min: 1.9, max: postsLimits.ratioMax },
] as const;

const same = (a: number, b: number) => Math.abs(a - b) < 1e-9;

// Indexes of the shapes a ratio range covers exactly, all of them for the
// default range and none for custom values
export function selectedShapes(ratioMin: number, ratioMax: number): number[] {
  const first = ratioShapes.findIndex((s) => same(s.min, ratioMin));
  const last = ratioShapes.findIndex((s) => same(s.max, ratioMax));
  if (first < 0 || last < first) return [];
  return Array.from({ length: last - first + 1 }, (_, i) => first + i);
}

// Ratio range after toggling a shape. With all or none selected it keeps just
// that one, picking one outside the selection fills the gap, dropping an inner
// one keeps just that one and dropping the last one selects all.
export function toggleShape(
  selected: number[],
  index: number
): { ratioMin: number; ratioMax: number } {
  let first = index;
  let last = index;
  if (selected.length && selected.length < ratioShapes.length) {
    const lo = selected[0];
    const hi = selected[selected.length - 1];
    if (index < lo || index > hi) {
      first = Math.min(lo, index);
      last = Math.max(hi, index);
    } else if (lo === hi) {
      first = 0;
      last = ratioShapes.length - 1;
    } else if (index === lo) {
      first = lo + 1;
      last = hi;
    } else if (index === hi) {
      first = lo;
      last = hi - 1;
    }
  }
  return { ratioMin: ratioShapes[first].min, ratioMax: ratioShapes[last].max };
}

export const postsParsers = {
  sort: parseAsStringLiteral(sortOpts).withDefault("time"),
  order: parseAsStringLiteral(orderOpts).withDefault("desc"),
  seed: parseAsInteger,
  nsfw: parseAsStringLiteral(filterOpts).withDefault("exclude"),
  bw: parseAsStringLiteral(filterOpts).withDefault("exclude"),
  sprocket: parseAsStringLiteral(filterOpts).withDefault("include"),
  color: parseAsArrayOf(parseAsString),
  text: parseAsString,
  widthMin: parseAsInteger.withDefault(postsLimits.widthMin),
  widthMax: parseAsInteger.withDefault(postsLimits.widthMax),
  heightMin: parseAsInteger.withDefault(postsLimits.heightMin),
  heightMax: parseAsInteger.withDefault(postsLimits.heightMax),
  ratioMin: parseAsFloat.withDefault(postsLimits.ratioMin),
  ratioMax: parseAsFloat.withDefault(postsLimits.ratioMax),
  film_make: parseAsString,
  film_type: parseAsString,
  camera_make: parseAsString,
  camera_model: parseAsString,
};

export type PostsFilters = inferParserType<typeof postsParsers>;

export type ActivePostsFilters = {
  camera: boolean;
  film: boolean;
  color: boolean;
  size: boolean;
  flags: boolean;
  text: boolean;
};

export function activePostsFilters(f: PostsFilters): ActivePostsFilters {
  const p = postsParsers;
  return {
    camera: Boolean(f.camera_make || f.camera_model),
    film: Boolean(f.film_make || f.film_type),
    color: Boolean(f.color?.length),
    size:
      f.widthMin !== p.widthMin.defaultValue ||
      f.widthMax !== p.widthMax.defaultValue ||
      f.heightMin !== p.heightMin.defaultValue ||
      f.heightMax !== p.heightMax.defaultValue ||
      f.ratioMin !== p.ratioMin.defaultValue ||
      f.ratioMax !== p.ratioMax.defaultValue,
    flags:
      f.nsfw !== p.nsfw.defaultValue ||
      f.bw !== p.bw.defaultValue ||
      f.sprocket !== p.sprocket.defaultValue,
    text: Boolean(f.text),
  };
}

// null drops the param from the url, so each filter falls back to its default
export const clearedPostsFilters = {
  nsfw: null,
  bw: null,
  sprocket: null,
  color: null,
  text: null,
  widthMin: null,
  widthMax: null,
  heightMin: null,
  heightMax: null,
  ratioMin: null,
  ratioMax: null,
  film_make: null,
  film_type: null,
  camera_make: null,
  camera_model: null,
};

export const searchParsers = {
  q: parseAsString,
  similar: parseAsInteger,
  nsfw: parseAsStringLiteral(filterOpts).withDefault("exclude"),
  bw: parseAsStringLiteral(filterOpts).withDefault("include"),
  sprocket: parseAsStringLiteral(filterOpts).withDefault("include"),
};

export type SearchFilters = inferParserType<typeof searchParsers>;

export type SearchFlags = {
  nsfw?: boolean;
  grayscale?: boolean;
  sprocket?: boolean;
};

export function toSearchFlags(
  filters: Pick<SearchFilters, "nsfw" | "bw" | "sprocket">
): SearchFlags {
  const flags: SearchFlags = {};
  if (filters.nsfw === "exclude") flags.nsfw = false;
  if (filters.nsfw === "only") flags.nsfw = true;
  if (filters.bw === "exclude") flags.grayscale = false;
  if (filters.bw === "only") flags.grayscale = true;
  if (filters.sprocket === "exclude") flags.sprocket = false;
  if (filters.sprocket === "only") flags.sprocket = true;
  return flags;
}

export const searchPageSize = 40;
export const similarPageSize = 50;

export function searchKey(
  mode: string,
  value: string | number,
  flags: SearchFlags
): string {
  return JSON.stringify([mode, value, flags]);
}

export const catalogFilterParsers = {
  q: parseAsString
    .withDefault("")
    .withOptions({ history: "replace", throttleMs: 300 }),
};

export const catalogParsers = {
  sort: parseAsStringLiteral(catalogSortOpts).withDefault("score"),
  order: parseAsStringLiteral(orderOpts).withDefault("desc"),
  seed: parseAsInteger,
};

export type CatalogMatch = Pick<
  PostsFilters,
  "film_make" | "film_type" | "camera_make" | "camera_model" | "text"
>;

export function catalogFilters(match: Partial<CatalogMatch>): PostsFilters {
  return {
    sort: "score",
    order: "desc",
    seed: null,
    nsfw: "exclude",
    bw: "include",
    sprocket: "include",
    color: null,
    text: null,
    ...postsLimits,
    film_make: null,
    film_type: null,
    camera_make: null,
    camera_model: null,
    ...match,
  };
}

export function toPostsRequest(filters: PostsFilters): PostsGetRequest {
  const params: PostsGetRequest = {
    sort: filters.sort,
    pageSize: 100,
    widthMin: filters.widthMin,
    widthMax: filters.widthMax,
    heightMin: filters.heightMin,
    heightMax: filters.heightMax,
    ratioMin: filters.ratioMin,
    ratioMax: filters.ratioMax,
  };

  if (filters.sort === "random" && isValidSeed(filters.seed)) {
    params.seed = filters.seed;
  }

  const ordered = filters.sort === "time" || filters.sort === "score";
  if (filters.order === "asc" && ordered) {
    params.order = "asc";
  }

  if (filters.nsfw === "exclude") params.nsfw = false;
  if (filters.nsfw === "only") params.nsfw = true;

  if (filters.bw === "exclude") params.grayscale = false;
  if (filters.bw === "only") params.grayscale = true;

  if (filters.sprocket === "exclude") params.sprocket = false;
  if (filters.sprocket === "only") params.sprocket = true;

  if (filters.text) {
    params.keyword = [filters.text];
  }

  const colors = filters.color?.filter(Boolean);
  if (colors?.length) {
    params.color = colors;
  }

  if (filters.film_make) params.filmMake = filters.film_make;
  if (filters.film_type) params.filmType = filters.film_type;
  if (filters.camera_make) params.cameraMake = filters.camera_make;
  if (filters.camera_model) params.cameraModel = filters.camera_model;

  return params;
}

// the public api call for these filters, leaving out values the api already defaults to
export function postsApiUrl(filters: PostsFilters): string {
  const request = toPostsRequest(filters);
  const defaults: Partial<Record<keyof PostsGetRequest, unknown>> = {
    sort: "time",
    pageSize: request.pageSize,
    ...postsLimits,
  };
  const query: HTTPQuery = {};
  for (const [key, value] of Object.entries(request)) {
    if (value == null || defaults[key as keyof PostsGetRequest] === value) {
      continue;
    }
    query[key.replace(/[A-Z]/g, (c) => `_${c.toLowerCase()}`)] = value;
  }
  const qs = querystring(query);
  return `${publicURL}/posts${qs ? `?${qs}` : ""}`;
}
