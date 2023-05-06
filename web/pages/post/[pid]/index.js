import { baseURL } from "../../../constants.ts";
import ImagePage from "../../../components/imagePage";

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
  const postURL = `${baseURL}/post/${params.pid}`;
  const response = await fetch(postURL);
  const post = await response.json();

  // only show nsfw results if the original image was nsfw
  let query = "?nsfw=false";
  if (post.nsfw) {
    query = "";
  }
  const similarURL = `${baseURL}/post/${params.pid}/similar` + query;
  const similarResp = await fetch(similarURL);
  const similar = await similarResp.json();
  return {
    props: {
      post,
      similar,
    },
    revalidate: 10,
  };
}

export default function Post({ post, similar }) {
  return ImagePage({ post, similar });
}
