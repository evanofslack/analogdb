"use server";

import * as data from "@lib/data/posts";
import {
  PostIdSimilarGetRequest,
  PostsGetRequest,
  ServerPostResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";

export async function getPosts(
  params: PostsGetRequest
): Promise<ServerPostResponse> {
  return data.getPosts(params);
}

export async function getPostsSimilar(
  params: PostIdSimilarGetRequest
): Promise<ServerSimilarPostsResponse> {
  return data.getPostsSimilar(params);
}
