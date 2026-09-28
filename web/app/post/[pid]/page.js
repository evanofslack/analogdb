import ImagePage from "@components/imagePage";
import { authorized_fetch } from "@lib/client";
import { notFound } from "next/navigation";
import { cache } from "react";

export const dynamicParams = true;
export const revalidate = 3600;

export async function generateStaticParams() {
  return [];
}

const getPost = cache(async (pid) => {
  const response = await authorized_fetch(`/post/${pid}`, "GET", revalidate);
  if (!response.ok) {
    return null;
  }
  return response.json();
});

// Generate metadata based on post
export async function generateMetadata({ params }) {
  const { pid } = await params;

  try {
    const post = await getPost(pid);
    if (!post) {
      return { title: "Post | AnalogDB" };
    }

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
  const post = await getPost(pid);

  if (!post) {
    return notFound();
  }

  // only show nsfw results if the original image was nsfw
  let query = "?nsfw=false";
  if (post.nsfw) {
    query = "";
  }

  const similarRoute = `/post/${pid}/similar${query}`;
  let similar;

  try {
    const similarResponse = await authorized_fetch(
      similarRoute,
      "GET",
      revalidate
    );
    similar = await similarResponse.json();
  } catch (e) {
    similar = {};
  }

  return { post, similar };
}

export default async function Post({ params }) {
  const { pid } = await params;
  const { post, similar } = await getPostData(pid);

  return <ImagePage post={post} similar={similar} />;
}
