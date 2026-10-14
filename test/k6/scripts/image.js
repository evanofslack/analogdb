import { check } from "k6";
import http from "k6/http";
import {
  announce,
  scenario,
  systemTags,
  thresholds,
  USER_AGENT,
} from "../lib/config.js";
import { api, apiPost, expectLimits, json } from "../lib/http.js";

// Image search: CLIP encodes the upload. Two slots, so 503s come early
expectLimits();

export const options = {
  systemTags,
  scenarios: {
    image: scenario("image", { load: 1, stress: [1, 2, 3, 4], spike: 4 }),
  },
  thresholds: thresholds(),
};

// setup downloads a few real low resolution post images to upload
export function setup() {
  announce();
  const posts = json(api("/posts?sort=random&page_size=10", "/posts"), "posts");
  const images = [];
  for (const post of posts) {
    const low = post.images.find((i) => i.resolution === "low");
    if (!low) continue;
    const res = http.get(low.url, {
      headers: { "User-Agent": USER_AGENT },
      responseType: "binary",
      tags: { name: "image download" },
    });
    if (res.status === 200) images.push(Array.from(new Uint8Array(res.body)));
    if (images.length === 5) break;
  }
  if (images.length === 0)
    throw new Error("could not download any post images");
  return { images };
}

export function image(data) {
  const bytes = data.images[Math.floor(Math.random() * data.images.length)];
  const body = {
    image: http.file(new Uint8Array(bytes).buffer, "image.jpg", "image/jpeg"),
  };
  check(apiPost("/search/image?page_size=50", body, "/search/image"), {
    "status is 200 or 503": (r) => r.status === 200 || r.status === 503,
  });
}
