"use client";

import { IconClock, IconCompass, IconSparkles } from "@tabler/icons-react";
import Image from "next/image";
import styles from "./searchSuggestions.module.css";

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
  layout = "dropdown",
}) {
  const keywords = suggestions?.keywords ?? [];
  const topics = suggestions?.topics ?? [];
  const iconSize = 16;

  return (
    <div className={layout === "page" ? styles.page : styles.dropdown}>
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
          title="new this week"
        >
          <KeywordChips words={keywords} onSearch={onSearch} />
        </Section>
      )}
      {topics.length > 0 && (
        <Section
          icon={<IconCompass size={iconSize} stroke={1.5} />}
          title="explore"
        >
          <div className={styles.topics}>
            {topics.map((topic) => (
              <button
                type="button"
                key={topic.word}
                className={styles.topic}
                onClick={() => onSearch(topic.word)}
              >
                <span
                  className={styles.cover}
                  style={{ backgroundColor: topic.cover?.hex }}
                >
                  {topic.cover && (
                    <Image
                      src={topic.cover.url}
                      alt=""
                      fill
                      sizes="160px"
                      style={{ objectFit: "cover" }}
                    />
                  )}
                </span>
                <span className={styles.topicLabel}>{topic.word}</span>
              </button>
            ))}
          </div>
        </Section>
      )}
    </div>
  );
}
