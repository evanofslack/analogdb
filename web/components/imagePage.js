"use client";

import PostEditor from "@components/admin/postEditor";
import Footer from "@components/footer";
import Header from "@components/header";
import ImageTag from "@components/imageTag";
import ReportModal from "@components/reportModal";
import useIsAdmin from "@hooks/useIsAdmin";
import { postAlt } from "@lib/seo";
import { ActionIcon, Tooltip } from "@mantine/core";
import { useRouter } from "next/navigation";
import { useState } from "react";
import {
  AiOutlineArrowsAlt,
  AiOutlineDelete,
  AiOutlineDownload,
  AiOutlineEdit,
  AiOutlineFlag,
} from "react-icons/ai";
import styles from "./imagePage.module.css";
import ResponsiveImage from "./responsiveImage";

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

function Action({ label, icon: Icon, onClick, href }) {
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

export default function ImagePage(props) {
  let post = props.post;
  let similar = props.similar;
  const isAdmin = useIsAdmin();
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [reporting, setReporting] = useState(false);
  let image = post.images[2];

  const actions = [
    {
      label: "download",
      icon: AiOutlineDownload,
      onClick: () => downloadImage(post.id),
    },
    { label: "fullscreen", icon: AiOutlineArrowsAlt, href: post.images[3].url },
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
          style={{ "--ratio": `${image.width} / ${image.height}` }}
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
