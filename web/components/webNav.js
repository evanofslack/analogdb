"use client";

import { useBreakpoint } from "@providers/breakpoint";
import Link from "next/link";
import { usePathname } from "next/navigation";
import styles from "./webNav.module.css";

export default function WebNav(props) {
  let isAdmin = props.isAdmin;

  const pathname = usePathname();
  const breakpoints = useBreakpoint();

  let useMobile = false;
  if (breakpoints["sm"]) {
    useMobile = true;
  }
  if (useMobile) {
    return null;
  }
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
      </div>
    </nav>
  );
}
