import { CatalogEntry } from "@lib/catalog";
import CatalogCard from "./catalogCard";
import styles from "./catalogIndex.module.css";

interface CatalogGridProps {
  entries: CatalogEntry[];
  hrefFor: (entry: CatalogEntry) => string;
  labelFor?: (entry: CatalogEntry) => string;
  altFor?: (entry: CatalogEntry) => string;
  priorityCount?: number;
  offset?: number;
}

export default function CatalogGrid({
  entries,
  hrefFor,
  labelFor,
  altFor,
  priorityCount = 0,
  offset = 0,
}: CatalogGridProps) {
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
