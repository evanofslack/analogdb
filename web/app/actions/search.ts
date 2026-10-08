"use server";

import * as data from "@lib/data/search";
import { limitAction } from "@lib/rateLimit";
import { Limited } from "@lib/rateLimited";
import { SearchGetRequest, ServerSearchResponse } from "analogdb-generated";

export async function searchPosts(
  params: SearchGetRequest
): Promise<ServerSearchResponse | Limited> {
  return (await limitAction("search")) ?? data.searchPosts(params);
}

export async function getSearchSource(
  id: number
): Promise<data.SearchSource | Limited> {
  return (await limitAction("browse")) ?? data.getSearchSource(id);
}
