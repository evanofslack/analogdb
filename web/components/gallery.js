"use client";

import usePosts from "@hooks/usePosts";
import { pickSeed } from "@lib/seed";
import FilterBar from "./filterBar";
import Footer from "./footer";
import styles from "./gallery.module.css";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import ScrollTop from "./scrollTop";

export default function Gallery({
  initialPage,
  initialFilters,
  initialColumns,
  filmOptions,
  cameraOptions,
}) {
  const { filters, setFilters, limits, ...posts } = usePosts(
    initialPage,
    initialFilters
  );

  const textPlaceholder = "search pictures...";

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
          filmOptions={filmOptions}
          cameraOptions={cameraOptions}
          textPlaceholder={textPlaceholder}
          widthMinLimit={limits.widthMin}
          widthMaxLimit={limits.widthMax}
          heightMinLimit={limits.heightMin}
          heightMaxLimit={limits.heightMax}
          ratioMinLimit={limits.ratioMin}
          ratioMaxLimit={limits.ratioMax}
        />
        <InfiniteGallery {...posts} initialColumns={initialColumns} />
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
