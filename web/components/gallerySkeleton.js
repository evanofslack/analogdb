import { Skeleton } from "@mantine/core";
import styles from "./infiniteGallery.module.css";

const heights = [260, 340, 300, 380, 280];

export default function GallerySkeleton() {
  return (
    <div className={styles.skeletonContainer}>
      <div className={styles.skeletonGrid}>
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
