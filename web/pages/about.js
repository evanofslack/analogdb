import Head from "next/head";
import styles from "../components/gallery.module.css";
import Header from "../components/header";
import About from "../components/about";

export default function AboutPage() {
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
