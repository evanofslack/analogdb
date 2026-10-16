import {
  Report,
  ReportsPage,
  ReportStatus,
  reportStatuses,
} from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import { reportReasonLabel } from "@lib/reports";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import Pager from "./pager";
import ReportActions from "./reportActions";
import Segmented from "./segmented";

function redditLink(permalink: string | null): string | null {
  if (!permalink) return null;
  return permalink.startsWith("http")
    ? permalink
    : `https://www.reddit.com${permalink}`;
}

interface ReportRowProps {
  report: Report;
}

function ReportRow({ report }: ReportRowProps) {
  const { post } = report;
  const reddit = redditLink(post.permalink);
  const title = post.title ?? `Post ${report.post_id}`;
  const marks = [post.removed && "removed", post.nsfw && "nsfw"].filter(
    Boolean
  );
  return (
    <tr>
      <td>
        {post.low_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            className={styles.thumb}
            src={post.low_url}
            alt={title}
            loading="lazy"
          />
        ) : (
          <span className={styles.thumb} />
        )}
      </td>
      <td className={styles.wrap}>
        {post.removed ? (
          <span>{title}</span>
        ) : (
          <Link href={`/post/${report.post_id}`} className={styles.link}>
            {title}
          </Link>
        )}
        <div className={styles.meta}>
          {[post.author, ...marks].filter(Boolean).join(" · ")}
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
      </td>
      <td className={styles.wrap}>
        <div>{reportReasonLabel(report.reason)}</div>
        {report.message && (
          <div className={`${styles.message} ${styles.meta}`}>
            {report.message}
          </div>
        )}
        {report.email && (
          <div className={styles.meta}>
            <a href={`mailto:${report.email}`} className={styles.link}>
              {report.email}
            </a>
          </div>
        )}
      </td>
      <td className={styles.nowrap}>
        {formatDateTime(report.created_at)}
        {report.resolved_at && (
          <div className={styles.meta}>
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

interface ReportsTabProps {
  page: ReportsPage;
  status: ReportStatus;
  firstPage: boolean;
}

export default function ReportsTab({
  page,
  status,
  firstPage,
}: ReportsTabProps) {
  return (
    <>
      <div className={styles.controls}>
        <Segmented
          active={status}
          items={reportStatuses.map((s) => ({
            key: s,
            href: `/admin?tab=reports&status=${s}`,
            label: s,
          }))}
        />
      </div>
      {page.reports.length === 0 ? (
        <p className={styles.empty}>
          {status === "all" ? "No reports" : `No ${status} reports`}
        </p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={`${styles.table} ${styles.top}`}>
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
      <Pager
        newest={firstPage ? null : `/admin?tab=reports&status=${status}`}
        older={
          page.next_before
            ? `/admin?tab=reports&status=${status}&before=${page.next_before}`
            : null
        }
      />
    </>
  );
}
