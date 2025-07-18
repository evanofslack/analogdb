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

  // Generate film options from films response
  const filmOptions = useMemo(() => {
    if (!filmsResponse?.films) return { makes: [], types: [] };

    const films = filmsResponse.films;
    const makes = [...new Set(films.map((f) => f.make).filter(Boolean))];

    // Filter types based on selected make
    const typesForMake = filmFilters.make
      ? films.filter((f) => f.make === filmFilters.make)
      : films;
    const types = [...new Set(typesForMake.map((f) => f.type).filter(Boolean))];

    return { makes, types };
  }, [filmsResponse]);

  // Handle Enter key press for text search
  useEffect(() => {
    if (returnPress) {
      executeQuery();
    }
  }, [returnPress, executeQuery]);

  useEffect(() => {
    console.log(filmsResponse);
  });

  return (
    <div className={styles.main}>
      <Header isAdmin={isAdmin} />
      <div className={styles.margin}>
        <FilterBar
          {...filters}
          {...setters}
          filmMakeOptions={filmOptions.makes}
          filmTypeOptions={filmOptions.types}
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
