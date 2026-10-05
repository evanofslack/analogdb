"use client";

import { baseURL } from "@lib/constants";
import { postAlt } from "@lib/seo";
import { Tooltip } from "@mantine/core";
import { useClipboard } from "@mantine/hooks";
import {
  IconApi,
  IconBrandReddit,
  IconCalendarWeek,
  IconCamera,
  IconMovie,
  IconUser,
} from "@tabler/icons-react";
import Image from "next/image";
import Link from "next/link";
import styles from "./imageTag.module.css";
import Keywords from "./keywords";

export default function ImageTag(props) {
  const clipboard = useClipboard({ timeout: 1000 });

  let post = props.post;
  let similarPosts = props.similar.posts;

  const api_endpoint = baseURL + "/post/";
  const redditUserURL = "https://www.reddit.com/user/";
  const author = post.author.replace("u/", "");

  const uploaded = new Date(post.timestamp * 1000);
  const date = uploaded.toLocaleDateString("en-US", {
    timeZone: "UTC",
  });

  const cameraInfo =
    post.camera_make && post.camera_model
      ? `${post.camera_make}, ${post.camera_model}`
      : null;

  const filmInfo =
    post.film_make && post.film_type
      ? `${post.film_make}, ${post.film_type}`
      : null;

  const filmURL =
    props.filmHref ??
    `/?film_make=${encodeURIComponent(
      post.film_make
    )}&film_type=${encodeURIComponent(post.film_type)}`;

  const cameraURL =
    props.cameraHref ??
    `/?camera_make=${encodeURIComponent(
      post.camera_make
    )}&camera_model=${encodeURIComponent(post.camera_model)}`;

  let hexColors = new Array();
  post.colors.forEach(function (color) {
    hexColors.push(color.hex);
  });

  const color = (hex) => {
    return {
      backgroundColor: hex,
    };
  };

  return (
    <div className={styles.container}>
      <div className={styles.containerMetadata}>
        <h1 className={styles.title}>{post.title}</h1>
        <div className={styles.containerSub}>
          <div className={styles.containerAuthor}>
            <div className={styles.infoItemCal}>
              <Tooltip label={"uploaded"} position="bottom" color="gray">
                <IconCalendarWeek size={16} className={styles.icon} />
              </Tooltip>
              <time dateTime={uploaded.toISOString()}>{date}</time>
            </div>
            <a href={redditUserURL + author} className={styles.author}>
              <Tooltip label={"author"} position="bottom" color="gray">
                <IconUser size={16} className={styles.icon} />
              </Tooltip>
              {author}
            </a>
            <a href={post.permalink} className={styles.author}>
              <Tooltip label={"view on reddit"} position="bottom" color="gray">
                <IconBrandReddit size={16} className={styles.icon} />
              </Tooltip>
              reddit
            </a>
            {cameraInfo && (
              <Link
                href={cameraURL}
                prefetch={false}
                className={styles.infoItem}
              >
                <Tooltip label={"camera"} position="bottom" color="gray">
                  <IconCamera size={16} className={styles.icon} />
                </Tooltip>
                {cameraInfo}
              </Link>
            )}
            {filmInfo && (
              <Link href={filmURL} prefetch={false} className={styles.infoItem}>
                <Tooltip label={"film"} position="bottom" color="gray">
                  <IconMovie size={16} className={styles.icon} />
                </Tooltip>
                {filmInfo}
              </Link>
            )}
            <a href={api_endpoint + post.id} className={styles.id}>
              <Tooltip label={"api response"} position="bottom" color="gray">
                <IconApi size={16} className={styles.icon} />
              </Tooltip>
              #{post.id}
            </a>
          </div>
          <div className={styles.containerColorsAndKeywords}>
            <div className={styles.containerColors}>
              {hexColors.map((hex) => {
                return (
                  <Tooltip
                    key={hex}
                    label={clipboard.copied ? "copied" : hex}
                    position="top"
                    color="gray"
                  >
                    <div
                      style={color(hex)}
                      className={styles.colorSquare}
                      onClick={() => clipboard.copy(hex)}
                    ></div>
                  </Tooltip>
                );
              })}
            </div>
            <Keywords
              keywords={Object.hasOwn(post, "keywords") ? post.keywords : []}
              maxKeywords={15}
            />
          </div>
        </div>
      </div>
      {similarPosts && (
        <div className={styles.similar}>
          <div className={styles.similarHeader}>
            <h2 className={styles.similarTitle}>discover similar</h2>
            <Link
              href={`/search?similar=${post.id}`}
              prefetch={false}
              className={styles.findSimilar}
            >
              find similar
            </Link>
          </div>
          <div className={styles.similarContainer}>
            {similarPosts.map((post) => {
              return (
                <div key={post.id} className={styles.similarImage}>
                  <Link
                    href={`/post/${post.id}`}
                    passHref={true}
                    key={post.id}
                    prefetch={false}
                  >
                    <Image
                      key={post.id}
                      style={{ objectFit: "cover" }}
                      src={post.images[1].url}
                      alt={postAlt(post)}
                      sizes="(max-width: 720px) 50vw, 200px"
                      fill
                      quality={100}
                    />
                  </Link>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
