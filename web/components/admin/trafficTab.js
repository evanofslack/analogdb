import { formatDateTime, formatNumber, formatPercent } from "@lib/format";
import styles from "./adminPanel.module.css";
import Change from "./change";
import RangePicker from "./rangePicker";
import Stat from "./stat";
import TabError from "./tabError";
import TimeChart from "./timeChart";

const requestSeries = [
  { name: "web", label: "Web server", color: "blue.6" },
  { name: "scraper", label: "Scraper", color: "teal.6" },
  { name: "direct", label: "Direct", color: "gray.6" },
];

const errorSeries = [
  { name: "status_4xx", label: "4xx", color: "orange.6" },
  { name: "status_5xx", label: "5xx", color: "red.6" },
];

function ratio(a, b) {
  return b ? a / b : 0;
}

function derived(t) {
  return {
    ...t,
    direct_share: ratio(t.direct, t.requests),
    rate_4xx: ratio(t.status_4xx, t.requests),
    rate_5xx: ratio(t.status_5xx, t.requests),
    per_view: ratio(t.web, t.page_views),
  };
}

function seconds(ms) {
  return formatNumber(Number((ms / 1000).toFixed(1)));
}

export default function TrafficTab({ traffic, error, range }) {
  return (
    <>
      <RangePicker tab="traffic" range={range} />
      <p className={styles.note} style={{ marginBottom: "1.5rem" }}>
        Backend API requests. Visitors are on the Analytics tab.
      </p>

      {error ? <TabError error={error} /> : <TrafficBody traffic={traffic} />}
    </>
  );
}

function Summary({ summary }) {
  const current = derived(summary.current);
  const previous = derived(summary.previous);
  return (
    <section className={styles.section}>
      <div className={styles.stats}>
        <Stat
          label="Requests"
          value={formatNumber(current.requests)}
          detail={
            <Change current={current.requests} previous={previous.requests} />
          }
        />
        <Stat
          label="Direct share"
          value={formatPercent(current.direct, current.requests)}
          detail={
            <Change
              current={current.direct_share}
              previous={previous.direct_share}
              invert
            />
          }
        />
        <Stat
          label="5xx rate"
          value={formatPercent(current.status_5xx, current.requests)}
          detail={
            <Change
              current={current.rate_5xx}
              previous={previous.rate_5xx}
              invert
            />
          }
        />
        <Stat
          label="4xx rate"
          value={formatPercent(current.status_4xx, current.requests)}
          detail={
            <Change
              current={current.rate_4xx}
              previous={previous.rate_4xx}
              invert
            />
          }
        />
        <Stat
          label="p95 latency"
          value={`${formatNumber(Math.round(current.p95_ms))} ms`}
          detail={
            <Change
              current={current.p95_ms}
              previous={previous.p95_ms}
              invert
            />
          }
        />
        <Stat
          label="API calls per page view"
          value={current.page_views ? current.per_view.toFixed(1) : "–"}
          detail={
            <Change
              current={current.per_view}
              previous={previous.per_view}
              invert
            />
          }
        />
      </div>
    </section>
  );
}

function CallersTable({ callers, direct }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Direct callers</h2>
      {callers.length === 0 ? (
        <p className={styles.note}>None.</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Caller</th>
                <th>Kind</th>
                <th className={styles.num}>Requests</th>
                <th className={styles.num}>Share of direct</th>
                <th className={styles.num}>IPs</th>
              </tr>
            </thead>
            <tbody>
              {callers.map((c) => (
                <tr key={`${c.kind}|${c.name}`}>
                  <td className={styles.wrap}>{c.name || "(empty)"}</td>
                  <td>{c.kind}</td>
                  <td className={styles.num}>{formatNumber(c.requests)}</td>
                  <td className={styles.num}>
                    {formatPercent(c.requests, direct)}
                  </td>
                  <td className={styles.num}>{formatNumber(c.ips)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function RoutesTable({ routes }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Routes by total time</h2>
      {routes.length === 0 ? (
        <p className={styles.note}>None.</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Route</th>
                <th className={styles.num}>Requests</th>
                <th className={styles.num}>Total (s)</th>
                <th className={styles.num}>p50 ms</th>
                <th className={styles.num}>p95 ms</th>
                <th className={styles.num}>4xx</th>
                <th className={styles.num}>5xx</th>
              </tr>
            </thead>
            <tbody>
              {routes.map((r) => (
                <tr key={r.route}>
                  <td className={styles.mono}>{r.route}</td>
                  <td className={styles.num}>{formatNumber(r.requests)}</td>
                  <td className={styles.num}>{seconds(r.total_ms)}</td>
                  <td className={styles.num}>{Math.round(r.p50_ms)}</td>
                  <td className={styles.num}>{Math.round(r.p95_ms)}</td>
                  <td className={styles.num}>{formatNumber(r.status_4xx)}</td>
                  <td className={styles.num}>{formatNumber(r.status_5xx)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function StatusTable({ rows }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Errors by status</h2>
      {rows.length === 0 ? (
        <p className={styles.note}>No errors in this range.</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.num}>Status</th>
                <th>Route</th>
                <th className={styles.num}>Requests</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={`${r.status}|${r.route}`}>
                  <td className={styles.num}>{r.status}</td>
                  <td className={styles.mono}>{r.route}</td>
                  <td className={styles.num}>{formatNumber(r.requests)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function RecentTable({ rows }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Recent 5xx</h2>
      {rows.length === 0 ? (
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
              {rows.map((e) => (
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
  );
}

function TrafficBody({ traffic }) {
  return (
    <>
      <Summary summary={traffic.summary} />

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Requests per {traffic.bucket}</h2>
        <TimeChart
          data={traffic.series}
          series={requestSeries}
          bucket={traffic.bucket}
        />
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Errors per {traffic.bucket}</h2>
        <TimeChart
          data={traffic.series}
          series={errorSeries}
          bucket={traffic.bucket}
          height={140}
        />
      </section>

      <CallersTable
        callers={traffic.callers}
        direct={traffic.summary.current.direct}
      />
      <RoutesTable routes={traffic.routes} />
      <StatusTable rows={traffic.errors.by_status} />
      <RecentTable rows={traffic.errors.recent} />
    </>
  );
}
