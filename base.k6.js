import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 10 },
    { duration: '15s', target: 0 },
  ],
};

export default function () {
  const res = http.get('http://host.docker.internal:8080/v0/randos');
  check(res, { 'status was 200': (r) => r.status === 200 });
  sleep(1);
};