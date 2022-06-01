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
                        LATEST
                    </a>
                </Link>
                <Link href="/top">
                    <a className={router.pathname == "/latest" ? styles.linkOn : styles.linkOff}>
                        TOP
                    </a>
                </Link>
                <Link href="/random">
                    <a className={router.pathname == "/random" ? styles.linkOn : styles.linkOff}>
                        RANDOM
                    </a>
                </Link>
                <Link href="/bw">
                    <a className={router.pathname == "/bw" ? styles.linkOn : styles.linkOff}>B&W</a>
                </Link>
            </div>
        </nav>
    );
}
