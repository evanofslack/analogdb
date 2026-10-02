"use client";

import { catalogHref, catalogName } from "@lib/catalog";
import { TextInput } from "@mantine/core";
import { IconSearch } from "@tabler/icons-react";
import { useState } from "react";
import CatalogCard from "./catalogCard";
import styles from "./catalogIndex.module.css";
import Footer from "./footer";
import Header from "./header";
import ScrollTop from "./scrollTop";

const priorityCount = 8;

function matches(entry, query) {
  return catalogName(entry).toLowerCase().includes(query);
}

export default function CatalogIndex({
  kind,
  title,
  intro,
  placeholder,
  groups,
}) {
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();

  const shown = groups
    .map((group) => ({
      ...group,
      entries: needle
        ? group.entries.filter((entry) => matches(entry, needle))
        : group.entries,
    }))
    .filter((group) => group.entries.length > 0);

  let index = 0;

  return (
    <div className={styles.main}>
      <Header compact />
      <div className={styles.margin}>
        <div className={styles.intro}>
          <h1 className={styles.title}>{title}</h1>
          <p className={styles.text}>{intro}</p>
          <TextInput
            className={styles.filter}
            value={query}
            onChange={(event) => setQuery(event.currentTarget.value)}
            placeholder={placeholder}
            aria-label={placeholder}
            leftSection={<IconSearch size={16} stroke={1.5} />}
          />
        </div>
        {shown.length === 0 && (
          <h3 className={styles.empty}>nothing matches :(</h3>
        )}
        {shown.map((group) => (
          <section key={group.label} className={styles.group}>
            <h2 className={styles.groupTitle}>{group.label}</h2>
            <div className={styles.grid}>
              {group.entries.map((entry) => (
                <CatalogCard
                  key={entry.slug}
                  entry={entry}
                  href={catalogHref(kind, entry)}
                  priority={index++ < priorityCount}
                />
              ))}
            </div>
          </section>
        ))}
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
