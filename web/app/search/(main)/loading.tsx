import styles from "@components/gallery.module.css";
import Header from "@components/header";
import { SearchBarSkeleton } from "@components/searchSkeletons";
import searchStyles from "./search.module.css";

export default function Loading() {
  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.margin}>
        <div className={searchStyles.top}>
          <SearchBarSkeleton />
        </div>
      </div>
    </div>
  );
}
