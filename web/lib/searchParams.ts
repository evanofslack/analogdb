import { HTTPQuery, PostsGetRequest, querystring } from "analogdb-generated";
import {
  inferParserType,
  parseAsFloat,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
} from "nuqs/server";
import { publicURL } from "./constants";
import { isValidSeed } from "./seed";

const sortOpts = ["time", "score", "random"] as const;
const filterOpts = ["include", "exclude", "only"] as const;

export const postsLimits = {
  widthMin: 600,
  widthMax: 15000,
  heightMin: 400,
  heightMax: 12000,
  ratioMin: 0.3,
  ratioMax: 4.8,
};

export const postsParsers = {
  sort: parseAsStringLiteral(sortOpts).withDefault("time"),
  seed: parseAsInteger,
  nsfw: parseAsStringLiteral(filterOpts).withDefault("exclude"),
  bw: parseAsStringLiteral(filterOpts).withDefault("exclude"),
  sprocket: parseAsStringLiteral(filterOpts).withDefault("include"),
  color: parseAsString,
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
    color: Boolean(f.color),
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
  sort: parseAsStringLiteral(sortOpts).withDefault("score"),
  seed: parseAsInteger,
};

export type CatalogMatch = Pick<
  PostsFilters,
  "film_make" | "film_type" | "camera_make" | "camera_model" | "text"
>;

export function catalogFilters(match: Partial<CatalogMatch>): PostsFilters {
  return {
    sort: "score",
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

const COLOR_MIN_VALUES: Record<string, number> = {
  gray: 0.8,
  black: 0.7,
  white: 0.5,
  teal: 0.35,
  olive: 0.35,
  brown: 0.35,
  tan: 0.3,
  navy: 0.25,
  green: 0.25,
  default: 0.15,
};

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

  if (filters.nsfw === "exclude") params.nsfw = false;
  if (filters.nsfw === "only") params.nsfw = true;

  if (filters.bw === "exclude") params.grayscale = false;
  if (filters.bw === "only") params.grayscale = true;

  if (filters.sprocket === "exclude") params.sprocket = false;
  if (filters.sprocket === "only") params.sprocket = true;

  if (filters.text) {
    params.keyword = [filters.text];
  }

  if (filters.color) {
    params.color = [filters.color];
    params.minColor = [
      COLOR_MIN_VALUES[filters.color] || COLOR_MIN_VALUES.default,
    ];
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
