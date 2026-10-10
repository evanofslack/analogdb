"use client";

import { ActionIcon, Button, CloseButton, Code, Popover } from "@mantine/core";
import { useClipboard } from "@mantine/hooks";
import {
  IconCheck,
  IconCopy,
  IconInfoCircle,
  IconX,
} from "@tabler/icons-react";
import Link from "next/link";
import styles from "./filterSummary.module.css";

export default function FilterSummary({
  count,
  pending,
  onClear,
  items,
  apiUrl,
}) {
  const clipboard = useClipboard({ timeout: 1500 });

  return (
    <div className={styles.summary}>
      <span
        className={`${styles.count} ${pending ? styles.pending : ""}`}
        aria-live="polite"
      >
        {count != null &&
          `${count.toLocaleString("en-US")} ${
            count === 1 ? "photo" : "photos"
          }`}
      </span>
      <Popover width={320} position="bottom-start" shadow="md" withArrow>
        <Popover.Target>
          <ActionIcon
            variant="subtle"
            color="gray"
            size="md"
            className={styles.info}
            aria-label="Filter details"
          >
            <IconInfoCircle size={18} stroke={1.5} />
          </ActionIcon>
        </Popover.Target>
        <Popover.Dropdown className={styles.dropdown}>
          <p className={styles.heading}>active filters</p>
          <ul className={styles.items}>
            {items.map((item) => (
              <li key={item.key} className={styles.item}>
                <span>
                  <span className={styles.category}>{item.category}</span>{" "}
                  {item.value}
                </span>
                <CloseButton
                  size="sm"
                  onClick={item.onRemove}
                  aria-label={`Remove ${item.category} ${item.value}`}
                />
              </li>
            ))}
          </ul>
          <p className={styles.heading}>api call</p>
          <Code block className={styles.url}>
            {apiUrl}
          </Code>
          <div className={styles.actions}>
            <Button
              variant="subtle"
              color="gray"
              size="compact-sm"
              leftSection={
                clipboard.copied ? (
                  <IconCheck size={14} stroke={1.75} />
                ) : (
                  <IconCopy size={14} stroke={1.75} />
                )
              }
              className={styles.copy}
              onClick={() => clipboard.copy(apiUrl)}
            >
              {clipboard.copied ? "copied" : "copy api url"}
            </Button>
            <Link href="/docs" className={styles.docs}>
              api docs
            </Link>
          </div>
        </Popover.Dropdown>
      </Popover>
      <Button
        variant="subtle"
        color="gray"
        size="compact-sm"
        leftSection={<IconX size={14} stroke={1.75} />}
        className={styles.clear}
        onClick={onClear}
        aria-label="Clear filters"
      >
        clear
      </Button>
    </div>
  );
}
