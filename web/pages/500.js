import Head from "next/head";
import styles from "./errorpage.module.css";
import Header from "../components/header";
import Footer from "../components/footer";

export default function Custom404() {
  return (
    <div>
      <Head>
        <title>AnalogDB</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Header />
      <div className={styles.center}>
        <h3 className={styles.error}>
          sorry, something is broken on our end [500]
        </h3>
      </div>
      <Footer />
    </div>
  );
}
