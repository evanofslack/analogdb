import {
  Analytics,
  AnalyticsCount,
  AnalyticsTopKey,
  AnalyticsTotals,
  TrafficRange,
} from "@lib/adminClient";
import { formatNumber, formatPercent } from "@lib/format";
import { LineChartSeries } from "@mantine/charts";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import Change from "./change";
import Hint from "./hint";
import RangePicker from "./rangePicker";
import Section from "./section";
import Share from "./share";
import Stat, { Tone, toneClass } from "./stat";
import TabError, { LoadError } from "./tabError";
import TimeChart from "./timeChart";

interface AnalyticsTabProps {
  analytics?: Analytics;
  error?: LoadError;
  range: TrafficRange;
}

type VitalMetric = "lcp" | "inp" | "cls";

interface TopTable {
  key: AnalyticsTopKey;
  title: string;
  nameLabel: string;
  href?: (name: string) => string | null;
  mono?: boolean;
}

interface ViewsTableProps extends Omit<TopTable, "key"> {
  rows: AnalyticsCount[];
}

const viewSeries: LineChartSeries[] = [
  { name: "page_views", label: "Page views", color: "var(--adb-heading)" },
  { name: "visitors", label: "Visitor-days", color: "var(--adb-muted)" },
];

const vitalThresholds: Record<VitalMetric, [number, number]> = {
  lcp: [2500, 4000],
  inp: [200, 500],
  cls: [0.1, 0.25],
};

function sum(rows: { page_views: number }[]) {
  return rows.reduce((total, row) => total + row.page_views, 0);
}

function ViewsTable({ title, rows, nameLabel, href, mono }: ViewsTableProps) {
  const total = sum(rows);
  return (
    <Section title={title} level="h3">
      {rows.length === 0 ? (
        <p className={styles.empty}>None</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>{nameLabel}</th>
                <th className={styles.num}>Views</th>
                <th className={styles.num}>Visitor-days</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => {
                const name = row.name || "(empty)";
                const link = row.name && href ? href(row.name) : null;
                return (
                  <tr key={row.name}>
                    <td className={styles.wrap}>
                      <span className={mono ? styles.mono : undefined}>
                        {link ? (
                          <a
                            href={link}
                            className={styles.link}
                            target={
                              link.startsWith("http") ? "_blank" : undefined
                            }
                            rel="noreferrer"
                          >
                            {name}
                          </a>
                        ) : (
                          name
                        )}
                      </span>
                      <Share value={row.page_views} total={total} />
                    </td>
                    <td className={styles.num}>
                      {formatNumber(row.page_views)}
                    </td>
                    <td className={styles.num}>{formatNumber(row.visitors)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </Section>
  );
}

function PostsTable({ posts }: { posts: Analytics["posts"] }) {
  return (
    <Section title="Top posts" level="h3">
      {posts.length === 0 ? (
        <p className={styles.empty}>None</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Post</th>
                <th className={styles.num}>Views</th>
                <th className={styles.num}>Visitor-days</th>
              </tr>
            </thead>
            <tbody>
              {posts.map((p) => (
                <tr key={p.post_id}>
                  <td className={styles.wrap}>
                    <Link
                      href={`/post/${p.post_id}`}
                      className={`${styles.postCell} ${styles.link}`}
                    >
                      {p.low_url && (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          className={styles.thumb}
                          src={p.low_url}
                          alt=""
                          loading="lazy"
                        />
                      )}
                      <span>{p.title || `Post ${p.post_id}`}</span>
                    </Link>
                  </td>
                  <td className={styles.num}>{formatNumber(p.page_views)}</td>
                  <td className={styles.num}>{formatNumber(p.visitors)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Section>
  );
}

function vitalTone(metric: VitalMetric, value: number): Tone | null {
  const [good, poor] = vitalThresholds[metric];
  if (value <= good) return null;
  if (value <= poor) return "warn";
  return "bad";
}

function Vital({
  metric,
  value,
}: {
  metric: VitalMetric;
  value: number | null;
}) {
  if (value === null || value === undefined) {
    return <td className={`${styles.num} ${styles.muted}`}>–</td>;
  }
  const text =
    metric === "cls"
      ? value.toFixed(3)
      : metric === "lcp"
      ? `${(value / 1000).toFixed(2)} s`
      : `${Math.round(value)} ms`;
  const tone = vitalTone(metric, value);
  return (
    <td className={`${styles.num} ${tone ? toneClass[tone] : ""}`}>{text}</td>
  );
}

function VitalsTable({ vitals }: { vitals: Analytics["vitals"] }) {
  return (
    <Section title="Web vitals p75">
      {vitals.length === 0 ? (
        <p className={styles.empty}>No web vitals in this range</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Route</th>
                <th className={styles.num}>LCP</th>
                <th className={styles.num}>INP</th>
                <th className={styles.num}>CLS</th>
                <th className={styles.num}>Samples</th>
              </tr>
            </thead>
            <tbody>
              {vitals.map((v) => (
                <tr key={v.route}>
                  <td className={styles.mono}>{v.route}</td>
                  <Vital metric="lcp" value={v.lcp} />
                  <Vital metric="inp" value={v.inp} />
                  <Vital metric="cls" value={v.cls} />
                  <td className={styles.num}>{formatNumber(v.samples)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Section>
  );
}

function pagePath(route: string) {
  return route.startsWith("/") && !route.includes("[") ? route : null;
}

const topTables: TopTable[] = [
  {
    key: "pages",
    title: "Top pages",
    nameLabel: "Route",
    href: pagePath,
    mono: true,
  },
  {
    key: "referrers",
    title: "Referrers",
    nameLabel: "Host",
    href: (host) => `https://${host}`,
  },
  { key: "sources", title: "Sources", nameLabel: "utm_source" },
  { key: "campaigns", title: "Campaigns", nameLabel: "utm_campaign" },
  { key: "devices", title: "Devices", nameLabel: "Device" },
  { key: "browsers", title: "Browsers", nameLabel: "Browser" },
];

function ratio(a: number, b: number) {
  return b ? a / b : 0;
}

function derived(t: AnalyticsTotals) {
  return {
    ...t,
    per_visitor: ratio(t.page_views, t.visitors),
    bot_share: ratio(t.bot_views, t.page_views + t.bot_views),
  };
}

export default function AnalyticsTab({
  analytics,
  error,
  range,
}: AnalyticsTabProps) {
  const live = analytics?.summary.live;
  return (
    <>
      <div className={styles.controls}>
        <RangePicker tab="analytics" range={range} />
        {live && (
          <span className={styles.live}>
            {formatNumber(live.visitors)} visitors in last 30 min
          </span>
        )}
      </div>
      {error ? (
        <TabError error={error} />
      ) : (
        <AnalyticsBody analytics={analytics} />
      )}
    </>
  );
}

function AnalyticsBody({ analytics }: { analytics: Analytics }) {
  const current = derived(analytics.summary.current);
  const previous = derived(analytics.summary.previous);
  const [pages, referrers, sources, campaigns, devices, browsers] =
    topTables.map(({ key, ...t }) => (
      <ViewsTable key={key} {...t} rows={analytics.top[key]} />
    ));
  return (
    <>
      <section className={styles.section}>
        <div className={styles.stats}>
          <Stat
            label={
              <Hint
                label="Visitor-days"
                tip="Visitor ids reset every day, so one person on three days counts three times. Bots are left out of every number but bot share."
              />
            }
            value={formatNumber(current.visitors)}
            detail={
              <Change current={current.visitors} previous={previous.visitors} />
            }
          />
          <Stat
            label="Page views"
            value={formatNumber(current.page_views)}
            detail={
              <Change
                current={current.page_views}
                previous={previous.page_views}
              />
            }
          />
          <Stat
            label="Views per visitor-day"
            value={current.per_visitor.toFixed(2)}
            detail={
              <Change
                current={current.per_visitor}
                previous={previous.per_visitor}
              />
            }
          />
          <Stat
            label="Bot share"
            value={formatPercent(current.bot_share, 1)}
            detail={
              <Change
                current={current.bot_share}
                previous={previous.bot_share}
                invert
              />
            }
          />
        </div>
      </section>

      <Section title={`Page views per ${analytics.bucket}`}>
        <TimeChart
          data={analytics.series}
          series={viewSeries}
          bucket={analytics.bucket}
        />
      </Section>

      <div className={`${styles.section} ${styles.twoCol}`}>
        {pages}
        <PostsTable posts={analytics.posts} />
        {referrers}
        <div>
          <div className={styles.subsection}>{sources}</div>
          <div className={styles.subsection}>{campaigns}</div>
        </div>
        {devices}
        {browsers}
      </div>

      <VitalsTable vitals={analytics.vitals} />
    </>
  );
}
