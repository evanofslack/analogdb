// EventMap mirrors eventSchemas in backend/server/eventschema.go
export type EventMap = {
  page_view: Record<string, never>;
  web_vital: {
    metric: "LCP" | "INP" | "CLS" | "FCP" | "TTFB";
    value: number;
    rating: string;
  };
  search: {
    q_norm: string;
    mode: SearchMode;
    source: SearchOrigin;
    filters: Record<string, string>;
    result_count: number;
    outcome: "ok" | "empty" | "limited" | "error";
    page: number;
  };
  search_result_click: {
    position: number;
    post_id: number;
    search_id: string;
  };
  post_view: {
    from: "gallery" | "search" | "similar" | "catalog" | "keyword" | "direct";
    post_id: number;
    search_id?: string;
  };
  post_download: {
    post_id: number;
    search_id?: string;
  };
  outbound_click: {
    post_id: number;
    target: "original" | "reddit" | "reddit_user";
  };
  similar_click: {
    from_post_id: number;
    post_id: number;
    position: number;
  };
};

export type SearchMode = "text" | "similar" | "image";

export type SearchOrigin =
  | "typed"
  | "suggestion"
  | "recent"
  | "chip"
  | "related"
  | "example"
  | "upload"
  | "post";

type SearchResult = Omit<EventMap["search"], "q_norm" | "source"> & {
  key: string;
  q: string;
  source?: SearchOrigin;
};

type LastSearch = {
  id: string;
  key: string;
  mode: SearchMode;
  qNorm: string;
  source: SearchOrigin;
  pages: number[];
  at: number;
};

type Extra = { postId?: number; searchId?: string };

type Utm = { source: string; medium: string; campaign: string };

type UiEvent = {
  id: string;
  name: string;
  v: number;
  ts: number;
  path: string;
  route: string;
  referrer: string;
  utm: Utm;
  vw: number;
  search_id: string;
  post_id: number;
  props: Record<string, unknown>;
};

const endpoint = "/api/e";
const maxQueue = 25;
const flushAt = 10;
const flushAfterMs = 5000;
const searchTtlMs = 30 * 60 * 1000;
const maxQueryLength = 200;

const routes: [RegExp, string][] = [
  [/^\/post\/\d+\/?$/, "/post/[id]"],
  [/^\/films\/[^/]+\/?$/, "/films/[slug]"],
  [/^\/cameras\/[^/]+\/?$/, "/cameras/[slug]"],
  [/^\/search\/keyword\/[^/]+\/?$/, "/search/keyword/[word]"],
];

let queue: UiEvent[] = [];
let timer: ReturnType<typeof setTimeout> | null = null;
let listening = false;
let firstView = true;
let lastPathname: string | null = null;
let previousPathname: string | null = null;
let lastSearch: LastSearch | null = null;
let nextSource: SearchOrigin | null = null;

function routeFor(pathname: string): string {
  for (const [pattern, route] of routes) {
    if (pattern.test(pathname)) return route;
  }
  return pathname;
}

function enabled(): boolean {
  if (typeof window === "undefined") return false;
  const nav = navigator as Navigator & { globalPrivacyControl?: boolean };
  if (nav.doNotTrack === "1" || nav.globalPrivacyControl === true) {
    return false;
  }
  return !location.pathname.startsWith("/admin");
}

function postIdFor(path: string, route: string): number {
  if (route !== "/post/[id]") return 0;
  const id = Number(path.split("/")[2]);
  return Number.isSafeInteger(id) ? id : 0;
}

function referrerHost(): string {
  try {
    if (!document.referrer) return "";
    const host = new URL(document.referrer).host;
    return host === location.host ? "" : host;
  } catch {
    return "";
  }
}

function utmFromUrl(): Utm {
  const params = new URLSearchParams(location.search);
  return {
    source: params.get("utm_source") ?? "",
    medium: params.get("utm_medium") ?? "",
    campaign: params.get("utm_campaign") ?? "",
  };
}

function send(events: UiEvent[]) {
  const body = JSON.stringify({ events });
  try {
    if (navigator.sendBeacon && navigator.sendBeacon(endpoint, body)) return;
  } catch {}
  try {
    fetch(endpoint, { method: "POST", body, keepalive: true }).catch(() => {});
  } catch {}
}

function flush() {
  if (timer) {
    clearTimeout(timer);
    timer = null;
  }
  if (queue.length === 0) return;
  const events = queue;
  queue = [];
  send(events);
}

function listen() {
  if (listening) return;
  listening = true;
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") flush();
  });
  window.addEventListener("pagehide", flush);
}

function enqueue(event: UiEvent) {
  listen();
  if (queue.length >= maxQueue) return;
  queue.push(event);
  if (queue.length >= flushAt) {
    flush();
  } else if (!timer) {
    timer = setTimeout(flush, flushAfterMs);
  }
}

function build(
  name: string,
  props: Record<string, unknown>,
  extra: Extra = {},
  first = false
): UiEvent {
  const path = location.pathname;
  const route = routeFor(path);
  return {
    id: crypto.randomUUID(),
    name,
    v: 1,
    ts: Date.now(),
    path,
    route,
    referrer: first ? referrerHost() : "",
    utm: first ? utmFromUrl() : { source: "", medium: "", campaign: "" },
    vw: window.innerWidth,
    search_id: extra.searchId ?? "",
    post_id: extra.postId ?? postIdFor(path, route),
    props,
  };
}

export function track<K extends keyof EventMap>(
  name: K,
  props: EventMap[K],
  extra?: Extra
): void {
  try {
    if (!enabled()) return;
    enqueue(build(name, props, extra));
  } catch {}
}

export function pageView(pathname: string): void {
  try {
    if (!enabled() || pathname === lastPathname) return;
    previousPathname = lastPathname;
    lastPathname = pathname;
    enqueue(build("page_view", {}, {}, firstView));
    firstView = false;
  } catch {}
}

function normalizeQuery(q: string): string {
  return q.trim().toLowerCase().replace(/\s+/g, " ").slice(0, maxQueryLength);
}

export function setSearchSource(source: SearchOrigin): void {
  nextSource = source;
}

export function currentSearchId(): string {
  if (!lastSearch || Date.now() - lastSearch.at > searchTtlMs) return "";
  return lastSearch.id;
}

// Route the visitor came from, whether or not the page view for the current
// path has been recorded yet
export function previousRoute(): string {
  try {
    const path =
      location.pathname === lastPathname ? previousPathname : lastPathname;
    return path ? routeFor(path) : "";
  } catch {
    return "";
  }
}

// Sends one search event per search and page. A search is identified by key,
// so re-renders and returning to the same results reuse it
export function trackSearch(result: SearchResult): void {
  try {
    if (!enabled()) return;
    const now = Date.now();
    const qNorm = normalizeQuery(result.q);
    let search = lastSearch;
    if (!search || search.key !== result.key || !currentSearchId()) {
      const refined =
        search && search.mode === result.mode && search.qNorm === qNorm;
      const source =
        result.source ?? nextSource ?? (refined ? search.source : "typed");
      nextSource = null;
      search = {
        id: crypto.randomUUID(),
        key: result.key,
        mode: result.mode,
        qNorm,
        source,
        pages: [],
        at: now,
      };
      lastSearch = search;
    }
    if (search.pages.indexOf(result.page) !== -1) return;
    search.pages.push(result.page);
    search.at = now;
    track(
      "search",
      {
        q_norm: qNorm,
        mode: result.mode,
        source: search.source,
        filters: result.filters,
        result_count: result.result_count,
        outcome: result.outcome,
        page: result.page,
      },
      { searchId: search.id }
    );
  } catch {}
}
