"use client";

import { SearchDemo, shuffle } from "@lib/about";
import { IconSearch } from "@tabler/icons-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import AboutPhoto from "./aboutPhoto";
import styles from "./aboutSearch.module.css";
import KeywordRow from "./keywordRow";

const TYPE_MS = 50;
const HOLD_MS = 6000;

export default function AboutSearch({ searches }: { searches: SearchDemo[] }) {
  const [order, setOrder] = useState<SearchDemo[] | null>(null);
  const [index, setIndex] = useState(0);
  const [typed, setTyped] = useState(0);
  const [reduced, setReduced] = useState(false);
  const [visible, setVisible] = useState(false);
  const [hovered, setHovered] = useState(false);
  const [shownIndex, setShownIndex] = useState<number | null>(null);
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    setOrder(shuffle(searches));
    setReduced(window.matchMedia("(prefers-reduced-motion: reduce)").matches);
  }, [searches]);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    const observer = new IntersectionObserver(([entry]) =>
      setVisible(entry.isIntersecting)
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [order]);

  const current = order?.[index];
  const length = current?.query.length ?? 0;
  const done = Boolean(current) && (reduced || typed >= length);

  // the last finished query stays on screen while the next one types
  useEffect(() => {
    if (done) setShownIndex(index);
  }, [done, index]);

  // type the query, hold the results, then move on, only while on screen
  useEffect(() => {
    if (!order || !current || reduced || !visible || hovered) return;
    const timer = done
      ? setTimeout(() => {
          setIndex((i) => (i + 1) % order.length);
          setTyped(0);
        }, HOLD_MS)
      : setTimeout(() => setTyped((n) => n + 1), TYPE_MS);
    return () => clearTimeout(timer);
  }, [order, current, reduced, visible, hovered, done, typed]);

  if (!order || !current) return null;

  const text = reduced ? current.query : current.query.slice(0, typed);
  const shown = shownIndex === null ? null : order[shownIndex];
  const links = (shown?.keywords ?? []).map((word) => ({
    label: word,
    href: `/search/keyword/${encodeURIComponent(word)}`,
  }));

  return (
    <div
      ref={ref}
      className={styles.demo}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Link
        href={`/search?q=${encodeURIComponent(current.query)}`}
        prefetch={false}
        className={styles.bar}
        aria-label={`search for ${current.query}`}
      >
        <IconSearch size={18} stroke={1.5} className={styles.icon} />
        <span className={styles.query}>
          {text}
          {!reduced && <span className={styles.caret} aria-hidden />}
        </span>
      </Link>
      {shown ? (
        <div key={shown.query} className={`${styles.results} ${styles.fadeIn}`}>
          {shown.photos.map((photo) => (
            <AboutPhoto
              key={photo.id}
              photo={photo}
              small
              fill
              sizes="(max-width: 720px) 33vw, 180px"
              className={styles.tile}
            />
          ))}
        </div>
      ) : (
        <div className={styles.results} aria-hidden>
          {current.photos.map((photo) => (
            <div key={photo.id} className={`${styles.tile} ${styles.empty}`} />
          ))}
        </div>
      )}
      <div
        key={`${shown?.query}-keywords`}
        className={`${styles.keywords} ${shown ? styles.fadeIn : ""}`}
      >
        {shown && (
          <KeywordRow
            links={links}
            label="related keywords"
            words={undefined}
            onSelect={undefined}
            className={undefined}
          />
        )}
      </div>
    </div>
  );
}
