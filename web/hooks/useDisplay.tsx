"use client";

import {
  defaultDisplay,
  Display,
  displayCookie,
  parseDisplay,
  resolveColumns,
  serializeDisplay,
  WidthClass,
  widthQueries,
} from "@lib/display";
import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useSyncExternalStore,
} from "react";

export interface InitialDisplay {
  display: Display;
  width: WidthClass;
}

const DisplayContext = createContext<InitialDisplay>({
  display: defaultDisplay,
  width: "desktop",
});

export function DisplayProvider({
  value,
  children,
}: {
  value?: InitialDisplay;
  children: ReactNode;
}) {
  if (!value) return children;
  return (
    <DisplayContext.Provider value={value}>{children}</DisplayContext.Provider>
  );
}

const listeners = new Set<() => void>();
let cachedRaw: string | null | undefined;
let cached: Display = defaultDisplay;

function readCookie(): string | null {
  const prefix = `${displayCookie}=`;
  const found = document.cookie
    .split("; ")
    .find((part) => part.startsWith(prefix));
  return found ? decodeURIComponent(found.slice(prefix.length)) : null;
}

function displaySnapshot(): Display {
  const raw = readCookie();
  if (raw !== cachedRaw) {
    cachedRaw = raw;
    cached = parseDisplay(raw);
  }
  return cached;
}

function subscribeDisplay(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

type Anchor = { id: string; top: number };
let anchor: Anchor | null = null;

// The topmost image in view, so a layout change can keep it in place
function findAnchor(): Anchor | null {
  if (window.scrollY < 50) return null;
  let best: Anchor | null = null;
  document.querySelectorAll<HTMLElement>("[data-post-id]").forEach((el) => {
    const rect = el.getBoundingClientRect();
    if (rect.bottom <= 0 || rect.top >= window.innerHeight) return;
    if (!best || rect.top < best.top) {
      best = { id: el.dataset.postId!, top: rect.top };
    }
  });
  return best;
}

export function takeAnchor(): Anchor | null {
  const taken = anchor;
  anchor = null;
  return taken;
}

function writeDisplay(display: Display) {
  anchor = findAnchor();
  const year = 60 * 60 * 24 * 365;
  document.cookie = `${displayCookie}=${encodeURIComponent(
    serializeDisplay(display)
  )}; path=/; max-age=${year}; samesite=lax`;
  listeners.forEach((listener) => listener());
}

function subscribeWidth(listener: () => void) {
  const queries = Object.values(widthQueries).map((q) => window.matchMedia(q));
  queries.forEach((q) => q.addEventListener("change", listener));
  return () =>
    queries.forEach((q) => q.removeEventListener("change", listener));
}

function widthSnapshot(): WidthClass {
  if (window.matchMedia(widthQueries.mobile).matches) return "mobile";
  if (window.matchMedia(widthQueries.tablet).matches) return "tablet";
  return "desktop";
}

export default function useDisplay() {
  const initial = useContext(DisplayContext);
  const display = useSyncExternalStore(
    subscribeDisplay,
    displaySnapshot,
    () => initial.display
  );
  const width = useSyncExternalStore(
    subscribeWidth,
    widthSnapshot,
    () => initial.width
  );
  const setDisplay = useCallback(
    (change: Partial<Display>) =>
      writeDisplay({ ...displaySnapshot(), ...change }),
    []
  );

  return {
    display,
    width,
    columns: resolveColumns(display, width),
    setDisplay,
  };
}
