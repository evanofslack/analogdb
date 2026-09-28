"use client";

import Footer from "@components/footer";
import ImageTag from "@components/imageTag";
import useIsAdmin from "@hooks/useIsAdmin";
import { ActionIcon, Tooltip } from "@mantine/core";
import Image from "next/image";
import Link from "next/link";
import {
  AiOutlineArrowsAlt,
  AiOutlineDelete,
  AiOutlineDownload,
} from "react-icons/ai";
import styles from "./imagePage.module.css";

async function handleDelete(postId) {
  if (!confirm("Are you sure you want to delete this post?")) return;

  const route = `/api/post/${postId}`;
  try {
    const response = await fetch(route, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
    });

    if (response.ok) {
      window.location.href = "/";
    } else {
      alert("Failed to delete post");
    }
  } catch (error) {
    console.error("Error deleting post:", error);
    alert("Error deleting post");
  }
}

function colorPlaceholder(post, image) {
  const hex = post.colors?.[0]?.hex;
  if (!hex) return "empty";
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${image.width} ${image.height}"><rect width="100%" height="100%" fill="${hex}"/></svg>`;
  return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

async function downloadImage(id) {
  try {
    const link = document.createElement("a");
    link.href = `/api/download/${id}`;
    link.download = "";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  } catch (error) {
    console.error("Download failed:", error);
  }
}

export default function ImagePage(props) {
  let post = props.post;
  let similar = props.similar;
  const isAdmin = useIsAdmin();
  let image = post.images[2];

  return (
    <div>
      <div className={styles.fullscreen}>
        <div className={styles.headerIcons}>
          <Link href={`/`} passHref={true}>
            <h1 className={styles.title}>AnalogDB</h1>
          </Link>
        </div>
        <div className={styles.imageContainer}>
          <Image
            priority
            style={{ objectFit: "contain" }}
            fill
            src={image.url}
            alt={`image ${post.id} by ${post.author}`}
            sizes="100vw"
            quality={100}
            placeholder={colorPlaceholder(post, image)}
          />
        </div>
        <div className={styles.footerIcons}>
          <ActionIcon.Group>
            <Tooltip label="download" withArrow className="px-2">
              <ActionIcon
                variant="subtle"
                color="gray"
                onClick={() => downloadImage(post.id)}
              >
                <AiOutlineDownload size="24px" />
              </ActionIcon>
            </Tooltip>
            <Tooltip label="fullscreen" withArrow className="px-2">
              <ActionIcon
                variant="subtle"
                color="gray"
                component="a"
                href={post.images[3].url}
              >
                <AiOutlineArrowsAlt size="24px"></AiOutlineArrowsAlt>
              </ActionIcon>
            </Tooltip>
            {/* Show delete button only to admin */}
            {isAdmin && (
              <Tooltip label="delete" withArrow className="px-2">
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  onClick={() => handleDelete(post.id)}
                >
                  <AiOutlineDelete size="24px"></AiOutlineDelete>
                </ActionIcon>
              </Tooltip>
            )}
          </ActionIcon.Group>
        </div>
      </div>
      <ImageTag post={post} similar={similar} />

      <Footer />
    </div>
  );
}
