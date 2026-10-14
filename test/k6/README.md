# Load tests

k6 scripts for the API and the web app. They run natively (`brew install k6`) and target prod by default,
through Cloudflare.

## Setup

```bash
brew install k6
export K6_PROMETHEUS_RW_SERVER_URL=http://<prometheus>:9090/api/v1/write   # optional
export K6_USERNAME=... K6_PASSWORD=...                                      # the web credential pair
```

`browser.js` needs Chrome or Chromium. Without Google Chrome in `/Applications`, point k6 at another one,
such as a Playwright download:

```bash
export K6_BROWSER_EXECUTABLE_PATH="$HOME/Library/Caches/ms-playwright/chromium-<n>/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
```

Prometheus needs `--web.enable-remote-write-receiver` and native histograms enabled. Import
`infra/config/grafana/dashboards/Analogdb/dashboard-results.json` into Grafana to see runs, picked by
`testid`.

Without `K6_USERNAME`/`K6_PASSWORD`, API calls share the anonymous 60/min limit and capacity runs hit 429s
right away. Web pages never get the credentials.

Every request sends `User-Agent: analogdb-k6/1`, so the traffic can be matched by a WAF skip rule and
filtered out of analytics afterwards.

## Env

| Var | Default | Meaning |
|---|---|---|
| `API_URL` | `https://api.analogdb.com/v1` | Backend base URL |
| `WEB_URL` | `https://analogdb.com` | Web app base URL |
| `PROFILE` | `smoke` | `smoke`, `load`, `stress` or `spike` |
| `RATE` | per script | Requests per second for `load` and `spike` |
| `STEPS` | per script | Comma separated rates for `stress`, for example `1,2,4,8` |
| `STEP` | `2m` | How long `stress` holds each step |
| `HOLD` | `10m` | How long `load` holds its rate |

Every profile except `smoke` aborts when more than 20% of requests fail unexpectedly or p95 passes 15s.
Ctrl-C stops a run at any time. Summaries land in `results/` (gitignored).

## Scripts

| Script | What it does | Defaults (rps) |
|---|---|---|
| `smoke.js` | Every public endpoint and page, 1 VU for 1 min | |
| `baseline.js` | Cache-busted `/post/{id}`, measures the network path | load 20, stress 5 to 80, spike 80 |
| `limits.js` | Checks anonymous 429s, web search limit, web pair not limited | |
| `search.js` | Text search with unique queries, some go to page 2 | load 4, stress 1 to 32, spike 12 |
| `image.js` | Image search with real post images | load 1, stress 1 to 4, spike 4 |
| `browse.js` | Server rendered home, post, camera and film pages | load 10, stress 2 to 40, spike 30 |
| `browser.js` | Chromium page loads with Web Vitals, 3 VUs for 2 min | |

Search queries are built from live keywords, cameras and films, and end with a short random token, so
neither Cloudflare nor Redis can answer them from cache. Every one reaches CLIP.

## Run order

From the repo root:

```bash
just load scripts/smoke.js
PROFILE=stress just load scripts/baseline.js
just load scripts/limits.js
PROFILE=stress just load scripts/search.js
PROFILE=spike RATE=<3x knee> just load scripts/search.js
PROFILE=load RATE=<half knee> just load scripts/search.js
PROFILE=stress just load scripts/image.js
PROFILE=spike just load scripts/browse.js
just load scripts/browser.js        # alongside a load run of search.js
```

Extra arguments go straight to `k6 run`, for example `just load scripts/search.js --http-debug`.
