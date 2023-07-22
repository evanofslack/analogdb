import http from "k6/http";
import { check } from "k6";
export const options = {
  vus: 1,
  duration: '3600s',
  thresholds: {
    http_req_failed: ['rate<0.01'], // http errors should be less than 1%
    http_req_duration: ['p(95)<100'], // 95% of requests should be below 200ms
  },
};


const baseURL = "https://api.analogdb.com"

export function setup() {
  const postIDsURL = `${baseURL}/ids`;
  const res = http.get(postIDsURL);
  const body = res.body
  const data = JSON.parse(body);
  const ids = data.ids;
  return { ids: ids };
}


export default function (data) {

  let ids = data.ids;
  let id = ids[Math.floor(Math.random()*ids.length)];
  // console.log(`fetching post id=${id}`);

  check(http.get(`${baseURL}/post/${id}`), {
    "status is 200": (r) => r.status == 200,
    "protocol is HTTP/2": (r) => r.proto == "HTTP/2.0",
  });
}

