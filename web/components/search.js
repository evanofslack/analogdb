"use client";

import useRecentSearches from "@hooks/useRecentSearches";
import { useRouter } from "next/navigation";
import styles from "./search.module.css";
import SearchBar from "./searchBar";
import SearchSuggestions from "./searchSuggestions";

export default function Search({ text, suggestions, onClose }) {
  const router = useRouter();
  const { recent, add, clear } = useRecentSearches();

  const handleSearch = (query) => {
    add(query);
    onClose();
    router.push(`/search?q=${encodeURIComponent(query)}`);
  };

  const handleVisual = () => {
    onClose();
    router.push("/search?visual=1");
  };

  return (
    <div className={styles.searchContainer}>
      <div className={styles.searchInput}>
        <SearchBar
          value={text}
          onSearch={handleSearch}
          onVisual={handleVisual}
          dropdown={false}
          autoFocus
        />
      </div>
      <SearchSuggestions
        suggestions={suggestions}
        recent={recent}
        onClearRecent={clear}
        onSearch={handleSearch}
      />
    </div>
  );
}
