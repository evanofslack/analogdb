"use client";

import Footer from "@components/footer";
import Header from "@components/header";
import { checkAdminAuth } from "@lib/auth";
import styles from "@styles/Error.module.css";

export default async function Error({ error, reset }) {
  const isAdmin = await checkAdminAuth();
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
