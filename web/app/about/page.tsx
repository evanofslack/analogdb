import { getAuthorsTotalCount } from "@app/actions/authors";
import {
  getPosts,
  getPostsSimilar,
  getPostsTotalCount,
} from "@app/actions/posts";
import About from "@components/about";
import styles from "@components/gallery.module.css";
import Header from "@components/header";
import { checkAdminAuth } from "@lib/auth";
import {
  PostIdSimilarGetRequest,
  PostsGetRequest,
  PostsGetSortEnum,
  ServerPostResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "AnalogDB",
  description: "Film photography database",
};

export const revalidate = 60;

interface ColorData {
  red: NonNullable<ServerPostResponse["posts"]>;
  navy: NonNullable<ServerPostResponse["posts"]>;
  olive: NonNullable<ServerPostResponse["posts"]>;
}

interface SimilarityData {
  centerPost: NonNullable<ServerPostResponse["posts"]>[0];
  similarPosts: NonNullable<ServerSimilarPostsResponse["posts"]>;
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
    return response.posts || [];
  });

  const [red, navy, olive] = await Promise.all(promises);
  return { red, navy, olive };
}

async function fetchSimilarityData(): Promise<SimilarityData[]> {
  const ids = [
    32298, 34246, 34252, 533, 30293, 4501, 5211, 1043, 4385, 2235, 6912, 33116,
    2941, 1862, 30131,
  ];

  for (let i = ids.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [ids[i], ids[j]] = [ids[j], ids[i]];
  }

  const BATCH_SIZE = 5;
  const allData: SimilarityData[] = [];

  for (let i = 0; i < ids.length; i += BATCH_SIZE) {
    const batch = ids.slice(i, i + BATCH_SIZE);

    const promises = batch.map(async (id) => {
      try {
        const postParams: PostsGetRequest = { id };
        const postResponse = await getPosts(postParams);
        const post = postResponse.posts?.[0];

        if (!post) return null;

        const similarParams: PostIdSimilarGetRequest = {
          id: post.id,
          pageSize: 6,
          nsfw: false,
          grayscale: false,
        };

        const similarResponse = await getPostsSimilar(similarParams);

        return {
          centerPost: post,
          similarPosts: similarResponse.posts || [],
        };
      } catch (error) {
        console.error("Fail fetch post or similar posts for id:", id, error);
        return null;
      }
    });

    const batchResults = await Promise.all(promises);
    allData.push(
      ...batchResults.filter((data): data is SimilarityData => data !== null)
    );
  }
  return allData;
}

async function getData(): Promise<AboutData> {
  const numPosts = await getPostsTotalCount();
  const numAuthors = await getAuthorsTotalCount();

  const [colorData, allSimilarityData] = await Promise.all([
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

export default async function AboutPage() {
  const data = await getData();
  const isAdmin = await checkAdminAuth();

  return (
    <div className={styles.container}>
      <Header isAdmin={isAdmin} />
      <About data={data} />
    </div>
  );
}
