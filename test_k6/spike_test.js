import http from 'k6/http';

const API_URL = 'http://localhost:3000';

export const options = {

    stages: [
        { duration: '30s', target: 5 },
        { duration: '10s', target: 500 },
        { duration: '1m', target: 500 },
        { duration: '20s', target: 0 },
    ],
};

export default function () {

    http.get(`${API_URL}/events`);
    http.get(`${API_URL}/alerts`);
}