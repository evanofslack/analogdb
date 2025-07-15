"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { AiOutlineMenu } from "react-icons/ai";
import { BiCheck } from "react-icons/bi";
import { FiGithub } from "react-icons/fi";
import { GrClose } from "react-icons/gr";
import styles from "./mobileNav.module.css";

export default function MobileNav(props) {
  let isAdmin = props.isAdmin;
  const pathname = usePathname();
  const [isOpen, setIsOpen] = useState(false);
  const toggle = () => setIsOpen((value) => !value);

  return (
    <div>
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
                    <h1 className={styles.iconText}>GALLERY</h1>
                    {pathname === "/" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/about" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <h1 className={styles.iconText}>ABOUT</h1>
                    {pathname === "/about" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/docs" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <h1 className={styles.iconText}>API</h1>
                    {pathname === "/docs" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              {isAdmin && (
                <Link href="/admin" className={styles.link}>
                  <div className={styles.icon}>
                    <div className={styles.check}>
                      <h1 className={styles.iconText}>ADMIN</h1>
                      {pathname === "/admin" && <BiCheck size="2rem" />}
                    </div>
                  </div>
                </Link>
              )}
            </nav>
            <div className={styles.footer}>
              <p> &copy; 2022 analogdb </p>
              <a href="https://github.com/evanofslack/analogdb">
                <FiGithub size="1.2rem" />
              </a>
            </div>
            <div className={styles.footer}>
              <p> &copy; 2022 analogdb </p>
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
