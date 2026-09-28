import Documentation from "@components/documentation";
import styles from "@components/gallery.module.css";
import Header from "@components/header";

export const metadata = {
  title: "AnalogDB",
  description: "Film photography database",
};

export default function Docs() {
  return (
    <div className={styles.container}>
      <Header />
      <Documentation />
    </div>
  );
}
