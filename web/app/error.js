"use client";

import Footer from "@components/footer";
import Header from "@components/header";
import styles from "@styles/Error.module.css";

export default function Error({ error, reset }) {
  const isAdmin = false;
  return (
    <div>
      <Header isAdmin={isAdmin} />
      <div className={styles.center}>
        <h3 className={styles.error}>
          sorry, something is broken on our end [500]
        </h3>
      </div>
      <Footer />
    </div>
  );
}
