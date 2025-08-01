"use client";

import { postsApi } from "@lib/client";
import { CodeHighlight } from "@mantine/code-highlight";
import { useBreakpoint } from "@providers/breakpoint";
import { IconPolaroid, IconUsers } from "@tabler/icons-react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useState } from "react";
import styles from "./about.module.css";
import Footer from "./footer";

export default function About(props) {
  const breakpoints = useBreakpoint();
  let isMobile = false;
  if (breakpoints["sm"]) {
    isMobile = true;
  }

  let numPosts = props.data.numPosts;
  let numAuthors = props.data.numAuthors;

  const [colorData, setColorData] = useState({
    red: [],
    blue: [],
    green: [],
  });
  const [loading, setLoading] = useState(true);

  const COLOR_MIN_VALUES = {
    red: 0.4,
    navy: 0.4,
    green: 0.4,
  };

  async function makeRequest(params) {
    try {
      const response = await postsApi.postsGet(params);
      return response;
    } catch (error) {
      console.error("API request failed:", error);
      throw error;
    }
  }

  useEffect(() => {
    const fetchColorData = async () => {
      try {
        const colors = ["red", "navy", "green"];
        const promises = colors.map((color) => {
          const params = {
            color: [color],
            minColor: [COLOR_MIN_VALUES[color]],
            pageSize: 30,
            nsfw: false,
            sort: "random",
          };
          return makeRequest(params);
        });

        const results = await Promise.all(promises);
        setColorData({
          red: results[0].posts || [],
          blue: results[1].posts || [],
          green: results[2].posts || [],
        });
      } catch (error) {
        console.error("Failed to fetch color data:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchColorData();
  }, []);

  const apiQuery = "curl https://api.analogdb.com/posts";

  const apiResponse = `
"meta":{
  "total_posts":3637,
  "page_size":20,
  "next_page_id":1672251647,
  "next_page_url":"/posts?sort=latest&page_size=20&page_id=1672251647"
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

  const renderColorRow = (images, direction) => {
    if (!images.length) return null;

    // Duplicate images for seamless loop
    const duplicatedImages = [...images, ...images];

    return (
      <div className={styles.colorRow}>
        <div
          className={`${styles.colorScrollContainer} ${
            direction === "left" ? styles.scrollLeft : styles.scrollRight
          }`}
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
                {renderColorRow(colorData.red, "right")}
                {renderColorRow(colorData.blue, "left")}
                {renderColorRow(colorData.green, "right")}
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
