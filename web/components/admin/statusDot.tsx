import { DependencyStatus } from "@lib/adminClient";
import styles from "./adminPanel.module.css";

interface StatusDotProps {
  label: string;
  status?: DependencyStatus;
}

export default function StatusDot({ label, status }: StatusDotProps) {
  const down = status !== "ok" && status !== "disabled";
  return (
    <span
      className={`${styles.statusDot} ${down ? styles.veryStale : ""}`}
      data-status={status ?? "unknown"}
    >
      {label}
      {status === "disabled" && " off"}
      {down && ` ${status ?? "unknown"}`}
    </span>
  );
}
