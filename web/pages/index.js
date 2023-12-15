import Head from "next/head";
import Gallery from "../components/gallery";
import { authorized_fetch } from "../fetch.js";

export async function getServerSideProps() {
  const numPosts = 50;
  const route = `/posts?sort=latest&page_size=${numPosts}&grayscale=false&nsfw=false`;
  const response = await authorized_fetch(route, "GET");
  const data = await response.json();
  return {
    props: { data },
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
