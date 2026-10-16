"use client";

import { removePost, updatePost } from "@app/actions/admin";
import { ReviewPost } from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import { Button, Switch } from "@mantine/core";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ReactNode, useEffect, useState } from "react";
import styles from "./adminPanel.module.css";
import PostEditor from "./postEditor";
import Section from "./section";

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

function gear(post: ReviewPost) {
  const camera = [post.camera_make, post.camera_model]
    .filter(Boolean)
    .join(" ");
  const film = [post.film_make, post.film_type].filter(Boolean).join(" ");
  return { camera, film };
}

interface IncomingRowProps {
  post: ReviewPost;
  isNew: boolean;
  onDeleted: (id: number) => void;
}

function IncomingRow({ post, isNew, onDeleted }: IncomingRowProps) {
  const router = useRouter();
  const [values, setValues] = useState(post);
  const [pending, setPending] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const image =
    post.images.find((i) => i.resolution === "low") ?? post.images[0];
  const { camera, film } = gear(values);

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
    <div className={styles.incoming}>
      <Link href={`/post/${post.id}`}>
        {image && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            className={styles.thumb}
            src={image.url}
            alt={post.title}
            loading="lazy"
          />
        )}
      </Link>
      <div className={styles.rowText}>
        <div>
          <Link href={`/post/${post.id}`} className={styles.link}>
            {post.title}
          </Link>
        </div>
        <div className={styles.meta}>
          #{post.id} · {post.author} · {formatDateTime(post.timestamp * 1000)}
          {isNew && (
            <>
              {" · "}
              <span className={styles.newDot} />
              new
            </>
          )}
        </div>
        <div className={styles.meta}>
          {camera || <span className={styles.faint}>no camera</span>}
          {" · "}
          {film || <span className={styles.faint}>no film</span>}
        </div>
        {error && <div className={styles.veryStale}>{error}</div>}
      </div>
      <div className={styles.incomingControls}>
        {flags.map((flag) => (
          <Switch
            key={flag.key}
            size="xs"
            color="gray"
            label={flag.label}
            checked={!!values[flag.key]}
            disabled={pending !== null}
            onChange={(e) => toggle(flag.key, e.currentTarget.checked)}
          />
        ))}
        <div className={styles.actions}>
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
        </div>
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

interface IncomingListProps {
  posts: ReviewPost[];
  firstPage: boolean;
  pager: ReactNode;
}

export default function IncomingList({
  posts,
  firstPage,
  pager,
}: IncomingListProps) {
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
    <Section
      title="Incoming"
      right={
        <>
          {reviewed > 0 && <span>{newCount} new</span>}
          <button
            type="button"
            className={styles.textButton}
            onClick={markReviewed}
            disabled={!firstPage || newest <= reviewed}
          >
            Mark reviewed
          </button>
        </>
      }
    >
      {visible.length === 0 ? (
        <p className={styles.empty}>No posts</p>
      ) : (
        <div className={styles.rows}>
          {visible.map((post) => (
            <IncomingRow
              key={post.id}
              post={post}
              isNew={reviewed > 0 && post.id > reviewed}
              onDeleted={(id) => setDeleted((d) => [...d, id])}
            />
          ))}
        </div>
      )}
      {pager}
    </Section>
  );
}
