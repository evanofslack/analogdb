"use client";

import { useEffect, useMemo, useState } from "react";
import styles from "./grid.module.css";
import GridImage from "./gridImage";
import { toColumns } from "@lib/masonry";

const priorityCount = 6;

export default function Grid({ pages, initialColumns = 4 }) {
  const [numColumn, setNumColumn] = useState(initialColumns);

  useEffect(() => {
    const small = window.matchMedia("(max-width: 720px)");
    const medium = window.matchMedia("(max-width: 1440px)");
    const update = () => {
      if (small.matches) setNumColumn(2);
      else if (medium.matches) setNumColumn(3);
      else setNumColumn(4);
    };
    update();
    small.addEventListener("change", update);
    medium.addEventListener("change", update);
    return () => {
      small.removeEventListener("change", update);
      medium.removeEventListener("change", update);
    };
  }, []);

  const columns = useMemo(
    () => toColumns(pages, numColumn),
    [pages, numColumn]
  );

  return (
    <div className={styles.grid}>
      {columns.map((column, i) => (
        <div key={i} className={styles.column}>
          {column.map(({ post, index }) => (
            <GridImage
              key={post.id}
              post={post}
              priority={index < priorityCount}
            />
          ))}
        </div>
      ))}
    </div>
  );
}
