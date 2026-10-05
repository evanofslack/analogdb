"use client";

import { keywordHref, keywordLabel } from "@lib/keywords";
import { TextInput } from "@mantine/core";
import {
  IconClock,
  IconSearch,
  IconSparkles,
  IconTags,
} from "@tabler/icons-react";
import Link from "next/link";
import { useState } from "react";
import CatalogGrid from "./catalogGrid";
import catalogStyles from "./catalogIndex.module.css";
import styles from "./searchSuggestions.module.css";

const iconSize = 16;
const priorityCount = 4;

function Section({ icon, title, action, children }) {
  return (
    <section className={styles.section}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>
          {icon}
          {title}
        </h3>
        {action}
      </div>
      {children}
    </section>
  );
}

export function KeywordChips({ words, onSearch, className }) {
  if (!words || words.length === 0) return null;
  return (
    <div className={className ?? styles.chips}>
      {words.map((word) => (
        <button
          type="button"
          key={word}
          className={styles.chip}
          onClick={() => onSearch(word)}
        >
          {word}
        </button>
      ))}
    </div>
  );
}

export default function SearchSuggestions({
  suggestions,
  recent = [],
  onClearRecent,
  onSearch,
  catalog,
}) {
  const keywords = suggestions?.keywords ?? [];

  return (
    <div className={styles.suggestions}>
      {recent.length > 0 && (
        <Section
          icon={<IconClock size={iconSize} stroke={1.5} />}
          title="recent searches"
          action={
            onClearRecent && (
              <button
                type="button"
                className={styles.clear}
                onClick={onClearRecent}
              >
                clear
              </button>
            )
          }
        >
          <KeywordChips words={recent} onSearch={onSearch} />
        </Section>
      )}
      {keywords.length > 0 && (
        <Section
          icon={<IconSparkles size={iconSize} stroke={1.5} />}
          title="trending this week"
        >
          <KeywordChips words={keywords} onSearch={onSearch} />
        </Section>
      )}
      {catalog && <KeywordBrowser catalog={catalog} onSearch={onSearch} />}
    </div>
  );
}

export function KeywordBrowser({ catalog, onSearch }) {
  const [query, setQuery] = useState("");
  const text = query.trim();
  const needle = text.toLowerCase();

  if (!catalog || catalog.length === 0) return null;

  const shown = needle
    ? catalog.filter((entry) => entry.make.includes(needle))
    : catalog;

  return (
    <Section
      icon={<IconTags size={iconSize} stroke={1.5} />}
      title="browse by keyword"
    >
      <TextInput
        className={catalogStyles.filter}
        value={query}
        onChange={(event) => setQuery(event.currentTarget.value)}
        placeholder="filter keywords"
        aria-label="filter keywords"
        leftSection={<IconSearch size={16} stroke={1.5} />}
      />
      {shown.length > 0 ? (
        <div className={styles.keywordGrid}>
          <CatalogGrid
            entries={shown}
            hrefFor={(entry) => keywordHref(entry.make)}
            labelFor={(entry) => keywordLabel(entry.make)}
            altFor={(entry) =>
              `${entry.cover?.title || entry.make}, tagged ${entry.make}`
            }
            priorityCount={priorityCount}
          />
        </div>
      ) : (
        <p className={styles.noKeywords}>
          <Link
            href={`/search?q=${encodeURIComponent(text)}`}
            prefetch={false}
            className={styles.searchFor}
            onClick={(event) => {
              event.preventDefault();
              onSearch(text);
            }}
          >
            search for &ldquo;{text}&rdquo;
          </Link>
        </p>
      )}
    </Section>
  );
}
