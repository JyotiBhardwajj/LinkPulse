import http from 'k6/http';
import { check, sleep } from 'k6';

export function runSuite(config) {
  const baseUrl = __ENV.BASE_URL || 'http://localhost:8080';
  const tag = config.name;

  // Generate a random user per VU iteration to simulate realistic usage
  const rand = Math.floor(Math.random() * 1000000) + Date.now();
  const testUser = {
    email: `loadtest_${rand}@example.com`,
    password: 'SecurePassword123!',
  };

  // --- Step 1: Liveness and Readiness check ---
  const liveRes = http.get(`${baseUrl}/health/live`, { tags: { name: 'health_live', type: tag } });
  check(liveRes, {
    'liveness status is 200': (r) => r.status === 200,
  });

  const readyRes = http.get(`${baseUrl}/health/ready`, { tags: { name: 'health_ready', type: tag } });
  check(readyRes, {
    'readiness status is 200': (r) => r.status === 200,
  });

  // --- Step 2: Register user ---
  const registerPayload = JSON.stringify(testUser);
  const registerHeaders = { 'Content-Type': 'application/json' };
  const registerRes = http.post(`${baseUrl}/api/v1/auth/register`, registerPayload, {
    headers: registerHeaders,
    tags: { name: 'auth_register', type: tag },
  });
  
  check(registerRes, {
    'register status is 201': (r) => r.status === 201,
  });

  // --- Step 3: Login user ---
  const loginPayload = JSON.stringify({
    email: testUser.email,
    password: testUser.password,
  });
  const loginRes = http.post(`${baseUrl}/api/v1/auth/login`, loginPayload, {
    headers: registerHeaders,
    tags: { name: 'auth_login', type: tag },
  });

  const loginSuccess = check(loginRes, {
    'login status is 200': (r) => r.status === 200,
    'login returns token': (r) => r.json('data.access_token') !== undefined,
  });

  if (!loginSuccess) {
    // Fail iteration early if auth failed
    return;
  }

  const accessToken = loginRes.json('data.access_token');
  const authHeaders = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${accessToken}`,
  };

  // --- Step 4: Create shortened links ---
  const createPayload = JSON.stringify({
    original_url: 'https://github.com/grafana/k6',
    title: 'k6 Load Testing',
  });
  const createRes = http.post(`${baseUrl}/api/v1/links`, createPayload, {
    headers: authHeaders,
    tags: { name: 'create_link', type: tag },
  });

  const createSuccess = check(createRes, {
    'create link status is 201': (r) => r.status === 201,
    'create link returns short code': (r) => r.json('data.short_code') !== undefined,
  });

  if (!createSuccess) {
    return;
  }

  const shortCode = createRes.json('data.short_code');

  // --- Step 5: Resolve / Redirect code (simulate high-volume redirect path) ---
  // The resolve path (/r/:code) does not require auth
  const resolveRes = http.get(`${baseUrl}/r/${shortCode}`, {
    redirects: 0, // do not follow redirect to measure shortener latency accurately
    tags: { name: 'resolve_link', type: tag },
  });
  check(resolveRes, {
    'resolve redirects with 302': (r) => r.status === 302,
  });

  // --- Step 6: Query analytics overview ---
  const overviewRes = http.get(`${baseUrl}/api/v1/analytics/overview`, {
    headers: authHeaders,
    tags: { name: 'analytics_overview', type: tag },
  });
  check(overviewRes, {
    'analytics status is 200': (r) => r.status === 200,
  });

  sleep(1);
}

export function generateThresholds() {
  return {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% under 500ms, 99% under 1s
    http_req_failed: ['rate<0.01'],                 // less than 1% errors
    http_reqs: ['count>10'],                        // should run successfully
  };
}

export function handleSummary(data) {
  return {
    'loadtest/summary.json': JSON.stringify(data),
    'loadtest/report.html': `
      <!DOCTYPE html>
      <html>
        <head>
          <title>LinkPulse Load Test Report</title>
          <style>
            body { font-family: sans-serif; margin: 40px; background: #f9f9f9; color: #333; }
            h1 { color: #111; }
            .metric-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.05); margin-bottom: 20px; }
            pre { background: #eee; padding: 10px; border-radius: 4px; overflow-x: auto; }
          </style>
        </head>
        <body>
          <h1>LinkPulse Load Test Summary</h1>
          <div class="metric-card">
            <h2>Execution Details</h2>
            <p><strong>VUs:</strong> \${data.state.testRunDurationMs} ms duration</p>
            <p><strong>Total Requests:</strong> \${data.metrics.http_reqs.values.count}</p>
            <p><strong>RPS:</strong> \${Math.round(data.metrics.http_reqs.values.rate)} req/s</p>
            <p><strong>Fail Rate:</strong> \${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%</p>
          </div>
          <div class="metric-card">
            <h2>Latency Metrics</h2>
            <p><strong>Min:</strong> \${data.metrics.http_req_duration.values.min.toFixed(2)} ms</p>
            <p><strong>Avg:</strong> \${data.metrics.http_req_duration.values.avg.toFixed(2)} ms</p>
            <p><strong>Med:</strong> \${data.metrics.http_req_duration.values.med.toFixed(2)} ms</p>
            <p><strong>Max:</strong> \${data.metrics.http_req_duration.values.max.toFixed(2)} ms</p>
            <p><strong>p(95):</strong> \${data.metrics.http_req_duration.values['p(95)'].toFixed(2)} ms</p>
            <p><strong>p(99):</strong> \${data.metrics.http_req_duration.values['p(99)'].toFixed(2)} ms</p>
          </div>
          <div class="metric-card">
            <h2>Raw Output JSON</h2>
            <pre><code>\${JSON.stringify(data, null, 2)}</code></pre>
          </div>
        </body>
      </html>
    `,
  };
}
