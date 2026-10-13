import Documentation from "@components/documentation";
import styles from "@components/gallery.module.css";
import Header from "@components/header";
import type { Metadata } from "next";

export const revalidate = 86400;

export const metadata: Metadata = {
  title: "API docs",
  description: "Documentation for the AnalogDB public API",
};

export default function Docs() {
  return (
    <div className={styles.container}>
      <Header />
      <Documentation />
    </div>
  );
}
