import Documentation from "@components/documentation";
import styles from "@components/gallery.module.css";
import Header from "@components/header";

export const metadata = {
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
