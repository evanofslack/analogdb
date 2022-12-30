import styles from "./hamburger.module.css";
import { useRouter } from "next/router";
import Link from "next/link";
import { AiOutlineTrophy } from "react-icons/ai";
import { BiTimeFive } from "react-icons/bi";
import { BiShuffle } from "react-icons/bi";
import { RiCameraLensFill } from "react-icons/ri";

export default function Hamburger() {
  const router = useRouter();

  return (
    <div className={styles.blur}>
      <nav>
        <div className={styles.headerContainer}>
          <Link
            href="/"
            className={router.pathname == "/" ? styles.linkOn : styles.linkOff}
          >
            <div className={styles.icon}>
              <AiOutlineTrophy size="1.8rem" />
              <span className={styles.iconText}>top</span>
            </div>
          </Link>
          <Link
            href="/latest"
            className={
              router.pathname == "/latest" ? styles.linkOn : styles.linkOff
            }
          >
            <div className={styles.icon}>
              <BiTimeFive size="1.8rem" />
              <span className={styles.iconText}>latest</span>
            </div>
          </Link>
          <Link
            href="/random"
            className={
              router.pathname == "/random" ? styles.linkOn : styles.linkOff
            }
          >
            <div className={styles.icon}>
              <BiShuffle size="1.8rem" />
              <span className={styles.iconText}>random</span>
            </div>
          </Link>
          <Link
            href="/bw"
            className={
              router.pathname == "/bw" ? styles.linkOn : styles.linkOff
            }
          >
            <div className={styles.icon}>
              <RiCameraLensFill size="1.8rem" />
              <span className={styles.iconText}>b&w</span>
            </div>
          </Link>
        </div>
      </nav>
    </div>
  );
}
