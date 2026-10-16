"use client";

import { removePost, updatePost } from "@app/actions/admin";
import { ReviewPost } from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import { Button, Group, Switch, Text } from "@mantine/core";
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
] as const;

function readReviewed() {
  try {
    return Number(window.localStorage.getItem(reviewedKey)) || 0;
  } catch {
    return 0;
  }
}

function writeReviewed(id: number) {
  try {
    window.localStorage.setItem(reviewedKey, String(id));
  } catch {}
}

type FlagKey = (typeof flags)[number]["key"];

interface ReviewCardProps {
  post: ReviewPost;
  isNew: boolean;
  onDeleted: (id: number) => void;
}

function ReviewCard({ post, isNew, onDeleted }: ReviewCardProps) {
  const router = useRouter();
  const [values, setValues] = useState(post);
  const [pending, setPending] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const image =
    post.images.find((i) => i.resolution === "low") ?? post.images[0];

  async function toggle(key: FlagKey, checked: boolean) {
    setPending(key);
    setError(null);
    setValues((v) => ({ ...v, [key]: checked }));
    const result = await updatePost(post.id, { [key]: checked });
    setPending(null);
    if (result.ok === false) {
      setValues((v) => ({ ...v, [key]: !checked }));
      setError(result.error);
    }
  }

  async function remove() {
    if (!confirm(`Delete post ${post.id}?`)) return;
    setPending("delete");
    const result = await removePost(post.id);
    setPending(null);
    if (result.ok === true) {
      onDeleted(post.id);
    } else {
      setError(result.error);
    }
  }

  return (
    <div className={styles.card}>
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
        <div className={styles.meta}>
          {isNew && (
            <>
              <span className={styles.newDot} />
              new ·{" "}
            </>
          )}
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
        <Group gap={6} justify="flex-end">
          <Button size="xs" variant="default" onClick={() => setEditing(true)}>
            Edit
          </Button>
          <Button
            size="xs"
            variant="default"
            onClick={remove}
            loading={pending === "delete"}
          >
            Delete
          </Button>
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

interface ReviewGridProps {
  posts: ReviewPost[];
  firstPage: boolean;
}

export default function ReviewGrid({ posts, firstPage }: ReviewGridProps) {
  const [reviewed, setReviewed] = useState(0);
  const [deleted, setDeleted] = useState<number[]>([]);

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
          {reviewed ? `${newCount} new` : "Nothing reviewed yet"}
        </Text>
        {firstPage && newest > reviewed && (
          <button
            type="button"
            className={styles.textButton}
            onClick={markReviewed}
          >
            Mark reviewed
          </button>
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
