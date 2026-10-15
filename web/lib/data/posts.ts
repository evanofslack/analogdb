import "server-only";
import { errorStatus, postApi, postsApi } from "@lib/client";
import {
  AnalogdbPost,
  PostIdSimilarGetRequest,
  PostsGetRequest,
  ServerPostResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";
import { unstable_cache } from "next/cache";
import { cache } from "react";

const maxPageSize = 100;

const postsParamKeys = [
  "pageSize",
  "cursor",
  "sort",
  "order",
  "seed",
  "id",
  "nsfw",
  "grayscale",
  "sprocket",
  "keyword",
  "color",
  "minColor",
  "widthMin",
  "widthMax",
  "heightMin",
  "heightMax",
  "ratioMin",
  "ratioMax",
  "filmMake",
  "filmType",
  "cameraMake",
  "cameraModel",
];

const similarParamKeys = ["id", "pageSize", "nsfw", "grayscale", "sprocket"];

function sanitizeParams<T extends object>(params: T, keys: string[]): T {
  const source = (params ?? {}) as Record<string, unknown>;
  const clean: Record<string, unknown> = {};
  for (const key of keys) {
    if (source[key] !== undefined) clean[key] = source[key];
  }
  if (clean.pageSize !== undefined) {
    const pageSize = Math.trunc(Number(clean.pageSize));
    clean.pageSize = Number.isFinite(pageSize)
      ? Math.min(Math.max(pageSize, 1), maxPageSize)
      : undefined;
  }
  return clean as T;
}

export async function getPosts(
  params: PostsGetRequest
): Promise<ServerPostResponse> {
  try {
    const response = await postsApi.postsGet(
      sanitizeParams(params, postsParamKeys)
    );
    return response;
  } catch (error) {
    console.error("get posts request failed:", error);
    throw error;
  }
}

const getPostsCached = unstable_cache(
  (params: PostsGetRequest) => postsApi.postsGet(params),
  ["posts"],
  { revalidate: 60, tags: ["posts"] }
);

export async function getFirstPosts(
  params: PostsGetRequest
): Promise<ServerPostResponse> {
  try {
    const response = await getPostsCached(
      sanitizeParams(params, postsParamKeys)
    );
    return response;
  } catch (error) {
    console.error("get first posts request failed:", error);
    throw error;
  }
}

export async function getPostsSimilar(
  params: PostIdSimilarGetRequest
): Promise<ServerSimilarPostsResponse> {
  try {
    const response = await postApi.postIdSimilarGet(
      sanitizeParams(params, similarParamKeys)
    );
    return response;
  } catch (error) {
    console.error("get posts similar request failed:", error);
    throw error;
  }
}

export async function getPostsTotalCount(): Promise<number> {
  try {
    const params: PostsGetRequest = {};
    const response = await postsApi.postsGet(params);
    return response.meta.totalPosts;
  } catch (error) {
    console.error("get posts total count request failed:", error);
    throw error;
  }
}

// getPost loads one post for the post page and its layout, null when the id
// is invalid or the post doesn't exist. Its revalidate matches the page's ISR
export const getPost = cache(
  async (pid: string): Promise<AnalogdbPost | null> => {
    const id = Number(pid);
    if (!Number.isInteger(id) || id <= 0) {
      return null;
    }
    try {
      return await postApi.postIdGet({ id }, {
        next: { revalidate: 3600 },
      } as RequestInit);
    } catch (error) {
      const status = errorStatus(error);
      if (status === 400 || status === 404) {
        return null;
      }
      throw error;
    }
  }
);
