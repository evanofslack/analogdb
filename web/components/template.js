import Head from "next/head";
import styles from "../styles/Home.module.css";
import Header from "../components/header";
import InfiniteGallery from "../components/infiniteGallery";
import ScrollTop from "../components/scrollTop";

export default function Template(props) {
    return (
        <div className={styles.container}>
            <Head>
                <title>AnalogDB</title>
                <link rel="icon" href="/favicon.ico" />
            </Head>
            <Header />
            <InfiniteGallery response={props.data} />
            <ScrollTop />
        </div>
    );
}
