import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 1,
  duration: "60s",
  thresholds: {
    http_req_failed: ["rate<0.01"], // http errors should be less than 1%
    http_req_duration: ["p(95)<100"], // 95% of requests should be below 200ms
  },
};

export default function () {
  let baseURL="http://analogdb:8080"
  // baseURL="http://api.analogdb.com"
  let url = `${baseURL}/posts`;
  check(http.get(url), {
    "status is 200": (r) => r.status == 200,
    "protocol is HTTP/2": (r) => r.proto == "HTTP/2.0",
  });
}
