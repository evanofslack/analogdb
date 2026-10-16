"use client";

import { lowImage, statsParsers, StatsPost } from "@lib/stats";
import Link from "next/link";
import { useQueryStates } from "nuqs";
import ResponsiveImage from "../responsiveImage";
import styles from "./statsPhoto.module.css";

interface StatsPhotoProps {
  post: StatsPost;
  sizes: string;
  className?: string;
}

// StatsPhoto links a post's low res image to its page. NSFW photos stay
// blurred until the page switch is on, and clicking a blurred one turns it on.
export default function StatsPhoto({
  post,
  sizes,
  className,
}: StatsPhotoProps) {
  const [{ nsfw }, setParams] = useQueryStates(statsParsers);
  const image = lowImage(post);
  const blurred = Boolean(post.nsfw) && nsfw !== "include";

  return (
    <Link
      href={`/post/${post.id}`}
      prefetch={false}
      className={`${styles.photo} ${className ?? ""}`}
      onClick={(event) => {
        if (blurred) {
          event.preventDefault();
          setParams({ nsfw: "include" });
        }
      }}
      aria-label={blurred ? "NSFW photo, click to show" : post.title}
    >
      {image?.url && (
        <ResponsiveImage
          src={image.url}
          alt={blurred ? "NSFW photo" : post.title ?? ""}
          fill
          sizes={sizes}
          loading="lazy"
          className={blurred ? styles.blurred : undefined}
          style={{ objectFit: "cover" }}
        />
      )}
      {blurred && <span className={styles.label}>NSFW</span>}
    </Link>
  );
}
