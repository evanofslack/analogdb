import { check, sleep } from "k6";
import { announce, systemTags, thresholds } from "../lib/config.js";
import { api, json, web } from "../lib/http.js";
import { loadVocab, query } from "../lib/queries.js";

export const options = {
  systemTags,
  scenarios: {
    smoke: { executor: "constant-vus", vus: 1, duration: "1m" },
  },
  thresholds: thresholds(),
};

export function setup() {
  announce();
  const posts = json(api("/posts?page_size=20", "/posts"), "posts");
  return { ids: posts.map((p) => p.id), vocab: loadVocab() };
}

const ok = { "status is 200": (r) => r.status === 200 };

export default function (data) {
  const id = data.ids[__ITER % data.ids.length];

  check(api("/posts?page_size=50", "/posts"), ok);
  check(api("/posts?sort=random&page_size=50", "/posts"), ok);
  check(api(`/post/${id}`, "/post/{id}"), ok);
  check(api(`/post/${id}/similar`, "/post/{id}/similar"), ok);
  check(api("/cameras", "/cameras"), ok);
  check(api("/films", "/films"), ok);
  check(api("/authors", "/authors"), ok);
  check(api("/keywords/summary", "/keywords/summary"), ok);
  check(
    api(`/search?q=${encodeURIComponent(query(data.vocab))}`, "/search"),
    ok
  );

  check(web("/", "web /"), ok);
  check(web(`/post/${id}`, "web /post/{id}"), ok);
  // web limits search to 30 a minute per IP, so only every fifth iteration
  if (__ITER % 5 === 0) {
    check(
      web(`/search?q=${encodeURIComponent(query(data.vocab))}`, "web /search"),
      ok
    );
  }

  sleep(1);
}
