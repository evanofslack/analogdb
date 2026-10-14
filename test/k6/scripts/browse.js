import { check } from "k6";
import { announce, scenario, systemTags, thresholds } from "../lib/config.js";
import { api, json, web } from "../lib/http.js";

// Launch day browsing through the Next app: server rendered pages that each
// call the backend. These pages have no per-IP limit in web
export const options = {
  systemTags,
  scenarios: {
    browse: scenario("browse", {
      load: 10,
      stress: [2, 5, 10, 20, 40],
      spike: 30,
    }),
  },
  thresholds: thresholds(),
};

export function setup() {
  announce();
  const ids = json(api("/ids", "/ids"), "ids");
  // web only builds catalog pages for entries with at least 5 posts
  const catalog = "?min_count=5&page_size=500";
  const cameras = json(api(`/cameras${catalog}`, "/cameras"), "cameras");
  const films = json(api(`/films${catalog}`, "/films"), "films");
  return {
    ids,
    cameras: cameras.map((c) => c.slug),
    films: films.map((f) => f.slug),
  };
}

function pick(list) {
  return list[Math.floor(Math.random() * list.length)];
}

const pages = [
  { weight: 30, page: () => ["/", "web /"] },
  { weight: 45, page: (d) => [`/post/${pick(d.ids)}`, "web /post/{id}"] },
  { weight: 5, page: () => ["/cameras", "web /cameras"] },
  { weight: 5, page: () => ["/films", "web /films"] },
  {
    weight: 8,
    page: (d) => [`/cameras/${pick(d.cameras)}`, "web /cameras/{slug}"],
  },
  { weight: 7, page: (d) => [`/films/${pick(d.films)}`, "web /films/{slug}"] },
];
const total = pages.reduce((sum, p) => sum + p.weight, 0);

export function browse(data) {
  let roll = Math.random() * total;
  const choice = pages.find((p) => (roll -= p.weight) < 0) || pages[0];
  const [path, name] = choice.page(data);
  check(web(path, name), { "status is 200": (r) => r.status === 200 });
}
