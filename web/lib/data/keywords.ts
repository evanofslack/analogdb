import "server-only";

import {
  CatalogEntry,
  coverPosts,
  fitsThumb,
  hourlyRandom,
  pickCover,
  pickCovers,
} from "@lib/catalog";
import { errorStatus, keywordsApi } from "@lib/client";
import { KeywordsSummaryGetRequest } from "analogdb-generated";
import { unstable_cache } from "next/cache";

interface RelatedKeyword {
  word: string;
  count: number;
  thumb?: string;
}

interface KeywordDetail {
  entry: CatalogEntry;
  related: RelatedKeyword[];
}

const catalogParams: KeywordsSummaryGetRequest = {
  pageSize: 300,
  minCount: 100,
  topPosts: coverPosts,
};

const maxWordLength = 100;

const getKeywordCatalogCached = unstable_cache(
  async (): Promise<CatalogEntry[]> => {
    const response = await keywordsApi.keywordsSummaryGet(catalogParams);
    const keywords = (response.keywords ?? [])
      .filter((k) => k.word)
      .sort((a, b) => (b.count ?? 0) - (a.count ?? 0));
    const covers = pickCovers(
      keywords.map((k) => k.topPosts ?? []),
      { random: hourlyRandom() }
    );
    return keywords.map((k, i) => ({
      slug: k.word,
      make: k.word,
      name: "",
      postCount: k.count ?? 0,
      description: "",
      cover: covers[i],
    }));
  },
  ["keyword-catalog"],
  { revalidate: 3600, tags: ["catalog"] }
);

export async function getKeywordCatalog(): Promise<CatalogEntry[]> {
  try {
    return await getKeywordCatalogCached();
  } catch (error) {
    console.error("get keyword catalog request failed:", error);
    return [];
  }
}

const getKeywordCached = unstable_cache(
  async (word: string): Promise<KeywordDetail | null> => {
    try {
      const response = await keywordsApi.keywordWordGet({
        word,
        topPosts: 6,
        related: 12,
        relatedPosts: coverPosts,
      });
      const name = response.word ?? word;
      const related = (response.related ?? []).filter((k) => k.word);
      const thumbs = pickCovers(
        related.map((k) => k.topPosts ?? []),
        { random: hourlyRandom(), fits: fitsThumb }
      );
      return {
        entry: {
          slug: name,
          make: name,
          name: "",
          postCount: response.count ?? 0,
          description: "",
          cover: pickCover(response.topPosts),
        },
        related: related.map((k, i) => ({
          word: k.word,
          count: k.count ?? 0,
          thumb: thumbs[i]?.thumbUrl ?? thumbs[i]?.url,
        })),
      };
    } catch (error) {
      if (errorStatus(error) === 404) {
        return null;
      }
      throw error;
    }
  },
  ["keyword"],
  { revalidate: 3600 }
);

export async function getKeyword(word: string): Promise<KeywordDetail | null> {
  const clean = String(word ?? "")
    .trim()
    .toLowerCase()
    .slice(0, maxWordLength);
  if (!clean) return null;
  try {
    return await getKeywordCached(clean);
  } catch (error) {
    console.error("get keyword request failed:", error);
    return null;
  }
}
