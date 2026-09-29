import { Alert } from "@mantine/core";

export default function TabError({ error }) {
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
