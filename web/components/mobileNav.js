import { AiOutlineMenu } from "react-icons/ai";
import { useState } from "react";
import styles from "./mobileNav.module.css";
import { useRouter } from "next/router";
import Link from "next/link";
import { BiCheck } from "react-icons/bi";
import { GrClose } from "react-icons/gr";
import { FiGithub } from "react-icons/fi";

export default function MobileNav() {
  const router = useRouter();
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
                    {router.pathname == "/" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/about" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <h1 className={styles.iconText}>ABOUT</h1>
                    {router.pathname == "/about" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
              <Link href="/docs" className={styles.link}>
                <div className={styles.icon}>
                  <div className={styles.check}>
                    <h1 className={styles.iconText}>API</h1>
                    {router.pathname == "/docs" && <BiCheck size="2rem" />}
                  </div>
                </div>
              </Link>
            </nav>
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
