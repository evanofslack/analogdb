"use client";

import useIsAdmin from "@hooks/useIsAdmin";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { AiOutlineMenu } from "react-icons/ai";
import { BiCheck } from "react-icons/bi";
import { FiGithub } from "react-icons/fi";
import { GrClose } from "react-icons/gr";
import styles from "./mobileNav.module.css";
import ThemeToggle from "./themeToggle";

export default function MobileNav() {
  const isAdmin = useIsAdmin();
  const pathname = usePathname();
  const [isOpen, setIsOpen] = useState(false);
  const toggle = () => setIsOpen((value) => !value);

  return (
    <div className={styles.bar}>
      <ThemeToggle />
      <AiOutlineMenu size="1.8rem" onClick={toggle} />
      {isOpen && (
        <div className={styles.blur}>
          <div className={styles.headerContainer}>
            <div className={styles.close}>
              <GrClose size="1.5rem" onClick={toggle} />
            </div>

            <nav className={styles.navContainer}>
              <Link href="/" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <span className={styles.iconText}>GALLERY</span>
                    {pathname === "/" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/search" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <span className={styles.iconText}>SEARCH</span>
                    {pathname === "/search" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/films" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <span className={styles.iconText}>FILM</span>
                    {pathname.startsWith("/films") && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/cameras" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <span className={styles.iconText}>CAMERAS</span>
                    {pathname.startsWith("/cameras") && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/about" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <span className={styles.iconText}>ABOUT</span>
                    {pathname === "/about" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/docs" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <span className={styles.iconText}>API</span>
                    {pathname === "/docs" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              {isAdmin && (
                <Link href="/admin" className={styles.link}>
                  <div className={styles.icon}>
                    <div className={styles.check}>
                      <span className={styles.iconText}>ADMIN</span>
                      {pathname === "/admin" && <BiCheck size="2rem" />}
                    </div>
                  </div>
                </Link>
              )}
            </nav>
            <div className={styles.footer}>
              <p> &copy; 2025 AnalogDB </p>
              <a href="https://github.com/evanofslack/analogdb">
                <FiGithub size="1.2rem" />
              </a>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
