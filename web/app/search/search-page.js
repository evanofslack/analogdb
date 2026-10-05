"use client";

import { getKeywordCatalog } from "@app/actions/keywords";
import filterStyles from "@components/filterBar.module.css";
import Footer from "@components/footer";
import galleryStyles from "@components/gallery.module.css";
import Header from "@components/header";
import InfiniteGallery from "@components/infiniteGallery";
import ScrollTop from "@components/scrollTop";
import SearchBar from "@components/searchBar";
import SearchSuggestions, { KeywordChips } from "@components/searchSuggestions";
import useRecentSearches from "@hooks/useRecentSearches";
import useSearch, { useSimilar } from "@hooks/useSearch";
import {
  imageAccept,
  imageMaxSize,
  postImageSearch,
  rejectMessage,
} from "@lib/imageSearch";
import { resizeImage } from "@lib/resizeImage";
import { searchParsers, toSearchFlags } from "@lib/searchParams";
import { Button, Menu, SegmentedControl } from "@mantine/core";
import { Dropzone } from "@mantine/dropzone";
import { IconAdjustmentsHorizontal, IconPhotoScan } from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useEffect, useMemo, useRef, useState } from "react";
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
  keywordCatalog,
  initialPage,
  initialSimilar,
  initialSource,
  initialKey,
  initialColumns,
}) {
  const [filters, setFilters] = useQueryStates(searchParsers);
  const { nsfw, bw, sprocket } = filters;
  const flags = useMemo(
    () => toSearchFlags({ nsfw, bw, sprocket }),
    [nsfw, bw, sprocket]
  );
  const flagsKey = JSON.stringify(flags);
  const q = filters.q?.trim() || null;
  const similar = !q && filters.similar > 0 ? filters.similar : null;

  const { recent, add, clear } = useRecentSearches();
  const [panel, setPanel] = useState(null);
  const [upload, setUpload] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [uploadPending, setUploadPending] = useState(false);
  const [uploadError, setUploadError] = useState(null);

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
  else if (upload) mode = "upload";

  const replaceUpload = (next) =>
    setUpload((previous) => {
      if (previous && previous.thumb !== next?.thumb) {
        URL.revokeObjectURL(previous.thumb);
      }
      return next;
    });

  const showError = (message) => {
    setUploadError(message);
    setPanel("visual");
  };

  const searchImage = async (file) => {
    if (!imageAccept.includes(file.type)) {
      showError("use a JPEG, PNG or WebP image");
      return;
    }
    if (file.size > imageMaxSize) {
      showError("that image is over 10 MB");
      return;
    }
    setPanel("visual");
    setUploading(true);
    setUploadError(null);
    try {
      const blob = await resizeImage(file);
      const response = await postImageSearch(blob, flags);
      replaceUpload({
        blob,
        thumb: URL.createObjectURL(blob),
        flagsKey,
        response,
      });
      if (q || similar) {
        setFilters({ q: null, similar: null }, { history: "push" });
      }
      setPanel(null);
      window.scrollTo({ top: 0 });
    } catch (error) {
      setUploadError(error.message || "image search failed");
    } finally {
      setUploading(false);
    }
  };

  const searchImageRef = useRef(searchImage);
  searchImageRef.current = searchImage;

  const handleSearch = (query) => {
    add(query);
    setPanel(null);
    setFilters({ q: query, similar: null }, { history: "push" });
    window.scrollTo({ top: 0 });
  };

  const handleExample = (id) => {
    setPanel(null);
    setUploadError(null);
    setFilters({ q: null, similar: id }, { history: "push" });
    window.scrollTo({ top: 0 });
  };

  useEffect(() => {
    if (!q && !similar) return;
    setUpload((previous) => {
      if (previous) URL.revokeObjectURL(previous.thumb);
      return null;
    });
  }, [q, similar]);

  useEffect(() => {
    if (!upload || upload.flagsKey === flagsKey) return;
    let active = true;
    setUploadPending(true);
    postImageSearch(upload.blob, flags)
      .then((response) => {
        if (active)
          setUpload((prev) => prev && { ...prev, flagsKey, response });
      })
      .catch(() => {
        if (active) setUpload((prev) => prev && { ...prev, flagsKey });
      })
      .finally(() => {
        if (active) setUploadPending(false);
      });
    return () => {
      active = false;
    };
  }, [upload, flags, flagsKey]);

  useEffect(() => {
    const onPaste = (event) => {
      const item = Array.from(event.clipboardData?.items ?? []).find(
        (entry) => entry.kind === "file" && entry.type.startsWith("image/")
      );
      const file = item?.getAsFile();
      if (!file) return;
      event.preventDefault();
      searchImageRef.current(file);
    };
    document.addEventListener("paste", onPaste);
    return () => document.removeEventListener("paste", onPaste);
  }, []);

  let visual = null;
  let results = null;
  let chips = [];
  if (mode === "text") {
    results = text;
    chips = text.relatedKeywords;
  } else if (mode === "similar") {
    visual = {
      thumb: similarResults.source?.image?.url,
      label: `similar to #${similar}`,
      onClear: () => setFilters({ similar: null }, { history: "push" }),
    };
    results = similarResults;
    chips = similarKeywords(similarResults.pages, bw === "only");
  } else if (mode === "upload") {
    visual = {
      thumb: upload.thumb,
      label: "your image",
      onClear: () => replaceUpload(null),
    };
    results = {
      pages: [upload.response],
      isLoading: false,
      isError: false,
      isPlaceholderData: uploadPending,
      hasNextPage: false,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage: () => {},
      refetch: () => {},
    };
    chips = upload.response.relatedKeywords ?? [];
  }

  const catalog = useQuery({
    queryKey: ["keyword-catalog"],
    queryFn: () => getKeywordCatalog(),
    initialData: keywordCatalog ?? undefined,
    staleTime: Infinity,
    enabled: mode === "empty",
  });

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
          active={panel !== "visual" && !uploading}
          accept={imageAccept}
          maxSize={imageMaxSize}
          multiple={false}
          onDrop={(files) => files[0] && searchImage(files[0])}
          onReject={(rejections) => showError(rejectMessage(rejections))}
        >
          <div className={styles.fullscreen}>
            <IconPhotoScan size={48} stroke={1.5} />
            <p>drop an image to search</p>
          </div>
        </Dropzone.FullScreen>

        <div className={styles.top}>
          <div className={styles.bar}>
            <SearchBar
              value={q ?? ""}
              onSearch={handleSearch}
              visual={visual}
              suggestions={suggestions}
              recent={recent}
              onClearRecent={clear}
              panel={panel}
              onPanelChange={setPanel}
              suggestionsDropdown={mode !== "empty"}
              visualProps={{
                loading: uploading,
                error: uploadError,
                onImage: searchImage,
                onError: setUploadError,
                onExample: handleExample,
              }}
            />
          </div>
          <FlagsMenu filters={filters} setFilters={setFilters} />
        </div>

        {mode === "empty" ? (
          <div className={styles.suggestions}>
            <SearchSuggestions
              suggestions={suggestions}
              recent={recent}
              onClearRecent={clear}
              onSearch={handleSearch}
              catalog={catalog.data}
            />
          </div>
        ) : (
          <KeywordChips
            words={chips}
            onSearch={handleSearch}
            className={styles.related}
          />
        )}
        {noMatches && (
          <div className={styles.noMatches}>
            <h3 className={styles.noMatchesTitle}>
              no matches for &ldquo;{q}&rdquo;
            </h3>
            <KeywordChips
              words={suggestions?.keywords}
              onSearch={handleSearch}
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
