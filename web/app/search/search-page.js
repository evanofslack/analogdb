"use client";

import filterStyles from "@components/filterBar.module.css";
import Footer from "@components/footer";
import galleryStyles from "@components/gallery.module.css";
import Header from "@components/header";
import InfiniteGallery from "@components/infiniteGallery";
import ScrollTop from "@components/scrollTop";
import { SearchBarButton } from "@components/searchBar";
import { KeywordChips } from "@components/searchSuggestions";
import useSearch, { useSimilar } from "@hooks/useSearch";
import {
  getImageSearch,
  imageAccept,
  imageMaxSize,
  postImageSearch,
  rejectMessage,
  updateImageSearch,
} from "@lib/imageSearchStore";
import { searchParsers, toSearchFlags } from "@lib/searchParams";
import { Button, Menu, SegmentedControl } from "@mantine/core";
import { Dropzone } from "@mantine/dropzone";
import { useSearchModal } from "@providers/search";
import { IconAdjustmentsHorizontal, IconPhotoScan } from "@tabler/icons-react";
import { useQueryStates } from "nuqs";
import { useEffect, useMemo, useState } from "react";
import styles from "./search.module.css";

const filterData = [
  { label: "exclude", value: "exclude" },
  { label: "include", value: "include" },
  { label: "only", value: "only" },
];

const maxChips = 8;

function similarKeywords(pages, grayscaleOnly) {
  const weights = new Map();
  for (const post of pages[0]?.posts ?? []) {
    for (const keyword of post.keywords ?? []) {
      if (!keyword.word) continue;
      if (grayscaleOnly && keyword.word === "monochrome") continue;
      weights.set(
        keyword.word,
        (weights.get(keyword.word) ?? 0) + (keyword.weight ?? 1)
      );
    }
  }
  return Array.from(weights.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, maxChips)
    .map(([word]) => word);
}

function FlagsMenu({ filters, setFilters }) {
  return (
    <Menu shadow="md" width={250} position="bottom-end">
      <Menu.Target>
        <Button
          variant="outline"
          color="gray"
          className={styles.filterButton}
          leftSection={<IconAdjustmentsHorizontal size={18} stroke={1.5} />}
          aria-label="filter"
        >
          <span className={styles.filterLabel}>filter</span>
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>filter by</Menu.Label>
        <div className={filterStyles.segment}>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>18+</h5>
            <SegmentedControl
              value={filters.nsfw}
              onChange={(nsfw) => setFilters({ nsfw })}
              data={filterData}
            />
          </div>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>b&w</h5>
            <SegmentedControl
              value={filters.bw}
              onChange={(bw) => setFilters({ bw })}
              data={filterData}
            />
          </div>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>sprocket</h5>
            <SegmentedControl
              value={filters.sprocket}
              onChange={(sprocket) => setFilters({ sprocket })}
              data={filterData}
            />
          </div>
        </div>
      </Menu.Dropdown>
    </Menu>
  );
}

export default function SearchPage({
  suggestions,
  initialPage,
  initialSimilar,
  initialSource,
  initialKey,
  initialColumns,
}) {
  const searchModal = useSearchModal();
  const [filters, setFilters] = useQueryStates(searchParsers);
  const { nsfw, bw, sprocket } = filters;
  const flags = useMemo(
    () => toSearchFlags({ nsfw, bw, sprocket }),
    [nsfw, bw, sprocket]
  );
  const flagsKey = JSON.stringify(flags);
  const q = filters.q?.trim() || null;
  const similar = !q && filters.similar > 0 ? filters.similar : null;
  const token = !q && !similar ? filters.image : null;

  const [image, setImage] = useState(undefined);
  const [imagePending, setImagePending] = useState(false);

  const text = useSearch(q, flags, initialPage, initialKey);
  const similarResults = useSimilar(
    similar,
    flags,
    initialSimilar,
    initialSource,
    initialKey
  );

  let mode = "empty";
  if (q) mode = "text";
  else if (similar) mode = "similar";
  else if (token) mode = "image";

  const { open, showError } = searchModal;

  useEffect(() => {
    setImage(token ? getImageSearch(token) : undefined);
  }, [token]);

  useEffect(() => {
    if (mode === "empty") open();
  }, [mode, open]);

  useEffect(() => {
    if (mode === "image" && image === null) {
      showError("that image is no longer here, upload it again");
    }
  }, [mode, image, showError]);

  useEffect(() => {
    if (!token || !image || image.flagsKey === flagsKey) return;
    let active = true;
    setImagePending(true);
    postImageSearch(image.blob, flags)
      .then((response) => {
        if (!active) return;
        const next = { ...image, flagsKey, response };
        updateImageSearch(token, next);
        setImage(next);
      })
      .catch(() => {
        if (active) setImage({ ...image, flagsKey });
      })
      .finally(() => {
        if (active) setImagePending(false);
      });
    return () => {
      active = false;
    };
  }, [token, image, flags, flagsKey]);

  let bar = { label: q ?? "" };
  let results = null;
  let chips = [];
  if (mode === "text") {
    results = text;
    chips = text.relatedKeywords;
  } else if (mode === "similar") {
    bar = {
      label: `similar to #${similar}`,
      thumb: similarResults.source?.image?.url,
    };
    results = similarResults;
    chips = similarKeywords(similarResults.pages, bw === "only");
  } else if (mode === "image") {
    bar = { label: "your image", thumb: image?.thumb };
    if (image) {
      results = {
        pages: [image.response],
        isLoading: false,
        isError: false,
        isPlaceholderData: imagePending,
        hasNextPage: false,
        isFetchingNextPage: false,
        isFetchNextPageError: false,
        fetchNextPage: () => {},
        refetch: () => {},
      };
      chips = image.response.relatedKeywords ?? [];
    }
  }

  const noMatches =
    mode === "text" &&
    !results.isLoading &&
    !results.isError &&
    !results.isPlaceholderData &&
    results.pages.every((page) => (page.posts ?? []).length === 0);

  return (
    <div className={galleryStyles.main}>
      <Header />
      <div className={galleryStyles.margin}>
        <Dropzone.FullScreen
          active={!searchModal.opened}
          accept={imageAccept}
          maxSize={imageMaxSize}
          multiple={false}
          onDrop={(files) => files[0] && searchModal.searchImage(files[0])}
          onReject={(rejections) => showError(rejectMessage(rejections))}
        >
          <div className={styles.fullscreen}>
            <IconPhotoScan size={48} stroke={1.5} />
            <p>drop an image to search</p>
          </div>
        </Dropzone.FullScreen>

        <div className={styles.top}>
          <div className={styles.bar}>
            <SearchBarButton
              label={bar.label}
              thumb={bar.thumb}
              onOpen={() => open({ text: q ?? "" })}
              onVisual={() => open({ panel: "visual" })}
            />
          </div>
          <FlagsMenu filters={filters} setFilters={setFilters} />
        </div>

        <KeywordChips
          words={chips}
          onSearch={searchModal.search}
          className={styles.related}
        />

        {mode === "image" && image === null && (
          <div className={styles.noMatches}>
            <h3 className={styles.noMatchesTitle}>
              that image is no longer here
            </h3>
            <Button variant="default" onClick={() => open({ panel: "visual" })}>
              upload again
            </Button>
          </div>
        )}
        {noMatches && (
          <div className={styles.noMatches}>
            <h3 className={styles.noMatchesTitle}>
              no matches for &ldquo;{q}&rdquo;
            </h3>
            <KeywordChips
              words={suggestions?.keywords}
              onSearch={searchModal.search}
              className={styles.noMatchesChips}
            />
          </div>
        )}
        {results && !noMatches && (
          <InfiniteGallery {...results} initialColumns={initialColumns} />
        )}
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
