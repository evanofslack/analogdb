"use server";

import * as data from "@lib/data/posts";
import { limitAction } from "@lib/rateLimit";
import { Limited } from "@lib/rateLimited";
import {
  PostIdSimilarGetRequest,
  PostsGetRequest,
  ServerPostResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";

export async function getPosts(
  params: PostsGetRequest
): Promise<ServerPostResponse | Limited> {
  return (await limitAction("browse")) ?? data.getPosts(params);
}

export async function getPostsSimilar(
  params: PostIdSimilarGetRequest
): Promise<ServerSimilarPostsResponse | Limited> {
  return (await limitAction("browse")) ?? data.getPostsSimilar(params);
}
