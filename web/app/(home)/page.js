import { getCameraOptions } from "@lib/data/cameras";
import { getFilmOptions } from "@lib/data/films";
import { getFirstPosts } from "@lib/data/posts";
import { firstPageSize, guessColumns, slim } from "@lib/firstPage";
import { postsParsers, toPostsRequest } from "@lib/searchParams";
import { isValidSeed, pickSeed } from "@lib/seed";
import { jsonLd } from "@lib/seo";
import { headers } from "next/headers";
import { userAgent } from "next/server";
import { createLoader } from "nuqs/server";
import HomePage from "./home-page";

const loadPosts = createLoader(postsParsers);

const title = "AnalogDB: film photography database and API";
const description =
  "Thousands of film photographs from Reddit, searchable by camera, film stock and color, with a free open API.";

export const metadata = {
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

async function loadFirstPage(filters) {
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

export default async function Page({ searchParams }) {
  let filters = loadPosts(await searchParams);
  if (filters.sort === "random" && !isValidSeed(filters.seed)) {
    filters = { ...filters, seed: pickSeed() };
  }

  const { device } = userAgent({ headers: await headers() });

  const [initialPage, filmOptions, cameraOptions] = await Promise.all([
    loadFirstPage(filters),
    getFilmOptions(),
    getCameraOptions(),
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
        initialColumns={guessColumns(device)}
        filmOptions={filmOptions}
        cameraOptions={cameraOptions}
      />
    </>
  );
}
