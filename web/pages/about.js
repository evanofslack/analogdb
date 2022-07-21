import Head from "next/head";
import styles from "../styles/Home.module.css";
import Header from "../components/header";
import About from "../components/about";

export default function Template(props) {
    return (
        <div className={styles.container}>
            <Head>
                <title>AnalogDB</title>
                <link rel="icon" href="/favicon.ico" />
            </Head>
            <Header />
            <About />
        </div>
    );
}
