"use client";

import useDisplay from "@hooks/useDisplay";
import { spacingPx } from "@lib/display";
import { Skeleton } from "@mantine/core";
import { CSSProperties } from "react";
import styles from "./infiniteGallery.module.css";

const heights = [260, 340, 300, 380, 280];

export default function GallerySkeleton() {
  const { display, columns } = useDisplay();
  // auto falls back to the breakpoints in css, since the route loading
  // skeleton renders without the server's guess
  const style = {
    "--grid-columns": display.columns === "auto" ? undefined : columns,
    "--grid-gap": `${spacingPx[display.spacing]}px`,
  } as CSSProperties;

  return (
    <div className={styles.skeletonContainer}>
      <div className={styles.skeletonGrid} style={style}>
        {[...Array(24)].map((_, index) => (
          <Skeleton
            key={index}
            className={styles.skeletonTile}
            height={heights[index % heights.length]}
            radius="md"
            animate={true}
          />
        ))}
      </div>
    </div>
  );
}
