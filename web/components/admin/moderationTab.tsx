import {
  Report,
  ReportsPage,
  ReportStatus,
  reportStatuses,
  ReviewPage,
} from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import { reportReasonLabel } from "@lib/reports";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import IncomingList from "./incomingList";
import Pager from "./pager";
import ReportActions from "./reportActions";
import Section from "./section";
import Segmented from "./segmented";
import TabError, { LoadError } from "./tabError";

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

export interface ModerationParams {
  status: ReportStatus;
  before: number | null;
  cursor: string | null;
}

export function moderationHref(params: ModerationParams) {
  const search = new URLSearchParams({ tab: "moderation" });
  if (params.status !== "open") search.set("status", params.status);
  if (params.before) search.set("before", String(params.before));
  if (params.cursor) search.set("cursor", params.cursor);
  return `/admin?${search}`;
}

interface ModerationTabProps {
  params: ModerationParams;
  reports?: ReportsPage;
  reportsError?: LoadError;
  incoming?: ReviewPage;
  incomingError?: LoadError;
  openReports?: number;
}

export default function ModerationTab({
  params,
  reports,
  reportsError,
  incoming,
  incomingError,
  openReports,
}: ModerationTabProps) {
  const { status, before, cursor } = params;
  return (
    <>
      <Section
        title="Reports"
        count={openReports}
        right={
          <Segmented
            active={status}
            items={reportStatuses.map((s) => ({
              key: s,
              href: moderationHref({ status: s, before: null, cursor }),
              label: s,
            }))}
          />
        }
      >
        {reportsError ? (
          <TabError error={reportsError} />
        ) : reports.reports.length === 0 ? (
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
                {reports.reports.map((r) => (
                  <ReportRow key={r.id} report={r} />
                ))}
              </tbody>
            </table>
          </div>
        )}
        <Pager
          newest={
            before ? moderationHref({ status, before: null, cursor }) : null
          }
          older={
            reports?.next_before
              ? moderationHref({ status, before: reports.next_before, cursor })
              : null
          }
        />
      </Section>

      {incomingError ? (
        <Section title="Incoming">
          <TabError error={incomingError} />
        </Section>
      ) : (
        <IncomingList
          posts={incoming.posts}
          firstPage={!cursor}
          pager={
            <Pager
              newest={
                cursor ? moderationHref({ status, before, cursor: null }) : null
              }
              older={
                incoming.next_cursor
                  ? moderationHref({
                      status,
                      before,
                      cursor: incoming.next_cursor,
                    })
                  : null
              }
            />
          }
        />
      )}
    </>
  );
}
