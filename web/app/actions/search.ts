"use server";

import * as data from "@lib/data/search";
import { SearchGetRequest, ServerSearchResponse } from "analogdb-generated";

export async function searchPosts(
  params: SearchGetRequest
): Promise<ServerSearchResponse> {
  return data.searchPosts(params);
}

export async function getSearchSource(
  id: number
): Promise<data.SearchSource> {
  return data.getSearchSource(id);
}
