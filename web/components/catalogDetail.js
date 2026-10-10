"use client";

import usePosts from "@hooks/usePosts";
import { catalogHref, catalogName } from "@lib/catalog";
import { catalogParsers } from "@lib/searchParams";
import { pickSeed } from "@lib/seed";
import { SegmentedControl } from "@mantine/core";
import { IconAperture, IconPalette, IconPhoto } from "@tabler/icons-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import styles from "./catalogDetail.module.css";
import Footer from "./footer";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import KeywordRow from "./keywordRow";
import ScrollTop from "./scrollTop";

const factIcons = {
  photos: IconPhoto,
  speed: IconAperture,
  color: IconPalette,
};

const sortData = [
  { label: "top", value: "score" },
  { label: "newest", value: "time" },
  { label: "random", value: "random" },
];

export default function CatalogDetail({
  kind,
  entry,
  facts,
  related = [],
  fixed,
  title,
  titleLink,
  relatedTitle,
  relatedLinks,
  relatedRow = false,
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
  const [overflowing, setOverflowing] = useState(false);
  const descriptionRef = useRef(null);

  const setSort = (sort) => {
    if (sort === "random") {
      setFilters({ sort: sort, seed: pickSeed() });
    } else {
      setFilters({ sort: sort, seed: null });
    }
  };

  const description = entry.description?.trim();
  const heading = title ?? catalogName(entry);
  const links =
    relatedLinks ??
    related.map((other) => ({
      href: catalogHref(kind, other),
      label: catalogName(other),
    }));

  useEffect(() => {
    const element = descriptionRef.current;
    if (!element || expanded) return;
    const measure = () =>
      setOverflowing(element.scrollHeight > element.clientHeight + 1);
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [description, expanded]);

  return (
    <div className={styles.main}>
      <Header brandHeading={false} />
      <div className={styles.margin}>
        <div className={styles.intro}>
          {titleLink ? (
            <div className={styles.titleRow}>
              <h1 className={styles.title}>{heading}</h1>
              <Link
                href={titleLink.href}
                prefetch={false}
                className={styles.titleLink}
              >
                {titleLink.label}
              </Link>
            </div>
          ) : (
            <h1 className={styles.title}>{heading}</h1>
          )}
          <ul className={styles.facts}>
            {facts.map((fact) => {
              const Icon = factIcons[fact.icon];
              return (
                <li key={fact.label} className={styles.fact}>
                  <Icon size={16} className={styles.icon} />
                  {fact.label}
                </li>
              );
            })}
          </ul>
          {description && (
            <div className={styles.descriptionBox}>
              <p
                ref={descriptionRef}
                className={expanded ? styles.description : styles.clamped}
              >
                {description}
              </p>
              {(overflowing || expanded) && (
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
        {relatedRow && (
          <KeywordRow
            links={links}
            label={relatedTitle}
            className={styles.relatedRow}
          />
        )}
        {!relatedRow && links.length > 0 && (
          <section className={styles.related}>
            <h2 className={styles.relatedTitle}>
              {relatedTitle ?? `more from ${entry.make.toUpperCase()}`}
            </h2>
            <div className={styles.relatedLinks}>
              {links.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  prefetch={false}
                  className={styles.relatedLink}
                >
                  {link.label}
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
