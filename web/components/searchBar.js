"use client";

import { ActionIcon, CloseButton, Popover, Tooltip } from "@mantine/core";
import { useUncontrolled } from "@mantine/hooks";
import { IconCameraSearch, IconSearch } from "@tabler/icons-react";
import Image from "next/image";
import { useEffect, useState } from "react";
import styles from "./searchBar.module.css";
import SearchSuggestions from "./searchSuggestions";
import VisualSearch from "./visualSearch";

export default function SearchBar({
  value,
  onSearch,
  onVisual,
  visual,
  suggestions,
  recent,
  onClearRecent,
  panel,
  onPanelChange,
  visualProps,
  dropdown = true,
  size = "md",
  autoFocus = false,
  placeholder = "search photos...",
}) {
  const [text, setText] = useState(value ?? "");
  const [current, setPanel] = useUncontrolled({
    value: panel,
    defaultValue: null,
    finalValue: null,
    onChange: onPanelChange,
  });

  useEffect(() => {
    setText(value ?? "");
  }, [value]);

  const opened =
    dropdown &&
    (current === "visual" ||
      (current === "suggestions" && !text.trim() && !visual));

  const search = (query) => {
    setPanel(null);
    onSearch(query);
  };

  const submit = (event) => {
    event.preventDefault();
    const query = text.trim();
    if (query) search(query);
  };

  const toggleVisual = () => {
    if (onVisual) {
      onVisual();
      return;
    }
    setPanel(current === "visual" ? null : "visual");
  };

  return (
    <Popover
      opened={opened}
      onChange={(next) => !next && setPanel(null)}
      width="target"
      position="bottom"
      offset={8}
      shadow="md"
      radius="md"
      trapFocus={false}
      returnFocus={false}
    >
      <Popover.Target>
        <form
          role="search"
          className={
            size === "lg" ? `${styles.bar} ${styles.large}` : styles.bar
          }
          onSubmit={submit}
        >
          <IconSearch size={20} stroke={1.5} className={styles.icon} />
          {visual ? (
            <div className={styles.visual}>
              {visual.thumb && (
                <span className={styles.thumb}>
                  <Image
                    src={visual.thumb}
                    alt=""
                    fill
                    sizes="40px"
                    unoptimized
                    style={{ objectFit: "cover" }}
                  />
                </span>
              )}
              <span className={styles.visualLabel}>{visual.label}</span>
              <CloseButton
                size="sm"
                onClick={visual.onClear}
                aria-label="clear visual search"
              />
            </div>
          ) : (
            <input
              type="search"
              className={styles.input}
              value={text}
              onChange={(event) => setText(event.currentTarget.value)}
              onFocus={() => current !== "visual" && setPanel("suggestions")}
              onClick={() => current !== "visual" && setPanel("suggestions")}
              onKeyDown={(event) => event.key === "Escape" && setPanel(null)}
              placeholder={placeholder}
              aria-label="search photos"
              enterKeyHint="search"
              autoComplete="off"
              autoFocus={autoFocus}
            />
          )}
          <Tooltip label="visual search" withArrow>
            <ActionIcon
              variant={current === "visual" ? "light" : "subtle"}
              color="gray"
              size="lg"
              radius="xl"
              onClick={toggleVisual}
              aria-label="visual search"
              aria-expanded={onVisual ? undefined : current === "visual"}
            >
              <IconCameraSearch size={20} stroke={1.5} />
            </ActionIcon>
          </Tooltip>
        </form>
      </Popover.Target>
      <Popover.Dropdown className={styles.dropdown}>
        {current === "visual" ? (
          <VisualSearch examples={suggestions?.examples} {...visualProps} />
        ) : (
          <SearchSuggestions
            suggestions={suggestions}
            recent={recent}
            onClearRecent={onClearRecent}
            onSearch={search}
          />
        )}
      </Popover.Dropdown>
    </Popover>
  );
}
