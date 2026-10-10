import styles from "./adminPanel.module.css";

export default function Stat({ label, value, detail }) {
  return (
    <div className={styles.card}>
      <div className={styles.statLabel}>{label}</div>
      <div className={styles.statValue}>{value}</div>
      {detail && <div className={styles.statDetail}>{detail}</div>}
    </div>
  );
}
