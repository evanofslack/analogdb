import "server-only";
import { coverPosts, fitsThumb, hourlyRandom, pickCovers } from "@lib/catalog";
import { keywordsApi, postApi, searchApi } from "@lib/client";
import { getKeywordCatalog } from "@lib/data/keywords";
import {
  AnalogdbCatalogPost,
  AnalogdbPost,
  KeywordsSummaryGetSortEnum,
  SearchGetRequest,
  ServerSearchResponse,
} from "analogdb-generated";
import { unstable_cache } from "next/cache";

export interface SearchImage {
  url: string;
  width: number;
  height: number;
  hex?: string;
}

export interface SearchExample {
  id: number;
  title: string;
  image: SearchImage;
}

export interface TrendingKeyword {
  word: string;
  thumb?: string;
}

export interface SearchSuggestions {
  keywords: TrendingKeyword[];
  examples: SearchExample[];
}

export interface SearchSource {
  id: number;
  title: string;
  image: SearchImage | null;
}

const maxPageSize = 100;
const maxQueryLength = 200;
const keywordPageSize = 20;
const trendingDays = 30;
const exampleIds = [32768, 34336, 35099, 38512];

function sanitizeSearch(params: SearchGetRequest): SearchGetRequest {
  const source = (params ?? {}) as SearchGetRequest;
  const clean: SearchGetRequest = {
    q: String(source.q ?? "")
      .trim()
      .slice(0, maxQueryLength),
  };
  if (typeof source.cursor === "string" && source.cursor) {
    clean.cursor = source.cursor;
  }
  if (source.pageSize !== undefined) {
    const pageSize = Math.trunc(Number(source.pageSize));
    if (Number.isFinite(pageSize)) {
      clean.pageSize = Math.min(Math.max(pageSize, 1), maxPageSize);
    }
  }
  for (const key of ["nsfw", "grayscale", "sprocket"] as const) {
    if (typeof source[key] === "boolean") clean[key] = source[key];
  }
  return clean;
}

const searchPostsCached = unstable_cache(
  (params: SearchGetRequest) => searchApi.searchGet(params),
  ["search"],
  { revalidate: 600 }
);

export async function searchPosts(
  params: SearchGetRequest
): Promise<ServerSearchResponse> {
  const clean = sanitizeSearch(params);
  if (!clean.q) return { posts: [], relatedKeywords: [] };
  try {
    return await searchPostsCached(clean);
  } catch (error) {
    console.error("search posts request failed:", error);
    throw error;
  }
}

function mediumImage(post: AnalogdbPost): SearchImage | null {
  const image =
    post.images?.find((img) => img.resolution === "medium") ?? post.images?.[1];
  if (!image?.url) return null;
  return {
    url: image.url,
    width: image.width ?? 0,
    height: image.height ?? 0,
    hex: post.colors?.[0]?.hex,
  };
}

interface TrendingPosts {
  word: string;
  topPosts: AnalogdbCatalogPost[];
}

const getKeywordsCached = unstable_cache(
  async (): Promise<TrendingPosts[]> => {
    const response = await keywordsApi.keywordsSummaryGet({
      sort: KeywordsSummaryGetSortEnum.Trending,
      days: trendingDays,
      pageSize: keywordPageSize,
      topPosts: coverPosts,
    });
    return (response.keywords ?? [])
      .filter((keyword) => keyword.word)
      .map((keyword) => ({
        word: keyword.word,
        topPosts: keyword.topPosts ?? [],
      }));
  },
  ["search-trending"],
  { revalidate: 3600 }
);

async function getTrendingKeywords(): Promise<TrendingKeyword[]> {
  const [trending, catalog] = await Promise.all([
    getKeywordsCached(),
    getKeywordCatalog(),
  ]);
  const used = new Set(catalog.map((entry) => entry.cover?.id).filter(Boolean));
  const covers = pickCovers(
    trending.map((keyword) => keyword.topPosts),
    { random: hourlyRandom(), fits: fitsThumb, used }
  );
  return trending.map((keyword, i) => ({
    word: keyword.word,
    thumb: covers[i]?.thumbUrl ?? covers[i]?.url,
  }));
}

const getExamplesCached = unstable_cache(
  async (): Promise<SearchExample[]> => {
    const examples = await Promise.all(
      exampleIds.map((id) => postApi.postIdGet({ id }))
    );
    return examples
      .map((post) => ({
        id: post.id,
        title: post.title ?? "",
        image: mediumImage(post),
      }))
      .filter((example) => example.image);
  },
  ["search-examples"],
  { revalidate: 86400 }
);

export async function getSearchSuggestions(): Promise<SearchSuggestions> {
  const [keywords, examples] = await Promise.all([
    getTrendingKeywords().catch((error) => {
      console.error("get search keywords request failed:", error);
      return [] as TrendingKeyword[];
    }),
    getExamplesCached().catch((error) => {
      console.error("get search examples request failed:", error);
      return [] as SearchExample[];
    }),
  ]);
  return { keywords, examples };
}

const getSearchSourceCached = unstable_cache(
  async (id: number): Promise<SearchSource> => {
    const post = await postApi.postIdGet({ id });
    return { id: post.id, title: post.title ?? "", image: mediumImage(post) };
  },
  ["search-source"],
  { revalidate: 3600 }
);

export async function getSearchSource(id: number): Promise<SearchSource> {
  const postId = Math.trunc(Number(id));
  if (!Number.isFinite(postId) || postId <= 0) return null;
  try {
    return await getSearchSourceCached(postId);
  } catch (error) {
    console.error("get search source request failed:", error);
    return null;
  }
}
