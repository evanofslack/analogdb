"use client";

import {
  ActionResult,
  dismissReport,
  takedownReportedPost,
} from "@app/actions/admin";
import { Button, Tooltip } from "@mantine/core";
import { useRouter } from "next/navigation";
import { useState } from "react";
import styles from "./adminPanel.module.css";

interface DuplicateActionsProps {
  reportId: number;
  postId: number;
  removed: boolean;
}

// The side by side pair is the confirmation, so neither button asks again
export default function DuplicateActions({
  reportId,
  postId,
  removed,
}: DuplicateActionsProps) {
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

  return (
    <div className={styles.actions} title={error ?? undefined}>
      {!removed && (
        <Tooltip label="Take down the reported post" withArrow>
          <Button
            size="xs"
            color="red"
            variant="subtle"
            onClick={() =>
              act("takedown", () => takedownReportedPost(reportId, postId))
            }
            loading={pending === "takedown"}
            disabled={pending !== null}
          >
            Duplicate
          </Button>
        </Tooltip>
      )}
      <Button
        size="xs"
        variant="default"
        onClick={() => act("dismiss", () => dismissReport(reportId))}
        loading={pending === "dismiss"}
        disabled={pending !== null}
        color={error ? "red" : undefined}
      >
        {error ? "Retry" : "Not duplicate"}
      </Button>
    </div>
  );
}
