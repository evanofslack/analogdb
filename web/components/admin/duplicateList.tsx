import { Report, ReportPost } from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import { redditLink } from "@lib/reports";
import Link from "next/link";
import { ReactNode } from "react";
import styles from "./adminPanel.module.css";
import DuplicateActions from "./duplicateActions";

interface DuplicateSideProps {
  label: string;
  postId: number | null;
  post: ReportPost | null;
}

function DuplicateSide({ label, postId, post }: DuplicateSideProps) {
  const src = post?.med_url ?? post?.low_url;
  const reddit = redditLink(post?.permalink ?? null);
  const title = post?.title ?? (postId ? `Post ${postId}` : "Unknown post");
  const facts = [
    post?.author,
    post?.posted_at && formatDateTime(post.posted_at),
    post?.width && post?.height && `${post.width}×${post.height}`,
    post?.removed && "removed",
    post?.nsfw && "nsfw",
  ].filter(Boolean);
  return (
    <div className={styles.dupeSide}>
      <div className={styles.meta}>{label}</div>
      <div className={styles.dupeImage}>
        {src && (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={src} alt={title} loading="lazy" />
        )}
      </div>
      {postId && !post?.removed ? (
        <Link href={`/post/${postId}`} className={styles.link}>
          {title}
        </Link>
      ) : (
        <span>{title}</span>
      )}
      <div className={styles.meta}>
        {facts.join(" · ")}
        {reddit && (
          <>
            {" · "}
            <a
              href={reddit}
              target="_blank"
              rel="noreferrer"
              className={styles.link}
            >
              reddit ↗
            </a>
          </>
        )}
      </div>
    </div>
  );
}

function DuplicatePair({ report }: { report: Report }) {
  return (
    <div className={styles.dupePair}>
      <div className={styles.dupeSides}>
        <DuplicateSide
          label="Original"
          postId={report.duplicate_of}
          post={report.original}
        />
        <DuplicateSide
          label="Reported"
          postId={report.post_id}
          post={report.post}
        />
      </div>
      <div className={styles.dupeFoot}>
        <div className={styles.meta}>
          {[report.message, formatDateTime(report.created_at)]
            .filter(Boolean)
            .join(" · ")}
        </div>
        <DuplicateActions
          reportId={report.id}
          postId={report.post_id}
          removed={report.post.removed}
        />
      </div>
    </div>
  );
}

interface DuplicateListProps {
  reports: Report[];
  pager: ReactNode;
}

export default function DuplicateList({ reports, pager }: DuplicateListProps) {
  return (
    <>
      {reports.length === 0 ? (
        <p className={styles.empty}>No open duplicate reports</p>
      ) : (
        <div className={styles.dupeList}>
          {reports.map((r) => (
            <DuplicatePair key={r.id} report={r} />
          ))}
        </div>
      )}
      {pager}
    </>
  );
}
