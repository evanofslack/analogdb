"use client";

import { CodeHighlight } from "@mantine/code-highlight";
import { useBreakpoint } from "@providers/breakpoint";
import { IconPolaroid, IconUsers } from "@tabler/icons-react";
import { AnalogdbPost } from "analogdb-generated";
import Image from "next/image";
import Link from "next/link";
import React, { useEffect, useRef, useState } from "react";
import styles from "./about.module.css";
import Footer from "./footer";

interface ColorData {
  red: AnalogdbPost[];
  navy: AnalogdbPost[];
  olive: AnalogdbPost[];
}

interface SimilarityData {
  centerPost: AnalogdbPost;
  similarPosts: AnalogdbPost[];
}

interface AboutProps {
  data: {
    numPosts: number;
    numAuthors: number;
    colorData: ColorData;
    allSimilarityData: SimilarityData[];
  };
}

export default function About(props: AboutProps) {
  const breakpoints = useBreakpoint();
  let isMobile = false;
  if (breakpoints["sm"]) {
    isMobile = true;
  }

  const { numPosts, numAuthors, colorData, allSimilarityData } = props.data;

  const [currentSimilarityIndex, setCurrentSimilarityIndex] =
    useState<number>(0);
  const [isTransitioning, setIsTransitioning] = useState<boolean>(false);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const currentSimilarityData = allSimilarityData[currentSimilarityIndex] || {
    centerPost: null,
    similarPosts: [],
  };

  useEffect(() => {
    if (allSimilarityData.length <= 1) return;

    const startCycling = () => {
      intervalRef.current = setInterval(() => {
        setIsTransitioning(true);

        setTimeout(() => {
          setCurrentSimilarityIndex((prevIndex) => {
            return (prevIndex + 1) % allSimilarityData.length;
          });

          setTimeout(() => {
            setIsTransitioning(false);
          }, 50);
        }, 250);
      }, 7000);
    };

    const timer = setTimeout(() => {
      startCycling();
    }, 7000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
      clearTimeout(timer);
    };
  }, [allSimilarityData]);

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
    if (
      !currentSimilarityData.centerPost ||
      !currentSimilarityData.similarPosts.length
    )
      return null;

    const clusterPosition = { left: "50%", top: "20%" };

    const similarPositions = [
      { top: "-160px", left: "-40px" },
      { top: "-130px", right: "-120px" },
      { bottom: "-150px", left: "-130px" },
      { bottom: "-165px", right: "-65px" },
      { top: "45%", left: "-170px", transform: "translateY(-50%)" },
      { top: "55%", right: "-195px", transform: "translateY(-50%)" },
    ];

    const centerPost = currentSimilarityData.centerPost;
    const centerImage =
      centerPost.images?.find((img) => img.resolution === "medium") ||
      centerPost.images?.[0];
    if (!centerImage) return null;

    const similarPosts = currentSimilarityData.similarPosts;

    const centerWidth = centerImage.width || 400;
    const centerHeight = centerImage.height || 400;
    const centerAspectRatio = centerWidth / centerHeight;
    const centerMaxHeight = Math.min(
      typeof window !== "undefined" ? window.innerWidth * 0.35 : 420,
      420
    );
    const centerContainerWidth = centerMaxHeight * centerAspectRatio;

    return (
      <div className={styles.clustersContainer}>
        <div
          key={centerPost.id}
          className={styles.clusterContainer}
          style={clusterPosition}
        >
          <div
            className={`${styles.clusterCenterContainer} ${
              isTransitioning ? styles.transitioning : ""
            }`}
            style={{
              width: `${centerContainerWidth}px`,
              height: `${centerMaxHeight}px`,
            }}
          >
            <Image
              src={centerImage.url}
              alt={centerPost.title}
              fill
              sizes="(max-width: 768px) 200px, 420px"
              className={styles.clusterCenterImage}
              style={{ objectFit: "cover" }}
            />
          </div>

          {similarPosts.slice(0, 6).map((post, index) => {
            const image =
              post.images?.find((img) => img.resolution === "medium") ||
              post.images?.[0];
            if (!image || !similarPositions[index]) return null;

            const width = image.width || 200;
            const height = image.height || 200;
            const aspectRatio = width / height;
            const maxHeight = Math.min(
              typeof window !== "undefined" ? window.innerWidth * 0.15 : 180,
              180
            );
            const containerWidth = maxHeight * aspectRatio;

            return (
              <div
                key={post.id}
                className={`${styles.clusterSimilarContainer} ${
                  isTransitioning ? styles.transitioning : ""
                }`}
                style={{
                  ...similarPositions[index],
                  width: `${containerWidth}px`,
                  height: `${maxHeight}px`,
                }}
              >
                <div className={styles.clusterConnectionLine} />
                <Image
                  src={image.url}
                  alt={post.title}
                  fill
                  sizes="(max-width: 768px) 90px, 180px"
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
            {renderColorRow(colorData.red, "right", 0)}
            {renderColorRow(colorData.navy, "left", 0)}
            {renderColorRow(colorData.olive, "right", 0)}
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
                Every image is encoded with vector embeddings, enabling
                intelligent visual similarity search. Discover photos that share
                composition and visual patterns.
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
