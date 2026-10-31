// Baseline load test for GET /items (read-only, no auth, no Xsolla calls).
// Shows the service's own capacity (Go + Postgres) at 50 -> MAX_RPS,
// to compare against POST /orders, which is limited by the Xsolla sandbox.
//
// Run:
//   k6 run -e MAX_RPS=1000 --summary-export=loadtest/results-items.json loadtest/items.js

import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const MAX_RPS = parseInt(__ENV.MAX_RPS || '1000', 10);

export const options = {
  summaryTrendStats: ['min', 'med', 'avg', 'p(90)', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    items: {
      executor: 'ramping-arrival-rate',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 1000,
      stages: [
        { target: 50, duration: '30s' },      // warm-up at 50 RPS
        { target: MAX_RPS, duration: '1m' },  // ramp 50 -> MAX_RPS
        { target: MAX_RPS, duration: '1m' },  // hold at MAX_RPS
        { target: 0, duration: '15s' },       // ramp down
      ],
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.01'],
    'http_req_duration{name:GET /items}': ['p(95)<300', 'p(99)<1000'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/items?limit=20`, { tags: { name: 'GET /items' } });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'not a timeout (504)': (r) => r.status !== 504,
  });
}
