import encoding from "k6/encoding";

export const API_URL = __ENV.API_URL || "https://api.analogdb.com/v1";
export const WEB_URL = __ENV.WEB_URL || "https://analogdb.com";
export const USER_AGENT = "analogdb-k6/1";
export const PROFILE = __ENV.PROFILE || "smoke";

const username = __ENV.K6_USERNAME || "";
const password = __ENV.K6_PASSWORD || "";

export const authenticated = username !== "" && password !== "";
export const authorization = authenticated
  ? `Basic ${encoding.b64encode(`${username}:${password}`)}`
  : null;

// every request keeps its name tag but drops the url tag, since unique search
// queries would give Prometheus a new series per request
export const systemTags = [
  "proto",
  "status",
  "method",
  "name",
  "group",
  "check",
  "error_code",
  "scenario",
  "expected_response",
];

const stepDuration = __ENV.STEP || "2m";
const holdDuration = __ENV.HOLD || "10m";

function steps(fallback) {
  if (!__ENV.STEPS) return fallback;
  return __ENV.STEPS.split(",").map(Number);
}

function rate(fallback) {
  return __ENV.RATE ? Number(__ENV.RATE) : fallback;
}

// arrival counts per minute, since k6 rates are whole numbers and RATE can
// be fractional. Stage targets come in per second
function arrival(stages, peak) {
  return {
    executor: "ramping-arrival-rate",
    startRate: 0,
    timeUnit: "1m",
    preAllocatedVUs: Math.min(Math.ceil(peak * 2), 50),
    maxVUs: Math.min(Math.ceil(peak * 20) + 10, 600),
    stages: stages.map((s) => ({ ...s, target: Math.round(s.target * 60) })),
  };
}

// scenario builds the executor for PROFILE. defaults holds the script's
// load and spike rate and its stress steps, all in requests per second
export function scenario(exec, defaults) {
  switch (PROFILE) {
    case "smoke":
      return {
        exec,
        executor: "constant-arrival-rate",
        rate: 1,
        timeUnit: "1s",
        duration: "1m",
        preAllocatedVUs: 2,
        maxVUs: 10,
      };
    case "load": {
      const target = rate(defaults.load);
      return {
        exec,
        ...arrival(
          [
            { target, duration: "1m" },
            { target, duration: holdDuration },
            { target: 0, duration: "30s" },
          ],
          target
        ),
      };
    }
    case "stress": {
      const rates = steps(defaults.stress);
      const stages = rates.flatMap((target) => [
        { target, duration: "15s" },
        { target, duration: stepDuration },
      ]);
      stages.push({ target: 0, duration: "30s" });
      return { exec, ...arrival(stages, Math.max(...rates)) };
    }
    case "spike": {
      const target = rate(defaults.spike);
      return {
        exec,
        ...arrival(
          [
            { target, duration: "10s" },
            { target, duration: "1m" },
            { target: 1, duration: "10s" },
            { target: 1, duration: "1m" },
          ],
          target
        ),
      };
    }
    default:
      throw new Error(
        `unknown PROFILE ${PROFILE}, use smoke, load, stress or spike`
      );
  }
}

// thresholds stops a run that is clearly overloading prod
export function thresholds() {
  if (PROFILE === "smoke") {
    return { http_req_failed: ["rate<0.01"], checks: ["rate>0.99"] };
  }
  return {
    http_req_failed: [
      { threshold: "rate<0.2", abortOnFail: true, delayAbortEval: "1m" },
    ],
    http_req_duration: [
      { threshold: "p(95)<15000", abortOnFail: true, delayAbortEval: "1m" },
    ],
  };
}

export function announce() {
  console.log(
    `profile=${PROFILE} api=${API_URL} web=${WEB_URL} auth=${
      authenticated ? "web pair" : "anonymous"
    }`
  );
  if (!authenticated) {
    console.warn(
      "K6_USERNAME and K6_PASSWORD are unset, API calls share the anonymous 60/min limit"
    );
  }
}
