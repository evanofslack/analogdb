import { baseURL } from "../../constants.ts";
import ImagePage from "../../components/imagePage";

export async function getStaticPaths() {
  const url = `${baseURL}/ids`;
  const response = await fetch(url);
  const data = await response.json();
  const paths = data.ids.map((id) => ({
    params: { pid: id.toString() },
  }));

  return { paths, fallback: "blocking" };
}

export async function getStaticProps({ params }) {
  const url = `${baseURL}/post/${params.pid}`;
  const response = await fetch(url);
  const post = await response.json();
  return {
    props: {
      post,
    },
    revalidate: 10,
  };
}

export default function Post({ post }) {
  return ImagePage((post = { post }));
}
