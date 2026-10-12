import { Alert } from "@mantine/core";

export interface LoadError {
  status: number;
  message?: string;
}

interface TabErrorProps {
  error: LoadError;
}

export default function TabError({ error }: TabErrorProps) {
  const unavailable = error?.status === 503;
  return (
    <Alert
      color={unavailable ? "yellow" : "red"}
      variant="light"
      title={unavailable ? "Unavailable" : "Failed to load"}
    >
      {error?.message || "Something went wrong"}
    </Alert>
  );
}
