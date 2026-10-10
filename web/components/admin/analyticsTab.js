import { formatNumber, formatPercent } from "@lib/format";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import RangePicker from "./rangePicker";
import Stat from "./stat";
import TabError from "./tabError";
import { bucketLabel } from "./trafficTab";

const vitalThresholds = {
  lcp: [2500, 4000],
  inp: [200, 500],
  cls: [0.1, 0.25],
};

function Change({ current, previous, invert }) {
  if (!previous) {
    return current ? <span>New this period</span> : null;
  }
  const pct = ((current - previous) / previous) * 100;
  const good = invert ? pct < 0 : pct > 0;
  const className = pct === 0 ? "" : good ? styles.fresh : styles.veryStale;
  return (
    <span className={className}>
      {pct > 0 ? "+" : ""}
      {pct.toFixed(1)}% vs previous
    </span>
  );
}

function Chart({ series, bucket }) {
  if (series.length === 0) {
    return <p className={styles.note}>No page views in this range.</p>;
  }
  const max = Math.max(1, ...series.map((b) => b.page_views));
  return (
    <div className={styles.card} style={{ padding: 0 }}>
      <div className={styles.chart}>
        {series.map((b) => {
          const visitors = Math.min(b.visitors, b.page_views);
          const title = `${bucketLabel(b.time, bucket)}: ${formatNumber(
            b.page_views
          )} page views, ${formatNumber(b.visitors)} visitors`;
          return (
            <div key={b.time} className={styles.barColumn} title={title}>
              <div
                className={styles.barWeb}
                style={{ height: `${(visitors / max) * 100}%` }}
              />
              <div
                className={styles.barScraper}
                style={{
                  height: `${((b.page_views - visitors) / max) * 100}%`,
                }}
              />
            </div>
          );
        })}
      </div>
      <div className={styles.axis}>
        <span>{bucketLabel(series[0].time, bucket)}</span>
        <span>{bucketLabel(series[series.length - 1].time, bucket)} UTC</span>
      </div>
      <div className={styles.legend}>
        <span className={styles.legendItem}>
          <span className={`${styles.swatch} ${styles.barWeb}`} />
          Visitors
        </span>
        <span className={styles.legendItem}>
          <span className={`${styles.swatch} ${styles.barScraper}`} />
          Further page views
        </span>
        <span>
          Peak {formatNumber(max)} page views per {bucket}
        </span>
      </div>
    </div>
  );
}

function ViewsTable({ title, rows, nameLabel, href, mono }) {
  return (
    <div>
      <h3 className={styles.sectionTitle}>{title}</h3>
      {rows.length === 0 ? (
        <p className={styles.note}>None.</p>
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
                    <td className={`${styles.wrap} ${mono ? styles.mono : ""}`}>
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
    </div>
  );
}

function PostsTable({ posts }) {
  return (
    <div>
      <h3 className={styles.sectionTitle}>Top posts</h3>
      {posts.length === 0 ? (
        <p className={styles.note}>None.</p>
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
                      className={styles.postCell}
                    >
                      {p.low_url && (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          className={styles.postThumb}
                          src={p.low_url}
                          alt=""
                          loading="lazy"
                        />
                      )}
                      <span className={styles.link}>
                        {p.title || `Post ${p.post_id}`}
                      </span>
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
    </div>
  );
}

function vitalClass(metric, value) {
  const [good, poor] = vitalThresholds[metric];
  if (value <= good) return styles.fresh;
  if (value <= poor) return styles.stale;
  return styles.veryStale;
}

function Vital({ metric, value }) {
  if (value === null || value === undefined) {
    return <td className={styles.num}>–</td>;
  }
  const text =
    metric === "cls"
      ? value.toFixed(3)
      : metric === "lcp"
      ? `${(value / 1000).toFixed(2)} s`
      : `${Math.round(value)} ms`;
  return (
    <td className={`${styles.num} ${vitalClass(metric, value)}`}>{text}</td>
  );
}

function VitalsTable({ vitals }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Web vitals (p75)</h2>
      {vitals.length === 0 ? (
        <p className={styles.note}>No web vitals in this range.</p>
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
    </section>
  );
}

function pagePath(route) {
  return route.startsWith("/") && !route.includes("[") ? route : null;
}

const topTables = [
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

function ratio(a, b) {
  return b ? a / b : 0;
}

function derived(t) {
  return {
    ...t,
    per_visitor: ratio(t.page_views, t.visitors),
    bot_share: ratio(t.bot_views, t.page_views + t.bot_views),
  };
}

export default function AnalyticsTab({ analytics, error, range }) {
  const live = analytics?.summary.live;
  return (
    <>
      <RangePicker tab="analytics" range={range}>
        {live && (
          <span className={styles.liveBadge}>
            <span className={styles.liveDot} />
            {formatNumber(live.visitors)} visitors,{" "}
            {formatNumber(live.page_views)} views in the last 30 min
          </span>
        )}
      </RangePicker>
      {error ? (
        <TabError error={error} />
      ) : (
        <AnalyticsBody analytics={analytics} />
      )}
    </>
  );
}

function AnalyticsBody({ analytics }) {
  const current = derived(analytics.summary.current);
  const previous = derived(analytics.summary.previous);
  const [pages, referrers, sources, campaigns, devices, browsers] =
    topTables.map((t) => (
      <ViewsTable key={t.key} {...t} rows={analytics.top[t.key]} />
    ));
  return (
    <>
      <section className={styles.section}>
        <div className={styles.stats}>
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
            label="Visitor-days"
            value={formatNumber(current.visitors)}
            detail={
              <Change current={current.visitors} previous={previous.visitors} />
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
        <p className={styles.note}>
          {current.page_views === 0 && "No page views in this range yet. "}
          Visitor ids reset every day, so one person visiting on three days
          counts as three visitor-days. Bots are left out of every number but
          bot share.
        </p>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>
          Page views per {analytics.bucket}
        </h2>
        <Chart series={analytics.series} bucket={analytics.bucket} />
      </section>

      <section className={`${styles.section} ${styles.twoCol}`}>
        {pages}
        <PostsTable posts={analytics.posts} />
        {referrers}
        <div>
          {sources}
          <div style={{ marginTop: "1.5rem" }}>{campaigns}</div>
        </div>
        {devices}
        {browsers}
      </section>

      <VitalsTable vitals={analytics.vitals} />
    </>
  );
}
