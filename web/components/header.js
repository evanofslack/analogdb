"use client";
import styles from "./header.module.css";
import WebNav from "./webNav";
import MobileNav from "./mobileNav";
import Link from "next/link";
import { useBreakpoint } from "@providers/breakpoint.js";

export default function Header(props) {
  let isAdmin = props.isAdmin;
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
      {useMobile && <MobileNav isAdmin={isAdmin} />}
      {!useMobile && <WebNav isAdmin={isAdmin} />}
    </main>
  );
}
