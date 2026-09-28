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

const defaultImage = {
  url: "/opengraph-image.jpg",
  width: 1200,
  height: 630,
  alt: "AnalogDB",
};

function describe(post) {
  const author = post.author.replace("u/", "");
  const camera = [post.camera_make, post.camera_model]
    .filter(Boolean)
    .join(" ");
  const film = [post.film_make, post.film_type].filter(Boolean).join(" ");

  let description = "Shot";
  if (camera) description += ` on ${camera}`;
  if (film) description += ` with ${film}`;
  return `${description} by ${author}`;
}

export async function generateMetadata({ params }) {
  const { pid } = await params;

  try {
    const post = await getPost(pid);
    if (!post) {
      return { title: "Post" };
    }

    const image = post.images?.find((img) => img.resolution === "high");
    const shareImage =
      image && !post.nsfw
        ? {
            url: image.url,
            width: image.width,
            height: image.height,
            alt: post.title,
          }
        : defaultImage;
    const description = describe(post);

    return {
      title: post.title,
      description,
      alternates: { canonical: `/post/${pid}` },
      openGraph: {
        siteName: "AnalogDB",
        type: "article",
        title: post.title,
        description,
        url: `/post/${pid}`,
        images: [shareImage],
      },
      twitter: {
        card: "summary_large_image",
        title: post.title,
        description,
        images: [shareImage],
      },
      ...(post.nsfw && { robots: { index: false } }),
    };
  } catch (error) {
    return { title: "Post" };
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
