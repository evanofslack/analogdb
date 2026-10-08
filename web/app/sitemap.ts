import { getCameraCatalog } from "@lib/data/cameras";
import { getFilmCatalog } from "@lib/data/films";
import { catalogHref } from "@lib/catalog";
import { authorized_fetch } from "@lib/client";
import { MetadataRoute } from "next";
import { unstable_cache } from "next/cache";

export const dynamic = "force-dynamic";

const siteURL = "https://analogdb.com";

async function fetchIds(): Promise<number[]> {
  const response = await authorized_fetch("/ids", "GET", 86400);
  if (!response.ok) {
    throw new Error(`Fail fetch post ids: ${response.status}`);
  }
  const data = await response.json();
  return data.ids || [];
}

const getCachedIds = unstable_cache(fetchIds, ["sitemap-ids"], {
  revalidate: 86400,
});

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const [ids, films, cameras] = await Promise.all([
    getCachedIds(),
    getFilmCatalog(),
    getCameraCatalog(),
  ]);

  return [
    { url: siteURL },
    { url: `${siteURL}/about` },
    { url: `${siteURL}/docs` },
    { url: `${siteURL}/films` },
    { url: `${siteURL}/cameras` },
    ...films.map((film) => ({
      url: `${siteURL}${catalogHref("films", film)}`,
    })),
    ...cameras.map((camera) => ({
      url: `${siteURL}${catalogHref("cameras", camera)}`,
    })),
    ...ids.map((id) => ({ url: `${siteURL}/post/${id}` })),
  ];
}
