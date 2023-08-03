import encoding from "k6/encoding";
import http from "k6/http";
import { sleep } from "k6";

const baseURL = __ENV.BASE_URL;
const username = __ENV.USERNAME;
const password = __ENV.PASSWORD;
const delay = __ENV.DELAY;
const duration = __ENV.DURATION;

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
    http.get(`${baseURL}${path}`, httpOpts);
  } else {
    http.get(`${baseURL}${path}`);
  }

  // sleep if configured
  if (delay != 0) {
    sleep(delay);
  }
}

export let options = {
  discardResponseBodies: true,
  scenarios: {
    Scenario_Random: {
      exec: "Random",
      executor: "ramping-vus",
      startTime: "0s",
      startVUs: 1,
      stages: [
        { duration: duration, target: 2 },
        { duration: duration, target: 3 },
        { duration: duration, target: 2 },
        { duration: duration, target: 1 },
      ],
    },
    Scenario_Top: {
      exec: "Top",
      executor: "ramping-vus",
      startTime: "0s",
      startVUs: 1,
      stages: [
        { duration: duration, target: 2 },
        { duration: duration, target: 3 },
        { duration: duration, target: 2 },
        { duration: duration, target: 1 },
      ],
    },
    Scenario_Latest: {
      exec: "Latest",
      executor: "ramping-vus",
      startTime: "0s",
      startVUs: 1,
      stages: [
        { duration: duration, target: 2 },
        { duration: duration, target: 3 },
        { duration: duration, target: 2 },
        { duration: duration, target: 1 },
      ],
    },
    Scenario_Post: {
      exec: "Post",
      executor: "ramping-vus",
      startTime: "0s",
      startVUs: 1,
      stages: [
        { duration: duration, target: 2 },
        { duration: duration, target: 3 },
        { duration: duration, target: 2 },
        { duration: duration, target: 1 },
      ],
    },
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

export function Random() {
  // http.get(`${baseURL}/posts?sort=random`);
  get("/posts?sort=random");
}

export function Top() {
  // http.get(`${baseURL}/posts?sort=top`);
  get("/posts?sort=top");
}

export function Latest() {
  // http.get(`${baseURL}/posts?sort=latest`);
  get("/posts?sort=latest");
}

export function Post(data) {
  let ids = data.ids;
  let id = ids[Math.floor(Math.random() * ids.length)];
  // http.get(`${baseURL}/post/${id}`);
  get(`/post/${id}`);
  get(`/post/${id}/similar`);
}
