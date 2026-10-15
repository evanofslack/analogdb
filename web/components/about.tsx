"use client";

import {
  AboutData,
  ColorData,
  AboutPhoto as Photo,
  pickSimilarRows,
  RainbowGroup,
  shuffle,
  SimilarityData,
} from "@lib/about";
import {
  IconBrandGithub,
  IconCamera,
  IconMovie,
  IconPolaroid,
  IconUsers,
} from "@tabler/icons-react";
import Link from "next/link";
import React, { useEffect, useState } from "react";
import styles from "./about.module.css";
import AboutApi from "./aboutApi";
import AboutFilms from "./aboutFilms";
import AboutPhoto, { AboutPhotoProvider } from "./aboutPhoto";
import AboutRepo from "./aboutRepo";
import AboutSearch from "./aboutSearch";
import AboutSimilar, { SIMILAR_MATCHES, SIMILAR_ROWS } from "./aboutSimilar";
import Footer from "./footer";

interface AboutProps {
  data: AboutData;
}

const COLOR_ROW_SIZE = 16;
const RAINBOW_SECONDS = 4;
const NO_HUE_SORT = ["black", "white"];

// pick each color's photos, then order them by hue so the belt reads as a gradient
function buildRainbow(groups: RainbowGroup[]): Photo[] {
  const shift = (hue: number) => (hue + 30) % 360;
  return groups.flatMap(({ color, count, photos }) => {
    const picks = shuffle(photos).slice(0, count);
    if (!NO_HUE_SORT.includes(color)) {
      picks.sort((a, b) => shift(a.hue) - shift(b.hue));
    }
    return picks.map((item) => item.photo);
  });
}

function pickColorRows(colorData: ColorData, random: boolean): ColorData {
  const pick = (images: Photo[]) =>
    (random ? shuffle(images) : images).slice(0, COLOR_ROW_SIZE);
  return {
    teal: pick(colorData.teal),
    red: pick(colorData.red),
    navy: pick(colorData.navy),
  };
}

export default function About(props: AboutProps) {
  const { numPosts, numAuthors, numCameras, numFilms, apiDemos } = props.data;

  const [colorData, setColorData] = useState<ColorData>(() =>
    pickColorRows(props.data.colorData, false)
  );
  const [similarRows, setSimilarRows] = useState<SimilarityData[] | null>(null);

  const [shuffled, setShuffled] = useState(false);
  const [rainbow, setRainbow] = useState<Photo[]>([]);

  // images render only after the shuffle, so the unshuffled server picks never download
  useEffect(() => {
    setColorData(pickColorRows(props.data.colorData, true));
    setRainbow(buildRainbow(props.data.rainbow));
    setSimilarRows(
      pickSimilarRows(
        props.data.allSimilarityData,
        SIMILAR_ROWS,
        SIMILAR_MATCHES
      )
    );
    setShuffled(true);
  }, [props.data]);

  const renderMobileColorRows = (): React.ReactElement | null => {
    if (!shuffled || !rainbow.length) return null;

    return (
      <div className={`${styles.mobileOnly} ${styles.visual}`}>
        <div className={styles.mobileColorRows}>
          <div className={styles.mobileColorRow}>
            <div
              className={`${styles.colorScrollContainer} ${styles.scrollLeft} ${styles.mobileScroll}`}
              style={{
                animationDuration: `${rainbow.length * RAINBOW_SECONDS}s`,
              }}
            >
              {[...rainbow, ...rainbow].map((image, index) => (
                <AboutPhoto
                  key={`${image.id}-${index}`}
                  photo={image}
                  small
                  className={styles.colorImageContainer}
                  imageClassName={styles.mobileColorImage}
                />
              ))}
            </div>
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
                <h1 className={styles.title}>Film for All</h1>
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
              {renderColorRow(colorData.teal, "right", 0)}
              {renderColorRow(colorData.red, "left", 0)}
              {renderColorRow(colorData.navy, "right", 0)}
              {renderMobileColorRows()}
              <div className={styles.colorTextOverlay}>
                <h2 className={styles.title}>Color Intelligence</h2>
                <p className={styles.subtitle}>
                  Dominant colors are extracted from every photo, allowing you
                  to discover images by their visual palette. Search and analyze
                  images by their distinct colors.
                </p>
                <Link href="/?color=red&sort=color" className={styles.link}>
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
                <div className={`${styles.visual} ${styles.filmVisual}`}>
                  <AboutFilms films={props.data.films} />
                </div>
              </div>
            </div>
          )}

          {props.data.searches.length > 0 && (
            <div className={styles.band}>
              <div className={styles.split}>
                <div className={`${styles.visual} ${styles.searchVisual}`}>
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

          {props.data.allSimilarityData.length > 0 && (
            <div className={styles.band}>
              <div className={`${styles.split} ${styles.wide}`}>
                <div className={styles.text}>
                  <h2 className={styles.title}>Similar Photos</h2>
                  <p className={styles.subtitle}>
                    Every photo is mapped by CLIP, so photos with the same
                    light, color and mood turn up together.
                  </p>
                  <Link
                    href={
                      similarRows?.length
                        ? `/search?similar=${similarRows[0].centerPost.id}`
                        : "/search"
                    }
                    prefetch={false}
                    className={styles.link}
                  >
                    see similar photos
                  </Link>
                </div>
                <div className={`${styles.visual} ${styles.similarVisual}`}>
                  <AboutSimilar rows={similarRows} />
                </div>
              </div>
            </div>
          )}

          <div className={styles.band}>
            <div className={`${styles.split} ${styles.apiSplit}`}>
              {apiDemos.length > 0 && (
                <div className={`${styles.visual} ${styles.apiVisual}`}>
                  <AboutApi groups={apiDemos} />
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
                <h2 className={styles.title}>Open Source</h2>
                <p className={styles.subtitle}>
                  Every part of AnalogDB is on GitHub: the Go API, this site,
                  and the pipeline that scrapes and analyzes each photo. Issues
                  and pull requests welcome.
                </p>
                <a
                  className={styles.link}
                  href="https://github.com/evanofslack/analogdb"
                >
                  view source
                </a>
              </div>
              {props.data.repo ? (
                <div className={`${styles.visual} ${styles.repoVisual}`}>
                  <AboutRepo repo={props.data.repo} />
                </div>
              ) : (
                <div className={styles.desktopOnly}>
                  <IconBrandGithub
                    size={160}
                    stroke={1}
                    className={styles.githubIcon}
                    aria-hidden
                  />
                </div>
              )}
            </div>
          </div>
        </div>
        <Footer />
      </main>
    </AboutPhotoProvider>
  );
}
