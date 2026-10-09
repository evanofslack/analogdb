import { getAuthorsTotalCount } from "@lib/data/authors";
import {
  getPosts,
  getPostsSimilar,
  getPostsTotalCount,
} from "@lib/data/posts";
import About, { AboutImage } from "@components/about";
import styles from "@components/gallery.module.css";
import Header from "@components/header";
import {
  AnalogdbPost,
  PostIdSimilarGetRequest,
  PostsGetRequest,
  PostsGetSortEnum,
} from "analogdb-generated";
import { Metadata } from "next";
import { unstable_cache } from "next/cache";

export const metadata: Metadata = {
  title: "About",
  description: "What AnalogDB is, with colors and similar photos from the archive",
};

export const dynamic = "force-dynamic";

interface ColorData {
  red: AboutImage[];
  navy: AboutImage[];
  olive: AboutImage[];
}

interface SimilarityData {
  centerPost: AboutImage;
  similarPosts: AboutImage[];
}

interface AboutData {
  numPosts: number;
  numAuthors: number;
  colorData: ColorData;
  allSimilarityData: SimilarityData[];
}

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

function toImage(post: AnalogdbPost): AboutImage | null {
  const image =
    post.images?.find((img) => img.resolution === "medium") ||
    post.images?.[0];
  if (!image?.url) return null;
  return {
    id: post.id,
    url: image.url,
    width: image.width,
    height: image.height,
  };
}

function toImages(posts: AnalogdbPost[] | undefined): AboutImage[] {
  return (posts || [])
    .map(toImage)
    .filter((image): image is AboutImage => image !== null);
}

async function fetchColorData(): Promise<ColorData> {
  const colors = ["red", "navy", "olive"] as const;

  const promises = colors.map(async (color) => {
    const params: PostsGetRequest = {
      color: [color],
      minColor: [COLOR_MIN_VALUES[color]],
      pageSize: 30,
      nsfw: false,
      sort: PostsGetSortEnum.Random,
      ratioMin: 0.7,
      ratioMax: 1.5,
    };

    const response = await getPosts(params);
    return toImages(response.posts);
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
      const similarParams: PostIdSimilarGetRequest = {
        id,
        pageSize: 6,
        nsfw: false,
        grayscale: false,
      };

      const [postResponse, similarResponse] = await Promise.all([
        getPosts({ id }),
        getPostsSimilar(similarParams),
      ]);
      const post = postResponse.posts?.[0];
      const centerPost = post ? toImage(post) : null;

      if (!centerPost) return null;

      return {
        centerPost,
        similarPosts: toImages(similarResponse.posts),
      };
    } catch (error) {
      console.error("Fail fetch post or similar posts for id:", id, error);
      return null;
    }
  });

  const results = await Promise.all(promises);
  return results.filter((data): data is SimilarityData => data !== null);
}

async function getData(): Promise<AboutData> {
  const [numPosts, numAuthors, colorData, allSimilarityData] =
    await Promise.all([
      getPostsTotalCount(),
      getAuthorsTotalCount(),
      fetchColorData(),
      fetchSimilarityData(),
    ]);

  return {
    numPosts,
    numAuthors,
    colorData,
    allSimilarityData,
  };
}

const getCachedData = unstable_cache(getData, ["about-data"], {
  revalidate: 3600,
});

export default async function AboutPage() {
  const data = await getCachedData();

  return (
    <div className={styles.container}>
      <Header />
      <About data={data} />
    </div>
  );
}
