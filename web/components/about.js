import styles from "./about.module.css";

export default function About() {
    return (
        <main>
            <div className={styles.sectionLeft}>
                <div className={styles.title}>Film for all</div>
                <p className={styles.subtitle}>
                    Thousands of analog photographs compiled from reddit and curated for your
                    enjoyment.
                </p>
            </div>

            <div className={styles.sectionRight}>
                <div className={styles.title}>Free and open-source</div>
                <p className={styles.subtitle}>
                    All code made publically avaliable on Github with no restrictions.
                </p>
            </div>

            <div className={styles.sectionLeft}>
                <div className={styles.title}>Built with love</div>
                <p className={styles.subtitle}>
                    Go on the backend, React on the frontend, with Python for ingestion and Postgres
                    for storage.
                </p>
            </div>
        </main>
    );
}
