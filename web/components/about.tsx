"use client";

import { postApi, postsApi } from "@lib/client";
import { CodeHighlight } from "@mantine/code-highlight";
import { useBreakpoint } from "@providers/breakpoint";
import { IconPolaroid, IconUsers } from "@tabler/icons-react";
import {
  AnalogdbPost,
  PostIdSimilarGetRequest,
  PostsGetRequest,
  PostsGetSortEnum,
  ServerPostResponse,
} from "analogdb-generated";
import Image from "next/image";
import Link from "next/link";
import React, { useEffect, useState } from "react";
import styles from "./about.module.css";
import Footer from "./footer";

interface AboutProps {
  data: {
    numPosts: number;
    numAuthors: number;
  };
}

interface ColorData {
  red: AnalogdbPost[];
  navy: AnalogdbPost[];
  olive: AnalogdbPost[];
}

interface SimilarityData {
  centerPost: AnalogdbPost | null;
  similarPosts: AnalogdbPost[];
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
    console.log("Posts API request params:", params);
    const response = await postsApi.postsGet(params);
    console.log("Posts API response:", response.posts?.length, "posts");
    return response;
  } catch (error: any) {
    console.error("Posts API request failed:", error.status, error.message);
    if (error.response) {
      console.error(
        "Error response:",
        await error.response.text().catch(() => "Could not read response")
      );
    }
    throw error;
  }
}

async function makeSimilarRequest(
  params: PostIdSimilarGetRequest
): Promise<ServerPostResponse> {
  try {
    console.log("Similar API request for post ID:", params.id);
    const response = await postApi.postIdSimilarGet(params);
    console.log(
      "Similar API response:",
      response.posts?.length,
      "similar posts"
    );
    return response;
  } catch (error: any) {
    console.error(
      "Similar API request failed for post",
      params.id,
      ":",
      error.status,
      error.message
    );
    if (error.response) {
      console.error(
        "Error response:",
        await error.response.text().catch(() => "Could not read response")
      );
    }
    throw error;
  }
}

export default function About(props: AboutProps) {
  const breakpoints = useBreakpoint();
  let isMobile = false;
  if (breakpoints["sm"]) {
    isMobile = true;
  }

  const numPosts: number = props.data.numPosts;
  const numAuthors: number = props.data.numAuthors;

  const [colorData, setColorData] = useState<ColorData>({
    red: [],
    navy: [],
    olive: [],
  });
  const [similarityData, setSimilarityData] = useState<SimilarityData>({
    centerPost: null,
    similarPosts: [],
  });
  const [loading, setLoading] = useState<boolean>(true);
  const [similarityLoading, setSimilarityLoading] = useState<boolean>(true);

  useEffect(() => {
    const fetchColorData = async (): Promise<void> => {
      try {
        const colors: (keyof ColorData)[] = ["red", "navy", "olive"];
        const apiColors: string[] = ["red", "navy", "olive"];

        const promises = colors.map((_, index) => {
          const apiColor = apiColors[index];
          const params: PostsGetRequest = {
            color: [apiColor],
            minColor: [COLOR_MIN_VALUES[apiColor]],
            pageSize: 30,
            nsfw: false,
            sort: PostsGetSortEnum.Random,
            ratioMin: 0.7,
            ratioMax: 1.5,
          };
          return makeRequest(params);
        });

        const results = await Promise.all(promises);
        setColorData({
          red: results[0].posts || [],
          navy: results[1].posts || [],
          olive: results[2].posts || [],
        });
      } catch (error) {
        console.error("Failed to fetch color data:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchColorData();
  }, []);

  useEffect(() => {
    const fetchSimilarityData = async (): Promise<void> => {
      try {
        console.log("Starting similarity data fetch");

        const topPostsParams: PostsGetRequest = {
          pageSize: 20,
          nsfw: false,
          grayscale: false,
          sort: PostsGetSortEnum.Score,
          ratioMin: 0.7,
          ratioMax: 1.5,
        };

        const topPostsResponse = await makeRequest(topPostsParams);
        const topPosts = topPostsResponse.posts || [];
        console.log("Fetched", topPosts.length, "top posts");

        if (topPosts.length < 3) {
          console.log("Not enough posts for similarity clusters");
          return;
        }

        const shuffled = [...topPosts].sort(() => 0.5 - Math.random());
        const centerPost = shuffled[0];

        const similarParams: PostIdSimilarGetRequest = {
          id: centerPost.id,
          pageSize: 6,
          nsfw: false,
        };

        const similarResponse = await makeSimilarRequest(similarParams);
        const similarPosts = similarResponse.posts || [];

        setSimilarityData({ centerPost, similarPosts });
      } catch (error) {
        console.error("Failed to fetch similarity data:", error);
      } finally {
        setSimilarityLoading(false);
      }
    };

    fetchSimilarityData();
  }, []);

  const apiQuery: string = "curl https://api.analogdb.com/posts";

  const apiResponse: string = `
"meta":{
  "total_posts":3637,
  "page_size":20,
  "next_page_id":1672251647,
  "next_page_url":"/posts?sort=time&page_size=20&page_id=1672251647"
},
"posts":[
  {
    "id":5127,
    "title":"Exam | Olympus OM-2n | 50mm 1.8 | Vision3 250D",
    "author":"Crazylyric",
    "permalink":"https://www.reddit.com/r/analog/comments/zyk2sp/exam_olympus_om2n_50mm_18_vision3_250d/",
    "score":163,
    "nsfw":false,
    "grayscale":false,
    "timestamp":1672356457,
    "sprocket":false
    "images":[
      {
        "resolution":"low",
        "url":"https://d3i73ktnzbi69i.cloudfront.net/8ed69a77-83fc-4a82-8994-935f82cada2e.jpeg",
        "width":720,
        "height":477
      },
      {
        "resolution":"medium",
        "url":"https://d3i73ktnzbi69i.cloudfront.net/d3ed07e5-b094-452f-b567-6d24b7d93f39.jpeg"
        "width":720,
        "height":477
      },
      {
        "resolution":"high",
        "url":"https://d3i73ktnzbi69i.cloudfront.net/b68bb45b-e723-4010-81d7-2c1a38cdffe1.jpeg"
        "width":1440,
        "height":955
      },
      {
        "resolution":"raw",
        "url":"https://d3i73ktnzbi69i.cloudfront.net/de6a9627-5127-4920-b6f4-d1078e7d3c35.jpeg"
        "width":3089,
        "height":2048
      }
     ],
  },
  ...
]`;

  const renderColorRow = (
    images: AnalogdbPost[],
    direction: "left" | "right",
    delay: number = 0
  ): React.ReactElement | null => {
    if (!images.length) return null;

    const duplicatedImages = [...images, ...images];

    return (
      <div className={styles.colorRow}>
        <div
          className={`${styles.colorScrollContainer} ${
            direction === "left" ? styles.scrollLeft : styles.scrollRight
          }`}
          style={{ animationDelay: `${delay}s` }}
        >
          {duplicatedImages.map((post, index) => {
            const image =
              post.images?.find((img) => img.resolution === "medium") ||
              post.images?.[0];
            if (!image) return null;

            return (
              <div
                key={`${post.id}-${index}`}
                className={styles.colorImageContainer}
              >
                <Image
                  src={image.url}
                  alt={post.title}
                  width={image.width}
                  height={image.height}
                  className={styles.colorImage}
                />
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  const renderSimilarityClusters = (): React.ReactElement | null => {
    if (similarityLoading || !similarityData.similarPosts.length) return null;

    const clusterPosition = { left: "50%", top: "20%" };

    const similarPositions = [
      { top: "-160px", left: "-40px" }, // top-left
      { top: "-130px", right: "-120px" }, // top-right
      { bottom: "-150px", left: "-130px" }, // bottom-left
      { bottom: "-165px", right: "-65px" }, // bottom-right
      { top: "45%", left: "-190px", transform: "translateY(-50%)" }, // middle-left
      { top: "55%", right: "-175px", transform: "translateY(-50%)" }, // middle-right
    ];
    const centerPost = similarityData.centerPost;
    const centerImage =
      centerPost.images?.find((img) => img.resolution === "medium") ||
      similarityData.centerPost.images?.[0];
    if (!centerImage) return null;
    const similarPosts = similarityData.similarPosts;

    return (
      <div className={styles.clustersContainer}>
        <div
          key={centerPost.id}
          className={styles.clusterContainer}
          style={clusterPosition}
        >
          <div className={styles.clusterCenterContainer}>
            <Image
              src={centerImage.url}
              alt={centerPost.title}
              fill
              sizes="(max-width: 768px) 140px, 300px"
              className={styles.clusterCenterImage}
              style={{ objectFit: "cover" }}
            />
          </div>

          {similarPosts.slice(0, 6).map((post, index) => {
            const image =
              post.images?.find((img) => img.resolution === "medium") ||
              post.images?.[0];
            if (!image || !similarPositions[index]) return null;

            return (
              <div
                key={post.id}
                className={styles.clusterSimilarContainer}
                style={similarPositions[index]}
              >
                <Image
                  src={image.url}
                  alt={post.title}
                  fill
                  sizes="(max-width: 768px) 60px, 100px"
                  className={styles.clusterSimilarImage}
                  style={{ objectFit: "cover" }}
                />
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  return (
    <main>
      <div className={styles.container}>
        <div className={styles.sectionOne}>
          <div className={styles.subSection}>
            <div className={styles.title}>Film for all</div>
            <p className={styles.subtitle}>
              AnalogDB is a curated database featuring thousands of film
              photographs. And it is always growing, with new pictures added
              every day.
            </p>
            <Link href="/" className={styles.link}>
              view latest
            </Link>
          </div>
          {!isMobile && (
            <div className={styles.stats}>
              <div className={styles.statRow}>
                <IconPolaroid
                  size={40}
                  color="#cacaca"
                  stroke={1.1}
                  className={styles.statIcon}
                />
                <div className={styles.statCol}>
                  <p className={styles.statNum}>{numPosts.toLocaleString()}</p>
                  <p className={styles.statTitle}>photos</p>
                </div>
              </div>

              <div className={styles.statRow}>
                <IconUsers
                  size={36}
                  color="#cacaca"
                  stroke={1.5}
                  className={styles.statIcon}
                />
                <div className={styles.statCol}>
                  <p className={styles.statNum}>
                    {numAuthors.toLocaleString()}
                  </p>
                  <p className={styles.statTitle}>photographers</p>
                </div>
              </div>
            </div>
          )}
        </div>

        <div className={styles.sectionTwoBg}>
          <div className={styles.colorSection}>
            {!loading && (
              <>
                {renderColorRow(colorData.red, "right", 0)}
                {renderColorRow(colorData.navy, "left", 0)}
                {renderColorRow(colorData.olive, "right", 0)}
              </>
            )}
            <div className={styles.colorTextOverlay}>
              <div className={styles.title}>Color Intelligence</div>
              <p className={styles.subtitle}>
                Dominant colors are extracted from every photo, allowing you to
                discover images by their visual palette. Search and analyze
                images by their distinct colors.
              </p>
              <Link href="/search?color=red" className={styles.link}>
                explore colors
              </Link>
            </div>
          </div>
        </div>

        <div className={styles.sectionSimilarityBg}>
          <div className={styles.similaritySection}>
            {renderSimilarityClusters()}
            <div className={styles.similarityTextOverlay}>
              <div className={styles.title}>Vector Similarity</div>
              <p className={styles.subtitle}>
                Every image is encoded with AI-powered vector embeddings,
                enabling intelligent visual similarity search. Discover photos
                that share composition, subject matter, and aesthetic qualities.
              </p>
              <Link href="/search" className={styles.link}>
                find similar
              </Link>
            </div>
          </div>
        </div>

        <div className={styles.sectionThreeBg}>
          <div className={styles.sectionThree}>
            <div>
              {!isMobile && (
                <div className={styles.apiDemoContainer}>
                  <div className={styles.apiDemo}>
                    <CodeHighlight
                      code={apiQuery}
                      language="javascript"
                      styles={{
                        code: {
                          fontSize: "0.75rem",
                          maxWidth: "40vw",
                        },
                      }}
                    />
                  </div>
                  <div className={styles.apiDemo}>
                    <CodeHighlight
                      code={apiResponse}
                      language="javascript"
                      styles={{
                        code: {
                          fontSize: "0.75rem",
                          maxHeight: "70vh",
                          maxWidth: "40vw",
                        },
                      }}
                    />
                  </div>
                </div>
              )}
            </div>
            <div>
              <div className={styles.title}>Accessible API</div>
              <p className={styles.subtitle}>
                The entire collection of film is exposed through a simple and
                modern JSON API. Embedding beautiful film photos in your
                projects has never been easier.
              </p>
              <Link href="/docs" className={styles.link}>
                read the docs
              </Link>
            </div>
          </div>
        </div>

        <div className={styles.sectionFourBg}>
          <div className={styles.sectionFour}>
            <div>
              <div className={styles.title}>Open-source</div>
              <p className={styles.subtitle}>
                All code made publicly available on Github with flexible
                licensing. AnalogDB is an open community where all contributions
                are welcome!
              </p>
              <a
                className={styles.link}
                href="https://github.com/evanofslack/analogdb"
              >
                view source
              </a>
            </div>
            {!isMobile && (
              <div className={styles.imageThree}>
                <Image
                  src={"/github_logo.png"}
                  alt={`example AnalogDB API call`}
                  width="384"
                  height="216"
                  quality={100}
                />
              </div>
            )}
          </div>
        </div>
      </div>
      <Footer />
    </main>
  );
}
