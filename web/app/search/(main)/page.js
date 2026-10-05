import { getKeywordCatalog } from "@app/actions/keywords";
import { getPostsSimilar } from "@app/actions/posts";
import {
  getSearchSource,
  getSearchSuggestions,
  searchPosts,
} from "@app/actions/search";
import { guessColumns, slim } from "@lib/firstPage";
import {
  searchKey,
  searchPageSize,
  searchParsers,
  similarPageSize,
  toSearchFlags,
} from "@lib/searchParams";
import { headers } from "next/headers";
import { userAgent } from "next/server";
import { createLoader } from "nuqs/server";
import SearchPage from "./search-page";

const loadSearch = createLoader(searchParsers);

const robots = { index: false, follow: true };

export async function generateMetadata({ searchParams }) {
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

function withKeywords(response) {
  const slimmed = slim(response);
  return {
    ...slimmed,
    posts: slimmed.posts.map((post, index) => ({
      ...post,
      keywords: response.posts[index].keywords,
    })),
  };
}

async function loadText(q, flags) {
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

async function loadSimilar(id, flags) {
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

export default async function Page({ searchParams }) {
  const filters = loadSearch(await searchParams);
  const flags = toSearchFlags(filters);
  const q = filters.q?.trim() || null;
  const similar = !q && filters.similar > 0 ? filters.similar : null;

  const { device } = userAgent({ headers: await headers() });

  const [
    suggestions,
    keywordCatalog,
    initialPage,
    initialSimilar,
    initialSource,
  ] = await Promise.all([
    getSearchSuggestions(),
    !q && !similar ? getKeywordCatalog() : null,
    q ? loadText(q, flags) : null,
    similar ? loadSimilar(similar, flags) : null,
    similar ? getSearchSource(similar) : null,
  ]);

  let initialKey = null;
  if (q) initialKey = searchKey("q", q, flags);
  else if (similar) initialKey = searchKey("similar", similar, flags);

  return (
    <SearchPage
      suggestions={suggestions}
      keywordCatalog={keywordCatalog}
      initialPage={initialPage}
      initialSimilar={initialSimilar}
      initialSource={initialSource}
      initialKey={initialKey}
      initialColumns={guessColumns(device)}
    />
  );
}
