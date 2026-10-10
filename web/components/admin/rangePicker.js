import { trafficRanges } from "@lib/adminClient";
import Link from "next/link";
import styles from "./adminPanel.module.css";

export default function RangePicker({ tab, range, children }) {
  return (
    <div className={styles.controls}>
      {trafficRanges.map((r) => (
        <Link
          key={r}
          href={`/admin?tab=${tab}&range=${r}`}
          className={`${styles.chip} ${r === range ? styles.chipActive : ""}`}
        >
          {r}
        </Link>
      ))}
      {children}
    </div>
  );
}
