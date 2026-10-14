import { check } from "k6";
import { announce, scenario, systemTags, thresholds } from "../lib/config.js";
import { api, json } from "../lib/http.js";

// Measures the path itself: Cloudflare, Caddy and Tailscale in front of a
// cheap Redis cached post. The random param keeps Cloudflare from answering
export const options = {
  systemTags,
  scenarios: {
    baseline: scenario("baseline", {
      load: 20,
      stress: [5, 10, 20, 40, 80],
      spike: 80,
    }),
  },
  thresholds: thresholds(),
};

export function setup() {
  announce();
  const posts = json(api("/posts?page_size=50", "/posts"), "posts");
  return { ids: posts.map((p) => p.id) };
}

export function baseline(data) {
  const id = data.ids[Math.floor(Math.random() * data.ids.length)];
  const bust = Math.random().toString(36).slice(2, 10);
  check(api(`/post/${id}?_=${bust}`, "/post/{id}"), {
    "status is 200": (r) => r.status === 200,
  });
}
