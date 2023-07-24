import Head from "next/head";
import styles from "../components/gallery.module.css";
import Header from "../components/header";
import About from "../components/about";
import { authorized_fetch } from "../fetch.js";

export async function getStaticProps() {
  const postsRoute = "/posts";
  const postsResponse = await authorized_fetch(postsRoute, "GET");
  const postsData = await postsResponse.json();
  const numPosts = postsData.meta.total_posts;

  const authorsRoute = "/authors";
  const authorsResponse = await authorized_fetch(authorsRoute, "GET");
  const authorsData = await authorsResponse.json();
  const numAuthors = [...new Set(authorsData.authors)].length;

  const data = { numPosts: numPosts, numAuthors: numAuthors };

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
