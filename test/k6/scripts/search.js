import { check } from "k6";
import { announce, scenario, systemTags, thresholds } from "../lib/config.js";
import { api, expectLimits } from "../lib/http.js";
import { loadVocab, query } from "../lib/queries.js";

// Text search: CLIP encodes every query, then Weaviate runs a hybrid search.
// 503s mean every text slot was taken, which is the overload we want to see
expectLimits();

export const options = {
  systemTags,
  scenarios: {
    search: scenario("search", {
      load: 4,
      stress: [1, 2, 4, 8, 16, 32],
      spike: 12,
    }),
  },
  thresholds: thresholds(),
};

export function setup() {
  announce();
  return { vocab: loadVocab() };
}

const served = {
  "status is 200 or 503": (r) => r.status === 200 || r.status === 503,
};

export function search(data) {
  const q = encodeURIComponent(query(data.vocab));
  const res = api(`/search?q=${q}&page_size=50`, "/search", {
    tags: { page: "1" },
  });
  check(res, served);

  // some people scroll on to the next page
  if (res.status !== 200 || Math.random() >= 0.2) return;
  const cursor = res.json("meta.next_cursor");
  if (!cursor) return;
  check(
    api(`/search?q=${q}&page_size=50&cursor=${cursor}`, "/search", {
      tags: { page: "2" },
    }),
    served
  );
}
