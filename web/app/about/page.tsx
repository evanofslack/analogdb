import About from "@components/about";
import styles from "@components/gallery.module.css";
import Header from "@components/header";
import { getAboutData } from "@lib/data/about";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "About",
  description:
    "What AnalogDB is: film photos analyzed for color, gear and content, with search and similar photos from the archive",
};

export const dynamic = "force-dynamic";

export default async function AboutPage() {
  const data = await getAboutData();

  return (
    <div className={styles.container}>
      <Header />
      <About data={data} />
    </div>
  );
}
