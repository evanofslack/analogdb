import { authorized_fetch } from "../../../fetch.js";
import ImagePage from "../../../components/imagePage";

export async function getStaticPaths() {
  // for debug environments, skip rendering static pages
  if (process.env.NODE_ENV == "development") {
    console.log("Development env, skipping static page generation");
    return {
      paths: [],
      fallback: "blocking",
    };
  }

  // for production, get all ids and generate static pages for latest 500 posts
  const response = await authorized_fetch("/ids", "GET");
  const data = await response.json();

    // only 500 latest posts
  const paths = data.ids.slice(-500).map((id) => ({
    params: { pid: id.toString() },
  }));

  return { paths, fallback: "blocking" };
}

export async function getStaticProps({ params }) {
  const postRoute = `/post/${params.pid}`;

  const response = await authorized_fetch(postRoute, "GET");

  if (!response.ok) {
    return {
      notFound: true,
    };
  }
  const post = await response.json();

  // only show nsfw results if the original image was nsfw
  let query = "?nsfw=false";
  if (post.nsfw) {
    query = "";
  }

  const similarRoute = `/post/${params.pid}/similar` + query;
  let similar;
  try {
    const similarResponse = await authorized_fetch(similarRoute, "GET");
    similar = await similarResponse.json();
  } catch (e) {
    similar = {};
  }

  return {
    props: {
      post,
      similar,
    },
    revalidate: 60,
  };
}

export default function Post({ post, similar }) {
  return ImagePage({ post, similar });
}
