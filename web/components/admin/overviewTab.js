import {
  formatAgo,
  formatBytes,
  formatDuration,
  formatNumber,
} from "@lib/format";
import { Badge } from "@mantine/core";
import styles from "./adminPanel.module.css";

const statusColors = { ok: "green", down: "red", disabled: "gray" };
const dependencies = ["postgres", "redis", "weaviate", "clickhouse"];

const day = 24 * 60 * 60;

function freshnessClass(unixSeconds, now) {
  if (!unixSeconds) return styles.veryStale;
  const age = now / 1000 - unixSeconds;
  if (age > 4 * day) return styles.veryStale;
  if (age > 2 * day) return styles.stale;
  return styles.fresh;
}

function Stat({ label, value, detail, className }) {
  return (
    <div className={styles.card}>
      <div className={styles.statLabel}>{label}</div>
      <div className={`${styles.statValue} ${className ?? ""}`}>{value}</div>
      {detail && <div className={styles.statDetail}>{detail}</div>}
    </div>
  );
}

function vectorDetail({ objects, posts }) {
  if (objects === null) return "Could not count objects";
  const diff = objects - posts;
  if (diff === 0) return "Matches post count";
  if (diff > 0)
    return `${formatNumber(diff)} more objects than posts (duplicates)`;
  return `${formatNumber(-diff)} posts without an object`;
}

export default function OverviewTab({ overview, now }) {
  const { status, app, counts, posted, freshness, database, vectors } =
    overview;

  const freshnessItems = [
    { label: "Newest post", value: freshness.newest_post },
    { label: "Score update", value: freshness.score_update },
    { label: "Keywords update", value: freshness.keywords_update },
    { label: "Colors update", value: freshness.colors_update },
  ];

  return (
    <>
      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Status</h2>
        <div className={styles.statusRow}>
          {dependencies.map((name) => (
            <Badge
              key={name}
              color={statusColors[status[name]] ?? "gray"}
              variant="light"
              size="lg"
            >
              {name}: {status[name] ?? "unknown"}
            </Badge>
          ))}
        </div>
        <p className={styles.note}>
          API {app.version} ({app.env}) on {app.hostname}, up{" "}
          {formatDuration(app.uptime_seconds)}
        </p>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Content</h2>
        <div className={styles.stats}>
          <Stat label="Posts" value={formatNumber(counts.posts)} />
          <Stat label="Authors" value={formatNumber(counts.authors)} />
          <Stat label="Cameras" value={formatNumber(counts.cameras)} />
          <Stat label="Films" value={formatNumber(counts.films)} />
          <Stat label="Keywords" value={formatNumber(counts.keywords)} />
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Posted</h2>
        <div className={styles.stats}>
          <Stat label="Last 24 hours" value={formatNumber(posted.day)} />
          <Stat label="Last 7 days" value={formatNumber(posted.week)} />
          <Stat label="Last 30 days" value={formatNumber(posted.month)} />
        </div>
        <p className={styles.note}>
          By Reddit post date, not the date we added the post.
        </p>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Pipeline freshness</h2>
        <div className={styles.stats}>
          {freshnessItems.map((item) => (
            <Stat
              key={item.label}
              label={item.label}
              value={formatAgo(item.value, now)}
              className={freshnessClass(item.value, now)}
            />
          ))}
        </div>
        <p className={styles.note}>
          Amber after 2 days, red after 4. Scrapes run a day or two behind.
        </p>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Similarity index</h2>
        <div className={styles.stats}>
          <Stat
            label="Weaviate objects"
            value={
              vectors.objects === null ? "—" : formatNumber(vectors.objects)
            }
            detail={vectorDetail(vectors)}
          />
          <Stat label="Posts" value={formatNumber(vectors.posts)} />
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Database</h2>
        <div className={styles.twoCol}>
          <div className={styles.stats}>
            <Stat label="Total size" value={formatBytes(database.bytes)} />
            <Stat
              label="Migration"
              value={database.migration_version}
              detail={database.migration_dirty ? "Dirty" : "Clean"}
              className={database.migration_dirty ? styles.veryStale : ""}
            />
          </div>
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Table</th>
                  <th className={styles.num}>Size</th>
                </tr>
              </thead>
              <tbody>
                {database.tables.map((table) => (
                  <tr key={table.name}>
                    <td className={styles.mono}>{table.name}</td>
                    <td className={styles.num}>{formatBytes(table.bytes)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </>
  );
}
