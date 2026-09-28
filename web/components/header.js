"use client";

import { useBreakpoint } from "@providers/breakpoint.js";
import Link from "next/link";
import styles from "./header.module.css";
import MobileNav from "./mobileNav";
import WebNav from "./webNav";

export default function Header() {
  const breakpoints = useBreakpoint();

  let useMobile = false;
  if (breakpoints["sm"]) {
    useMobile = true;
  }
  return (
    <main className={styles.main}>
      <h1 className={styles.title}>
        <Link href="/">AnalogDB</Link>
        <p className={styles.description}>the collection of film photography</p>
      </h1>
      {useMobile && <MobileNav />}
      {!useMobile && <WebNav />}
    </main>
  );
}
