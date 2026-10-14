import { api, json } from "./http.js";

const phrases = [
  "kodak portra beach at sunset",
  "black and white street portrait",
  "foggy mountain lake",
  "dog running in the snow",
  "city lights at night",
  "portrait by a window",
  "desert road trip",
  "old car in the countryside",
  "light leak summer",
  "people at a concert",
];

// loadVocab fetches the words queries are built from. Call it in setup
export function loadVocab() {
  const keywords = json(
    api("/keywords/summary?page_size=200", "/keywords/summary"),
    "keywords"
  ).map((k) => k.word);
  const cameras = json(
    api("/cameras?exclude_zero_counts=true", "/cameras"),
    "cameras"
  ).map((c) => `${c.make} ${c.model}`);
  const films = json(
    api("/films?exclude_zero_counts=true", "/films"),
    "films"
  ).map((f) => `${f.make} ${f.type}`);
  if (keywords.length === 0) throw new Error("no keywords to build queries");
  return { keywords, cameras, films };
}

function pick(list) {
  return list[Math.floor(Math.random() * list.length)];
}

// query builds a search that no earlier request used. The short token at the
// end keeps both the Cloudflare and the Redis cache from answering it
export function query(vocab) {
  const parts = [];
  if (Math.random() < 0.3) {
    parts.push(pick(phrases));
  } else {
    const n = 1 + Math.floor(Math.random() * 3);
    for (let i = 0; i < n; i++) parts.push(pick(vocab.keywords));
  }
  if (Math.random() < 0.2 && vocab.cameras.length)
    parts.push(pick(vocab.cameras));
  if (Math.random() < 0.2 && vocab.films.length) parts.push(pick(vocab.films));
  const salt = Math.random().toString(36).slice(2, 5);
  parts.push(`${__VU.toString(36)}${__ITER.toString(36)}${salt}`);
  return parts.join(" ").slice(0, 200);
}
