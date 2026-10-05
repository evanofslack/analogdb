"use client";

import styles from "./search.module.css";
import SearchBar from "./searchBar";
import SearchSuggestions from "./searchSuggestions";
import VisualSearch from "./visualSearch";

export default function Search({
  text,
  panel,
  setPanel,
  suggestions,
  recent,
  onClearRecent,
  onSearch,
  visualProps,
}) {
  const visual = panel === "visual";

  return (
    <div className={styles.searchContainer}>
      <div className={styles.searchInput}>
        <SearchBar
          value={text}
          onSearch={onSearch}
          onVisual={() => setPanel(visual ? "suggestions" : "visual")}
          visualActive={visual}
          onFocus={() => setPanel("suggestions")}
          autoFocus={!visual}
        />
      </div>
      {visual ? (
        <VisualSearch examples={suggestions?.examples} {...visualProps} />
      ) : (
        <SearchSuggestions
          suggestions={suggestions}
          recent={recent}
          onClearRecent={onClearRecent}
          onSearch={onSearch}
        />
      )}
    </div>
  );
}
