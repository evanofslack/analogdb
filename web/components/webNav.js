import styles from "./webNav.module.css";
import Link from "next/link";
import { useRouter } from "next/router";
import { useBreakpoint } from "../providers/breakpoint.js";

export default function WebNav() {
    const router = useRouter();
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
                <Link href="/">
                    <a className={router.pathname == "/" ? styles.linkOn : styles.linkOff}>
                        GALLERY
                    </a>
                </Link>
                <Link href="/about">
                    <a className={router.pathname == "/about" ? styles.linkOn : styles.linkOff}>
                        ABOUT
                    </a>
                </Link>
                <Link href="/docs">
                    <a className={router.pathname == "/docs" ? styles.linkOn : styles.linkOff}>
                        API
                    </a>
                </Link>
            </div>
        </nav>
    );
}
