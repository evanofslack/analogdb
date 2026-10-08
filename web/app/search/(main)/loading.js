import styles from "@components/gallery.module.css";
import GallerySkeleton from "@components/gallerySkeleton";
import Header from "@components/header";

export default function Loading() {
  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.margin}>
        <GallerySkeleton />
      </div>
    </div>
  );
}
