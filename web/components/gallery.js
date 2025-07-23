"use client";

import useFilms from "@hooks/useFilms";
import useKeyPress from "@hooks/useKeyPress";
import usePosts from "@hooks/usePosts";
import { useBreakpoint } from "@providers/breakpoint";
import { useEffect, useMemo } from "react";
import FilterBar from "./filterBar";
import Footer from "./footer";
import styles from "./gallery.module.css";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import ScrollTop from "./scrollTop";

export default function Gallery({ isAdmin }) {
  const { response, isLoading, filters, setters, executeQuery, limits } =
    usePosts();

  const {
    response: filmsResponse, // Rename response to filmsResponse
    isLoading: isFilmLoading,
    filters: filmFilters,
    setters: filmSetters,
    executeQuery: executeFilmQuery,
  } = useFilms(500);

  const returnPress = useKeyPress("Enter");
  const breakpoints = useBreakpoint();

  const onlyIcon = breakpoints["xs"] || breakpoints["sm"];
  const textPlaceholder = onlyIcon
    ? "films, cameras..."
    : "films, cameras, places...";

  const filmOptions = useMemo(() => {
    if (!filmsResponse?.films) return [];

    return filmsResponse.films
      .filter((f) => f.make && f.type)
      .map((f) => ({
        make: f.make,
        type: f.type,
        label: `${f.make} - ${f.type}`,
      }))
      .filter((v, i, arr) => arr.findIndex((x) => x.label === v.label) === i);
  }, [filmsResponse]);

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
          filmOptions={filmOptions}
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
