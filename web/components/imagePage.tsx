"use client";

import PostEditor from "@components/admin/postEditor";
import Footer from "@components/footer";
import Header from "@components/header";
import ImageTag from "@components/imageTag";
import ReportModal from "@components/reportModal";
import useIsAdmin from "@hooks/useIsAdmin";
import {
  currentSearchId,
  EventMap,
  previousRoute,
  track,
} from "@lib/analytics";
import { postAlt } from "@lib/seo";
import { ActionIcon, Tooltip } from "@mantine/core";
import {
  AnalogdbColor,
  AnalogdbImage,
  AnalogdbKeyword,
} from "analogdb-generated";
import { type ImageProps } from "next/image";
import { useRouter } from "next/navigation";
import { CSSProperties, useEffect, useState } from "react";
import { IconType } from "react-icons";
import {
  AiOutlineArrowsAlt,
  AiOutlineDelete,
  AiOutlineDownload,
  AiOutlineEdit,
  AiOutlineFlag,
} from "react-icons/ai";
import styles from "./imagePage.module.css";
import ResponsiveImage from "./responsiveImage";

export interface PagePost {
  id: number;
  title: string;
  author: string;
  permalink: string;
  timestamp: number;
  score?: number;
  nsfw: boolean;
  grayscale: boolean;
  sprocket: boolean;
  description?: string;
  caption?: string;
  images: AnalogdbImage[];
  colors: AnalogdbColor[];
  keywords?: AnalogdbKeyword[];
  camera_make?: string;
  camera_model?: string;
  film_make?: string;
  film_type?: string;
  film_speed?: number;
  focal_length?: number;
  aperture?: string;
}

export interface SimilarPosts {
  posts?: PagePost[];
}

interface ImagePageProps {
  post: PagePost;
  similar: SimilarPosts;
  filmHref: string | null;
  cameraHref: string | null;
}

interface ActionProps {
  label: string;
  icon: IconType;
  onClick?: () => void;
  href?: string;
}

async function handleDelete(postId: number) {
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

function colorPlaceholder(
  post: PagePost,
  image: AnalogdbImage
): ImageProps["placeholder"] {
  const hex = post.colors?.[0]?.hex;
  if (!hex) return "empty";
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${image.width} ${image.height}"><rect width="100%" height="100%" fill="${hex}"/></svg>`;
  return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

type PostFrom = EventMap["post_view"]["from"];

function postFrom(searchId: string): PostFrom {
  const route = previousRoute();
  if (route === "/search" && searchId) return "search";
  if (route === "/") return "gallery";
  if (route === "/post/[id]") return "similar";
  if (route.startsWith("/films/") || route.startsWith("/cameras/")) {
    return "catalog";
  }
  if (route === "/search/keyword/[word]") return "keyword";
  return "direct";
}

function trackPostView(id: number) {
  const searchId = currentSearchId();
  const from = postFrom(searchId);
  if (from === "search") {
    track(
      "post_view",
      { from, post_id: id, search_id: searchId },
      { postId: id, searchId }
    );
  } else {
    track("post_view", { from, post_id: id }, { postId: id });
  }
}

async function downloadImage(id: number) {
  const searchId = currentSearchId();
  track(
    "post_download",
    searchId ? { post_id: id, search_id: searchId } : { post_id: id },
    { postId: id, searchId }
  );
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

function Action({ label, icon: Icon, onClick, href }: ActionProps) {
  return (
    <Tooltip
      label={label}
      withArrow
      events={{ hover: true, focus: true, touch: true }}
      className="px-2"
    >
      <ActionIcon
        variant="subtle"
        color="gray"
        aria-label={label}
        onClick={onClick}
        {...(href && { component: "a", href })}
      >
        <Icon size="24px" />
      </ActionIcon>
    </Tooltip>
  );
}

export default function ImagePage(props: ImagePageProps) {
  let post = props.post;
  let similar = props.similar;
  const isAdmin = useIsAdmin();
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [reporting, setReporting] = useState(false);
  let image = post.images[2];

  useEffect(() => {
    trackPostView(post.id);
  }, [post.id]);

  const actions = [
    {
      label: "download",
      icon: AiOutlineDownload,
      onClick: () => downloadImage(post.id),
    },
    {
      label: "fullscreen",
      icon: AiOutlineArrowsAlt,
      href: post.images[3].url,
      onClick: () =>
        track("outbound_click", { post_id: post.id, target: "original" }),
    },
    { label: "report", icon: AiOutlineFlag, onClick: () => setReporting(true) },
    {
      label: "edit",
      icon: AiOutlineEdit,
      onClick: () => setEditing(true),
      admin: true,
    },
    {
      label: "delete",
      icon: AiOutlineDelete,
      onClick: () => handleDelete(post.id),
      admin: true,
    },
  ];

  return (
    <div>
      <div className={styles.fullscreen}>
        <div className={styles.header}>
          <Header compact />
        </div>
        <div
          className={styles.imageContainer}
          style={
            { "--ratio": `${image.width} / ${image.height}` } as CSSProperties
          }
        >
          <ResponsiveImage
            loading="eager"
            fetchPriority="high"
            style={{ objectFit: "contain" }}
            fill
            src={image.url}
            phoneSrc={post.images[1]?.url}
            alt={postAlt(post)}
            sizes="100vw"
            quality={100}
            placeholder={colorPlaceholder(post, image)}
          />
        </div>
        <div className={styles.footerIcons}>
          <ActionIcon.Group>
            {actions
              .filter((action) => !action.admin || isAdmin)
              .map((action) => (
                <Action key={action.label} {...action} />
              ))}
          </ActionIcon.Group>
        </div>
      </div>
      <ImageTag
        post={post}
        similar={similar}
        filmHref={props.filmHref}
        cameraHref={props.cameraHref}
      />
      <ReportModal
        postId={post.id}
        opened={reporting}
        onClose={() => setReporting(false)}
      />
      {isAdmin && (
        <PostEditor
          post={post}
          opened={editing}
          onClose={() => setEditing(false)}
          onSaved={() => router.refresh()}
        />
      )}

      <Footer />
    </div>
  );
}
