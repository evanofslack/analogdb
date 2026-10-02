"use server";

import { CatalogEntry, minPosts, pickCover } from "@lib/catalog";
import { filmsApi } from "@lib/client";
import { FilmsGetRequest } from "analogdb-generated";
import { unstable_cache } from "next/cache";

export interface FilmOption {
  make: string;
  type: string;
  label: string;
}

const params: FilmsGetRequest = {
  sort: "counts",
  pageSize: 500,
  includeCounts: true,
  excludeZeroCounts: true,
};

const getFilmOptionsCached = unstable_cache(
  async (): Promise<FilmOption[]> => {
    const response = await filmsApi.filmsGet(params);
    return (response.films ?? [])
      .filter((f) => f.make && f.type)
      .map((f) => ({
        make: f.make,
        type: f.type,
        label: `${f.make} - ${f.type}`,
      }))
      .filter((v, i, arr) => arr.findIndex((x) => x.label === v.label) === i);
  },
  ["film-options"],
  { revalidate: 3600 }
);

export async function getFilmOptions(): Promise<FilmOption[]> {
  try {
    return await getFilmOptionsCached();
  } catch (error) {
    console.error("get film options request failed:", error);
    return [];
  }
}

const catalogParams: FilmsGetRequest = {
  sort: "counts",
  pageSize: 500,
  includeCounts: true,
  minCount: minPosts,
  topPosts: 6,
};

const getFilmCatalogCached = unstable_cache(
  async (): Promise<CatalogEntry[]> => {
    const response = await filmsApi.filmsGet(catalogParams);
    return (response.films ?? [])
      .filter(
        (f) => f.slug && f.make && f.type && (f.postCount ?? 0) >= minPosts
      )
      .map((f) => ({
        slug: f.slug,
        make: f.make,
        name: f.type,
        postCount: f.postCount ?? 0,
        description: f.description ?? "",
        speed: f.speed,
        colorType: f.colorType,
        cover: pickCover(f.topPosts),
      }));
  },
  ["film-catalog"],
  { revalidate: 3600, tags: ["catalog"] }
);

export async function getFilmCatalog(): Promise<CatalogEntry[]> {
  try {
    return await getFilmCatalogCached();
  } catch (error) {
    console.error("get film catalog request failed:", error);
    return [];
  }
}
