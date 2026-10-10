import { catalogName } from "@lib/catalog";
import Link from "next/link";
import styles from "./catalogCard.module.css";
import ResponsiveImage from "./responsiveImage";

export default function CatalogCard({ entry, href, priority, label, alt }) {
  const name = label ?? catalogName(entry);
  return (
    <Link href={href} prefetch={false} className={styles.card}>
      <div className={styles.cover}>
        {entry.cover && (
          <ResponsiveImage
            src={entry.cover.url}
            phoneSrc={entry.cover.phoneUrl}
            alt={alt ?? `${entry.cover.title || name}, shot on ${name}`}
            fill
            sizes="(max-width: 720px) 50vw, (max-width: 1024px) 33vw, (max-width: 1440px) 25vw, 20vw"
            loading={priority ? "eager" : "lazy"}
            fetchPriority={priority ? "high" : undefined}
            style={{ objectFit: "cover" }}
          />
        )}
      </div>
      <div className={styles.name}>{name}</div>
      <div className={styles.count}>
        {entry.postCount.toLocaleString("en-US")} photos
      </div>
    </Link>
  );
}
