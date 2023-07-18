import Head from "next/head";
import Gallery from "../components/gallery";
import { baseURL } from "../constants.ts";

export async function getStaticProps() {
  const numPosts = 40
  const url =
    baseURL + `/posts?sort=latest&page_size=${numPosts}&grayscale=false&nsfw=false`;
  const response = await fetch(url);
  const data = await response.json();
  return {
    props: {
      data,
    },
    revalidate: 60,
  };
}

export default function Home({ data }) {
  return (
    <div>
      <Head>
        <title>AnalogDB</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Gallery data={data}></Gallery>;
    </div>
  );
}
