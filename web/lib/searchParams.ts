import { PostsGetRequest } from "analogdb-generated";
import {
  inferParserType,
  parseAsFloat,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
} from "nuqs/server";
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
