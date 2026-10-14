"use client";

import { DisplayProvider, InitialDisplay } from "@hooks/useDisplay";
import usePosts from "@hooks/usePosts";
import {
  CatalogEntry,
  catalogHref,
  CatalogKind,
  catalogName,
} from "@lib/catalog";
import { EntryFact } from "@lib/catalogPage";
import { catalogParsers, PostsFilters } from "@lib/searchParams";
import { pickSeed } from "@lib/seed";
import { Button, SegmentedControl } from "@mantine/core";
import {
  IconAperture,
  IconPalette,
  IconPhoto,
  IconSortAscending,
  IconSortDescending,
} from "@tabler/icons-react";
import { ServerPostResponse } from "analogdb-generated";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import styles from "./catalogDetail.module.css";
import DisplayMenu from "./displayMenu";
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

type DetailLink = { href: string; label: string };

interface CatalogDetailProps {
  kind?: CatalogKind;
  entry: CatalogEntry;
  facts: EntryFact[];
  related?: CatalogEntry[];
  fixed?: PostsFilters;
  title?: string;
  titleLink?: DetailLink;
  relatedTitle?: string;
  relatedLinks?: DetailLink[];
  initialPage?: ServerPostResponse | null;
  initialFilters?: PostsFilters;
  initialDisplay?: InitialDisplay;
}

function sortData(ascending: boolean) {
  return [
    { label: ascending ? "lowest" : "top", value: "score" },
    { label: ascending ? "oldest" : "newest", value: "time" },
    { label: "random", value: "random" },
  ];
}

function orderLabel(sort: PostsFilters["sort"], ascending: boolean): string {
  if (sort === "time") return ascending ? "oldest first" : "newest first";
  if (sort === "score") return ascending ? "lowest first" : "highest first";
  return "order";
}

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
  initialPage,
  initialFilters,
  initialDisplay,
}: CatalogDetailProps) {
  const { filters, setFilters, limits, ...posts } = usePosts(
    initialPage,
    initialFilters,
    { parsers: catalogParsers, fixed }
  );
  const [expanded, setExpanded] = useState(false);
  const [overflowing, setOverflowing] = useState(false);
  const descriptionRef = useRef<HTMLParagraphElement>(null);

  const setSort = (sort: PostsFilters["sort"]) => {
    if (sort === "random") {
      setFilters({ sort: sort, order: null, seed: pickSeed() });
    } else {
      setFilters({ sort: sort, seed: null });
    }
  };

  const ascending = filters.order === "asc";
  const OrderIcon = ascending ? IconSortAscending : IconSortDescending;

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
    <DisplayProvider value={initialDisplay}>
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
          <KeywordRow
            links={links}
            label={relatedTitle ?? `more from ${entry.make.toUpperCase()}`}
            className={styles.relatedRow}
          />
          <div className={styles.sort}>
            <SegmentedControl
              value={filters.sort}
              onChange={setSort}
              data={sortData(ascending)}
              aria-label="sort"
            />
            <Button
              variant="outline"
              color="gray"
              className={styles.orderButton}
              onClick={() => setFilters({ order: ascending ? null : "asc" })}
              disabled={filters.sort === "random"}
              aria-label="reverse order"
              aria-pressed={ascending}
              title={orderLabel(filters.sort, ascending)}
            >
              <OrderIcon size={18} stroke={1.5} />
            </Button>
            <DisplayMenu
              buttonClassName={styles.displayButton}
              labelClassName={styles.displayLabel}
            />
          </div>
          <InfiniteGallery {...posts} />
          <ScrollTop />
        </div>
        <Footer />
      </div>
    </DisplayProvider>
  );
}
