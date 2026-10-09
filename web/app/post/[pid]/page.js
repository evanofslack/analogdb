import { getCameraCatalog } from "@lib/data/cameras";
import { getFilmCatalog } from "@lib/data/films";
import ImagePage from "@components/imagePage";
import { catalogHref, findByName } from "@lib/catalog";
import { authorized_fetch } from "@lib/client";
import { authorName, jsonLd, postDescription } from "@lib/seo";
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

function imageObject(post) {
  const image = (resolution) =>
    post.images?.find((img) => img.resolution === resolution)?.url;
  const author = authorName(post);
  return {
    "@context": "https://schema.org",
    "@type": "ImageObject",
    contentUrl: image("high"),
    thumbnailUrl: image("medium"),
    name: post.title,
    description: postDescription(post),
    url: `https://analogdb.com/post/${post.id}`,
    creator: {
      "@type": "Person",
      name: author,
      url: `https://www.reddit.com/user/${author}`,
    },
    creditText: author,
    copyrightNotice: author,
    datePublished: new Date(post.timestamp * 1000).toISOString(),
  };
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
    const description = postDescription(post);

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

async function getCatalogHrefs(post) {
  const [films, cameras] = await Promise.all([
    getFilmCatalog(),
    getCameraCatalog(),
  ]);
  const film = findByName(films, post.film_make, post.film_type);
  const camera = findByName(cameras, post.camera_make, post.camera_model);
  return {
    filmHref: film ? catalogHref("films", film) : null,
    cameraHref: camera ? catalogHref("cameras", camera) : null,
  };
}

export default async function Post({ params }) {
  const { pid } = await params;
  const { post, similar } = await getPostData(pid);
  const { filmHref, cameraHref } = await getCatalogHrefs(post);

  return (
    <>
      {!post.nsfw && (
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: jsonLd(imageObject(post)) }}
        />
      )}
      <ImagePage
        post={post}
        similar={similar}
        filmHref={filmHref}
        cameraHref={cameraHref}
      />
    </>
  );
}
