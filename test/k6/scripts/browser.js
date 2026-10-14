import { check } from "k6";
import { browser } from "k6/browser";
import { announce, systemTags, USER_AGENT, WEB_URL } from "../lib/config.js";
import { api, json } from "../lib/http.js";
import { loadVocab, query } from "../lib/queries.js";

// Real page loads in Chromium, for Web Vitals while another script adds load.
// Pages run their JS, so these visits send page view events
export const options = {
  systemTags,
  scenarios: {
    browser: {
      executor: "constant-vus",
      vus: 3,
      duration: "2m",
      options: { browser: { type: "chromium" } },
    },
  },
  thresholds: {
    checks: ["rate>0.95"],
  },
};

export function setup() {
  announce();
  const posts = json(api("/posts?page_size=20", "/posts"), "posts");
  return { ids: posts.map((p) => p.id), vocab: loadVocab() };
}

export default async function (data) {
  const context = await browser.newContext({
    userAgent: `${USER_AGENT} browser`,
  });
  const page = await context.newPage();
  try {
    const home = await page.goto(WEB_URL, { waitUntil: "load" });
    check(home, { "home is 200": (r) => r && r.status() === 200 });

    const id = data.ids[Math.floor(Math.random() * data.ids.length)];
    const post = await page.goto(`${WEB_URL}/post/${id}`, {
      waitUntil: "load",
    });
    check(post, { "post is 200": (r) => r && r.status() === 200 });

    // three VUs stay under web's 30 searches a minute per IP
    if (__ITER % 2 === 0) {
      const q = encodeURIComponent(query(data.vocab));
      const search = await page.goto(`${WEB_URL}/search?q=${q}`, {
        waitUntil: "load",
      });
      check(search, { "search is 200": (r) => r && r.status() === 200 });
    }
  } finally {
    await page.close();
    await context.close();
  }
}
