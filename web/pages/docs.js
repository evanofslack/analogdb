import Head from "next/head";
import styles from "../components/gallery.module.css";
import Header from "../components/header";
import Documentation from "../components/documentation";

export default function Docs() {
  return (
    <div className={styles.container}>
      <Head>
        <title>AnalogDB</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Header />
      <Documentation />
    </div>
  );
}
