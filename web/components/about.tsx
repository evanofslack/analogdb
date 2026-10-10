"use client";

import { CodeHighlight } from "@mantine/code-highlight";
import { IconPolaroid, IconUsers } from "@tabler/icons-react";
import Image from "next/image";
import Link from "next/link";
import React, { useEffect, useRef, useState } from "react";
import styles from "./about.module.css";
import Footer from "./footer";

export interface AboutImage {
  id: number;
  url: string;
  smallUrl?: string;
  width?: number;
  height?: number;
}

interface ColorData {
  red: AboutImage[];
  navy: AboutImage[];
  olive: AboutImage[];
}

interface SimilarityData {
  centerPost: AboutImage;
  similarPosts: AboutImage[];
}

interface AboutProps {
  data: {
    numPosts: number;
    numAuthors: number;
    colorData: ColorData;
    allSimilarityData: SimilarityData[];
  };
}

const COLOR_ROW_SIZE = 16;
const MOBILE_ROW_SIZE = 8;

function shuffle<T>(items: T[]): T[] {
  const result = [...items];
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]];
  }
  return result;
}

function pickColorRows(colorData: ColorData, random: boolean): ColorData {
  const pick = (images: AboutImage[]) =>
    (random ? shuffle(images) : images).slice(0, COLOR_ROW_SIZE);
  return {
    red: pick(colorData.red),
    navy: pick(colorData.navy),
    olive: pick(colorData.olive),
  };
}

export default function About(props: AboutProps) {
  const { numPosts, numAuthors } = props.data;

  const [colorData, setColorData] = useState<ColorData>(() =>
    pickColorRows(props.data.colorData, false)
  );
  const [allSimilarityData, setAllSimilarityData] = useState<SimilarityData[]>(
    props.data.allSimilarityData
  );

  const [viewportWidth, setViewportWidth] = useState<number | null>(null);
  const [shuffled, setShuffled] = useState(false);

  // images render only after the shuffle, so the unshuffled server picks never download
  useEffect(() => {
    setColorData(pickColorRows(props.data.colorData, true));
    setAllSimilarityData(shuffle(props.data.allSimilarityData));
    setShuffled(true);
  }, [props.data]);

  useEffect(() => {
    setViewportWidth(window.innerWidth);
  }, []);

  const [currentSimilarityIndex, setCurrentSimilarityIndex] =
    useState<number>(0);
  const [isTransitioning, setIsTransitioning] = useState<boolean>(false);
  const similarityRef = useRef<HTMLDivElement | null>(null);
  const [isSimilarityVisible, setIsSimilarityVisible] = useState(false);

  const currentSimilarityData = allSimilarityData[currentSimilarityIndex] || {
    centerPost: null,
    similarPosts: [],
  };

  useEffect(() => {
    const node = similarityRef.current;
    if (!node) return;
    const observer = new IntersectionObserver(([entry]) =>
      setIsSimilarityVisible(entry.isIntersecting)
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (allSimilarityData.length <= 1 || !isSimilarityVisible) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    const interval = setInterval(() => {
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

    return () => clearInterval(interval);
  }, [allSimilarityData, isSimilarityVisible]);

  const apiQuery: string = "curl https://api.analogdb.com/v1/posts";

  const apiResponse: string = `
"meta":{
  "total_posts":18233,
  "page_size":20,
  "next_cursor":"eyJzIjoidGltZSIsInYiOjE2NzIyNTE2NDcsImlkIjo1MTA4fQ",
  "next_page_url":"/posts?cursor=eyJzIjoidGltZSIsInYiOjE2NzIyNTE2NDcsImlkIjo1MTA4fQ&page_size=20&sort=time"
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

  const apiResponseShort: string = `
"meta":{
  "total_posts":18233,
  "page_size":20,
  "next_cursor":"eyJzIjoidGltZSIsInYiOjE2NzIyNTE2NDcsImlkIjo1MTA4fQ"
},
"posts":[
  {
    "id":5127,
    "title":"Exam | Olympus OM-2n | 50mm 1.8 | Vision3 250D",
    "author":"Crazylyric",
    "score":163,
    "images":[
      {
        "resolution":"low",
        "url":"https://d3i73ktnzbi69i.cloudfront.net/8ed69a77-83fc-4a82-8994-935f82cada2e.jpeg",
        "width":720,
        "height":477
      },
      ...
    ]
  },
  ...
]`;

  const renderMobileColorRows = (): React.ReactElement | null => {
    const rows: [AboutImage[], "left" | "right"][] = [
      [colorData.red.slice(0, MOBILE_ROW_SIZE), "right"],
      [colorData.navy.slice(0, MOBILE_ROW_SIZE), "left"],
    ];
    if (rows.every(([images]) => !images.length)) return null;

    return (
      <div className={styles.mobileOnly}>
        <div className={styles.mobileColorRows}>
          {rows.map(([images, direction]) => (
            <div key={direction} className={styles.mobileColorRow}>
              <div
                className={`${styles.colorScrollContainer} ${
                  direction === "left" ? styles.scrollLeft : styles.scrollRight
                } ${styles.mobileScroll}`}
              >
                {(shuffled ? [...images, ...images] : []).map(
                  (image, index) => (
                    <div
                      key={`${image.id}-${index}`}
                      className={styles.colorImageContainer}
                    >
                      <Image
                        src={image.smallUrl ?? image.url}
                        alt={`image ${image.id}`}
                        width={image.width}
                        height={image.height}
                        className={styles.mobileColorImage}
                      />
                    </div>
                  )
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  };

  const renderMobileSimilarity = (): React.ReactElement | null => {
    const { centerPost, similarPosts } = currentSimilarityData;
    if (!shuffled || !centerPost || !similarPosts.length) return null;
    const fade = isTransitioning ? styles.transitioning : "";

    return (
      <div className={styles.mobileOnly}>
        <div className={styles.mobileSimilarity}>
          <div
            className={`${styles.mobileSimilarCenter} ${fade}`}
            style={{
              aspectRatio: `${centerPost.width || 4} / ${
                centerPost.height || 3
              }`,
            }}
          >
            <Image
              src={centerPost.smallUrl ?? centerPost.url}
              alt={`image ${centerPost.id}`}
              fill
              sizes="100vw"
              style={{ objectFit: "cover" }}
            />
          </div>
          <div className={styles.mobileSimilarGrid}>
            {similarPosts.slice(0, 6).map((image) => (
              <div
                key={image.id}
                className={`${styles.mobileSimilarTile} ${fade}`}
              >
                <Image
                  src={image.smallUrl ?? image.url}
                  alt={`image ${image.id}`}
                  fill
                  sizes="33vw"
                  style={{ objectFit: "cover" }}
                />
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  };

  const renderColorRow = (
    images: AboutImage[],
    direction: "left" | "right",
    delay: number = 0
  ): React.ReactElement | null => {
    if (!shuffled || !images.length) return null;

    const duplicatedImages = [...images, ...images];

    return (
      <div className={`${styles.colorRow} ${styles.desktopOnly}`}>
        <div
          className={`${styles.colorScrollContainer} ${
            direction === "left" ? styles.scrollLeft : styles.scrollRight
          }`}
          style={{ animationDelay: `${delay}s` }}
        >
          {duplicatedImages.map((image, index) => {
            return (
              <div
                key={`${image.id}-${index}`}
                className={styles.colorImageContainer}
              >
                <Image
                  src={image.smallUrl ?? image.url}
                  alt={`image ${image.id}`}
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
      !shuffled ||
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

    const centerImage = currentSimilarityData.centerPost;

    const similarPosts = currentSimilarityData.similarPosts;

    const centerWidth = centerImage.width || 400;
    const centerHeight = centerImage.height || 400;
    const centerAspectRatio = centerWidth / centerHeight;
    const centerMaxHeight = Math.min(
      viewportWidth ? viewportWidth * 0.35 : 420,
      420
    );
    const centerContainerWidth = centerMaxHeight * centerAspectRatio;

    return (
      <div className={`${styles.clustersContainer} ${styles.desktopOnly}`}>
        <div
          key={centerImage.id}
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
              alt={`image ${centerImage.id}`}
              fill
              sizes="(max-width: 768px) 200px, 420px"
              className={styles.clusterCenterImage}
              style={{ objectFit: "cover" }}
            />
          </div>

          {similarPosts.slice(0, 6).map((image, index) => {
            if (!similarPositions[index]) return null;

            const width = image.width || 200;
            const height = image.height || 200;
            const aspectRatio = width / height;
            const maxHeight = Math.min(
              viewportWidth ? viewportWidth * 0.15 : 180,
              180
            );
            const containerWidth = maxHeight * aspectRatio;

            return (
              <div
                key={image.id}
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
                  src={image.smallUrl ?? image.url}
                  alt={`image ${image.id}`}
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
                <p className={styles.statNum}>{numAuthors.toLocaleString()}</p>
                <p className={styles.statTitle}>photographers</p>
              </div>
            </div>
          </div>
        </div>

        <div className={styles.sectionTwoBg}>
          <div className={styles.colorSection}>
            {renderColorRow(colorData.red, "right", 0)}
            {renderColorRow(colorData.navy, "left", 0)}
            {renderColorRow(colorData.olive, "right", 0)}
            {renderMobileColorRows()}
            <div className={styles.colorTextOverlay}>
              <div className={styles.title}>Color Intelligence</div>
              <p className={styles.subtitle}>
                Dominant colors are extracted from every photo, allowing you to
                discover images by their visual palette. Search and analyze
                images by their distinct colors.
              </p>
              <Link href="/?color=red" className={styles.link}>
                explore colors
              </Link>
            </div>
          </div>
        </div>

        <div className={styles.sectionSimilarityBg} ref={similarityRef}>
          <div className={styles.similaritySection}>
            {renderSimilarityClusters()}
            {renderMobileSimilarity()}
            <div className={styles.similarityTextOverlay}>
              <div className={styles.title}>Vector Similarity</div>
              <p className={styles.subtitle}>
                Every image is encoded with vector embeddings, enabling
                intelligent visual similarity search. Discover photos that share
                composition and visual patterns.
              </p>
              <Link href="/" className={styles.link}>
                find similar
              </Link>
            </div>
          </div>
        </div>

        <div className={styles.sectionThreeBg}>
          <div className={styles.sectionThree}>
            <div>
              <div
                className={`${styles.apiDemoContainer} ${styles.desktopOnly}`}
              >
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
              <div className={styles.mobileOnly}>
                <div className={styles.mobileApiDemo}>
                  <CodeHighlight
                    code={`${apiQuery}\n${apiResponseShort}`}
                    language="javascript"
                    styles={{
                      code: {
                        fontSize: "0.75rem",
                        maxHeight: "320px",
                      },
                    }}
                  />
                </div>
              </div>
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
            <div className={`${styles.imageThree} ${styles.desktopOnly}`}>
              <Image
                src={"/github_logo.png"}
                alt={`example AnalogDB API call`}
                width="384"
                height="216"
                quality={100}
              />
            </div>
          </div>
        </div>
      </div>
      <Footer />
    </main>
  );
}
