import Documentation from "../../components/documentation";
import Header from "../../components/header";
import styles from "../../components/gallery.module.css";

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
