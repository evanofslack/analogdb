"use client";

import {
  ActionResult,
  dismissReport,
  takedownReportedPost,
} from "@app/actions/admin";
import { Button, Group } from "@mantine/core";
import { useRouter } from "next/navigation";
import { useState } from "react";
import EditPostButton from "./editPostButton";

interface ReportActionsProps {
  reportId: number;
  postId: number;
  removed: boolean;
  resolved: boolean;
}

export default function ReportActions({
  reportId,
  postId,
  removed,
  resolved,
}: ReportActionsProps) {
  const router = useRouter();
  const [pending, setPending] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function act(name: string, action: () => Promise<ActionResult>) {
    setPending(name);
    setError(null);
    const result = await action();
    setPending(null);
    if (result.ok === false) {
      setError(result.error);
      return;
    }
    router.refresh();
  }

  function takedown() {
    if (!confirm(`Take down post ${postId}? This deletes it from AnalogDB.`)) {
      return;
    }
    act("takedown", () => takedownReportedPost(reportId, postId));
  }

  return (
    <Group gap="xs" wrap="nowrap" title={error ?? undefined}>
      {!removed && <EditPostButton id={postId} />}
      {!removed && (
        <Button
          size="xs"
          color="red"
          variant="light"
          onClick={takedown}
          loading={pending === "takedown"}
          disabled={pending !== null}
        >
          Take down
        </Button>
      )}
      {!resolved && (
        <Button
          size="xs"
          variant="default"
          onClick={() => act("dismiss", () => dismissReport(reportId))}
          loading={pending === "dismiss"}
          disabled={pending !== null}
          color={error ? "red" : undefined}
        >
          {error ? "Retry" : "Dismiss"}
        </Button>
      )}
    </Group>
  );
}
