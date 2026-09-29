import Link from "next/link";
import styles from "./header.module.css";
import MobileNav from "./mobileNav";
import WebNav from "./webNav";

export default function Header() {
  return (
    <main className={styles.main}>
      <h1 className={styles.title}>
        <Link href="/">AnalogDB</Link>
        <p className={styles.description}>the collection of film photography</p>
      </h1>
      <div className={styles.webNav}>
        <WebNav />
      </div>
      <div className={styles.mobileNav}>
        <MobileNav />
      </div>
    </main>
  );
}
