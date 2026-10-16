import { Alert } from "@mantine/core";
import styles from "./adminPanel.module.css";

export interface LoadError {
  status: number;
  message?: string;
}

interface TabErrorProps {
  error: LoadError;
}

export default function TabError({ error }: TabErrorProps) {
  const status = error?.status ?? 500;
  const message = error?.message || "Something went wrong";
  if (status < 500) {
    return (
      <p className={styles.errorText}>
        {status} · {message}
      </p>
    );
  }
  const unavailable = status === 503;
  return (
    <Alert
      color={unavailable ? "yellow" : "red"}
      variant="light"
      title={unavailable ? "Unavailable" : `Failed to load (${status})`}
    >
      {message}
    </Alert>
  );
}
