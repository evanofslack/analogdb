"use client";

import { ActionIcon } from "@mantine/core";
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import styles from "./keywordRow.module.css";

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
    row?.scrollBy({
      left: direction * row.clientWidth * 0.8,
      behavior: "smooth",
    });
  };

  let fade = styles.row;
  if (canLeft && canRight) fade = `${styles.row} ${styles.fadeBoth}`;
  else if (canLeft) fade = `${styles.row} ${styles.fadeLeft}`;
  else if (canRight) fade = `${styles.row} ${styles.fadeRight}`;

  return (
    <div className={className ? `${styles.wrap} ${className}` : styles.wrap}>
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
        className={fade}
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
