"use server";

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
