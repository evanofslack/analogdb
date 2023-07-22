import http from 'k6/http';

export let options = {
  discardResponseBodies: true,
  scenarios: {
    Scenario_Random: {
      exec: 'Random',
      executor: 'ramping-vus',
      startTime: '0s',
      startVUs: 1,
      stages: [
        { duration: '60s', target: 2 },
        { duration: '60s', target: 3 },
        { duration: '60s', target: 2 },
        { duration: '60s', target: 1 },
      ],
    },
    Scenario_Top: {
      exec: 'Top',
      executor: 'ramping-vus',
      startTime: '0s',
      startVUs: 1,
      stages: [
        { duration: '60s', target: 2 },
        { duration: '60s', target: 3 },
        { duration: '60s', target: 2 },
        { duration: '60s', target: 1 },
      ],
    },
    Scenario_Latest: {
      exec: 'Latest',
      executor: 'ramping-vus',
      startTime: '0s',
      startVUs: 1,
      stages: [
        { duration: '60s', target: 2 },
        { duration: '60s', target: 3 },
        { duration: '60s', target: 2 },
        { duration: '60s', target: 1 },
      ],
    },
  },
};

export function Random() {
  http.get('https://api.analogdb.com/posts?sort=random');
}

export function Top() {
  http.get('https://api.analogdb.com/posts?sort=top');
}

export function Latest() {
  http.get('https://api.analogdb.com/posts?sort=latest');
}

