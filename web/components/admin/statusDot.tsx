import { DependencyStatus } from "@lib/adminClient";
import styles from "./adminPanel.module.css";

interface StatusDotProps {
  label: string;
  status?: DependencyStatus;
}

export default function StatusDot({ label, status }: StatusDotProps) {
  const state = status ?? "unknown";
  const down = state !== "ok" && state !== "disabled";
  return (
    <span
      className={`${styles.statusDot} ${down ? styles.veryStale : ""}`}
      data-status={state}
    >
      {label}
      {state === "disabled" && " disabled"}
      {down && ` ${state}`}
    </span>
  );
}
