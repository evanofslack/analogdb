import { getCameraOptions } from "@lib/data/cameras";
import { getFilmOptions } from "@lib/data/films";
import { getFirstPosts } from "@lib/data/posts";
import { firstPageSize, loadDisplay, slim } from "@lib/firstPage";
import { PostsFilters, postsParsers, toPostsRequest } from "@lib/searchParams";
import { isValidSeed, pickSeed } from "@lib/seed";
import { jsonLd } from "@lib/seo";
import { Metadata } from "next";
import { headers } from "next/headers";
import { userAgent } from "next/server";
import { createLoader, SearchParams } from "nuqs/server";
import HomePage from "./home-page";

const loadPosts = createLoader(postsParsers);

const title = "AnalogDB: film photography database and API";
const description =
  "Thousands of film photographs from Reddit, searchable by camera, film stock and color, with a free open API.";

export const metadata: Metadata = {
  title: { absolute: title },
  description,
  alternates: { canonical: "/" },
  openGraph: { title, description, url: "/" },
};

const website = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  name: "AnalogDB",
  url: "https://analogdb.com",
  potentialAction: {
    "@type": "SearchAction",
    target: "https://analogdb.com/?text={search_term_string}",
    "query-input": "required name=search_term_string",
  },
};

async function loadFirstPage(filters: PostsFilters) {
  try {
    const response = await getFirstPosts({
      ...toPostsRequest(filters),
      pageSize: firstPageSize,
    });
    return slim(response);
  } catch {
    return null;
  }
}

interface PageProps {
  searchParams: Promise<SearchParams>;
}

export default async function Page({ searchParams }: PageProps) {
  let filters = loadPosts(await searchParams);
  if (filters.sort === "random" && !isValidSeed(filters.seed)) {
    filters = { ...filters, seed: pickSeed() };
  }

  const { device } = userAgent({ headers: await headers() });

  const [initialPage, filmOptions, cameraOptions, display] = await Promise.all([
    loadFirstPage(filters),
    getFilmOptions(),
    getCameraOptions(),
    loadDisplay(device),
  ]);

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: jsonLd(website) }}
      />
      <HomePage
        initialPage={initialPage}
        initialFilters={filters}
        initialDisplay={display}
        filmOptions={filmOptions}
        cameraOptions={cameraOptions}
      />
    </>
  );
}
