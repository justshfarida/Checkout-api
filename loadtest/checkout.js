// k6 load test for the checkout endpoint (POST /orders).
//
// Run (Docker Compose backend on localhost:8080):
//   k6 run -e EMAIL=you@example.com -e PASSWORD=yourpassword loadtest/checkout.js
//
// Optional:
//   -e BASE_URL=http://localhost:8080   API address
//   -e MAX_RPS=1000                     peak requests per second (start low!)
//   --summary-export=loadtest/results.json
import http from 'k6/http';
import { check, fail } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const MAX_RPS = parseInt(__ENV.MAX_RPS || '1000', 10);

// Ramp the arrival rate from 50 to MAX_RPS, hold, then ramp down.
// "ramping-arrival-rate" fixes requests per second regardless of how slow
// the server gets, so latency growth shows up clearly.
const stage = (fraction) => Math.max(50, Math.round(MAX_RPS * fraction));

export const options = {
  scenarios: {
    checkout: {
      executor: 'ramping-arrival-rate',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 2000,
      stages: [
        { target: stage(0.2), duration: '30s' },
        { target: stage(0.5), duration: '30s' },
        { target: stage(1.0), duration: '30s' },
        { target: stage(1.0), duration: '1m' },
        { target: 0, duration: '15s' },
      ],
    },
  },
  // Report the percentiles the homework asks for.
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
  thresholds: {
    'http_req_duration{name:POST /orders}': ['p(95)<500', 'p(99)<1000'],
    'http_req_failed{name:POST /orders}': ['rate<0.01'],
  },
};

// setup() runs once before the test: log in, and make sure the cart holds
// a non-merch item (unlimited stock, so orders never fail on stock).
export function setup() {
  if (!__ENV.EMAIL || !__ENV.PASSWORD) {
    fail('Pass -e EMAIL=... -e PASSWORD=... for an existing user');
  }

  const login = http.post(
    `${BASE_URL}/login`,
    JSON.stringify({ email: __ENV.EMAIL, password: __ENV.PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  if (login.status !== 200) fail(`login failed: ${login.status} ${login.body}`);
  const token = login.json('jwt');

  const items = http.get(`${BASE_URL}/items?limit=100`).json('items') || [];
  const unlimited = items.find((i) => i.item_type && i.item_type !== 'merch');
  if (!unlimited) fail('no non-merch item found to put in the cart');

  const cart = http.patch(
    `${BASE_URL}/user/cart/items/${unlimited.id}`,
    JSON.stringify({ quantity: 1 }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
    },
  );
  if (cart.status !== 204) fail(`could not fill cart: ${cart.status} ${cart.body}`);

  console.log(`Using item ${unlimited.id} (${unlimited.name}), peak ${MAX_RPS} RPS`);
  return { token };
}

export default function (data) {
  const res = http.post(`${BASE_URL}/orders`, null, {
    headers: {
      Authorization: `Bearer ${data.token}`,
      // A fresh key per request, otherwise the idempotency cache would
      // answer from memory and nothing real would be tested.
      'Idempotency-Key': uuidv4(),
    },
    tags: { name: 'POST /orders' },
  });

  check(res, {
    'order created (201)': (r) => r.status === 201,
  });
}
