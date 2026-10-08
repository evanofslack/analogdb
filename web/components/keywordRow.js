"use client";

import { ActionIcon } from "@mantine/core";
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import styles from "./keywordRow.module.css";

const step = 2;

export default function KeywordRow({
  words,
  links,
  onSelect,
  label,
  className,
}) {
  const rowRef = useRef(null);
  const [canLeft, setCanLeft] = useState(false);
  const [canRight, setCanRight] = useState(false);

  const measure = useCallback(() => {
    const row = rowRef.current;
    if (!row) return;
    setCanLeft(row.scrollLeft > 1);
    setCanRight(row.scrollLeft + row.clientWidth < row.scrollWidth - 1);
  }, []);

  useEffect(() => {
    const row = rowRef.current;
    if (!row) return;
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(row);
    return () => observer.disconnect();
  }, [measure, words, links]);

  const items = links ?? (words ?? []).map((word) => ({ label: word }));
  if (items.length === 0) return null;

  const scroll = (direction) => {
    const row = rowRef.current;
    if (!row) return;
    const offsets = [...row.children].map((child) => child.offsetLeft);
    const first = row.children[0]?.offsetLeft ?? 0;
    const current = offsets.findIndex(
      (left) => left - first >= row.scrollLeft - 1
    );
    const start = current === -1 ? offsets.length - 1 : current;
    const target = Math.min(
      Math.max(start + direction * step, 0),
      offsets.length - 1
    );
    row.scrollTo({ left: offsets[target] - first, behavior: "smooth" });
  };

  return (
    <div className={className ? `${styles.wrap} ${className}` : styles.wrap}>
      {canLeft && (
        <div className={`${styles.fade} ${styles.fadeLeft}`} aria-hidden />
      )}
      {canRight && (
        <div className={`${styles.fade} ${styles.fadeRight}`} aria-hidden />
      )}
      {canLeft && (
        <ActionIcon
          variant="default"
          radius="xl"
          size="lg"
          className={`${styles.arrow} ${styles.left}`}
          onClick={() => scroll(-1)}
          aria-label="scroll keywords left"
        >
          <IconChevronLeft size={18} stroke={1.5} />
        </ActionIcon>
      )}
      <div
        ref={rowRef}
        className={styles.row}
        onScroll={measure}
        role="list"
        aria-label={label}
        tabIndex={0}
      >
        {items.map((item) => (
          <div role="listitem" key={item.label} className={styles.item}>
            {item.href ? (
              <Link href={item.href} prefetch={false} className={styles.chip}>
                {item.label}
              </Link>
            ) : (
              <button
                type="button"
                className={styles.chip}
                onClick={() => onSelect(item.label)}
              >
                {item.label}
              </button>
            )}
          </div>
        ))}
      </div>
      {canRight && (
        <ActionIcon
          variant="default"
          radius="xl"
          size="lg"
          className={`${styles.arrow} ${styles.right}`}
          onClick={() => scroll(1)}
          aria-label="scroll keywords right"
        >
          <IconChevronRight size={18} stroke={1.5} />
        </ActionIcon>
      )}
    </div>
  );
}
