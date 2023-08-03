import encoding from "k6/encoding";
import http from "k6/http";
import { check, sleep } from "k6";

const baseURL = __ENV.BASE_URL;
const username = __ENV.USERNAME;
const password = __ENV.PASSWORD;
const delay = __ENV.DELAY;
const duration = __ENV.DURATION;
const vus = __ENV.VUS;

const credentials = `${username}:${password}`;
const encodedCredentials = encoding.b64encode(credentials);

const httpOpts = {
  headers: {
    Authorization: `Basic ${encodedCredentials}`,
  },
};

function get(path) {
  let url = `${baseURL}${path}`;

  // use basic auth if configured
  if (username != "" && password != "") {
    return http.get(`${baseURL}${path}`, httpOpts);
  } else {
    return http.get(`${baseURL}${path}`);
  }
}

export const options = {
  vus: vus,
  duration: duration,
  thresholds: {
    http_req_failed: ["rate<0.01"], // http errors should be less than 1%
    http_req_duration: ["p(95)<100"], // 95% of requests should be below 200ms
  },
};

export function setup() {
  const postIDsURL = `${baseURL}/ids`;
  const res = http.get(postIDsURL);
  const body = res.body;
  const data = JSON.parse(body);
  const ids = data.ids;
  return { ids: ids };
}

export default function (data) {
  let ids = data.ids;

  // only take a portion of all ids
  ids = ids.slice(0, 100);

  let id = ids[Math.floor(Math.random() * ids.length)];

  check(get(`/post/${id}`), {
    "status is 200": (r) => r.status == 200,
  });

  // sleep if configured
  if (delay != 0) {
    sleep(delay);
  }
}
