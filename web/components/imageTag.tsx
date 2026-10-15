"use client";

import { setSearchSource, track } from "@lib/analytics";
import { bareHex, similarPalette } from "@lib/color";
import { publicURL } from "@lib/constants";
import { smallImage } from "@lib/images";
import { postAlt } from "@lib/seo";
import { Tooltip } from "@mantine/core";
import {
  IconApi,
  IconBrandReddit,
  IconCalendarWeek,
  IconCamera,
  IconMovie,
  IconUser,
} from "@tabler/icons-react";
import { AnalogdbPost, ServerSimilarPostsResponse } from "analogdb-generated";
import Image from "next/image";
import Link from "next/link";
import styles from "./imageTag.module.css";
import Keywords from "./keywords";

interface ImageTagProps {
  post: AnalogdbPost;
  similar: ServerSimilarPostsResponse;
  filmHref?: string | null;
  cameraHref?: string | null;
}

export default function ImageTag(props: ImageTagProps) {
  let post = props.post;
  let similarPosts = props.similar.posts;

  const api_endpoint = publicURL + "/post/";
  const redditUserURL = "https://www.reddit.com/user/";
  const author = post.author.replace("u/", "");

  const uploaded = new Date(post.timestamp * 1000);
  const date = uploaded.toLocaleDateString("en-US", {
    timeZone: "UTC",
  });

  const cameraInfo =
    post.cameraMake && post.cameraModel
      ? `${post.cameraMake}, ${post.cameraModel}`
      : null;

  const filmInfo =
    post.filmMake && post.filmType
      ? `${post.filmMake}, ${post.filmType}`
      : null;

  const filmURL =
    props.filmHref ??
    `/?film_make=${encodeURIComponent(
      post.filmMake
    )}&film_type=${encodeURIComponent(post.filmType)}`;

  const cameraURL =
    props.cameraHref ??
    `/?camera_make=${encodeURIComponent(
      post.cameraMake
    )}&camera_model=${encodeURIComponent(post.cameraModel)}`;

  const palette = (post.colors ?? [])
    .filter((color) => color.hex)
    .map((color) => ({
      hex: bareHex(color.hex),
      percent: Math.max(0, Math.min(1, color.percent ?? 0)),
    }))
    .sort((a, b) => b.percent - a.percent);
  const paletteTotal = palette.reduce((sum, { percent }) => sum + percent, 0);
  const paletteHexes = similarPalette(post.colors);

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
            <a
              href={redditUserURL + author}
              className={styles.author}
              onClick={() =>
                track("outbound_click", {
                  post_id: post.id,
                  target: "reddit_user",
                })
              }
            >
              <Tooltip label={"author"} position="bottom" color="gray">
                <IconUser size={16} className={styles.icon} />
              </Tooltip>
              {author}
            </a>
            <a
              href={post.permalink}
              className={styles.author}
              onClick={() =>
                track("outbound_click", { post_id: post.id, target: "reddit" })
              }
            >
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
            {palette.length > 0 && (
              <div className={styles.containerColors}>
                <div className={styles.strip}>
                  {palette.map(({ hex, percent }, index) => (
                    <Tooltip
                      key={index}
                      label={`#${hex} · ${Math.round(percent * 100)}%`}
                      position="top"
                      color="gray"
                    >
                      <Link
                        href={`/?color=${hex}&sort=color`}
                        prefetch={false}
                        className={styles.segment}
                        style={{
                          flexGrow:
                            paletteTotal > 0 ? percent / paletteTotal : 1,
                          backgroundColor: `#${hex}`,
                        }}
                        aria-label={`Photos with #${hex}`}
                      />
                    </Tooltip>
                  ))}
                </div>
                {paletteHexes.length > 0 && (
                  <Link
                    href={`/?color=${paletteHexes.join(",")}&sort=color`}
                    prefetch={false}
                    className={styles.similarPalette}
                  >
                    similar palette
                  </Link>
                )}
              </div>
            )}
            <Keywords
              keywords={
                Object.prototype.hasOwnProperty.call(post, "keywords")
                  ? post.keywords
                  : []
              }
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
              onClick={() => setSearchSource("post")}
              className={styles.findSimilar}
            >
              see more...
            </Link>
          </div>
          <div className={styles.similarContainer}>
            {similarPosts.map((post, position) => {
              return (
                <div key={post.id} className={styles.similarImage}>
                  <Link
                    href={`/post/${post.id}`}
                    passHref={true}
                    key={post.id}
                    prefetch={false}
                    onClick={() =>
                      track("similar_click", {
                        from_post_id: props.post.id,
                        post_id: post.id,
                        position,
                      })
                    }
                  >
                    <Image
                      key={post.id}
                      style={{ objectFit: "cover" }}
                      src={smallImage(post.images)?.url}
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
