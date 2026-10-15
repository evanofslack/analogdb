"use client";

import { DisplayProvider, InitialDisplay } from "@hooks/useDisplay";
import usePosts from "@hooks/usePosts";
import type { CameraOption } from "@lib/data/cameras";
import type { FilmOption } from "@lib/data/films";
import {
  activePostsFilters,
  clearedPostsFilters,
  postsApiUrl,
  PostsFilters,
} from "@lib/searchParams";
import { pickSeed } from "@lib/seed";
import { ServerPostResponse } from "analogdb-generated";
import FilterBar from "./filterBar";
import Footer from "./footer";
import styles from "./gallery.module.css";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import ScrollTop from "./scrollTop";

interface GalleryProps {
  initialPage?: ServerPostResponse | null;
  initialFilters?: PostsFilters;
  initialDisplay?: InitialDisplay;
  filmOptions: FilmOption[];
  cameraOptions: CameraOption[];
}

export default function Gallery({
  initialPage,
  initialFilters,
  initialDisplay,
  filmOptions,
  cameraOptions,
}: GalleryProps) {
  const { filters, setFilters, limits, ...posts } = usePosts(
    initialPage,
    initialFilters
  );

  const textPlaceholder = "search pictures...";
  const active = activePostsFilters(filters);
  const anyActive = Object.values(active).some(Boolean);

  const setSort = (sort: PostsFilters["sort"]) => {
    if (sort === "random") {
      setFilters({ sort: sort, order: null, seed: pickSeed() });
    } else if (sort === "color") {
      setFilters({ sort: sort, order: null, seed: null });
    } else {
      setFilters({ sort: sort, seed: null });
    }
  };

  return (
    <DisplayProvider value={initialDisplay}>
      <div className={styles.main}>
        <Header />
        <div className={styles.margin}>
          <FilterBar
            sort={filters.sort}
            order={filters.order}
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
            setOrder={(order) => setFilters({ order })}
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
            active={active}
            onClear={
              anyActive ? () => setFilters(clearedPostsFilters) : undefined
            }
            resultCount={anyActive ? posts.totalPosts : null}
            resultPending={posts.isPlaceholderData}
            apiUrl={anyActive ? postsApiUrl(filters) : undefined}
          />
          <InfiniteGallery {...posts} />
          <ScrollTop />
        </div>
        <Footer />
      </div>
    </DisplayProvider>
  );
}
