import "server-only";
import {
  AboutData,
  AboutPhoto,
  ApiSample,
  ColorData,
  FilmSet,
  SearchDemo,
  SimilarityData,
} from "@lib/about";
import { authorized_fetch } from "@lib/client";
import { publicURL } from "@lib/constants";
import { getAuthorsTotalCount } from "@lib/data/authors";
import { getCameraOptions } from "@lib/data/cameras";
import { getFilmCatalog, getFilmOptions } from "@lib/data/films";
import { getPosts, getPostsSimilar, getPostsTotalCount } from "@lib/data/posts";
import { searchPosts } from "@lib/data/search";
import { smallImage } from "@lib/images";
import { cameraName, filmName, postAlt } from "@lib/seo";
import { AnalogdbPost, PostsGetSortEnum } from "analogdb-generated";
import { unstable_cache } from "next/cache";

const COLOR_MIN_VALUES: Record<string, number> = {
  red: 0.4,
  navy: 0.4,
  olive: 0.4,
};

const SIMILARITY_IDS = [
  32298, 34246, 34252, 533, 30293, 4501, 5211, 1043, 4385, 2235, 6912, 33116,
  2941, 1862, 30131,
];

const NUM_CLUSTERS = 8;

// labels are short so each fits on one line under its set
const FILMS = [
  {
    make: "kodak",
    type: "portra 400",
    slug: "kodak-portra-400",
    label: "portra 400",
  },
  {
    make: "kodak",
    type: "ektar 100",
    slug: "kodak-ektar-100",
    label: "ektar 100",
  },
  {
    make: "cinestill",
    type: "800t",
    slug: "cinestill-800t",
    label: "cinestill 800t",
  },
  {
    make: "fujifilm",
    type: "fujichrome velvia 50",
    slug: "fujifilm-fujichrome-velvia-50",
    label: "velvia 50",
  },
  {
    make: "lomography",
    type: "lomochrome purple xr",
    slug: "lomography-lomochrome-purple-xr",
    label: "lomochrome purple",
  },
  {
    make: "kodak",
    type: "aerochrome",
    slug: "kodak-aerochrome",
    label: "aerochrome",
  },
];

const FILM_SET_SIZE = 4;

const QUERIES = [
  "man in black",
  "neon at night",
  "cat in window",
  "red flowers",
  "foggy forest",
  "girl at sunset",
  "snow covered hill",
];

const SEARCH_RESULTS = 6;
const SEARCH_KEYWORDS = 5;
const SEARCH_CONCURRENCY = 2;

const API_SAMPLE_IDS = [4777, 3995];

function lensName(post: AnalogdbPost): string | undefined {
  const focal = post.focalLength ? `${post.focalLength}mm` : "";
  const lens = [focal, post.aperture].filter(Boolean).join(" ");
  return lens || undefined;
}

function toPhoto(post: AnalogdbPost): AboutPhoto | null {
  const image =
    post.images?.find((img) => img.resolution === "medium") || post.images?.[0];
  if (!post.id || !image?.url) return null;
  return {
    id: post.id,
    url: image.url,
    smallUrl: smallImage(post.images)?.url ?? image.url,
    width: image.width,
    height: image.height,
    alt: postAlt(post),
    caption: post.caption || undefined,
    camera: cameraName(post) ?? undefined,
    film: filmName(post) ?? undefined,
    lens: lensName(post),
    keywords: [...(post.keywords ?? [])]
      .sort((a, b) => (b.weight ?? 0) - (a.weight ?? 0))
      .map((keyword) => keyword.word)
      .filter((word): word is string => Boolean(word))
      .slice(0, 3),
    colors: (post.colors ?? [])
      .map((color) => color.hex)
      .filter((hex): hex is string => Boolean(hex))
      .slice(0, 5),
  };
}

function toPhotos(posts: AnalogdbPost[] | undefined): AboutPhoto[] {
  return (posts || [])
    .map(toPhoto)
    .filter((photo): photo is AboutPhoto => photo !== null);
}

async function fetchColorData(): Promise<ColorData> {
  const colors = ["red", "navy", "olive"] as const;

  const promises = colors.map(async (color) => {
    try {
      const response = await getPosts({
        color: [color],
        minColor: [COLOR_MIN_VALUES[color]],
        pageSize: 30,
        nsfw: false,
        sort: PostsGetSortEnum.Random,
        ratioMin: 0.7,
        ratioMax: 1.5,
      });
      return toPhotos(response.posts);
    } catch (error) {
      console.error("Fail fetch color posts for:", color, error);
      return [];
    }
  });

  const [red, navy, olive] = await Promise.all(promises);
  return { red, navy, olive };
}

async function fetchSimilarityData(): Promise<SimilarityData[]> {
  const ids = [...SIMILARITY_IDS];

  for (let i = ids.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [ids[i], ids[j]] = [ids[j], ids[i]];
  }

  const promises = ids.slice(0, NUM_CLUSTERS).map(async (id) => {
    try {
      const [postResponse, similarResponse] = await Promise.all([
        getPosts({ id }),
        getPostsSimilar({ id, pageSize: 6, nsfw: false, grayscale: false }),
      ]);
      const post = postResponse.posts?.[0];
      const centerPost = post ? toPhoto(post) : null;

      if (!centerPost) return null;

      return {
        centerPost,
        similarPosts: toPhotos(similarResponse.posts),
      };
    } catch (error) {
      console.error("Fail fetch post or similar posts for id:", id, error);
      return null;
    }
  });

  const results = await Promise.all(promises);
  return results.filter((data): data is SimilarityData => data !== null);
}

async function fetchFilms(): Promise<FilmSet[]> {
  const catalog = await getFilmCatalog();

  const promises = FILMS.map(async (film) => {
    try {
      const response = await getPosts({
        filmMake: film.make,
        filmType: film.type,
        sort: PostsGetSortEnum.Score,
        pageSize: 30,
        nsfw: false,
        ratioMin: 0.7,
        ratioMax: 1.5,
      });
      const photos = toPhotos(response.posts);
      if (photos.length < FILM_SET_SIZE) return null;
      const entry = catalog.find((item) => item.slug === film.slug);
      return {
        slug: film.slug,
        label: film.label,
        postCount: entry?.postCount ?? 0,
        photos,
      };
    } catch (error) {
      console.error("Fail fetch film posts for:", film.slug, error);
      return null;
    }
  });

  const results = await Promise.all(promises);
  return results.filter((film): film is FilmSet => film !== null);
}

async function fetchSearch(query: string): Promise<SearchDemo | null> {
  try {
    const response = await searchPosts({
      q: query,
      pageSize: SEARCH_RESULTS,
      nsfw: false,
      grayscale: false,
    });
    const photos = toPhotos(response.posts);
    if (photos.length < SEARCH_RESULTS) return null;
    return {
      query,
      photos,
      keywords: (response.relatedKeywords ?? []).slice(0, SEARCH_KEYWORDS),
    };
  } catch (error) {
    console.error("Fail fetch search for:", query, error);
    return null;
  }
}

// backend text search has few slots, so only a couple run at once
async function fetchSearches(): Promise<SearchDemo[]> {
  const results: (SearchDemo | null)[] = [];
  for (let i = 0; i < QUERIES.length; i += SEARCH_CONCURRENCY) {
    const batch = QUERIES.slice(i, i + SEARCH_CONCURRENCY);
    results.push(...(await Promise.all(batch.map(fetchSearch))));
  }
  return results.filter((search): search is SearchDemo => search !== null);
}

type RawPost = Record<string, unknown> & {
  images?: { resolution?: string }[];
  colors?: unknown[];
  keywords?: unknown[];
};

function toApiSample(id: number, post: RawPost): ApiSample {
  const images = post.images ?? [];
  const full = {
    ...post,
    images: images.filter(
      (image) => image.resolution === "low" || image.resolution === "raw"
    ),
    colors: (post.colors ?? []).slice(0, 3),
  };
  const short = {
    id: post.id,
    title: post.title,
    caption: post.caption,
    camera_make: post.camera_make,
    camera_model: post.camera_model,
    film_make: post.film_make,
    film_type: post.film_type,
    keywords: (post.keywords ?? []).slice(0, 4),
    images: images.filter((image) => image.resolution === "low"),
  };
  return {
    query: `curl ${publicURL}/post/${id}`,
    full: JSON.stringify(full, null, 2),
    short: JSON.stringify(short, null, 2),
  };
}

// raw json keeps the field names and order the api actually returns
async function fetchApiSample(): Promise<ApiSample | null> {
  for (const id of API_SAMPLE_IDS) {
    try {
      const response = await authorized_fetch(`/post/${id}`, "GET", 3600);
      if (!response.ok) continue;
      return toApiSample(id, await response.json());
    } catch (error) {
      console.error("Fail fetch api sample post:", id, error);
    }
  }
  return null;
}

async function getData(): Promise<AboutData> {
  const [
    numPosts,
    numAuthors,
    cameras,
    filmOptions,
    colorData,
    allSimilarityData,
    films,
    searches,
    sample,
  ] = await Promise.all([
    getPostsTotalCount(),
    getAuthorsTotalCount(),
    getCameraOptions(),
    getFilmOptions(),
    fetchColorData(),
    fetchSimilarityData(),
    fetchFilms(),
    fetchSearches(),
    fetchApiSample(),
  ]);

  return {
    numPosts,
    numAuthors,
    numCameras: cameras.length,
    numFilms: filmOptions.length,
    colorData,
    allSimilarityData,
    films,
    searches,
    apiSample: sample,
  };
}

export const getAboutData = unstable_cache(getData, ["about-data-v2"], {
  revalidate: 3600,
});
