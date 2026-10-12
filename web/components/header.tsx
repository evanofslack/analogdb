"use client";

import useScrollUp from "@hooks/useScrollUp";
import Link from "next/link";
import { useRef, useState } from "react";
import styles from "./header.module.css";
import MobileNav from "./mobileNav";
import WebNav from "./webNav";

interface HeaderProps {
  compact?: boolean;
  brandHeading?: boolean;
}

export default function Header({
  compact = false,
  brandHeading = true,
}: HeaderProps) {
  const Brand = brandHeading ? "h1" : "p";
  const headerRef = useRef<HTMLElement>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const shown = useScrollUp(headerRef, { paused: menuOpen });

  return (
    <>
      <header
        ref={headerRef}
        className={compact ? styles.compact : styles.main}
      >
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
      <div
        className={shown ? `${styles.bar} ${styles.barShown}` : styles.bar}
        inert={!shown}
      >
        <p className={styles.compactTitle}>
          <Link href="/">AnalogDB</Link>
        </p>
        <MobileNav onOpenChange={setMenuOpen} />
      </div>
    </>
  );
}
