import Gallery from "../components/gallery";
import { baseURL } from "../constants.ts";

export async function getStaticProps(context) {
  const url = baseURL + "/posts/latest?page_size=50&bw=false&nsfw=false";
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
  return <Gallery data={data}></Gallery>;
}
