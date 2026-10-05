import Link from "next/link";
import styles from "./header.module.css";
import MobileNav from "./mobileNav";
import SearchButton from "./searchButton";
import WebNav from "./webNav";

export default function Header({ compact = false }) {
  return (
    <header className={compact ? styles.compact : styles.main}>
      {compact ? (
        <p className={styles.compactTitle}>
          <Link href="/">AnalogDB</Link>
        </p>
      ) : (
        <div className={styles.brand}>
          <h1 className={styles.title}>
            <Link href="/">AnalogDB</Link>
          </h1>
          <p className={styles.description}>
            the collection of film photography
          </p>
        </div>
      )}
      <div className={styles.actions}>
        <div className={styles.search}>
          <SearchButton />
        </div>
        <div className={styles.webNav}>
          <WebNav />
        </div>
        <div className={styles.mobileNav}>
          <MobileNav />
        </div>
      </div>
    </header>
  );
}
