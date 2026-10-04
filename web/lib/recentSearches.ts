const storageKey = "analogdb:recent-searches";
const maxRecent = 8;

export function readRecentSearches(): string[] {
  try {
    const stored = JSON.parse(window.localStorage.getItem(storageKey) ?? "[]");
    return Array.isArray(stored)
      ? stored.filter((item) => typeof item === "string").slice(0, maxRecent)
      : [];
  } catch {
    return [];
  }
}

export function saveRecentSearch(query: string): string[] {
  const text = query.trim();
  if (!text) return readRecentSearches();
  const next = [
    text,
    ...readRecentSearches().filter(
      (item) => item.toLowerCase() !== text.toLowerCase()
    ),
  ].slice(0, maxRecent);
  try {
    window.localStorage.setItem(storageKey, JSON.stringify(next));
  } catch {}
  return next;
}

export function clearRecentSearches(): void {
  try {
    window.localStorage.removeItem(storageKey);
  } catch {}
}
