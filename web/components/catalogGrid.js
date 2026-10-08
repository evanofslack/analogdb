import CatalogCard from "./catalogCard";
import styles from "./catalogIndex.module.css";

export default function CatalogGrid({
  entries,
  hrefFor,
  labelFor,
  altFor,
  priorityCount = 0,
  offset = 0,
}) {
  return (
    <div className={styles.grid}>
      {entries.map((entry, i) => (
        <CatalogCard
          key={entry.slug}
          entry={entry}
          href={hrefFor(entry)}
          label={labelFor?.(entry)}
          alt={altFor?.(entry)}
          priority={offset + i < priorityCount}
        />
      ))}
    </div>
  );
}
