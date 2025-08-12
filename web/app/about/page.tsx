import About from "@components/about";
import styles from "@components/gallery.module.css";
import Header from "@components/header";
import { checkAdminAuth } from "@lib/auth";
import { authorized_fetch } from "@lib/client";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "AnalogDB",
  description: "Film photography database",
};

export const revalidate = 60;

interface PostsResponse {
  meta: {
    total_posts: number;
  };
}

interface AuthorsResponse {
  authors: string[];
}

interface AboutData {
  numPosts: number;
  numAuthors: number;
}

async function getData(): Promise<AboutData> {
  const postsRoute = "/posts";
  const postsResponse = await authorized_fetch(postsRoute, "GET");
  const postsData: PostsResponse = await postsResponse.json();
  const numPosts = postsData.meta.total_posts;

  const authorsRoute = "/authors";
  const authorsResponse = await authorized_fetch(authorsRoute, "GET");
  const authorsData: AuthorsResponse = await authorsResponse.json();
  const numAuthors = Array.from(new Set(authorsData.authors)).length;

  return { numPosts, numAuthors };
}

export default async function AboutPage() {
  const data = await getData();
  const isAdmin = await checkAdminAuth();

  return (
    <div className={styles.container}>
      <Header isAdmin={isAdmin} />
      <About data={data} />
    </div>
  );
}
