import { getKeywordCatalog } from "@lib/data/keywords";
import { getPostsSimilar } from "@lib/data/posts";
import {
  getSearchSource,
  getSearchSuggestions,
  searchPosts,
} from "@lib/data/search";
import { guessColumns, slim } from "@lib/firstPage";
import { limit } from "@lib/rateLimit";
import {
  SearchFlags,
  searchKey,
  searchPageSize,
  searchParsers,
  similarPageSize,
  toSearchFlags,
} from "@lib/searchParams";
import {
  ServerSearchResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";
import { Metadata } from "next";
import { headers } from "next/headers";
import { userAgent } from "next/server";
import { createLoader, SearchParams } from "nuqs/server";
import SearchPage from "./search-page";

const loadSearch = createLoader(searchParsers);

const robots = { index: false, follow: true };

interface PageProps {
  searchParams: Promise<SearchParams>;
}

export async function generateMetadata({
  searchParams,
}: PageProps): Promise<Metadata> {
  const { q, similar } = loadSearch(await searchParams);
  let title = "Search · AnalogDB";
  if (q) title = `"${q}" photos · AnalogDB`;
  else if (similar) title = "Similar photos · AnalogDB";
  return {
    title: { absolute: title },
    description: "Search film photographs by words or by image.",
    robots,
  };
}

function withKeywords(
  response: ServerSimilarPostsResponse
): ServerSimilarPostsResponse {
  const slimmed = slim(response);
  return {
    ...slimmed,
    posts: slimmed.posts.map((post, index) => ({
      ...post,
      keywords: response.posts[index].keywords,
    })),
  };
}

async function loadText(
  q: string,
  flags: SearchFlags
): Promise<ServerSearchResponse | null> {
  try {
    const response = await searchPosts({
      q,
      pageSize: searchPageSize,
      ...flags,
    });
    return {
      ...slim(response),
      relatedKeywords: response.relatedKeywords ?? [],
    };
  } catch {
    return null;
  }
}

async function loadSimilar(
  id: number,
  flags: SearchFlags
): Promise<ServerSimilarPostsResponse | null> {
  try {
    const response = await getPostsSimilar({
      id,
      pageSize: similarPageSize,
      ...flags,
    });
    return withKeywords(response);
  } catch {
    return null;
  }
}

export default async function Page({ searchParams }: PageProps) {
  const filters = loadSearch(await searchParams);
  const flags = toSearchFlags(filters);
  const q = filters.q?.trim() || null;
  const similar = !q && filters.similar > 0 ? filters.similar : null;

  const { device } = userAgent({ headers: await headers() });

  let allowed = true;
  if (q) allowed = (await limit("search")).ok;
  else if (similar) allowed = (await limit("browse")).ok;

  let initialKey: string | null = null;
  if (q) initialKey = searchKey("q", q, flags);
  else if (similar) initialKey = searchKey("similar", similar, flags);

  return (
    <SearchPage
      suggestionsPromise={getSearchSuggestions()}
      catalogPromise={!q && !similar ? getKeywordCatalog() : null}
      textPromise={q && allowed ? loadText(q, flags) : null}
      similarPromise={similar && allowed ? loadSimilar(similar, flags) : null}
      sourcePromise={similar && allowed ? getSearchSource(similar) : null}
      sourceId={similar}
      initialKey={initialKey}
      initialColumns={guessColumns(device)}
    />
  );
}
