import styles from "./webNav.module.css";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useBreakpoint } from "../providers/breakpoint.js";

export default function WebNav() {
    const pathname = usePathname();
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
                    className={router.pathname == "/" ? styles.linkOn : styles.linkOff}
                >
                    GALLERY
                </Link>
                <Link
                    href="/about"
                    className={
                        pathname == "/about" ? styles.linkOn : styles.linkOff
                    }
                >
                    ABOUT
                </Link>
                <Link
                    href="/docs"
                    className={
                        pathname == "/docs" ? styles.linkOn : styles.linkOff
                    }
                >
                    API
                </Link>
            </div>
        </nav>
    );
}
