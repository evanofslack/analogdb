import Link from "next/link";
import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

export type Tone = "warn" | "bad";

export const toneClass: Record<Tone, string> = {
  warn: styles.stale,
  bad: styles.veryStale,
};

interface StatProps {
  label: ReactNode;
  value: ReactNode;
  detail?: ReactNode;
  tone?: Tone | null;
  href?: string;
}

export default function Stat({ label, value, detail, tone, href }: StatProps) {
  const body = (
    <>
      <div className={styles.statLabel}>{label}</div>
      <div className={`${styles.statValue} ${tone ? toneClass[tone] : ""}`}>
        {value}
      </div>
      {detail && <div className={styles.statDetail}>{detail}</div>}
    </>
  );
  if (href) {
    return (
      <Link href={href} className={styles.stat}>
        {body}
      </Link>
    );
  }
  return <div className={styles.stat}>{body}</div>;
}
