"use client";

import styles from "../styles/Error.module.css";
import Header from "../components/header";
import Footer from "../components/footer";

export default function Error({ error, reset }) {
  return (
    <div>
      <Header />
      <div className={styles.center}>
        <h3 className={styles.error}>
          sorry, something is broken on our end [500]
        </h3>
      </div>
      <Footer />
    </div>
  );
}
