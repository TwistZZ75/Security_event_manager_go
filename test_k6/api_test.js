import http from 'k6/http';
import { check, sleep } from 'k6';

const API_URL = 'http://localhost:3000';

export const options = {
    vus: 1,
    duration: '10s',

    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<100'],
    },
};

export function setup() {

    const loginPayload = JSON.stringify({
        username: 'Hikki',
        password: 'qwe_1234'
    });

    const loginRes = http.post(
        `${API_URL}/api/auth/login`,
        loginPayload,
        {
            headers: {
                'Content-Type': 'application/json',
            },
        }
    );

    // console.log(`LOGIN STATUS: ${loginRes.status}`);
    // console.log(`LOGIN BODY: ${loginRes.body}`);

    check(loginRes, {
        'login success': (r) => r.status === 200,
        'login response < 500ms': (r) => r.timings.duration < 500,
    });

    // если есть JWT/token
    let token = null;

    try {
        token = loginRes.json('token');
    } catch (e) {
        console.log('Token not found in response');
    }

    return { token };
}

export default function (data) {

    const headers = data.token
        ? {
            Authorization: `Bearer ${data.token}`,
        }
        : {};

    const endpoints = [
        '/agents',
        '/rules',
        '/events',
        '/alerts',
        '/actions'
    ];

    endpoints.forEach((endpoint) => {

        const res = http.get(
            `${API_URL}${endpoint}`,
            { headers }
        );

        check(res, {
            [`${endpoint} status 200`]:
                (r) => r.status === 200,

            [`${endpoint} response < 100ms`]:
                (r) => r.timings.duration < 100,
        });

        // console.log(
        //     `${endpoint} | status=${res.status} | duration=${res.timings.duration}ms`
        // );
    });

    sleep(1);
}