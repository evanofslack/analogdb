import { check } from "k6";
import { Counter } from "k6/metrics";
import { announce, authenticated, systemTags } from "../lib/config.js";
import { api, expectLimits, web } from "../lib/http.js";
import { loadVocab, query } from "../lib/queries.js";

// Checks each limit fires where expected:
//   anonymous API calls get 429 past 60 a minute per IP
//   web search pages stop loading results past 30 a minute per IP
//   the web pair is not limited at a rate far above both
expectLimits();

const webSearchServed = new Counter("web_search_served");
const webSearchLimited = new Counter("web_search_limited");

export const options = {
  systemTags,
  scenarios: {
    anonymous: {
      exec: "anonymous",
      executor: "shared-iterations",
      vus: 2,
      iterations: 75,
      maxDuration: "40s",
    },
    web_search: {
      exec: "webSearch",
      executor: "shared-iterations",
      vus: 2,
      iterations: 40,
      startTime: "45s",
      maxDuration: "40s",
    },
    web_pair: {
      exec: "webPair",
      executor: "constant-arrival-rate",
      rate: 3,
      timeUnit: "1s",
      duration: "1m",
      preAllocatedVUs: 5,
      startTime: "1m30s",
    },
  },
  thresholds: {
    "rate_limited{scenario:anonymous}": ["count>0"],
    web_search_limited: ["count>0"],
    web_search_served: ["count>0", "count<=30"],
    "rate_limited{scenario:web_pair}": ["count==0"],
  },
};

export function setup() {
  announce();
  return { vocab: loadVocab() };
}

function bust() {
  return Math.random().toString(36).slice(2, 10);
}

export function anonymous() {
  const res = api(`/posts?page_size=1&_=${bust()}`, "/posts", { auth: false });
  check(res, { "200 or 429": (r) => r.status === 200 || r.status === 429 });
}

// a page that loaded results streams them with their related keywords, a
// limited page renders without them
export function webSearch(data) {
  const res = web(
    `/search?q=${encodeURIComponent(query(data.vocab))}`,
    "web /search"
  );
  check(res, { "status is 200": (r) => r.status === 200 });
  if (res.status !== 200) return;
  if (res.body.includes("relatedKeywords")) webSearchServed.add(1);
  else webSearchLimited.add(1);
}

export function webPair() {
  if (!authenticated) {
    console.warn("skipping web pair check, K6_USERNAME and K6_PASSWORD unset");
    return;
  }
  const res = api(`/posts?page_size=1&_=${bust()}`, "/posts");
  check(res, { "status is 200": (r) => r.status === 200 });
}
