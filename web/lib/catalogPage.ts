import "server-only";
import { getCameraCatalog } from "@app/actions/cameras";
import { getFilmCatalog } from "@app/actions/films";
import { getFirstPosts } from "@app/actions/posts";
import {
  CatalogEntry,
  catalogHref,
  CatalogKind,
  catalogName,
  findBySlug,
  sameMake,
} from "@lib/catalog";
import { firstPageSize, guessColumns, slim } from "@lib/firstPage";
import {
  catalogFilters,
  CatalogMatch,
  catalogParsers,
  PostsFilters,
  toPostsRequest,
} from "@lib/searchParams";
import { isValidSeed, pickSeed } from "@lib/seed";
import { truncate } from "@lib/seo";
import { Metadata } from "next";
import { headers } from "next/headers";
import { userAgent } from "next/server";
import { createLoader } from "nuqs/server";
import { cache } from "react";

const siteURL = "https://analogdb.com";
const maxDescription = 160;
const maxRelated = 12;
const loadSort = createLoader(catalogParsers);

const labels: Record<CatalogKind, string> = {
  films: "Film",
  cameras: "Cameras",
};

export const getCatalog = cache(
  (kind: CatalogKind): Promise<CatalogEntry[]> =>
    kind === "films" ? getFilmCatalog() : getCameraCatalog()
);

export async function getEntry(kind: CatalogKind, slug: string) {
  const list = await getCatalog(kind);
  return { list, entry: findBySlug(list, slug) };
}

export function pageTitle(kind: CatalogKind, entry: CatalogEntry): string {
  const name = catalogName(entry);
  return kind === "films" ? `${name} film photos` : `${name} photos`;
}

export function pageDescription(
  kind: CatalogKind,
  entry: CatalogEntry
): string {
  const text = entry.description?.trim();
  if (text) return truncate(text, maxDescription);
  const name = catalogName(entry);
  const shot =
    kind === "films" ? `shot on ${name} film` : `shot on the ${name}`;
  return `${entry.postCount} photographs ${shot}, from the AnalogDB archive.`;
}

export function entryMatch(
  kind: CatalogKind,
  entry: CatalogEntry
): Partial<CatalogMatch> {
  return kind === "films"
    ? { film_make: entry.make, film_type: entry.name }
    : { camera_make: entry.make, camera_model: entry.name };
}

export type EntryFact = {
  icon: "photos" | "speed" | "color";
  label: string;
};

function photosFact(entry: CatalogEntry): EntryFact {
  return {
    icon: "photos",
    label: `${entry.postCount.toLocaleString("en-US")} photos`,
  };
}

export function entryFacts(
  kind: CatalogKind,
  entry: CatalogEntry
): EntryFact[] {
  const facts: EntryFact[] = [photosFact(entry)];
  if (kind === "films") {
    if (entry.speed) facts.push({ icon: "speed", label: `ISO ${entry.speed}` });
    if (entry.colorType === "color") {
      facts.push({ icon: "color", label: "color" });
    }
    if (entry.colorType === "bw") {
      facts.push({ icon: "color", label: "black & white" });
    }
  }
  return facts;
}

export function detailMetadata(
  kind: CatalogKind,
  entry: CatalogEntry | null
): Metadata {
  if (!entry) return { title: labels[kind] };
  const title = pageTitle(kind, entry);
  const description = pageDescription(kind, entry);
  const url = catalogHref(kind, entry);
  const images = entry.cover
    ? [
        {
          url: entry.cover.url,
          width: entry.cover.width,
          height: entry.cover.height,
          alt: title,
        },
      ]
    : undefined;
  return {
    title,
    description,
    alternates: { canonical: url },
    openGraph: {
      siteName: "AnalogDB",
      type: "website",
      title,
      description,
      url,
      images,
    },
    twitter: { card: "summary_large_image", title, description, images },
  };
}

async function loadFirstPage(
  fixed: PostsFilters,
  searchParams: Record<string, string | string[] | undefined>
) {
  let filters = { ...fixed, ...loadSort(searchParams) };
  if (filters.sort === "random" && !isValidSeed(filters.seed)) {
    filters = { ...filters, seed: pickSeed() };
  }

  const { device } = userAgent({ headers: await headers() });

  let initialPage = null;
  try {
    const response = await getFirstPosts({
      ...toPostsRequest(filters),
      pageSize: firstPageSize,
    });
    initialPage = slim(response);
  } catch {}

  return {
    fixed,
    initialFilters: filters,
    initialPage,
    initialColumns: guessColumns(device),
  };
}

export async function loadDetail(
  kind: CatalogKind,
  entry: CatalogEntry,
  list: CatalogEntry[],
  searchParams: Record<string, string | string[] | undefined>
) {
  const fixed = catalogFilters(entryMatch(kind, entry));
  return {
    ...(await loadFirstPage(fixed, searchParams)),
    facts: entryFacts(kind, entry),
    related: sameMake(list, entry).slice(0, maxRelated),
  };
}

export async function loadKeywordDetail(
  entry: CatalogEntry,
  searchParams: Record<string, string | string[] | undefined>
) {
  const fixed = catalogFilters({ text: entry.make });
  return {
    ...(await loadFirstPage(fixed, searchParams)),
    facts: [photosFact(entry)],
  };
}

export function detailJsonLd(kind: CatalogKind, entry: CatalogEntry) {
  const url = `${siteURL}${catalogHref(kind, entry)}`;
  const name = catalogName(entry);
  return [
    {
      "@context": "https://schema.org",
      "@type": "CollectionPage",
      name: pageTitle(kind, entry),
      description: pageDescription(kind, entry),
      url,
      ...(entry.cover && {
        primaryImageOfPage: {
          "@type": "ImageObject",
          contentUrl: entry.cover.url,
          url: `${siteURL}/post/${entry.cover.id}`,
        },
      }),
    },
    {
      "@context": "https://schema.org",
      "@type": "BreadcrumbList",
      itemListElement: [
        { "@type": "ListItem", position: 1, name: "Home", item: siteURL },
        {
          "@type": "ListItem",
          position: 2,
          name: labels[kind],
          item: `${siteURL}/${kind}`,
        },
        { "@type": "ListItem", position: 3, name, item: url },
      ],
    },
  ];
}

export function indexJsonLd(
  kind: CatalogKind,
  title: string,
  description: string,
  list: CatalogEntry[]
) {
  return {
    "@context": "https://schema.org",
    "@type": "CollectionPage",
    name: title,
    description,
    url: `${siteURL}/${kind}`,
    mainEntity: {
      "@type": "ItemList",
      numberOfItems: list.length,
      itemListElement: list.map((entry, i) => ({
        "@type": "ListItem",
        position: i + 1,
        name: catalogName(entry),
        url: `${siteURL}${catalogHref(kind, entry)}`,
      })),
    },
  };
}
