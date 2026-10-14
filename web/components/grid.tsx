"use client";

import useDisplay, { takeAnchor } from "@hooks/useDisplay";
import { columnWidth, gearPx, spacingPx } from "@lib/display";
import { toColumns } from "@lib/masonry";
import { AnalogdbPost } from "analogdb-generated";
import { CSSProperties, useLayoutEffect, useMemo } from "react";
import styles from "./grid.module.css";
import GridImage from "./gridImage";

const priorityCount = 6;

interface GridProps {
  pages: AnalogdbPost[][];
  onClick?: (post: AnalogdbPost, position: number) => void;
}

export default function Grid({ pages, onClick }: GridProps) {
  const { display, width, columns } = useDisplay();
  const gap = spacingPx[display.spacing];
  const extra =
    (gap + (display.gear === "always" ? gearPx : 0)) /
    columnWidth(width, columns);

  const layout = useMemo(
    () => toColumns(pages, columns, extra),
    [pages, columns, extra]
  );

  // Keep the image that was at the top of the screen in place when the
  // layout changes under it
  useLayoutEffect(() => {
    const anchor = takeAnchor();
    if (!anchor) return;
    const el = document.querySelector(`[data-post-id="${anchor.id}"]`);
    if (!el) return;
    window.scrollBy(0, el.getBoundingClientRect().top - anchor.top);
  }, [columns, gap, display.gear]);

  const sizes = `${Math.ceil(100 / columns)}vw`;

  return (
    <div
      className={styles.grid}
      style={{ "--grid-gap": `${gap}px` } as CSSProperties}
    >
      {layout.map((column, i) => (
        <div key={i} className={styles.column}>
          {column.map(({ post, index }) => (
            <GridImage
              key={post.id}
              post={post}
              sizes={sizes}
              gear={display.gear}
              priority={index < priorityCount}
              onClick={onClick && (() => onClick(post, index))}
            />
          ))}
        </div>
      ))}
    </div>
  );
}
