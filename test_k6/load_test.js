import http from 'k6/http';
import { check, sleep } from 'k6';

const API_URL = 'http://localhost:3000';

export const options = {

    stages: [
        { duration: '1m', target: 50 },
        { duration: '3m', target: 100 },
        { duration: '1m', target: 0 },
    ],

    thresholds: {
        http_req_duration: ['p(95)<500'],
        http_req_failed: ['rate<0.01'],
    },
};

export default function () {

    http.get(`${API_URL}/events`);
    http.get(`${API_URL}/alerts`);
    http.get(`${API_URL}/agents`);

    sleep(1);
}