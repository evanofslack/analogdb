"use client";

import { getKeywordCatalog } from "@app/actions/keywords";
import DisplayMenu from "@components/displayMenu";
import filterStyles from "@components/filterBar.module.css";
import Footer from "@components/footer";
import galleryStyles from "@components/gallery.module.css";
import Header from "@components/header";
import InfiniteGallery from "@components/infiniteGallery";
import KeywordRow from "@components/keywordRow";
import ScrollTop from "@components/scrollTop";
import SearchBar, { SearchPanel, SearchVisual } from "@components/searchBar";
import {
  ResultsSkeleton,
  SuggestionsSkeleton,
} from "@components/searchSkeletons";
import SearchSuggestions, { KeywordChips } from "@components/searchSuggestions";
import { DisplayProvider, InitialDisplay } from "@hooks/useDisplay";
import useRecentSearches from "@hooks/useRecentSearches";
import useSearch, { useSearchSource, useSimilar } from "@hooks/useSearch";
import {
  currentSearchId,
  EventMap,
  SearchMode,
  SearchOrigin,
  setSearchSource,
  track,
  trackSearch,
} from "@lib/analytics";
import type { CatalogEntry } from "@lib/catalog";
import type {
  SearchSource,
  SearchSuggestions as Suggestions,
} from "@lib/data/search";
import {
  imageAccept,
  imageMaxSize,
  postImageSearch,
  rejectMessage,
} from "@lib/imageSearch";
import { RateLimitedError, retryUnlessLimited, unwrap } from "@lib/rateLimited";
import { resizeImage } from "@lib/resizeImage";
import {
  SearchFilters,
  SearchFlags,
  searchKey,
  searchParsers,
  toSearchFlags,
} from "@lib/searchParams";
import { Button, Menu, SegmentedControl } from "@mantine/core";
import { Dropzone, FileRejection, FileWithPath } from "@mantine/dropzone";
import { IconAdjustmentsHorizontal, IconPhotoScan } from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import {
  AnalogdbPost,
  ServerSearchResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";
import { useQueryStates } from "nuqs";
import {
  ReactNode,
  Suspense,
  use,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import styles from "./search.module.css";

const filterData = [
  { label: "exclude", value: "exclude" },
  { label: "include", value: "include" },
  { label: "only", value: "only" },
];

const maxChips = 8;

type SearchOutcome = EventMap["search"]["outcome"];

let imageSearches = 0;

type ResultsPage = ServerSearchResponse | ServerSimilarPostsResponse;

interface ResultsData {
  pages: ResultsPage[];
  isLoading: boolean;
  isError: boolean;
  error: Error | null;
  isPlaceholderData: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  isFetchNextPageError: boolean;
  fetchNextPage: () => void;
  refetch: () => void;
}

interface Upload {
  blob: Blob;
  thumb: string;
  flagsKey: string;
  response: ServerSearchResponse;
}

function similarKeywords(
  pages: ResultsPage[],
  grayscaleOnly: boolean
): string[] {
  const weights = new Map<string, number>();
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

function trackResultClick(post: AnalogdbPost, position: number) {
  const searchId = currentSearchId();
  track(
    "search_result_click",
    { position, post_id: post.id, search_id: searchId },
    { postId: post.id, searchId }
  );
}

function outcomeFor(count: number, page: number): SearchOutcome {
  return count > 0 || page > 0 ? "ok" : "empty";
}

function failureFor(error: unknown): SearchOutcome {
  return error instanceof RateLimitedError ? "limited" : "error";
}

function useSearchEvents(
  key: string,
  mode: SearchMode,
  q: string,
  filters: Record<string, string>,
  results: ResultsData
) {
  const {
    pages,
    isLoading,
    isError,
    error,
    isPlaceholderData,
    isFetchNextPageError,
  } = results;

  const sent = useRef("");

  useEffect(() => {
    if (isLoading || isPlaceholderData) return;
    const failed = isError && (pages.length === 0 || isFetchNextPageError);
    const signature = `${key}:${pages.length}:${failed}`;
    if (sent.current === signature) return;
    sent.current = signature;
    const search = { key, mode, q, filters };
    pages.forEach((page, index) => {
      const count = (page.posts ?? []).length;
      trackSearch({
        ...search,
        page: index,
        result_count: count,
        outcome: outcomeFor(count, index),
      });
    });
    if (failed) {
      trackSearch({
        ...search,
        page: pages.length,
        result_count: 0,
        outcome: failureFor(error),
      });
    }
  }, [
    key,
    mode,
    q,
    filters,
    pages,
    isLoading,
    isError,
    error,
    isPlaceholderData,
    isFetchNextPageError,
  ]);
}

async function searchImageTracked(
  blob: Blob,
  flags: SearchFlags,
  filters: Record<string, string>
): Promise<ServerSearchResponse> {
  imageSearches += 1;
  const search = {
    key: `image:${imageSearches}`,
    mode: "image" as const,
    q: "",
    filters,
    source: "upload" as const,
    page: 0,
  };
  try {
    const response = await postImageSearch(blob, flags);
    const count = (response.posts ?? []).length;
    trackSearch({
      ...search,
      result_count: count,
      outcome: outcomeFor(count, 0),
    });
    return response;
  } catch (error) {
    trackSearch({ ...search, result_count: 0, outcome: failureFor(error) });
    throw error;
  }
}

interface FlagsMenuProps {
  filters: SearchFilters;
  setFilters: (values: Partial<SearchFilters>) => void;
}

function FlagsMenu({ filters, setFilters }: FlagsMenuProps) {
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
              onChange={(nsfw) =>
                setFilters({ nsfw: nsfw as SearchFilters["nsfw"] })
              }
              data={filterData}
            />
          </div>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>b&w</h5>
            <SegmentedControl
              value={filters.bw}
              onChange={(bw) => setFilters({ bw: bw as SearchFilters["bw"] })}
              data={filterData}
            />
          </div>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>sprocket</h5>
            <SegmentedControl
              value={filters.sprocket}
              onChange={(sprocket) =>
                setFilters({ sprocket: sprocket as SearchFilters["sprocket"] })
              }
              data={filterData}
            />
          </div>
        </div>
      </Menu.Dropdown>
    </Menu>
  );
}

interface ResultsProps {
  results: ResultsData;
  chips: string[];
  onSearch: (query: string, source?: SearchOrigin) => void;
  noMatches?: ReactNode;
}

function Results({ results, chips, onSearch, noMatches }: ResultsProps) {
  return (
    <>
      <KeywordRow
        words={chips}
        onSelect={(word) => onSearch(word, "related")}
        label="related keywords"
        className={styles.related}
      />
      {noMatches}
      {!noMatches && (
        <InfiniteGallery {...results} onClick={trackResultClick} />
      )}
    </>
  );
}

interface EmptyBodyProps {
  suggestionsPromise: Promise<Suggestions>;
  catalogPromise: Promise<CatalogEntry[]> | null;
  recent: string[];
  onClearRecent: () => void;
  onSearch: (query: string, source?: SearchOrigin) => void;
}

function EmptyBody({
  suggestionsPromise,
  catalogPromise,
  recent,
  onClearRecent,
  onSearch,
}: EmptyBodyProps) {
  const suggestions = use(suggestionsPromise);
  const initialCatalog = catalogPromise ? use(catalogPromise) : null;
  const catalog = useQuery({
    queryKey: ["keyword-catalog"],
    queryFn: async () => unwrap(await getKeywordCatalog()),
    initialData: initialCatalog ?? undefined,
    staleTime: Infinity,
    retry: retryUnlessLimited,
  });

  return (
    <div className={styles.suggestions}>
      <SearchSuggestions
        suggestions={suggestions}
        recent={recent}
        onClearRecent={onClearRecent}
        onSearch={onSearch}
        catalog={catalog.data}
        trendingRow
      />
    </div>
  );
}

interface TextResultsProps {
  q: string;
  flags: SearchFlags;
  filters: Record<string, string>;
  textPromise: Promise<ServerSearchResponse | null> | null;
  initialKey: string | null;
  suggestions?: Suggestions;
  onSearch: (query: string, source?: SearchOrigin) => void;
}

function TextResults({
  q,
  flags,
  filters,
  textPromise,
  initialKey,
  suggestions,
  onSearch,
}: TextResultsProps) {
  const initialPage = textPromise ? use(textPromise) : null;
  const results = useSearch(q, flags, initialPage, initialKey);
  useSearchEvents(searchKey("q", q, flags), "text", q, filters, results);

  const empty =
    !results.isLoading &&
    !results.isError &&
    !results.isPlaceholderData &&
    results.pages.every((page) => (page.posts ?? []).length === 0);

  return (
    <Results
      results={results}
      chips={results.relatedKeywords}
      onSearch={onSearch}
      noMatches={
        empty && (
          <div className={styles.noMatches}>
            <h3 className={styles.noMatchesTitle}>
              no matches for &ldquo;{q}&rdquo;
            </h3>
            <KeywordChips
              words={suggestions?.keywords}
              onSearch={(word) => onSearch(word, "suggestion")}
              className={styles.noMatchesChips}
            />
          </div>
        )
      }
    />
  );
}

interface SimilarResultsProps {
  similar: number;
  flags: SearchFlags;
  filters: Record<string, string>;
  similarPromise: Promise<ServerSimilarPostsResponse | null> | null;
  initialKey: string | null;
  grayscaleOnly: boolean;
  onSearch: (query: string, source?: SearchOrigin) => void;
}

function SimilarResults({
  similar,
  flags,
  filters,
  similarPromise,
  initialKey,
  grayscaleOnly,
  onSearch,
}: SimilarResultsProps) {
  const initialPage = similarPromise ? use(similarPromise) : null;
  const results = useSimilar(similar, flags, initialPage, initialKey);
  useSearchEvents(
    searchKey("similar", similar, flags),
    "similar",
    "",
    filters,
    results
  );

  return (
    <Results
      results={results}
      chips={similarKeywords(results.pages, grayscaleOnly)}
      onSearch={onSearch}
    />
  );
}

interface SearchPageProps {
  suggestionsPromise: Promise<Suggestions>;
  catalogPromise: Promise<CatalogEntry[]> | null;
  textPromise: Promise<ServerSearchResponse | null> | null;
  similarPromise: Promise<ServerSimilarPostsResponse | null> | null;
  sourcePromise: Promise<SearchSource | null> | null;
  sourceId: number | null;
  initialKey: string | null;
  initialDisplay: InitialDisplay;
}

export default function SearchPage({
  suggestionsPromise,
  catalogPromise,
  textPromise,
  similarPromise,
  sourcePromise,
  sourceId,
  initialKey,
  initialDisplay,
}: SearchPageProps) {
  const [filters, setFilters] = useQueryStates(searchParsers);
  const { nsfw, bw, sprocket } = filters;
  const flags = useMemo(
    () => toSearchFlags({ nsfw, bw, sprocket }),
    [nsfw, bw, sprocket]
  );
  const searchFilters = useMemo(
    () => ({ nsfw, bw, sprocket }),
    [nsfw, bw, sprocket]
  );
  const flagsKey = JSON.stringify(flags);
  const q = filters.q?.trim() || null;
  const similar = !q && filters.similar > 0 ? filters.similar : null;

  const { recent, add, clear } = useRecentSearches();
  const [panel, setPanel] = useState<SearchPanel>(null);
  const [upload, setUpload] = useState<Upload | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadPending, setUploadPending] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const { data: suggestions } = useQuery({
    queryKey: ["search-suggestions"],
    queryFn: () => suggestionsPromise,
    staleTime: Infinity,
  });
  const source = useSearchSource(similar, sourceId, sourcePromise);

  let mode = "empty";
  if (q) mode = "text";
  else if (similar) mode = "similar";
  else if (upload) mode = "upload";

  const replaceUpload = (next: Upload | null) =>
    setUpload((previous) => {
      if (previous && previous.thumb !== next?.thumb) {
        URL.revokeObjectURL(previous.thumb);
      }
      return next;
    });

  const showError = (message: string) => {
    setUploadError(message);
    setPanel("visual");
  };

  const searchImage = async (file: File) => {
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
      const response = await searchImageTracked(blob, flags, searchFilters);
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

  const handleSearch = (query: string, source: SearchOrigin = "typed") => {
    setSearchSource(source);
    add(query);
    setPanel(null);
    setFilters({ q: query, similar: null }, { history: "push" });
    window.scrollTo({ top: 0 });
  };

  const handleExample = (id: number) => {
    setSearchSource("example");
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
    searchImageTracked(upload.blob, flags, searchFilters)
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
  }, [upload, flags, flagsKey, searchFilters]);

  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
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

  let visual: SearchVisual | null = null;
  let body: ReactNode = null;
  if (mode === "empty") {
    body = (
      <Suspense key="empty" fallback={<SuggestionsSkeleton />}>
        <EmptyBody
          suggestionsPromise={suggestionsPromise}
          catalogPromise={catalogPromise}
          recent={recent}
          onClearRecent={clear}
          onSearch={handleSearch}
        />
      </Suspense>
    );
  } else if (mode === "text") {
    body = (
      <Suspense
        key="text"
        fallback={<ResultsSkeleton className={styles.related} />}
      >
        <TextResults
          q={q}
          flags={flags}
          filters={searchFilters}
          textPromise={textPromise}
          initialKey={initialKey}
          suggestions={suggestions}
          onSearch={handleSearch}
        />
      </Suspense>
    );
  } else if (mode === "similar") {
    visual = {
      thumb: source?.image?.url,
      label: `similar to #${similar}`,
      onClear: () => setFilters({ similar: null }, { history: "push" }),
    };
    body = (
      <Suspense
        key="similar"
        fallback={<ResultsSkeleton className={styles.related} />}
      >
        <SimilarResults
          similar={similar}
          flags={flags}
          filters={searchFilters}
          similarPromise={similarPromise}
          initialKey={initialKey}
          grayscaleOnly={bw === "only"}
          onSearch={handleSearch}
        />
      </Suspense>
    );
  } else if (mode === "upload") {
    visual = {
      thumb: upload.thumb,
      label: "your image",
      onClear: () => replaceUpload(null),
    };
    body = (
      <Results
        results={{
          pages: [upload.response],
          isLoading: false,
          isError: false,
          error: null,
          isPlaceholderData: uploadPending,
          hasNextPage: false,
          isFetchingNextPage: false,
          isFetchNextPageError: false,
          fetchNextPage: () => {},
          refetch: () => {},
        }}
        chips={upload.response.relatedKeywords ?? []}
        onSearch={handleSearch}
      />
    );
  }

  return (
    <DisplayProvider value={initialDisplay}>
      <div className={galleryStyles.main}>
        <Header />
        <div className={galleryStyles.margin}>
          <Dropzone.FullScreen
            active={panel !== "visual" && !uploading}
            accept={imageAccept}
            maxSize={imageMaxSize}
            multiple={false}
            onDrop={(files: FileWithPath[]) =>
              files[0] && searchImage(files[0])
            }
            onReject={(rejections: FileRejection[]) =>
              showError(rejectMessage(rejections))
            }
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
            <DisplayMenu
              buttonClassName={styles.filterButton}
              labelClassName={styles.filterLabel}
              position="bottom-end"
            />
          </div>

          {body}
          <ScrollTop />
        </div>
        <Footer />
      </div>
    </DisplayProvider>
  );
}
