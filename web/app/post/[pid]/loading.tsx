import Header from "@components/header";
import styles from "@components/imagePage.module.css";
import { Skeleton } from "@mantine/core";
import { CSSProperties } from "react";

export default function Loading() {
  return (
    <div className={styles.fullscreen}>
      <div className={styles.header}>
        <Header compact />
      </div>
      <div
        className={styles.imageContainer}
        style={{ "--ratio": "3 / 2" } as CSSProperties}
      >
        <Skeleton height="100%" radius={0} animate={true} />
      </div>
    </div>
  );
}
