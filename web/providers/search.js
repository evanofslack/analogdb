"use client";

import { getSearchSuggestions } from "@app/actions/search";
import useRecentSearches from "@hooks/useRecentSearches";
import {
  imageAccept,
  imageMaxSize,
  postImageSearch,
  saveImageSearch,
} from "@lib/imageSearchStore";
import { resizeImage } from "@lib/resizeImage";
import { searchParsers, toSearchFlags } from "@lib/searchParams";
import { Modal } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { createLoader, createSerializer } from "nuqs";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

const Search = dynamic(() => import("@components/search"), { ssr: false });

const SearchContext = createContext(null);

const serialize = createSerializer(searchParsers);
const loadSearch = createLoader(searchParsers);

function currentFlags() {
  const { nsfw, bw, sprocket } = loadSearch(
    new URLSearchParams(window.location.search)
  );
  return { nsfw, bw, sprocket };
}

export function useSearchModal() {
  return useContext(SearchContext);
}

export function SearchProvider({ children }) {
  const router = useRouter();
  const { recent, add, clear } = useRecentSearches();

  const [opened, setOpened] = useState(false);
  const [panel, setPanel] = useState("suggestions");
  const [text, setText] = useState("");
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState(null);

  const suggestions = useQuery({
    queryKey: ["search-suggestions"],
    queryFn: () => getSearchSuggestions(),
    staleTime: 60 * 60 * 1000,
    enabled: opened,
  });

  const open = useCallback(
    ({ panel = "suggestions", text = "", error = null } = {}) => {
      setPanel(panel);
      setText(text ?? "");
      setUploadError(error);
      setOpened(true);
    },
    []
  );

  const close = useCallback(() => setOpened(false), []);

  const search = useCallback(
    (query) => {
      add(query);
      setOpened(false);
      router.push(serialize("/search", { q: query, ...currentFlags() }));
    },
    [add, router]
  );

  const findSimilar = useCallback(
    (id) => {
      setOpened(false);
      router.push(serialize("/search", { similar: id, ...currentFlags() }));
    },
    [router]
  );

  const showError = useCallback((message) => {
    setUploadError(message);
    setPanel("visual");
    setOpened(true);
  }, []);

  const searchImage = useCallback(
    async (file) => {
      if (!imageAccept.includes(file.type)) {
        showError("use a JPEG, PNG or WebP image");
        return;
      }
      if (file.size > imageMaxSize) {
        showError("that image is over 10 MB");
        return;
      }
      setPanel("visual");
      setOpened(true);
      setUploading(true);
      setUploadError(null);
      try {
        const flagFilters = currentFlags();
        const flags = toSearchFlags(flagFilters);
        const blob = await resizeImage(file);
        const response = await postImageSearch(blob, flags);
        const token = saveImageSearch({
          blob,
          thumb: URL.createObjectURL(blob),
          flagsKey: JSON.stringify(flags),
          response,
        });
        setOpened(false);
        router.push(serialize("/search", { image: token, ...flagFilters }));
      } catch (error) {
        setUploadError(error.message || "image search failed");
      } finally {
        setUploading(false);
      }
    },
    [router, showError]
  );

  const searchImageRef = useRef(searchImage);
  searchImageRef.current = searchImage;

  useEffect(() => {
    if (!opened) return;
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
  }, [opened]);

  const value = useMemo(
    () => ({ opened, open, close, search, searchImage, showError }),
    [opened, open, close, search, searchImage, showError]
  );

  return (
    <SearchContext.Provider value={value}>
      {children}
      <Modal
        opened={opened}
        onClose={close}
        size="xl"
        title="search"
        overlayProps={{ backgroundOpacity: 0.4, blur: 2 }}
      >
        <Search
          text={text}
          panel={panel}
          setPanel={setPanel}
          suggestions={suggestions.data}
          recent={recent}
          onClearRecent={clear}
          onSearch={search}
          visualProps={{
            loading: uploading,
            error: uploadError,
            onImage: searchImage,
            onError: setUploadError,
            onExample: findSimilar,
          }}
        />
      </Modal>
    </SearchContext.Provider>
  );
}
