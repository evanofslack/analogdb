"use client";

import useIsAdmin from "@hooks/useIsAdmin";
import Link from "next/link";
import { usePathname } from "next/navigation";
import ThemeToggle from "./themeToggle";
import styles from "./webNav.module.css";

export default function WebNav() {
  const isAdmin = useIsAdmin();
  const pathname = usePathname();
  return (
    <nav>
      <div className={styles.headerContainer}>
        <Link
          href="/"
          className={pathname == "/" ? styles.linkOn : styles.linkOff}
        >
          GALLERY
        </Link>
        <Link
          href="/about"
          className={pathname == "/about" ? styles.linkOn : styles.linkOff}
        >
          ABOUT
        </Link>
        <Link
          href="/docs"
          className={pathname == "/docs" ? styles.linkOn : styles.linkOff}
        >
          API
        </Link>
        {isAdmin && (
          <Link
            href="/admin"
            className={pathname == "/admin" ? styles.linkOn : styles.linkOff}
          >
            ADMIN
          </Link>
        )}
        <ThemeToggle />
      </div>
    </nav>
  );
}
