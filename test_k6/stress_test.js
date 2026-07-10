import http from 'k6/http';

const API_URL = 'http://localhost:3000';

export const options = {

    stages: [
        { duration: '1m', target: 500 },
        { duration: '1m', target: 700 },
        { duration: '1m', target: 1000 },
        { duration: '1m', target: 2000 },
        { duration: '1m', target: 5000 },
        { duration: '30s', target: 0 },
    ],
};

export default function () {

    http.get(`${API_URL}/events`);
}