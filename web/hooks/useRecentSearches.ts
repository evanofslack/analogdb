import {
  clearRecentSearches,
  readRecentSearches,
  saveRecentSearch,
} from "@lib/recentSearches";
import { useCallback, useEffect, useState } from "react";

const changeEvent = "analogdb:recent-searches";

export default function useRecentSearches() {
  const [recent, setRecent] = useState<string[]>([]);

  useEffect(() => {
    const update = () => setRecent(readRecentSearches());
    update();
    window.addEventListener(changeEvent, update);
    window.addEventListener("storage", update);
    return () => {
      window.removeEventListener(changeEvent, update);
      window.removeEventListener("storage", update);
    };
  }, []);

  const add = useCallback((query: string) => {
    saveRecentSearch(query);
    window.dispatchEvent(new Event(changeEvent));
  }, []);

  const clear = useCallback(() => {
    clearRecentSearches();
    window.dispatchEvent(new Event(changeEvent));
  }, []);

  return { recent, add, clear };
}
