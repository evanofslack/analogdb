import { AuditEntry, AuditPage } from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import Link from "next/link";
import { ReactNode } from "react";
import styles from "./adminPanel.module.css";
import Pager from "./pager";
import Segmented from "./segmented";

export const activityViews = ["admin", "all"] as const;
export type ActivityView = (typeof activityViews)[number];

const viewLabels: Record<ActivityView, string> = {
  admin: "Admin",
  all: "All",
};

interface Action {
  pattern: RegExp;
  label: string;
  target?: (match: RegExpMatchArray) => ReactNode;
}

const postTarget = (match: RegExpMatchArray) => (
  <Link href={`/post/${match[1]}`} className={styles.link}>
    #{match[1]}
  </Link>
);

const actions: Record<string, Action[]> = {
  PATCH: [
    { pattern: /^\/post\/(\d+)$/, label: "Edited post", target: postTarget },
  ],
  DELETE: [
    { pattern: /^\/post\/(\d+)$/, label: "Deleted post", target: postTarget },
  ],
  POST: [
    {
      pattern: /^\/admin\/reports\/(\d+)\/takedown$/,
      label: "Took down",
      target: (m) => `report ${m[1]}`,
    },
    {
      pattern: /^\/admin\/reports\/(\d+)\/resolve$/,
      label: "Dismissed report",
      target: (m) => `report ${m[1]}`,
    },
    { pattern: /^\/cameras?$/, label: "Added camera" },
    { pattern: /^\/films?$/, label: "Added film" },
    { pattern: /^\/admin\/colors\/rederive$/, label: "Rederived colors" },
  ],
};
actions.PUT = actions.POST;

function describe(entry: AuditEntry): { action: string; target: ReactNode } {
  const path = entry.path.split("?")[0].replace(/^\/v1/, "").replace(/\/$/, "");
  for (const action of actions[entry.method] ?? []) {
    const match = path.match(action.pattern);
    if (match) {
      return { action: action.label, target: action.target?.(match) ?? null };
    }
  }
  return { action: `${entry.method} ${entry.path}`, target: null };
}

function activityHref(view: ActivityView, before?: number | null) {
  const search = new URLSearchParams({ tab: "activity" });
  if (view !== "admin") search.set("view", view);
  if (before) search.set("before", String(before));
  return `/admin?${search}`;
}

function failed(status: number) {
  return status === 401 || status === 403;
}

function AdminRows({ entries }: { entries: AuditEntry[] }) {
  return (
    <table className={`${styles.table} ${styles.narrow}`}>
      <thead>
        <tr>
          <th>Time</th>
          <th>Action</th>
          <th>Target</th>
          <th className={styles.num}>Status</th>
        </tr>
      </thead>
      <tbody>
        {entries.map((e) => {
          const { action, target } = describe(e);
          return (
            <tr
              key={e.request_id || e.start_ms}
              className={e.status >= 400 ? styles.errorRow : undefined}
            >
              <td className={styles.nowrap}>{formatDateTime(e.time)}</td>
              <td className={styles.wrap}>{action}</td>
              <td className={styles.nowrap}>{target}</td>
              <td className={styles.num}>{e.status}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function AllRows({ entries }: { entries: AuditEntry[] }) {
  return (
    <table className={styles.table}>
      <thead>
        <tr>
          <th>Time</th>
          <th>Method</th>
          <th>Path</th>
          <th className={styles.num}>Status</th>
          <th>Client</th>
          <th>IP</th>
          <th>User agent</th>
          <th>Request id</th>
        </tr>
      </thead>
      <tbody>
        {entries.map((e) => (
          <tr
            key={e.request_id || e.start_ms}
            className={failed(e.status) ? styles.errorRow : undefined}
          >
            <td className={styles.nowrap}>{formatDateTime(e.time)}</td>
            <td>{e.method}</td>
            <td className={styles.mono}>{e.path}</td>
            <td className={styles.num}>{e.status}</td>
            <td>{e.client}</td>
            <td className={styles.mono}>{e.remote_ip}</td>
            <td className={styles.wrap}>{e.user_agent}</td>
            <td className={styles.mono}>{e.request_id}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

interface ActivityTabProps {
  page: AuditPage;
  view: ActivityView;
  before: number | null;
}

export default function ActivityTab({ page, view, before }: ActivityTabProps) {
  return (
    <>
      <div className={styles.controls}>
        <Segmented
          active={view}
          items={activityViews.map((v) => ({
            key: v,
            href: activityHref(v),
            label: viewLabels[v],
          }))}
        />
      </div>
      {page.entries.length === 0 ? (
        <p className={styles.empty}>No activity</p>
      ) : (
        <div className={styles.tableWrap}>
          {view === "admin" ? (
            <AdminRows entries={page.entries} />
          ) : (
            <AllRows entries={page.entries} />
          )}
        </div>
      )}
      <Pager
        newest={before ? activityHref(view) : null}
        older={page.next_before ? activityHref(view, page.next_before) : null}
      />
    </>
  );
}
