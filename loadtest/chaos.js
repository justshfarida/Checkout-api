// Chaos experiment: GET /items at a steady rate while chaos.ps1 breaks the
// database on a fixed schedule. Every request is tagged with the phase it ran
// in, so the summary shows status codes and latency per phase.
//
// Don't run this file directly; run:  .\loadtest\chaos.ps1
//
// Timeline (seconds since the test started) - must match chaos.ps1:
//   0-60     steady       normal operation (steady state)
//   60-105   db_paused    `docker compose pause db`  -> DB hangs (no answers)
//   105-165  recovery_1   `docker compose unpause db`
//   165-210  db_stopped   `docker compose stop db`   -> DB is down (refused)
//   210-300  recovery_2   `docker compose start db`

import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const RATE = parseInt(__ENV.RATE || '100', 10);

const PHASES = [
  { name: 'steady', until: 60 },
  { name: 'db_paused', until: 105 },
  { name: 'recovery_1', until: 165 },
  { name: 'db_stopped', until: 210 },
  { name: 'recovery_2', until: 300 },
];
const STATUSES = ['200', '500', '503', '504', '0'];

// k6 only reports tag breakdowns (sub-metrics) that have a threshold, so
// every phase/status pair gets one that always passes.
const thresholds = {};
for (const p of PHASES) {
  thresholds[`http_req_duration{phase:${p.name},endpoint:items}`] = ['p(95)>=0'];
  for (const s of STATUSES) {
    thresholds[`http_reqs{phase:${p.name},status:${s}}`] = ['count>=0'];
  }
  thresholds[`http_reqs{phase:${p.name}}`] = ['count>=0'];
  thresholds[`http_reqs{phase:${p.name},endpoint:health}`] = ['count>=0'];
  thresholds[`http_reqs{phase:${p.name},endpoint:health,status:200}`] = ['count>=0'];
}

export const options = {
  summaryTrendStats: ['min', 'med', 'avg', 'p(95)', 'p(99)', 'max'],
  thresholds,
  scenarios: {
    items: {
      executor: 'constant-arrival-rate',
      exec: 'items',
      rate: RATE,
      timeUnit: '1s',
      duration: '300s',
      preAllocatedVUs: 50,
      maxVUs: 1000,
    },
    // What a Kubernetes probe would see while the DB is broken.
    health: {
      executor: 'constant-arrival-rate',
      exec: 'health',
      rate: 2,
      timeUnit: '1s',
      duration: '300s',
      preAllocatedVUs: 2,
      maxVUs: 10,
    },
  },
};

export function setup() {
  return { start: Date.now() };
}

function phaseAt(start) {
  const t = (Date.now() - start) / 1000;
  for (const p of PHASES) {
    if (t < p.until) return p.name;
  }
  return PHASES[PHASES.length - 1].name;
}

export function items(data) {
  const phase = phaseAt(data.start);
  const res = http.get(`${BASE_URL}/items?limit=20`, {
    timeout: '20s',
    tags: { name: 'GET /items', endpoint: 'items', phase },
  });

  if (res.status !== 200) {
    check(res, {
      // "Is the error clean?" - JSON with an error code, no DB details leaked.
      'error body is clean JSON': (r) => {
        try {
          const b = r.json();
          return typeof b.error === 'string' && !/pgx|postgres|sql|dial tcp|lookup db/i.test(r.body);
        } catch (e) {
          return false;
        }
      },
    }, { phase });
  }
}

export function health(data) {
  const phase = phaseAt(data.start);
  http.get(`${BASE_URL}/health`, {
    timeout: '5s',
    tags: { name: 'GET /health', endpoint: 'health', phase },
  });
}

function count(m, key) {
  const v = m[key];
  return v ? v.values.count : 0;
}

function ms(m, key, stat) {
  const v = m[key];
  if (!v || v.values[stat] === undefined) return '-';
  const x = v.values[stat];
  return x >= 1000 ? `${(x / 1000).toFixed(2)}s` : `${x.toFixed(1)}ms`;
}

export function handleSummary(data) {
  const m = data.metrics;
  const pad = (s, n) => String(s).padEnd(n);
  let out = '\nCHAOS EXPERIMENT - GET /items by phase\n\n';
  out += pad('phase', 12) + pad('total', 8) + STATUSES.map((s) => pad(s === '0' ? 'no resp' : s, 8)).join('')
    + pad('p50', 10) + pad('p95', 10) + pad('max', 10) + 'health 200/total\n';

  for (const p of PHASES) {
    const healthTotal = count(m, `http_reqs{phase:${p.name},endpoint:health}`);
    const healthOk = count(m, `http_reqs{phase:${p.name},endpoint:health,status:200}`);
    const total = count(m, `http_reqs{phase:${p.name}}`) - healthTotal;
    // Status counts include /health hits; subtract them from the 200 column.
    const cols = STATUSES.map((s) => {
      let c = count(m, `http_reqs{phase:${p.name},status:${s}}`);
      if (s === '200') c -= healthOk;
      return pad(c, 8);
    }).join('');
    const d = `http_req_duration{phase:${p.name},endpoint:items}`;
    out += pad(p.name, 12) + pad(total, 8) + cols
      + pad(ms(m, d, 'med'), 10) + pad(ms(m, d, 'p(95)'), 10) + pad(ms(m, d, 'max'), 10)
      + `${healthOk}/${healthTotal}\n`;
  }

  const clean = m['checks'] ? m['checks'].values : null;
  if (clean) {
    out += `\nError bodies clean: ${clean.passes} passed, ${clean.fails} failed\n`;
  }

  return {
    stdout: out,
    'loadtest/results-chaos.json': JSON.stringify(data, null, 2),
  };
}
