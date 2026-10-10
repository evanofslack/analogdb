import Link from "next/link";
import styles from "./header.module.css";
import MobileNav from "./mobileNav";
import WebNav from "./webNav";

export default function Header({ compact = false, brandHeading = true }) {
  const Brand = brandHeading ? "h1" : "p";
  return (
    <header className={compact ? styles.compact : styles.main}>
      {compact ? (
        <p className={styles.compactTitle}>
          <Link href="/">AnalogDB</Link>
        </p>
      ) : (
        <div className={styles.brand}>
          <Brand className={styles.title}>
            <Link href="/">AnalogDB</Link>
          </Brand>
          <p className={styles.description}>
            the collection of film photography
          </p>
        </div>
      )}
      <div className={styles.webNav}>
        <WebNav />
      </div>
      <div className={styles.mobileNav}>
        <MobileNav />
      </div>
    </header>
  );
}
