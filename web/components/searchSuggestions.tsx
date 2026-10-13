"use client";

import type { SearchOrigin } from "@lib/analytics";
import { CatalogEntry } from "@lib/catalog";
import type { SearchSuggestions as Suggestions } from "@lib/data/search";
import { keywordHref, keywordLabel } from "@lib/keywords";
import { IconClock, IconSparkles, IconTags } from "@tabler/icons-react";
import { ReactNode } from "react";
import CatalogGrid from "./catalogGrid";
import KeywordRow from "./keywordRow";
import styles from "./searchSuggestions.module.css";

const iconSize = 16;
const priorityCount = 4;

interface SectionProps {
  icon: ReactNode;
  title: string;
  action?: ReactNode;
  children: ReactNode;
}

function Section({ icon, title, action, children }: SectionProps) {
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

interface KeywordChipsProps {
  words?: string[];
  onSearch: (word: string) => void;
  className?: string;
}

export function KeywordChips({
  words,
  onSearch,
  className,
}: KeywordChipsProps) {
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

interface SearchSuggestionsProps {
  suggestions?: Suggestions | null;
  recent?: string[];
  onClearRecent?: () => void;
  onSearch: (query: string, source?: SearchOrigin) => void;
  catalog?: CatalogEntry[] | null;
  trendingRow?: boolean;
}

export default function SearchSuggestions({
  suggestions,
  recent = [],
  onClearRecent,
  onSearch,
  catalog,
  trendingRow = false,
}: SearchSuggestionsProps) {
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
          <KeywordChips
            words={recent}
            onSearch={(word) => onSearch(word, "recent")}
          />
        </Section>
      )}
      {keywords.length > 0 && (
        <Section
          icon={<IconSparkles size={iconSize} stroke={1.5} />}
          title="trending this month"
        >
          {trendingRow ? (
            <KeywordRow
              words={keywords}
              onSelect={(word) => onSearch(word, "chip")}
              label="trending this month"
            />
          ) : (
            <KeywordChips
              words={keywords}
              onSearch={(word) => onSearch(word, "suggestion")}
            />
          )}
        </Section>
      )}
      {catalog && <KeywordBrowser catalog={catalog} />}
    </div>
  );
}

interface KeywordBrowserProps {
  catalog?: CatalogEntry[] | null;
}

function KeywordBrowser({ catalog }: KeywordBrowserProps) {
  if (!catalog || catalog.length === 0) return null;

  return (
    <Section icon={<IconTags size={iconSize} stroke={1.5} />} title="keywords">
      <CatalogGrid
        entries={catalog}
        hrefFor={(entry: CatalogEntry) => keywordHref(entry.make)}
        labelFor={(entry: CatalogEntry) => keywordLabel(entry.make)}
        altFor={(entry: CatalogEntry) =>
          `${entry.cover?.title || entry.make}, tagged ${entry.make}`
        }
        priorityCount={priorityCount}
      />
    </Section>
  );
}
