"use client";

import { ActionIcon, Tooltip } from "@mantine/core";
import { IconCameraSearch, IconSearch } from "@tabler/icons-react";
import Image from "next/image";
import { useEffect, useState } from "react";
import styles from "./searchBar.module.css";

export default function SearchBar({
  value,
  onSearch,
  onVisual,
  visualActive = false,
  onFocus,
  autoFocus = false,
  placeholder = "search photos...",
}) {
  const [text, setText] = useState(value ?? "");

  useEffect(() => {
    setText(value ?? "");
  }, [value]);

  const submit = (event) => {
    event.preventDefault();
    const query = text.trim();
    if (query) onSearch(query);
  };

  return (
    <form role="search" className={styles.bar} onSubmit={submit}>
      <IconSearch size={20} stroke={1.5} className={styles.icon} />
      <input
        type="search"
        className={styles.input}
        value={text}
        onChange={(event) => setText(event.currentTarget.value)}
        onFocus={onFocus}
        placeholder={placeholder}
        aria-label="search photos"
        enterKeyHint="search"
        autoComplete="off"
        autoFocus={autoFocus}
      />
      <Tooltip label="visual search" withArrow>
        <ActionIcon
          variant={visualActive ? "light" : "subtle"}
          color="gray"
          size="lg"
          onClick={onVisual}
          aria-label="visual search"
          aria-pressed={visualActive}
        >
          <IconCameraSearch size={20} stroke={1.5} />
        </ActionIcon>
      </Tooltip>
    </form>
  );
}

export function SearchBarButton({ label, thumb, onOpen, onVisual }) {
  return (
    <div className={styles.bar}>
      <button type="button" className={styles.open} onClick={onOpen}>
        <IconSearch size={20} stroke={1.5} className={styles.icon} />
        {thumb && (
          <span className={styles.thumb}>
            <Image
              src={thumb}
              alt=""
              fill
              sizes="40px"
              unoptimized
              style={{ objectFit: "cover" }}
            />
          </span>
        )}
        <span
          className={
            label ? styles.label : `${styles.label} ${styles.placeholder}`
          }
        >
          {label || "search photos..."}
        </span>
      </button>
      <Tooltip label="visual search" withArrow>
        <ActionIcon
          variant="subtle"
          color="gray"
          size="lg"
          onClick={onVisual}
          aria-label="visual search"
        >
          <IconCameraSearch size={20} stroke={1.5} />
        </ActionIcon>
      </Tooltip>
    </div>
  );
}
