"use client";

import { reportPost } from "@app/actions/reports";
import { isLimited } from "@lib/rateLimited";
import { maxReportEmail, maxReportMessage, reportReasons } from "@lib/reports";
import {
  Alert,
  Button,
  Group,
  Modal,
  Radio,
  Stack,
  Text,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useState } from "react";
import styles from "./reportModal.module.css";

const empty = { reason: "", message: "", email: "", website: "" };

export default function ReportModal({ postId, opened, onClose }) {
  const [values, setValues] = useState(empty);
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState(null);

  const set = (key) => (event) =>
    setValues((v) => ({ ...v, [key]: event.currentTarget.value }));

  function close() {
    onClose();
    // reset after the close animation so the thank you doesn't flash away
    setTimeout(() => {
      setValues(empty);
      setSent(false);
      setError(null);
    }, 200);
  }

  async function submit(event) {
    event.preventDefault();
    if (!values.reason) {
      setError("Pick a reason");
      return;
    }
    setSending(true);
    setError(null);
    const result = await reportPost(postId, values);
    setSending(false);
    if (isLimited(result)) {
      setError("Too many reports, try again later");
      return;
    }
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setSent(true);
  }

  return (
    <Modal opened={opened} onClose={close} title="Report this post" size="md">
      {sent ? (
        <Stack gap="md">
          <Text>Thanks, the report was sent and will be reviewed.</Text>
          <Group justify="flex-end">
            <Button onClick={close}>Close</Button>
          </Group>
        </Stack>
      ) : (
        <form onSubmit={submit}>
          <Stack gap="md">
            <Radio.Group
              value={values.reason}
              onChange={(reason) => setValues((v) => ({ ...v, reason }))}
              label="What's wrong?"
            >
              <Stack gap="xs" mt="xs">
                {reportReasons.map((r) => (
                  <Radio key={r.value} value={r.value} label={r.label} />
                ))}
              </Stack>
            </Radio.Group>
            <Textarea
              label="Details"
              description={`Optional, ${values.message.length}/${maxReportMessage}`}
              autosize
              minRows={2}
              maxRows={6}
              maxLength={maxReportMessage}
              value={values.message}
              onChange={set("message")}
            />
            <TextInput
              type="email"
              label="Email"
              description={
                values.reason === "takedown"
                  ? "Optional, so we can confirm the takedown with you"
                  : "Optional, only if you'd like a reply"
              }
              maxLength={maxReportEmail}
              value={values.email}
              onChange={set("email")}
            />
            <input
              className={styles.honeypot}
              type="text"
              name="website"
              tabIndex={-1}
              autoComplete="off"
              aria-hidden="true"
              value={values.website}
              onChange={set("website")}
            />
            {error && (
              <Alert color="red" variant="light">
                {error}
              </Alert>
            )}
            <Group justify="flex-end">
              <Button variant="default" onClick={close} disabled={sending}>
                Cancel
              </Button>
              <Button type="submit" loading={sending}>
                Send report
              </Button>
            </Group>
          </Stack>
        </form>
      )}
    </Modal>
  );
}
