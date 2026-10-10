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

function CampaignsTable({ campaigns }) {
  return (
    <div>
      <h3 className={styles.sectionTitle}>Campaigns</h3>
      {campaigns.length === 0 ? (
        <p className={styles.note}>None.</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Source</th>
                <th>Campaign</th>
                <th className={styles.num}>Views</th>
                <th className={styles.num}>Visitor-days</th>
              </tr>
            </thead>
            <tbody>
              {campaigns.map((c) => (
                <tr key={`${c.source}|${c.campaign}`}>
                  <td>{c.source || "(none)"}</td>
                  <td>{c.campaign || "(none)"}</td>
                  <td className={styles.num}>{formatNumber(c.page_views)}</td>
                  <td className={styles.num}>{formatNumber(c.visitors)}</td>
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

export default function AnalyticsTab({ analytics, error, range }) {
  const live = analytics?.available ? analytics.live : null;
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
      ) : !analytics.available ? (
        <p className={styles.note}>
          No UI events yet. Needs the consumer migration and web page view
          tracking deployed.
        </p>
      ) : (
        <AnalyticsBody analytics={analytics} />
      )}
    </>
  );
}

function AnalyticsBody({ analytics }) {
  const { totals, previous } = analytics;
  return (
    <>
      <section className={styles.section}>
        <div className={styles.stats}>
          <Stat
            label="Page views"
            value={formatNumber(totals.page_views)}
            detail={
              <Change
                current={totals.page_views}
                previous={previous.page_views}
              />
            }
          />
          <Stat
            label="Visitor-days"
            value={formatNumber(totals.visitor_days)}
            detail={
              <Change
                current={totals.visitor_days}
                previous={previous.visitor_days}
              />
            }
          />
          <Stat
            label="Views per visitor-day"
            value={totals.views_per_visitor.toFixed(2)}
            detail={
              <Change
                current={totals.views_per_visitor}
                previous={previous.views_per_visitor}
              />
            }
          />
          <Stat
            label="Bot share"
            value={formatPercent(totals.bot_share, 1)}
            detail={
              <Change
                current={totals.bot_share}
                previous={previous.bot_share}
                invert
              />
            }
          />
        </div>
        <p className={styles.note}>
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
        <ViewsTable
          title="Top pages"
          rows={analytics.pages}
          nameLabel="Route"
          href={pagePath}
          mono
        />
        <PostsTable posts={analytics.posts} />
        <ViewsTable
          title="Referrers"
          rows={analytics.referrers}
          nameLabel="Host"
          href={(host) => `https://${host}`}
        />
        <CampaignsTable campaigns={analytics.campaigns} />
        <div>
          <ViewsTable
            title="Devices"
            rows={analytics.devices}
            nameLabel="Device"
          />
          <div style={{ marginTop: "1.5rem" }}>
            <ViewsTable
              title="Browsers"
              rows={analytics.browsers}
              nameLabel="Browser"
            />
          </div>
        </div>
        {analytics.countries.length > 0 && (
          <ViewsTable
            title="Countries"
            rows={analytics.countries}
            nameLabel="Country"
          />
        )}
      </section>

      <VitalsTable vitals={analytics.vitals} />
    </>
  );
}
