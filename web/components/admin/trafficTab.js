import { trafficRanges } from "@lib/adminClient";
import { formatDateTime, formatNumber, formatPercent } from "@lib/format";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import TabError from "./tabError";

const clients = [
  { key: "web", label: "Web server", className: styles.barWeb },
  { key: "scraper", label: "Scraper", className: styles.barScraper },
  { key: "other", label: "Other", className: styles.barOther },
];

function bucketLabel(time, bucket) {
  const date = new Date(time);
  if (bucket === "hour") {
    return `${date.toISOString().slice(11, 13)}:00`;
  }
  return date.toISOString().slice(5, 10);
}

function Chart({ series, bucket }) {
  if (series.length === 0) {
    return <p className={styles.note}>No requests in this range.</p>;
  }
  const max = Math.max(1, ...series.map((b) => b.web + b.scraper + b.other));
  return (
    <div className={styles.card} style={{ padding: 0 }}>
      <div className={styles.chart}>
        {series.map((b) => {
          const total = b.web + b.scraper + b.other;
          const title = `${bucketLabel(b.time, bucket)}: ${formatNumber(
            total
          )} requests, ${b.status_4xx} 4xx, ${b.status_5xx} 5xx`;
          return (
            <div key={b.time} className={styles.barColumn} title={title}>
              {clients.map((c) => (
                <div
                  key={c.key}
                  className={c.className}
                  style={{ height: `${(b[c.key] / max) * 100}%` }}
                />
              ))}
            </div>
          );
        })}
      </div>
      <div className={styles.axis}>
        <span>{bucketLabel(series[0].time, bucket)}</span>
        <span>{bucketLabel(series[series.length - 1].time, bucket)} UTC</span>
      </div>
      <div className={styles.legend}>
        {clients.map((c) => (
          <span key={c.key} className={styles.legendItem}>
            <span className={`${styles.swatch} ${c.className}`} />
            {c.label}
          </span>
        ))}
        <span>
          Peak {formatNumber(max)} per {bucket}
        </span>
      </div>
    </div>
  );
}

function Stat({ label, value, detail }) {
  return (
    <div className={styles.card}>
      <div className={styles.statLabel}>{label}</div>
      <div className={styles.statValue}>{value}</div>
      {detail && <div className={styles.statDetail}>{detail}</div>}
    </div>
  );
}

function CountTable({ title, rows, nameLabel, showClient, mono }) {
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
                {showClient && <th>Client</th>}
                <th className={styles.num}>Requests</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={`${row.name}|${row.client ?? ""}`}>
                  <td className={`${styles.wrap} ${mono ? styles.mono : ""}`}>
                    {row.name || "(empty)"}
                  </td>
                  {showClient && <td>{row.client}</td>}
                  <td className={styles.num}>{formatNumber(row.requests)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export default function TrafficTab({ traffic, error, range }) {
  return (
    <>
      <div className={styles.controls}>
        {trafficRanges.map((r) => (
          <Link
            key={r}
            href={`/admin?tab=traffic&range=${r}`}
            className={`${styles.chip} ${r === range ? styles.chipActive : ""}`}
          >
            {r}
          </Link>
        ))}
      </div>
      <p className={styles.note} style={{ marginBottom: "1.5rem" }}>
        Backend API requests from ClickHouse. Pages served from the web cache
        never reach the API, and page renders show as the web server&apos;s IP.
      </p>

      {error ? <TabError error={error} /> : <TrafficBody traffic={traffic} />}
    </>
  );
}

function TrafficBody({ traffic }) {
  const { totals } = traffic;
  return (
    <>
      <section className={styles.section}>
        <div className={styles.stats}>
          <Stat label="Requests" value={formatNumber(totals.requests)} />
          <Stat
            label="IPs seen by the API"
            value={formatNumber(totals.unique_ips)}
          />
          <Stat
            label="4xx"
            value={formatNumber(totals.status_4xx)}
            detail={formatPercent(totals.status_4xx, totals.requests)}
          />
          <Stat
            label="5xx"
            value={formatNumber(totals.status_5xx)}
            detail={formatPercent(totals.status_5xx, totals.requests)}
          />
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Requests per {traffic.bucket}</h2>
        <Chart series={traffic.series} bucket={traffic.bucket} />
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Top routes</h2>
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Route</th>
                <th className={styles.num}>Requests</th>
                <th className={styles.num}>p50 ms</th>
                <th className={styles.num}>p95 ms</th>
                <th className={styles.num}>5xx</th>
              </tr>
            </thead>
            <tbody>
              {traffic.routes.map((r) => (
                <tr key={r.route}>
                  <td className={styles.mono}>{r.route}</td>
                  <td className={styles.num}>{formatNumber(r.requests)}</td>
                  <td className={styles.num}>{Math.round(r.p50_ms)}</td>
                  <td className={styles.num}>{Math.round(r.p95_ms)}</td>
                  <td className={styles.num}>{formatNumber(r.errors)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className={`${styles.section} ${styles.twoCol}`}>
        <div>
          <h3 className={styles.sectionTitle}>Most requested posts</h3>
          {traffic.posts.length === 0 ? (
            <p className={styles.note}>None.</p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Post</th>
                    <th className={styles.num}>Requests</th>
                  </tr>
                </thead>
                <tbody>
                  {traffic.posts.map((p) => (
                    <tr key={p.post_id}>
                      <td>
                        <Link
                          href={`/post/${p.post_id}`}
                          className={styles.link}
                        >
                          #{p.post_id}
                        </Link>
                      </td>
                      <td className={styles.num}>{formatNumber(p.requests)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        <div>
          <h3 className={styles.sectionTitle}>Legacy routes (no /v1)</h3>
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Client</th>
                  <th className={styles.num}>Legacy</th>
                  <th className={styles.num}>Share</th>
                </tr>
              </thead>
              <tbody>
                {traffic.legacy.map((l) => (
                  <tr key={l.client}>
                    <td>{l.client}</td>
                    <td className={styles.num}>{formatNumber(l.legacy)}</td>
                    <td className={styles.num}>
                      {formatPercent(l.legacy, l.requests)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        <CountTable
          title="Filters used on /posts"
          rows={traffic.params}
          nameLabel="Param"
          mono
        />
        <CountTable
          title="Top IPs (not web or scraper)"
          rows={traffic.ips}
          nameLabel="IP"
          mono
        />
      </section>

      <section className={styles.section}>
        <CountTable
          title="Top user agents"
          rows={traffic.user_agents}
          nameLabel="User agent"
          showClient
        />
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Recent 5xx</h2>
        {traffic.errors.length === 0 ? (
          <p className={styles.note}>No server errors in this range.</p>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Method</th>
                  <th>Path</th>
                  <th className={styles.num}>Status</th>
                  <th>Request id</th>
                </tr>
              </thead>
              <tbody>
                {traffic.errors.map((e) => (
                  <tr key={e.request_id || e.time}>
                    <td>{formatDateTime(e.time)}</td>
                    <td>{e.method}</td>
                    <td className={styles.mono}>{e.path}</td>
                    <td className={styles.num}>{e.status}</td>
                    <td className={styles.mono}>{e.request_id}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  );
}
