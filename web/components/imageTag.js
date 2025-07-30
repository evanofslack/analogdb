"use client";

import { baseURL } from "@lib/constants";
import { Tooltip } from "@mantine/core";
import { useClipboard } from "@mantine/hooks";
import {
  IconApi,
  IconCalendarWeek,
  IconCamera,
  IconMovie,
  IconUser,
} from "@tabler/icons-react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import styles from "./imageTag.module.css";
import Keywords from "./keywords";

export default function ImageTag(props) {
  const router = useRouter();
  const clipboard = useClipboard({ timeout: 1000 });

  let post = props.post;
  let similarPosts = props.similar.posts;

  const api_endpoint = baseURL + "/post/";
  const redditUserURL = "https://www.reddit.com/user/";
  const author = post.author.replace("u/", "");

  const date = new Date(post.timestamp * 1000).toLocaleDateString("en-US");

  const cameraInfo =
    post.camera_make && post.camera_model
      ? `${post.camera_make}, ${post.camera_model}`
      : null;

  const filmInfo =
    post.film_make && post.film_type
      ? `${post.film_make}, ${post.film_type}`
      : null;

  const handleFilmClick = () => {
    router.push(
      `/?film_make=${encodeURIComponent(
        post.film_make
      )}&film_type=${encodeURIComponent(post.film_type)}`
    );
  };

  const handleCameraClick = () => {
    router.push(
      `/?camera_make=${encodeURIComponent(
        post.camera_make
      )}&camera_model=${encodeURIComponent(post.camera_model)}`
    );
  };

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
        <a href={post.permalink} className={styles.title}>
          {post.title}
        </a>
        <div className={styles.containerSub}>
          <div className={styles.containerAuthor}>
            <div className={styles.infoItemCal}>
              <Tooltip label={"uploaded"} position="bottom" color="gray">
                <IconCalendarWeek size={16} className={styles.icon} />
              </Tooltip>
              {date}
            </div>
            <a href={redditUserURL + author} className={styles.author}>
              <Tooltip label={"author"} position="bottom" color="gray">
                <IconUser size={16} className={styles.icon} />
              </Tooltip>
              {author}
            </a>
            {cameraInfo && (
              <div
                className={styles.infoItem}
                onClick={handleCameraClick}
                style={{ cursor: "pointer" }}
              >
                <Tooltip label={"camera"} position="bottom" color="gray">
                  <IconCamera size={16} className={styles.icon} />
                </Tooltip>
                {cameraInfo}
              </div>
            )}
            {filmInfo && (
              <div
                className={styles.infoItem}
                onClick={handleFilmClick}
                style={{ cursor: "pointer" }}
              >
                <Tooltip label={"film"} position="bottom" color="gray">
                  <IconMovie size={16} className={styles.icon} />
                </Tooltip>
                {filmInfo}
              </div>
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
                      key={hex.id}
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
          <h2 className={styles.similarTitle}>discover similar</h2>
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
                      priority
                      style={{ objectFit: "cover" }}
                      src={post.images[1].url}
                      alt={`image ${post.id} by ${post.author}`}
                      sizes="100vw"
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
