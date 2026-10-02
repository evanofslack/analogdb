"use client";

import usePosts from "@hooks/usePosts";
import { catalogHref, catalogName } from "@lib/catalog";
import { catalogParsers } from "@lib/searchParams";
import { pickSeed } from "@lib/seed";
import { SegmentedControl } from "@mantine/core";
import Link from "next/link";
import { useState } from "react";
import styles from "./catalogDetail.module.css";
import Footer from "./footer";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import ScrollTop from "./scrollTop";

const longDescription = 320;

const sortData = [
  { label: "top", value: "score" },
  { label: "newest", value: "time" },
  { label: "random", value: "random" },
];

export default function CatalogDetail({
  kind,
  entry,
  facts,
  related,
  fixed,
  initialPage,
  initialFilters,
  initialColumns,
}) {
  const { filters, setFilters, limits, ...posts } = usePosts(
    initialPage,
    initialFilters,
    { parsers: catalogParsers, fixed }
  );
  const [expanded, setExpanded] = useState(false);

  const setSort = (sort) => {
    if (sort === "random") {
      setFilters({ sort: sort, seed: pickSeed() });
    } else {
      setFilters({ sort: sort, seed: null });
    }
  };

  const description = entry.description?.trim();
  const isLong = description?.length > longDescription;

  return (
    <div className={styles.main}>
      <Header compact />
      <div className={styles.margin}>
        <div className={styles.intro}>
          <nav className={styles.crumbs} aria-label="breadcrumb">
            <Link href={`/${kind}`}>
              {kind === "films" ? "FILM" : "CAMERAS"}
            </Link>
          </nav>
          <h1 className={styles.title}>{catalogName(entry)}</h1>
          <ul className={styles.facts}>
            {facts.map((fact) => (
              <li key={fact}>{fact}</li>
            ))}
          </ul>
          {description && (
            <div className={styles.descriptionBox}>
              <p
                className={
                  isLong && !expanded ? styles.clamped : styles.description
                }
              >
                {description}
              </p>
              {isLong && (
                <button
                  type="button"
                  className={styles.more}
                  onClick={() => setExpanded((value) => !value)}
                >
                  {expanded ? "less" : "more"}
                </button>
              )}
            </div>
          )}
        </div>
        {related.length > 0 && (
          <section className={styles.related}>
            <h2 className={styles.relatedTitle}>
              more from {entry.make.toUpperCase()}
            </h2>
            <div className={styles.relatedLinks}>
              {related.map((other) => (
                <Link
                  key={other.slug}
                  href={catalogHref(kind, other)}
                  prefetch={false}
                  className={styles.relatedLink}
                >
                  {catalogName(other)}
                </Link>
              ))}
            </div>
          </section>
        )}
        <div className={styles.sort}>
          <SegmentedControl
            value={filters.sort}
            onChange={setSort}
            data={sortData}
            aria-label="sort"
          />
        </div>
        <InfiniteGallery {...posts} initialColumns={initialColumns} />
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
