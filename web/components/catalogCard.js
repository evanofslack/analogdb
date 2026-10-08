import { catalogName } from "@lib/catalog";
import Image from "next/image";
import Link from "next/link";
import styles from "./catalogCard.module.css";

export default function CatalogCard({ entry, href, priority, label, alt }) {
  const name = label ?? catalogName(entry);
  return (
    <Link href={href} prefetch={false} className={styles.card}>
      <div className={styles.cover}>
        {entry.cover && (
          <Image
            src={entry.cover.url}
            alt={alt ?? `${entry.cover.title || name}, shot on ${name}`}
            fill
            sizes="(max-width: 720px) 50vw, (max-width: 1024px) 33vw, (max-width: 1440px) 25vw, 20vw"
            priority={priority}
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
