"use client";

import { ActionIcon, CloseButton, Popover, Tooltip } from "@mantine/core";
import { IconCameraSearch, IconSearch } from "@tabler/icons-react";
import Image from "next/image";
import { useEffect, useState } from "react";
import styles from "./searchBar.module.css";
import SearchSuggestions from "./searchSuggestions";
import VisualSearch from "./visualSearch";

export default function SearchBar({
  value,
  onSearch,
  visual,
  suggestions,
  recent,
  onClearRecent,
  panel,
  onPanelChange,
  visualProps,
  suggestionsDropdown = true,
  placeholder = "search photos...",
}) {
  const [text, setText] = useState(value ?? "");

  useEffect(() => {
    setText(value ?? "");
  }, [value]);

  const opened =
    panel === "visual" ||
    (panel === "suggestions" && suggestionsDropdown && !text.trim() && !visual);

  const search = (query) => {
    onPanelChange(null);
    onSearch(query);
  };

  const submit = (event) => {
    event.preventDefault();
    const query = text.trim();
    if (query) search(query);
  };

  const showSuggestions = () => {
    if (panel !== "visual") onPanelChange("suggestions");
  };

  return (
    <Popover
      opened={opened}
      onChange={(next) => !next && onPanelChange(null)}
      width="target"
      position="bottom-start"
      offset={8}
      shadow="md"
      trapFocus={false}
      returnFocus={false}
    >
      <Popover.Target>
        <form role="search" className={styles.bar} onSubmit={submit}>
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
              <span className={styles.label}>{visual.label}</span>
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
              onFocus={showSuggestions}
              onClick={() => onPanelChange("suggestions")}
              onKeyDown={(event) =>
                event.key === "Escape" && onPanelChange(null)
              }
              placeholder={placeholder}
              aria-label="search photos"
              enterKeyHint="search"
              autoComplete="off"
            />
          )}
          <Tooltip label="visual search" withArrow>
            <ActionIcon
              variant={panel === "visual" ? "light" : "subtle"}
              color="gray"
              size="lg"
              onClick={() =>
                onPanelChange(panel === "visual" ? null : "visual")
              }
              aria-label="visual search"
              aria-expanded={panel === "visual"}
            >
              <IconCameraSearch size={20} stroke={1.5} />
            </ActionIcon>
          </Tooltip>
        </form>
      </Popover.Target>
      <Popover.Dropdown className={styles.dropdown}>
        {panel === "visual" ? (
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
