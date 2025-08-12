import About from "@components/about";
import styles from "@components/gallery.module.css";
import Header from "@components/header";
import { checkAdminAuth } from "@lib/auth";
import { authorized_fetch, postApi, postsApi } from "@lib/client";
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

interface PostsResponse {
  meta: {
    total_posts: number;
  };
  posts?: Array<{
    id: number;
    title: string;
    author: string;
    permalink: string;
    score: number;
    nsfw: boolean;
    grayscale: boolean;
    timestamp: number;
    sprocket: boolean;
    images?: Array<{
      resolution: string;
      url: string;
      width: number;
      height: number;
    }>;
  }>;
}

interface AuthorsResponse {
  authors: string[];
}

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

async function makeRequest(
  params: PostsGetRequest
): Promise<ServerPostResponse> {
  try {
    const response = await postsApi.postsGet(params);
    return response;
  } catch (error) {
    console.error("API request failed:", error);
    throw error;
  }
}

async function makeSimilarRequest(
  params: PostIdSimilarGetRequest
): Promise<ServerSimilarPostsResponse> {
  try {
    const response = await postApi.postIdSimilarGet(params);
    return response;
  } catch (error) {
    console.error("API request failed:", error);
    throw error;
  }
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

    const response = await makeRequest(params);
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

  // Shuffle array
  for (let i = ids.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [ids[i], ids[j]] = [ids[j], ids[i]];
  }

  const allData: SimilarityData[] = [];

  for (const id of ids) {
    try {
      // Fetch the center post
      const postParams: PostsGetRequest = { id: id };
      const postResponse = await makeRequest(postParams);
      const post = postResponse.posts?.[0];

      if (!post) continue;

      // Fetch similar posts
      const similarParams: PostIdSimilarGetRequest = {
        id: post.id,
        pageSize: 6,
        nsfw: false,
        grayscale: false,
      };

      const similarResponse = await makeSimilarRequest(similarParams);
      const similarPosts = similarResponse.posts || [];

      allData.push({
        centerPost: post,
        similarPosts,
      });
    } catch (error) {
      console.error("Failed to fetch data for ID:", id, error);
    }
  }

  return allData;
}

async function getData(): Promise<AboutData> {
  // Existing data fetching
  const postsRoute = "/posts";
  const postsResponse = await authorized_fetch(postsRoute, "GET");
  const postsData: PostsResponse = await postsResponse.json();
  const numPosts = postsData.meta.total_posts;

  const authorsRoute = "/authors";
  const authorsResponse = await authorized_fetch(authorsRoute, "GET");
  const authorsData: AuthorsResponse = await authorsResponse.json();
  const numAuthors = Array.from(new Set(authorsData.authors)).length;

  // New data fetching
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
