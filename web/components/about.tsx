"use client";

import {
  AboutData,
  ColorData,
  AboutPhoto as Photo,
  shuffle,
  SimilarityData,
} from "@lib/about";
import { CodeHighlight } from "@mantine/code-highlight";
import {
  IconBrandGithub,
  IconCamera,
  IconMovie,
  IconPolaroid,
  IconUsers,
} from "@tabler/icons-react";
import Link from "next/link";
import React, { useEffect, useRef, useState } from "react";
import styles from "./about.module.css";
import AboutFilms from "./aboutFilms";
import AboutPhoto, { AboutPhotoProvider } from "./aboutPhoto";
import AboutSearch from "./aboutSearch";
import Footer from "./footer";

interface AboutProps {
  data: AboutData;
}

// how far the similar photos reach past the center photo
const CLUSTER_REACH = { left: 170, right: 195, top: 160, bottom: 165 };
const CLUSTER_CENTER_HEIGHT = 420;
const CLUSTER_SIMILAR_HEIGHT = 180;

const COLOR_ROW_SIZE = 16;
const MOBILE_ROW_SIZE = 8;

function pickColorRows(colorData: ColorData, random: boolean): ColorData {
  const pick = (images: Photo[]) =>
    (random ? shuffle(images) : images).slice(0, COLOR_ROW_SIZE);
  return {
    red: pick(colorData.red),
    navy: pick(colorData.navy),
    olive: pick(colorData.olive),
  };
}

export default function About(props: AboutProps) {
  const { numPosts, numAuthors, numCameras, numFilms, apiSample } = props.data;

  const [colorData, setColorData] = useState<ColorData>(() =>
    pickColorRows(props.data.colorData, false)
  );
  const [allSimilarityData, setAllSimilarityData] = useState<SimilarityData[]>(
    props.data.allSimilarityData
  );

  const [shuffled, setShuffled] = useState(false);

  // images render only after the shuffle, so the unshuffled server picks never download
  useEffect(() => {
    setColorData(pickColorRows(props.data.colorData, true));
    setAllSimilarityData(shuffle(props.data.allSimilarityData));
    setShuffled(true);
  }, [props.data]);

  const [currentSimilarityIndex, setCurrentSimilarityIndex] =
    useState<number>(0);
  const [isTransitioning, setIsTransitioning] = useState<boolean>(false);
  const similarityRef = useRef<HTMLDivElement | null>(null);
  const [isSimilarityVisible, setIsSimilarityVisible] = useState(false);
  const [isSimilarityHovered, setIsSimilarityHovered] = useState(false);
  const clusterAreaRef = useRef<HTMLDivElement | null>(null);
  const [clusterArea, setClusterArea] = useState<{
    width: number;
    height: number;
  } | null>(null);

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

  // the cluster scales down to fit the space beside the text
  useEffect(() => {
    const node = clusterAreaRef.current;
    if (!node) return;
    const observer = new ResizeObserver(([entry]) =>
      setClusterArea({
        width: entry.contentRect.width,
        height: entry.contentRect.height,
      })
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  // a photo can leave while the pointer is on it, so a new cluster starts unpaused
  useEffect(() => {
    setIsSimilarityHovered(false);
  }, [currentSimilarityIndex]);

  useEffect(() => {
    if (
      allSimilarityData.length <= 1 ||
      !isSimilarityVisible ||
      isSimilarityHovered
    )
      return;
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
    }, 5000);

    return () => clearInterval(interval);
  }, [allSimilarityData, isSimilarityVisible, isSimilarityHovered]);

  const renderMobileColorRows = (): React.ReactElement | null => {
    const rows: [Photo[], "left" | "right"][] = [
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
                    <AboutPhoto
                      key={`${image.id}-${index}`}
                      photo={image}
                      small
                      className={styles.colorImageContainer}
                      imageClassName={styles.mobileColorImage}
                    />
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
          <AboutPhoto
            photo={centerPost}
            small
            fill
            sizes="100vw"
            className={`${styles.mobileSimilarCenter} ${fade}`}
            style={{
              aspectRatio: `${centerPost.width || 4} / ${
                centerPost.height || 3
              }`,
            }}
          />
          <div className={styles.mobileSimilarGrid}>
            {similarPosts.slice(0, 6).map((image) => (
              <AboutPhoto
                key={image.id}
                photo={image}
                small
                fill
                sizes="33vw"
                className={`${styles.mobileSimilarTile} ${fade}`}
              />
            ))}
          </div>
        </div>
      </div>
    );
  };

  const renderColorRow = (
    images: Photo[],
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
          {duplicatedImages.map((image, index) => (
            <AboutPhoto
              key={`${image.id}-${index}`}
              photo={image}
              small
              className={styles.colorImageContainer}
              imageClassName={styles.colorImage}
            />
          ))}
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

    const centerAspectRatio =
      (centerImage.width || 400) / (centerImage.height || 400);
    const centerHeight = CLUSTER_CENTER_HEIGHT;
    const centerWidth = centerHeight * centerAspectRatio;

    const stageWidth = centerWidth + CLUSTER_REACH.left + CLUSTER_REACH.right;
    const stageHeight = centerHeight + CLUSTER_REACH.top + CLUSTER_REACH.bottom;
    const scale = clusterArea
      ? Math.min(
          1,
          clusterArea.width / stageWidth,
          clusterArea.height / stageHeight
        )
      : 1;
    const fade = isTransitioning ? styles.transitioning : "";

    return (
      <div
        className={styles.clusterStage}
        style={{
          width: `${stageWidth}px`,
          height: `${stageHeight}px`,
          transform: `translate(-50%, -50%) scale(${scale})`,
        }}
      >
        <div
          key={centerImage.id}
          className={styles.clusterContainer}
          style={{
            left: `${CLUSTER_REACH.left}px`,
            top: `${CLUSTER_REACH.top}px`,
            width: `${centerWidth}px`,
            height: `${centerHeight}px`,
          }}
        >
          <AboutPhoto
            photo={centerImage}
            fill
            sizes="420px"
            className={`${styles.clusterCenterContainer} ${fade}`}
            style={{ width: "100%", height: "100%" }}
            onHover={setIsSimilarityHovered}
          />

          {similarPosts.slice(0, 6).map((image, index) => {
            if (!similarPositions[index]) return null;

            const aspectRatio = (image.width || 200) / (image.height || 200);

            return (
              <AboutPhoto
                key={image.id}
                photo={image}
                small
                fill
                sizes="180px"
                className={`${styles.clusterSimilarContainer} ${fade}`}
                style={{
                  ...similarPositions[index],
                  width: `${CLUSTER_SIMILAR_HEIGHT * aspectRatio}px`,
                  height: `${CLUSTER_SIMILAR_HEIGHT}px`,
                }}
                onHover={setIsSimilarityHovered}
              />
            );
          })}
        </div>
      </div>
    );
  };

  const stats = [
    { icon: IconPolaroid, count: numPosts, label: "photos" },
    { icon: IconUsers, count: numAuthors, label: "photographers" },
    { icon: IconCamera, count: numCameras, label: "cameras" },
    { icon: IconMovie, count: numFilms, label: "films" },
  ];

  return (
    <AboutPhotoProvider>
      <main>
        <div className={styles.container}>
          <div className={styles.band}>
            <div className={`${styles.split} ${styles.hero}`}>
              <div>
                <h1 className={styles.title}>Film for all</h1>
                <p className={styles.subtitle}>
                  AnalogDB is a curated database of over{" "}
                  {(Math.floor(numPosts / 1000) * 1000).toLocaleString()} film
                  photos, each one analyzed for color, gear and content. New
                  photos are added every day.
                </p>
                <Link href="/" className={styles.link}>
                  view latest
                </Link>
              </div>
              <div className={styles.stats}>
                {stats.map(({ icon: Icon, count, label }) => (
                  <div key={label} className={styles.statRow}>
                    <Icon size={36} stroke={1.2} className={styles.statIcon} />
                    <div className={styles.statCol}>
                      <p className={styles.statNum}>{count.toLocaleString()}</p>
                      <p className={styles.statTitle}>{label}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className={styles.band}>
            <div className={styles.colorSection}>
              {renderColorRow(colorData.red, "right", 0)}
              {renderColorRow(colorData.navy, "left", 0)}
              {renderColorRow(colorData.olive, "right", 0)}
              {renderMobileColorRows()}
              <div className={styles.colorTextOverlay}>
                <h2 className={styles.title}>Color Intelligence</h2>
                <p className={styles.subtitle}>
                  Dominant colors are extracted from every photo, allowing you
                  to discover images by their visual palette. Search and analyze
                  images by their distinct colors.
                </p>
                <Link href="/?color=red" className={styles.link}>
                  explore colors
                </Link>
              </div>
            </div>
          </div>

          {props.data.films.length > 0 && (
            <div className={styles.band}>
              <div className={`${styles.split} ${styles.wide}`}>
                <div className={styles.text}>
                  <h2 className={styles.title}>Film Stocks</h2>
                  <p className={styles.subtitle}>
                    Camera, lens and film are read from every post, so you can
                    browse by film stock and see what each one looks like.
                  </p>
                  <Link href="/films" className={styles.link}>
                    browse film
                  </Link>
                </div>
                <div className={`${styles.visualFirst} ${styles.filmVisual}`}>
                  <AboutFilms films={props.data.films} />
                </div>
              </div>
            </div>
          )}

          {props.data.searches.length > 0 && (
            <div className={styles.band}>
              <div className={styles.split}>
                <div className={`${styles.visualFirst} ${styles.searchVisual}`}>
                  <AboutSearch searches={props.data.searches} />
                </div>
                <div>
                  <h2 className={styles.title}>Search by Phrase</h2>
                  <p className={styles.subtitle}>
                    A vision model writes a caption and tags for every photo, so
                    you can search by what is in it.
                  </p>
                  <Link href="/search" className={styles.link}>
                    try search
                  </Link>
                </div>
              </div>
            </div>
          )}

          <div className={styles.band} ref={similarityRef}>
            <div className={`${styles.split} ${styles.wide}`}>
              <div className={styles.text}>
                <h2 className={styles.title}>Vector Similarity</h2>
                <p className={styles.subtitle}>
                  Every photo is embedded with CLIP, one vector space shared by
                  text search, image search and similar photos. Find photos that
                  share a look.
                </p>
                <Link href="/search" className={styles.link}>
                  find similar
                </Link>
              </div>
              <div
                ref={clusterAreaRef}
                className={`${styles.clusterArea} ${styles.desktopOnly}`}
              >
                {renderSimilarityClusters()}
              </div>
              <div className={`${styles.visualFirst} ${styles.mobileOnly}`}>
                {renderMobileSimilarity()}
              </div>
            </div>
          </div>

          <div className={styles.band}>
            <div className={styles.split}>
              {apiSample && (
                <div className={styles.visualFirst}>
                  <div
                    className={`${styles.apiDemoContainer} ${styles.desktopOnly}`}
                  >
                    <div className={styles.apiDemo}>
                      <CodeHighlight
                        code={apiSample.query}
                        language="bash"
                        classNames={{ code: styles.apiCode }}
                      />
                    </div>
                    <div className={styles.apiDemo}>
                      <CodeHighlight
                        code={apiSample.full}
                        language="json"
                        classNames={{ code: styles.apiCodeTall }}
                      />
                    </div>
                  </div>
                  <div className={styles.mobileOnly}>
                    <div className={styles.mobileApiDemo}>
                      <CodeHighlight
                        code={`${apiSample.query}\n\n${apiSample.short}`}
                        language="json"
                        classNames={{ code: styles.mobileApiCode }}
                      />
                    </div>
                  </div>
                </div>
              )}
              <div>
                <h2 className={styles.title}>Accessible API</h2>
                <p className={styles.subtitle}>
                  Posts, search, similar photos, colors and gear, all through
                  one simple JSON API.
                </p>
                <Link href="/docs" className={styles.link}>
                  read the docs
                </Link>
              </div>
            </div>
          </div>

          <div className={styles.band}>
            <div className={styles.split}>
              <div>
                <h2 className={styles.title}>Open-source</h2>
                <p className={styles.subtitle}>
                  All code made publicly available on Github with flexible
                  licensing. AnalogDB is an open community where all
                  contributions are welcome!
                </p>
                <a
                  className={styles.link}
                  href="https://github.com/evanofslack/analogdb"
                >
                  view source
                </a>
              </div>
              <div className={styles.desktopOnly}>
                <IconBrandGithub
                  size={160}
                  stroke={1}
                  className={styles.githubIcon}
                  aria-hidden
                />
              </div>
            </div>
          </div>
        </div>
        <Footer />
      </main>
    </AboutPhotoProvider>
  );
}
