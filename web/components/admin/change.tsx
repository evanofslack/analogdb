import styles from "./adminPanel.module.css";

interface ChangeProps {
  current: number;
  previous: number;
  invert?: boolean;
}

export default function Change({ current, previous, invert }: ChangeProps) {
  if (!previous) {
    return current ? <span className={styles.change}>new</span> : null;
  }
  const pct = ((current - previous) / previous) * 100;
  const rounded = Math.round(pct);
  const bad = invert ? pct > 10 : pct < -10;
  const text =
    rounded === 0
      ? "±0%"
      : rounded > 0
      ? `+${rounded}%`
      : `−${Math.abs(rounded)}%`;
  return (
    <span className={`${styles.change} ${bad ? styles.stale : ""}`}>
      {text}
    </span>
  );
}
