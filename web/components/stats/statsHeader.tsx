import { formatNumber } from "@lib/format";
import { formatMonthYear, Stats, windowStart } from "@lib/stats";
import NsfwSwitch from "./nsfwSwitch";
import styles from "./stats.module.css";

export default function StatsHeader({ stats }: { stats: Stats | null }) {
  const totals = stats?.totals;
  const since = stats ? formatMonthYear(windowStart(stats)) : "";
  const tiles = totals
    ? [
        { value: totals.photos, label: "photos" },
        { value: totals.photographers, label: "photographers" },
        { value: totals.films, label: "film stocks" },
        { value: totals.cameras, label: "cameras" },
      ]
    : [];

  return (
    <header className={styles.header}>
      <div className={styles.titleRow}>
        <h1 className={styles.title}>STATS</h1>
        {stats && (
          <div className={styles.switch}>
            <NsfwSwitch />
          </div>
        )}
      </div>
      {totals ? (
        <>
          <p className={styles.tagline}>
            What {formatNumber(totals.photos ?? 0)} photos say about film, gear
            and taste.
          </p>
          <ul className={styles.tiles}>
            {tiles.map((tile) => (
              <li key={tile.label} className={styles.tile}>
                <span className={styles.tileValue}>
                  {formatNumber(tile.value ?? 0)}
                </span>
                <span className={styles.tileLabel}>{tile.label}</span>
              </li>
            ))}
          </ul>
          {since && <p className={styles.since}>Since {since}</p>}
        </>
      ) : (
        <p className={styles.empty}>Stats are being built, check back soon.</p>
      )}
    </header>
  );
}
