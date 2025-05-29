import styles from "../styles/Error.module.css";
import Header from "../components/header";
import Footer from "../components/footer";

export default function NotFound() {
  return (
    <div>
      <Header />
      <div className={styles.center}>
        <h3 className={styles.error}>
          sorry, this page got lost somewhere along the way [404]
        </h3>
      </div>
      <Footer />
    </div>
  );
}
