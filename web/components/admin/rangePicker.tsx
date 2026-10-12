import { TrafficRange, trafficRanges } from "@lib/adminClient";
import Link from "next/link";
import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

interface RangePickerProps {
  tab: string;
  range: TrafficRange;
  children?: ReactNode;
}

export default function RangePicker({
  tab,
  range,
  children,
}: RangePickerProps) {
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
