"use server";

import { checkAdminAuth } from "@lib/auth";
import { postApi, postsApi } from "@lib/client";
import {
  PostIdDeleteRequest,
  PostIdSimilarGetRequest,
  PostsGetRequest,
  ServerDeleteResponse,
  ServerPostResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";
import { revalidatePath } from "next/cache";

const maxPageSize = 100;

const postsParamKeys = [
  "pageSize",
  "pageId",
  "cursor",
  "sort",
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

export async function deletePost(id: number): Promise<ServerDeleteResponse> {
  if (!(await checkAdminAuth())) {
    throw new Error("Unauthorized");
  }
  try {
    const params: PostIdDeleteRequest = { id: id };
    const response = await postApi.postIdDelete(params);
    revalidatePath(`/post/${id}`);
    revalidatePath("/");
    return response;
  } catch (error) {
    console.error("delete post request failed:", error);
    throw error;
  }
}
