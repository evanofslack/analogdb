import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

interface StatProps {
  label: ReactNode;
  value: ReactNode;
  detail?: ReactNode;
}

export default function Stat({ label, value, detail }: StatProps) {
  return (
    <div className={styles.card}>
      <div className={styles.statLabel}>{label}</div>
      <div className={styles.statValue}>{value}</div>
      {detail && <div className={styles.statDetail}>{detail}</div>}
    </div>
  );
}
