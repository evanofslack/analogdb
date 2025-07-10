import { authorized_fetch } from "@lib/fetch";
import ImagePage from "@components/imagePage";
import { notFound } from "next/navigation";
import { checkAdminAuth } from "@lib/auth";

export async function generateStaticParams() {
  if (process.env.NODE_ENV === "development") {
    return [];
  }

  // get all post IDs for production
  const response = await authorized_fetch("/ids", "GET");
  const data = await response.json();

  // only generate static pages for latest 500 posts
  return data.ids.slice(-500).map((id) => ({
    pid: id.toString(),
  }));
}

// Generate metadata based on post
export async function generateMetadata({ params }) {
  const { pid } = await params;

  try {
    const response = await authorized_fetch(`/post/${pid}`, "GET");
    const post = await response.json();

    return {
      title: `${post.title} | AnalogDB`,
      description: `${post.title} - photo by ${post.author}`,
    };
  } catch (error) {
    return {
      title: "Post | AnalogDB",
    };
  }
}

async function getPostData(pid) {
  const postRoute = `/post/${pid}`;
  const response = await authorized_fetch(postRoute, "GET");

  if (!response.ok) {
    return notFound();
  }

  const post = await response.json();

  // only show nsfw results if the original image was nsfw
  let query = "?nsfw=false";
  if (post.nsfw) {
    query = "";
  }

  const similarRoute = `/post/${pid}/similar${query}`;
  let similar;

  try {
    const similarResponse = await authorized_fetch(similarRoute, "GET");
    similar = await similarResponse.json();
  } catch (e) {
    similar = {};
  }

  return { post, similar };
}

export default async function Post({ params }) {
  const { pid } = await params;
  const { post, similar } = await getPostData(pid);
  const isAdmin = await checkAdminAuth();

  return <ImagePage post={post} similar={similar} isAdmin={isAdmin} />;
}

export const dynamic = "force-dynamic";
