"use client";

import { catalogHref, catalogName } from "@lib/catalog";
import { catalogFilterParsers } from "@lib/searchParams";
import { TextInput } from "@mantine/core";
import { IconSearch } from "@tabler/icons-react";
import { useQueryStates } from "nuqs";
import CatalogGrid from "./catalogGrid";
import styles from "./catalogIndex.module.css";
import Footer from "./footer";
import Header from "./header";
import ScrollTop from "./scrollTop";

const priorityCount = 8;

function matches(entry, query) {
  return catalogName(entry).toLowerCase().includes(query);
}

export default function CatalogIndex({ kind, intro, placeholder, groups }) {
  const [{ q: query }, setFilter] = useQueryStates(catalogFilterParsers);
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
      <Header brandHeading={false} />
      <div className={styles.margin}>
        <div className={styles.intro}>
          <h1 className={styles.text}>{intro}</h1>
          <TextInput
            className={styles.filter}
            value={query}
            onChange={(event) =>
              setFilter({ q: event.currentTarget.value || null })
            }
            placeholder={placeholder}
            aria-label={placeholder}
            leftSection={<IconSearch size={16} stroke={1.5} />}
          />
        </div>
        {shown.length === 0 && (
          <h3 className={styles.empty}>nothing matches :(</h3>
        )}
        {shown.map((group) => {
          const offset = index;
          index += group.entries.length;
          return (
            <section key={group.label} className={styles.group}>
              <h2 className={styles.groupTitle}>{group.label}</h2>
              <CatalogGrid
                entries={group.entries}
                hrefFor={(entry) => catalogHref(kind, entry)}
                priorityCount={priorityCount}
                offset={offset}
              />
            </section>
          );
        })}
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
