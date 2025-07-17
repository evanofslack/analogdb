"use client";

import useKeyPress from "@hooks/useKeyPress";
import usePostsQuery from "@hooks/usePostsQuery";
import { useBreakpoint } from "@providers/breakpoint.js";
import { useEffect } from "react";
import FilterBar from "./filterBar";
import Footer from "./footer";
import styles from "./gallery.module.css";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import ScrollTop from "./scrollTop";

export default function Gallery({ isAdmin }) {
  const { response, isLoading, filters, setters, executeQuery, limits } =
    usePostsQuery();

  const returnPress = useKeyPress("Enter");
  const breakpoints = useBreakpoint();

  const onlyIcon = breakpoints["xs"] || breakpoints["sm"];
  const textPlaceholder = onlyIcon
    ? "films, cameras..."
    : "films, cameras, places...";

  // Handle Enter key press for text search
  useEffect(() => {
    if (returnPress) {
      executeQuery();
    }
  }, [returnPress, executeQuery]);

  return (
    <div className={styles.main}>
      <Header isAdmin={isAdmin} />
      <div className={styles.margin}>
        <FilterBar
          {...filters}
          {...setters}
          onlyIcon={onlyIcon}
          textPlaceholder={textPlaceholder}
          widthMinLimit={limits.widthMin}
          widthMaxLimit={limits.widthMax}
          heightMinLimit={limits.heightMin}
          heightMaxLimit={limits.heightMax}
          ratioMinLimit={limits.ratioMin}
          ratioMaxLimit={limits.ratioMax}
        />
        <InfiniteGallery response={response} isLoading={isLoading} />
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
