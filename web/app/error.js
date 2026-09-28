"use client";

import Footer from "@components/footer";
import Header from "@components/header";
import { Button } from "@mantine/core";
import styles from "@styles/Error.module.css";
import { useEffect } from "react";

export default function Error({ error, reset }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div>
      <Header />
      <div className={styles.center}>
        <h3 className={styles.error}>
          sorry, something is broken on our end [500]
        </h3>
        <Button variant="outline" color="gray" onClick={() => reset()}>
          try again
        </Button>
      </div>
      <Footer />
    </div>
  );
}
