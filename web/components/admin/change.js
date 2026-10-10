import styles from "./adminPanel.module.css";

export default function Change({ current, previous, invert }) {
  if (!previous) {
    return current ? <span>New this period</span> : null;
  }
  const pct = ((current - previous) / previous) * 100;
  const good = invert ? pct < 0 : pct > 0;
  const className = pct === 0 ? "" : good ? styles.fresh : styles.veryStale;
  return (
    <span className={className}>
      {pct > 0 ? "+" : ""}
      {pct.toFixed(1)}% vs previous
    </span>
  );
}
