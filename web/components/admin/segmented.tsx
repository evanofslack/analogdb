import Link from "next/link";
import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

export interface SegmentedItem {
  key: string;
  href: string;
  label: ReactNode;
  count?: number;
}

interface SegmentedProps {
  items: SegmentedItem[];
  active: string;
}

export default function Segmented({ items, active }: SegmentedProps) {
  return (
    <div className={styles.segmented}>
      {items.map((item) => (
        <Link
          key={item.key}
          href={item.href}
          className={styles.segment}
          aria-current={item.key === active ? "page" : undefined}
          {...(item.key === active ? { "data-active": true } : {})}
        >
          {item.label}
          {item.count !== undefined && (
            <span className={styles.segmentCount}>{item.count}</span>
          )}
          <span className={styles.segmentDot} aria-hidden="true" />
        </Link>
      ))}
    </div>
  );
}
