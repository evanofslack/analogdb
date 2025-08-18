"use server";

import { postApi, postsApi } from "@lib/client";
import {
  PostIdSimilarGetRequest,
  PostsGetRequest,
  ServerPostResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";

export async function getPosts(
  params: PostsGetRequest
): Promise<ServerPostResponse> {
  try {
    const response = await postsApi.postsGet(params);
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
    const response = await postApi.postIdSimilarGet(params);
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
