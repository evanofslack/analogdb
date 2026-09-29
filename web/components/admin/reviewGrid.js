"use client";

import { removePost, updatePost } from "@app/actions/admin";
import { formatDateTime } from "@lib/format";
import {
  ActionIcon,
  Button,
  Group,
  Switch,
  Text,
  Tooltip,
} from "@mantine/core";
import { IconPencil, IconTrash } from "@tabler/icons-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import styles from "./adminPanel.module.css";
import PostEditor from "./postEditor";

const reviewedKey = "adb-admin-last-reviewed";
const flags = [
  { key: "nsfw", label: "NSFW" },
  { key: "grayscale", label: "B&W" },
  { key: "sprocket", label: "Sprocket" },
];

function readReviewed() {
  try {
    return Number(window.localStorage.getItem(reviewedKey)) || 0;
  } catch {
    return 0;
  }
}

function writeReviewed(id) {
  try {
    window.localStorage.setItem(reviewedKey, String(id));
  } catch {}
}

function ReviewCard({ post, isNew, onDeleted }) {
  const router = useRouter();
  const [values, setValues] = useState(post);
  const [pending, setPending] = useState(null);
  const [error, setError] = useState(null);
  const [editing, setEditing] = useState(false);
  const image =
    post.images.find((i) => i.resolution === "low") ?? post.images[0];

  async function toggle(key, checked) {
    setPending(key);
    setError(null);
    setValues((v) => ({ ...v, [key]: checked }));
    const result = await updatePost(post.id, { [key]: checked });
    setPending(null);
    if (!result.ok) {
      setValues((v) => ({ ...v, [key]: !checked }));
      setError(result.error);
    }
  }

  async function remove() {
    if (!confirm(`Delete post ${post.id}?`)) return;
    setPending("delete");
    const result = await removePost(post.id);
    setPending(null);
    if (result.ok) {
      onDeleted(post.id);
    } else {
      setError(result.error);
    }
  }

  return (
    <div
      className={`${styles.reviewCard} ${isNew ? styles.reviewCardNew : ""}`}
    >
      <Link href={`/post/${post.id}`} className={styles.reviewImage}>
        {image && (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={image.url} alt={post.title} loading="lazy" />
        )}
      </Link>
      <div className={styles.reviewBody}>
        <Link href={`/post/${post.id}`} className={styles.reviewTitle}>
          {post.title}
        </Link>
        <div className={styles.reviewMeta}>
          #{post.id} · {post.author} · {formatDateTime(post.timestamp * 1000)}
        </div>
        <Group gap="sm">
          {flags.map((flag) => (
            <Switch
              key={flag.key}
              size="xs"
              label={flag.label}
              checked={!!values[flag.key]}
              disabled={pending !== null}
              onChange={(e) => toggle(flag.key, e.currentTarget.checked)}
            />
          ))}
        </Group>
        <Group gap="xs" justify="flex-end">
          <Tooltip label="edit">
            <ActionIcon
              variant="subtle"
              color="gray"
              onClick={() => setEditing(true)}
              aria-label="edit"
            >
              <IconPencil size={18} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="delete">
            <ActionIcon
              variant="subtle"
              color="red"
              onClick={remove}
              loading={pending === "delete"}
              aria-label="delete"
            >
              <IconTrash size={18} />
            </ActionIcon>
          </Tooltip>
        </Group>
        {error && (
          <Text size="xs" c="red">
            {error}
          </Text>
        )}
      </div>
      <PostEditor
        post={values}
        opened={editing}
        onClose={() => setEditing(false)}
        onSaved={(patch) => {
          setValues((v) => ({ ...v, ...patch }));
          router.refresh();
        }}
      />
    </div>
  );
}

export default function ReviewGrid({ posts, firstPage }) {
  const [reviewed, setReviewed] = useState(0);
  const [deleted, setDeleted] = useState([]);

  useEffect(() => {
    setReviewed(readReviewed());
  }, []);

  const visible = posts.filter((p) => !deleted.includes(p.id));
  const newCount = visible.filter((p) => p.id > reviewed).length;
  const newest = posts.reduce((max, p) => Math.max(max, p.id), 0);

  function markReviewed() {
    writeReviewed(newest);
    setReviewed(newest);
  }

  return (
    <>
      <div className={styles.controls}>
        <Text size="sm" c="dimmed">
          {reviewed
            ? `${newCount} new since last review (post ${reviewed})`
            : "No review marker yet"}
        </Text>
        {firstPage && newest > reviewed && (
          <Button size="xs" variant="light" onClick={markReviewed}>
            Mark all reviewed
          </Button>
        )}
      </div>
      <div className={styles.reviewGrid}>
        {visible.map((post) => (
          <ReviewCard
            key={post.id}
            post={post}
            isNew={reviewed > 0 && post.id > reviewed}
            onDeleted={(id) => setDeleted((d) => [...d, id])}
          />
        ))}
      </div>
    </>
  );
}
