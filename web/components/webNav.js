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
                <Link
                    href="/"
                    className={router.pathname == "/" ? styles.linkOn : styles.linkOff}>
                    
                        GALLERY
                    
                </Link>
                <Link
                    href="/about"
                    className={router.pathname == "/about" ? styles.linkOn : styles.linkOff}>
                    
                        ABOUT
                    
                </Link>
                <Link
                    href="/docs"
                    className={router.pathname == "/docs" ? styles.linkOn : styles.linkOff}>
                    
                        API
                    
                </Link>
            </div>
        </nav>
    );
}
