export const keywordLinkMode: "page" | "search" = "page";

export function keywordPageHref(word: string): string {
  return `/search/keyword/${encodeURIComponent(word)}`;
}

export function keywordSearchHref(word: string): string {
  return `/search?q=${encodeURIComponent(word)}`;
}

export function keywordHref(word: string): string {
  return keywordLinkMode === "page"
    ? keywordPageHref(word)
    : keywordSearchHref(word);
}

export function keywordLabel(word: string): string {
  return word.toUpperCase();
}
