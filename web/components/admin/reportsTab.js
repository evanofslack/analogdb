import { reportStatuses } from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import { reportReasonLabel } from "@lib/reports";
import { Badge, Button } from "@mantine/core";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import ReportActions from "./reportActions";

function redditLink(permalink) {
  if (!permalink) return null;
  return permalink.startsWith("http")
    ? permalink
    : `https://www.reddit.com${permalink}`;
}

function ReportRow({ report }) {
  const { post } = report;
  const reddit = redditLink(post.permalink);
  return (
    <tr>
      <td>
        {post.low_url && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            className={styles.reportThumb}
            src={post.low_url}
            alt={post.title ?? ""}
            loading="lazy"
          />
        )}
      </td>
      <td className={styles.wrap}>
        {post.removed ? (
          <span>{post.title ?? `Post ${report.post_id}`}</span>
        ) : (
          <Link href={`/post/${report.post_id}`} className={styles.link}>
            {post.title ?? `Post ${report.post_id}`}
          </Link>
        )}
        <div className={styles.reportMeta}>
          {post.author}
          {reddit && (
            <>
              {" · "}
              <a href={reddit} target="_blank" rel="noreferrer">
                reddit ↗
              </a>
            </>
          )}
          {post.removed && (
            <Badge ml="xs" size="xs" color="red" variant="light">
              removed
            </Badge>
          )}
          {post.nsfw && (
            <Badge ml="xs" size="xs" color="orange" variant="light">
              nsfw
            </Badge>
          )}
        </div>
      </td>
      <td className={styles.wrap}>
        <strong>{reportReasonLabel(report.reason)}</strong>
        {report.message && (
          <div className={styles.reportMessage}>{report.message}</div>
        )}
        {report.email && (
          <div className={styles.reportMeta}>
            <a href={`mailto:${report.email}`}>{report.email}</a>
          </div>
        )}
      </td>
      <td>
        {formatDateTime(report.created_at)}
        {report.resolved_at && (
          <div className={styles.reportMeta}>
            resolved {formatDateTime(report.resolved_at)}
          </div>
        )}
      </td>
      <td>
        <ReportActions
          reportId={report.id}
          postId={report.post_id}
          removed={post.removed}
          resolved={!!report.resolved_at}
        />
      </td>
    </tr>
  );
}

export default function ReportsTab({ page, status, firstPage }) {
  return (
    <>
      <div className={styles.controls}>
        {reportStatuses.map((s) => (
          <Link
            key={s}
            href={`/admin?tab=reports&status=${s}`}
            className={`${styles.chip} ${
              s === status ? styles.chipActive : ""
            }`}
          >
            {s}
          </Link>
        ))}
      </div>
      <p className={styles.note} style={{ marginBottom: "1rem" }}>
        Reports from the post page, newest first. Take down deletes the post and
        its vector and stops the scraper adding it again, the S3 images stay.
        Dismiss resolves the report and clears its email.
      </p>
      {page.reports.length === 0 ? (
        <p className={styles.note}>No reports.</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th></th>
                <th>Post</th>
                <th>Report</th>
                <th>Time</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {page.reports.map((r) => (
                <ReportRow key={r.id} report={r} />
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className={styles.pager}>
        {!firstPage && (
          <Button
            component={Link}
            href={`/admin?tab=reports&status=${status}`}
            variant="default"
            mr="sm"
          >
            Newest
          </Button>
        )}
        {page.next_before && (
          <Button
            component={Link}
            href={`/admin?tab=reports&status=${status}&before=${page.next_before}`}
            variant="default"
          >
            Older
          </Button>
        )}
      </div>
    </>
  );
}
