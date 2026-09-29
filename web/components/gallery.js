"use client";

import useCameras from "@hooks/useCameras";
import useFilms from "@hooks/useFilms";
import usePosts from "@hooks/usePosts";
import { pickSeed } from "@lib/seed";
import { useBreakpoint } from "@providers/breakpoint";
import { useMemo, useState } from "react";
import FilterBar from "./filterBar";
import Footer from "./footer";
import styles from "./gallery.module.css";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import ScrollTop from "./scrollTop";

export default function Gallery() {
  const { filters, setFilters, limits, ...posts } = usePosts();

  const [filmMenuOpened, setFilmMenuOpened] = useState(false);
  const [cameraMenuOpened, setCameraMenuOpened] = useState(false);

  const { data: filmsResponse } = useFilms(500, filmMenuOpened);
  const { data: camerasResponse } = useCameras(500, cameraMenuOpened);

  const breakpoints = useBreakpoint();

  const onlyIcon = breakpoints["xs"] || breakpoints["sm"];
  const textPlaceholder = "search pictures...";

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

  const cameraOptions = useMemo(() => {
    if (!camerasResponse?.cameras) return [];

    return camerasResponse.cameras
      .filter((f) => f.make && f.model)
      .map((f) => ({
        make: f.make,
        model: f.model,
        label: `${f.make} - ${f.model}`,
      }))
      .filter((v, i, arr) => arr.findIndex((x) => x.label === v.label) === i);
  }, [camerasResponse]);

  const setSort = (sort) => {
    if (sort === "random") {
      setFilters({ sort: sort, seed: pickSeed() });
    } else {
      setFilters({ sort: sort, seed: null });
    }
  };

  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.margin}>
        <FilterBar
          sort={filters.sort}
          nsfw={filters.nsfw}
          bw={filters.bw}
          sprocket={filters.sprocket}
          color={filters.color}
          text={filters.text}
          widthMin={filters.widthMin}
          widthMax={filters.widthMax}
          heightMin={filters.heightMin}
          heightMax={filters.heightMax}
          ratioMin={filters.ratioMin}
          ratioMax={filters.ratioMax}
          filmMake={filters.film_make}
          filmType={filters.film_type}
          cameraMake={filters.camera_make}
          cameraModel={filters.camera_model}
          setSort={setSort}
          setNsfw={(nsfw) => setFilters({ nsfw })}
          setBw={(bw) => setFilters({ bw })}
          setSprocket={(sprocket) => setFilters({ sprocket })}
          setColor={(color) => setFilters({ color })}
          setText={(text) => setFilters({ text: text || null })}
          setSizes={(sizes) => setFilters(sizes)}
          setFilm={(make, type) =>
            setFilters({ film_make: make, film_type: type })
          }
          setCamera={(make, model) =>
            setFilters({ camera_make: make, camera_model: model })
          }
          onFilmMenuOpen={() => setFilmMenuOpened(true)}
          onCameraMenuOpen={() => setCameraMenuOpened(true)}
          filmOptions={filmOptions}
          cameraOptions={cameraOptions}
          onlyIcon={onlyIcon}
          textPlaceholder={textPlaceholder}
          widthMinLimit={limits.widthMin}
          widthMaxLimit={limits.widthMax}
          heightMinLimit={limits.heightMin}
          heightMaxLimit={limits.heightMax}
          ratioMinLimit={limits.ratioMin}
          ratioMaxLimit={limits.ratioMax}
        />
        <InfiniteGallery {...posts} />
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
