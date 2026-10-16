import { formatNumber } from "@lib/format";
import { formatMonthYear, generatedDate, Stats, windowStart } from "@lib/stats";
import styles from "./methodSection.module.css";
import shared from "./stats.module.css";

export default function MethodSection({ stats }: { stats: Stats }) {
  const method = stats.method ?? {};
  const generated = generatedDate(stats);
  return (
    <section className={shared.section}>
      <details className={styles.details}>
        <summary className={styles.summary}>How this is measured ↓</summary>
        <ul className={styles.list}>
          <li>
            {formatNumber(stats.totals?.photos ?? 0)} photos in AnalogDB since{" "}
            {formatMonthYear(windowStart(stats))}. They are photos that did well
            when posted, not every photo shot on a film.
          </li>
          <li>
            Popularity is a photo&apos;s score rank within its calendar month,
            from 0 to 100. Scores drift over the years, so they are only
            compared within a month.
          </li>
          <li>
            Colors are measured from the scans, so a film&apos;s look here
            includes scanning and editing, not the bare emulsion.
          </li>
          <li>
            Film families, styles, typical photos and themes use image
            embeddings ({method.model ?? "OpenCLIP ViT-B/16"}). Themes are
            groups of similar photos, named by hand.
          </li>
          <li>
            Films need {method.minFilmPosts ?? 80} photos and cameras{" "}
            {method.minCameraPosts ?? 100} to be shown. In trends each
            photographer counts at most {method.authorCap ?? 5} photos per film
            or camera per period.
          </li>
          {generated && (
            <li>
              Built{" "}
              {generated.toLocaleDateString("en-US", {
                day: "numeric",
                month: "long",
                year: "numeric",
                timeZone: "UTC",
              })}
              .
            </li>
          )}
        </ul>
      </details>
    </section>
  );
}
