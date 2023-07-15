import styles from "./about.module.css";
import Image from "next/legacy/image";
import Link from "next/link";
import Footer from "./footer";
import { useBreakpoint } from "../providers/breakpoint.js";
import { Prism } from "@mantine/prism";
import { IconPolaroid, IconUsers } from "@tabler/icons";

export default function About(props) {
  const breakpoints = useBreakpoint();
  let isMobile = false;
  if (breakpoints["sm"]) {
    isMobile = true;
  }

  let numPosts = props.data.numPosts
  let numAuthors = props.data.numAuthors

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
            <Link href="/" legacyBehavior>
              <a className={styles.link}>view latest</a>
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
                  <p className={styles.statNum}>{numPosts}</p>
                  <p className={styles.statTitle}>posts</p>
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
                  <p className={styles.statNum}>{numAuthors}</p>
                  <p className={styles.statTitle}>photographers</p>
                </div>
              </div>
            </div>
          )}
        </div>

        <div className={styles.sectionTwoBg}>
          <div className={styles.sectionTwo}>
            <div>
              {!isMobile && (
                <div className={styles.apiDemoContainer}>
                  <div className={styles.apiDemo}>
                    <Prism
                      language="javascript"
                      styles={() => ({
                        code: {
                          fontSize: "0.75rem",
                          maxWidth: "40vw",
                        },
                      })}
                    >
                      {apiQuery}
                    </Prism>
                  </div>
                  <div className={styles.apiDemo}>
                    <Prism
                      withLineNumbers
                      language="javascript"
                      styles={() => ({
                        code: {
                          fontSize: "0.75rem",
                          maxHeight: "70vh",
                          maxWidth: "40vw",
                        },
                      })}
                    >
                      {apiResponse}
                    </Prism>
                  </div>
                </div>
              )}
            </div>
            <div>
              <div className={styles.title}>Accesible API</div>
              <p className={styles.subtitle}>
                The entire collection of film is exposed through a simple and
                modern JSON API. Embeddeding beautiful film photos in your
                projects has never been easier.
              </p>
              <Link href="/docs" legacyBehavior>
                <a className={styles.link}>read the docs</a>
              </Link>
            </div>
          </div>
        </div>

        <div className={styles.sectionThreeBg}>
          <div className={styles.sectionThree}>
            <div>
              <div className={styles.title}>Open-source</div>
              <p className={styles.subtitle}>
                All code made publically avaliable on Github with flexible
                licensing. Analogdb is an open community where all contributions
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
                  width="3840"
                  height="2160"
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
