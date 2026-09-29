"use client";

import { useEffect, useMemo, useState } from "react";
import styles from "./grid.module.css";
import GridImage from "./gridImage";

const priorityCount = 6;

function ratio(post) {
  const image = post.images?.[0];
  if (!image?.width || !image?.height) return 1;
  return image.height / image.width;
}

// Each post goes in the shortest column, measured by image ratio, so the
// server and the client build the same columns before any image loads
function toColumns(posts, count) {
  const columns = Array.from({ length: count }, () => []);
  const heights = Array(count).fill(0);
  posts.forEach((post, index) => {
    const shortest = heights.indexOf(Math.min(...heights));
    columns[shortest].push({ post, index });
    heights[shortest] += ratio(post);
  });
  return columns;
}

export default function Grid({ posts, initialColumns = 4 }) {
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
    () => toColumns(posts, numColumn),
    [posts, numColumn]
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
