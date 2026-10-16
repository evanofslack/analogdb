import { AdminOverview, AdminQueue } from "@lib/adminClient";
import {
  formatAgo,
  formatBytes,
  formatDuration,
  formatNumber,
} from "@lib/format";
import styles from "./adminPanel.module.css";
import Hint from "./hint";
import Section from "./section";
import Stat, { Tone } from "./stat";
import StatusDot from "./statusDot";

interface OverviewTabProps {
  overview: AdminOverview;
  queue?: AdminQueue;
  now: number;
}

const postedTip = "By Reddit post date, not the date the post was added.";

function count(value?: number) {
  return value === undefined ? "—" : formatNumber(value);
}

const dependencies = ["postgres", "redis", "weaviate", "clickhouse"];

const day = 24 * 60 * 60;

function freshnessTone(unixSeconds: number | null, now: number): Tone | null {
  if (!unixSeconds) return "bad";
  const age = now / 1000 - unixSeconds;
  if (age > 4 * day) return "bad";
  if (age > 2 * day) return "warn";
  return null;
}

function vectorDetail({ objects, posts }: AdminOverview["vectors"]) {
  if (objects === null) return "could not count";
  const diff = objects - posts;
  if (diff === 0) return null;
  if (diff > 0) return `${formatNumber(diff)} extra objects`;
  return `${formatNumber(-diff)} posts missing`;
}

export default function OverviewTab({
  overview,
  queue,
  now,
}: OverviewTabProps) {
  const { status, app, counts, posted, freshness, database, vectors } =
    overview;

  const freshnessItems = [
    { label: "Newest post", value: freshness.newest_post },
    { label: "Scores", value: freshness.score_update },
    { label: "Keywords", value: freshness.keywords_update },
    { label: "Colors", value: freshness.colors_update },
  ];
  const vectorsOff =
    vectors.objects === null || vectors.objects !== vectors.posts;

  return (
    <>
      <Section title="Status">
        <div className={styles.statusRow}>
          {dependencies.map((name) => (
            <StatusDot key={name} label={name} status={status[name]} />
          ))}
          <span className={styles.appInfo}>
            {app.version} · {app.env} · {app.hostname} · up{" "}
            {formatDuration(app.uptime_seconds)}
          </span>
        </div>
      </Section>

      <Section title="To do">
        <div className={styles.stats}>
          <Stat
            label="Open reports"
            value={count(queue?.open_reports)}
            href="/admin?tab=moderation"
          />
          <Stat
            label="Unmatched cameras"
            value={count(queue?.unmatched_cameras)}
            href="/admin?tab=catalog"
          />
          <Stat
            label="Unmatched films"
            value={count(queue?.unmatched_films)}
            href="/admin?tab=catalog"
          />
        </div>
      </Section>

      <Section
        title={
          <Hint
            label="Freshness"
            tip="Amber after 2 days, red after 4. Scrapes run a day or two behind."
          />
        }
      >
        <div className={styles.stats}>
          {freshnessItems.map((item) => (
            <Stat
              key={item.label}
              label={item.label}
              value={formatAgo(item.value, now)}
              tone={freshnessTone(item.value, now)}
            />
          ))}
        </div>
      </Section>

      <Section title="Collection">
        <div className={styles.stats}>
          <Stat label="Posts" value={formatNumber(counts.posts)} />
          <Stat label="Authors" value={formatNumber(counts.authors)} />
          <Stat label="Cameras" value={formatNumber(counts.cameras)} />
          <Stat label="Films" value={formatNumber(counts.films)} />
          <Stat label="Keywords" value={formatNumber(counts.keywords)} />
        </div>
        <div className={`${styles.stats} ${styles.statsRow}`}>
          <Stat
            label={<Hint label="Posted 24h" tip={postedTip} />}
            value={formatNumber(posted.day)}
          />
          <Stat
            label={<Hint label="Posted 7d" tip={postedTip} />}
            value={formatNumber(posted.week)}
          />
          <Stat
            label={<Hint label="Posted 30d" tip={postedTip} />}
            value={formatNumber(posted.month)}
          />
        </div>
      </Section>

      <Section title="Similarity index">
        <div className={styles.stats}>
          <Stat
            label="Objects"
            value={
              vectors.objects === null ? "—" : formatNumber(vectors.objects)
            }
            detail={vectorDetail(vectors)}
            tone={vectorsOff ? "warn" : null}
          />
          <Stat label="Posts" value={formatNumber(vectors.posts)} />
        </div>
      </Section>

      <Section title="Database">
        <div className={styles.twoCol}>
          <div className={styles.stats}>
            <Stat label="Total size" value={formatBytes(database.bytes)} />
            <Stat
              label="Migration"
              value={database.migration_version}
              detail={database.migration_dirty ? "dirty" : "clean"}
              tone={database.migration_dirty ? "bad" : null}
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
      </Section>
    </>
  );
}
