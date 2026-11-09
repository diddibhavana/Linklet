import http from 'k6/http';
import { check, sleep } from 'k6';

const api = __ENV.API_URL || 'http://localhost:8080';
export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '2m', target: 100 },
    { duration: '30s', target: 0 },
  ],
  thresholds: { http_req_failed: ['rate<0.01'], http_req_duration: ['p(99)<250'] },
};

export default function () {
  const response = http.get(`${api}/api/v1/links?limit=25`);
  check(response, { 'list is successful': (r) => r.status === 200 });
  sleep(0.2);
}
