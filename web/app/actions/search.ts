"use server";

import { keywordsApi, postApi, searchApi } from "@lib/client";
import {
  AnalogdbPost,
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

export interface SearchSuggestions {
  keywords: string[];
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
const minKeywords = 8;
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

async function keywordSummary(days: number): Promise<string[]> {
  const response = await keywordsApi.keywordsSummaryGet({
    days,
    pageSize: keywordPageSize,
  });
  return (response.keywords ?? [])
    .map((keyword) => keyword.word)
    .filter(Boolean);
}

const getKeywordsCached = unstable_cache(
  async (): Promise<string[]> => {
    const week = await keywordSummary(7);
    if (week.length >= minKeywords) return week;
    return keywordSummary(30);
  },
  ["search-keywords"],
  { revalidate: 3600 }
);

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
    getKeywordsCached().catch((error) => {
      console.error("get search keywords request failed:", error);
      return [] as string[];
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
