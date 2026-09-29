import { Skeleton } from "@mantine/core";
import styles from "./infiniteGallery.module.css";

export default function GallerySkeleton() {
  return (
    <div className={styles.skeletonContainer}>
      <div className={styles.skeletonGrid}>
        {[...Array(25)].map((_, index) => (
          <Skeleton key={index} height={300} radius="md" animate={true} />
        ))}
      </div>
    </div>
  );
}
