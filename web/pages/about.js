import Head from "next/head";
import styles from "../components/gallery.module.css";
import Header from "../components/header";
import About from "../components/about";
import { baseURL } from "../constants.ts";

export async function getStaticProps() {
  const postsURL =
    baseURL + "/posts";
  const postsResp = await fetch(postsURL);
  const postsData = await postsResp.json();
  const numPosts = postsData.meta.total_posts;

  const authorsURL =
    baseURL + "/authors";
  const authorsResp = await fetch(authorsURL);
  const authorsData = await authorsResp.json();
  const numAuthors = [...new Set(authorsData.authors)].length;

  const data = {numPosts: numPosts, numAuthors: numAuthors}

  console.log(data)

  return {
    props: {
      data,
    },
    revalidate: 60,
  };
}

export default function AboutPage({ data }) {
  return (
    <div className={styles.container}>
      <Head>
        <title>AnalogDB</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Header />
      <About data={data} />
    </div>
  );
}
