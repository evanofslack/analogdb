import Link from "next/link";
import { FiGithub } from "react-icons/fi";
import styles from "./footer.module.css";

export default function Footer() {
  return (
    <footer className={styles.footer}>
      <p> &copy; 2026 AnalogDB </p>
      <div className={styles.links}>
        <Link href="/docs" className={styles.link}>
          API
        </Link>
        <a
          href="https://github.com/evanofslack/analogdb"
          className={styles.link}
          aria-label="GitHub"
        >
          <FiGithub size="18px" />
        </a>
      </div>
    </footer>
  );
}
