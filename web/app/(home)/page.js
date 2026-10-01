import { getCameraOptions } from "@app/actions/cameras";
import { getFilmOptions } from "@app/actions/films";
import { getFirstPosts } from "@app/actions/posts";
import { postsParsers, toPostsRequest } from "@lib/searchParams";
import { isValidSeed, pickSeed } from "@lib/seed";
import { jsonLd } from "@lib/seo";
import { headers } from "next/headers";
import { userAgent } from "next/server";
import { createLoader } from "nuqs/server";
import HomePage from "./home-page";

const loadPosts = createLoader(postsParsers);

const firstPageSize = 40;

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

function slim(response) {
  return {
    meta: response.meta,
    posts: (response.posts ?? []).map((post) => ({
      id: post.id,
      author: post.author,
      title: post.title,
      images: (post.images ?? []).slice(0, 2),
      colors: (post.colors ?? []).slice(0, 1),
    })),
  };
}

function guessColumns(device) {
  if (device.type === "mobile") return 2;
  if (device.type === "tablet") return 3;
  return 4;
}

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
