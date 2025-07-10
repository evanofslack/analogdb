import { authorized_fetch } from "@lib/fetch";
import { checkAdminAuth } from "@lib/auth";
import About from "@components/about";
import Header from "@components/header";
import styles from "@components/gallery.module.css";

export const metadata = {
  title: "AnalogDB",
  description: "Film photography database",
};

export const revalidate = 60; // Revalidate every 60 seconds

async function getData() {
  const postsRoute = "/posts";
  const postsResponse = await authorized_fetch(postsRoute, "GET");
  const postsData = await postsResponse.json();
  const numPosts = postsData.meta.total_posts;

  const authorsRoute = "/authors";
  const authorsResponse = await authorized_fetch(authorsRoute, "GET");
  const authorsData = await authorsResponse.json();
  const numAuthors = [...new Set(authorsData.authors)].length;

  return { numPosts: numPosts, numAuthors: numAuthors };
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
