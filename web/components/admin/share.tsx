import styles from "./adminPanel.module.css";

interface ShareProps {
  value: number;
  total: number;
}

export default function Share({ value, total }: ShareProps) {
  const pct = total > 0 ? Math.min(100, (value / total) * 100) : 0;
  return (
    <span className={styles.share} aria-hidden="true">
      <span style={{ width: `${pct}%` }} />
    </span>
  );
}
