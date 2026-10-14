import http from "k6/http";
import { Counter } from "k6/metrics";
import { API_URL, authorization, USER_AGENT, WEB_URL } from "./config.js";

export const rateLimited = new Counter("rate_limited");
export const slotBusy = new Counter("slot_busy");

// expectLimits makes 429 and 503 count as expected, for scripts that push
// past the limits on purpose
export function expectLimits() {
  http.setResponseCallback(
    http.expectedStatuses({ min: 200, max: 399 }, 429, 503)
  );
}

function params(name, opts = {}) {
  const headers = { "User-Agent": USER_AGENT, ...(opts.headers || {}) };
  if (opts.auth !== false && authorization) {
    headers.Authorization = authorization;
  }
  return {
    headers,
    tags: { name, ...(opts.tags || {}) },
    timeout: opts.timeout || "30s",
    responseType: opts.responseType || "text",
  };
}

function counted(res, name) {
  if (res.status === 429) rateLimited.add(1, { name });
  if (res.status === 503) slotBusy.add(1, { name });
  return res;
}

// api calls the backend. It sends the web pair when set, unless auth is false
export function api(path, name, opts = {}) {
  return counted(http.get(`${API_URL}${path}`, params(name, opts)), name);
}

export function apiPost(path, body, name, opts = {}) {
  return counted(
    http.post(`${API_URL}${path}`, body, params(name, opts)),
    name
  );
}

// web loads a page from the Next app. It never sends the web pair
export function web(path, name, opts = {}) {
  return counted(
    http.get(`${WEB_URL}${path}`, params(name, { ...opts, auth: false })),
    name
  );
}

export function json(res, selector) {
  if (res.status !== 200) {
    throw new Error(`${res.request.url} returned ${res.status}`);
  }
  return res.json(selector);
}
