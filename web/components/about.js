import styles from "./about.module.css";

export default function About() {
    return (
        <main>
            <div></div>
            <div></div>
            <div className={styles.sectionLeft}>
                <div className={styles.title}>Film for all</div>
                <p className={styles.subtitle}>
                    AnalogDB is a curated database featuring thousands of film photographs. And it
                    is always growing, with new pictures added every day.
                </p>
            </div>

            <div className={styles.sectionRight}>
                <div className={styles.title}>Accesible API</div>
                <p className={styles.subtitle}>
                    The entire collection of film is exposed through a simple and intuitive API.
                    Embedded any of our photos in your next project with ease.
                </p>
            </div>

            <div className={styles.sectionRight}>
                <div className={styles.title}>Free and open-source</div>
                <p className={styles.subtitle}>
                    All code made publically avaliable on Github with flexible liscening
                </p>
            </div>

            <div className={styles.sectionLeft}>
                <div className={styles.title}>Fast image loads</div>
                <p className={styles.subtitle}>
                    Photos are hosted on a CDN to deliever quick and efficient load times regardless
                    of region.
                </p>
            </div>
        </main>
    );
}
