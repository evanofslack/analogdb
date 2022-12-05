import styles from "./documentation.module.css";
import Footer from "./footer";
import Link from "next/link";

export default function Documentation() {
    return (
        <main>
            <div className={styles.sectionOne}>
                <div className={styles.construction}>
                    please excuse our appearance while we are remodeling...
                </div>
                <u>
                    <Link href="https://api.analogdb.com">visit the old docs</Link>
                </u>
            </div>
            <Footer />
        </main>
    );
}
