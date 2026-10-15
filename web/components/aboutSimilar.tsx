"use client";

import { SimilarityData } from "@lib/about";
import AboutPhoto from "./aboutPhoto";
import styles from "./aboutSimilar.module.css";

export const SIMILAR_ROWS = 3;
export const SIMILAR_MATCHES = 5;

// empty sleeves hold the space until the rows are picked
function Placeholder() {
  return (
    <div className={styles.rows} aria-hidden>
      {Array.from({ length: SIMILAR_ROWS }, (_, row) => (
        <div key={row} className={styles.row}>
          <div className={`${styles.source} ${styles.empty}`} />
          {Array.from({ length: SIMILAR_MATCHES }, (_, tile) => (
            <div key={tile} className={`${styles.tile} ${styles.empty}`} />
          ))}
        </div>
      ))}
    </div>
  );
}

// each row is a source photo, then its closest matches
export default function AboutSimilar({
  rows,
}: {
  rows: SimilarityData[] | null;
}) {
  if (!rows) return <Placeholder />;
  if (!rows.length) return null;

  return (
    <div className={styles.rows}>
      {rows.map(({ centerPost, similarPosts }) => (
        <div key={centerPost.id} className={styles.row}>
          <AboutPhoto
            photo={centerPost}
            small
            fill
            sizes="(max-width: 720px) 30vw, 200px"
            className={styles.source}
          />
          {similarPosts.map((photo) => (
            <AboutPhoto
              key={photo.id}
              photo={photo}
              small
              fill
              sizes="(max-width: 720px) 20vw, 130px"
              className={styles.tile}
            />
          ))}
        </div>
      ))}
    </div>
  );
}
