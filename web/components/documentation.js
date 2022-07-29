import styles from "./documentation.module.css";
import Footer from "./footer";

export default function Documentation() {
    return (
        <main>
            <div className={styles.sectionOne}>
                <div className={styles.construction}>
                    please excuse our appearance while we are remodeling...
                </div>
                <u>
                    <a href="https://analogdb.herokuapp.com/">visit the old docs -> </a>
                </u>
            </div>
            <Footer />
        </main>
    );
}
