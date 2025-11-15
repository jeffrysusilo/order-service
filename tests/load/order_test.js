import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 200 },  // Spike to 200 users
    { duration: '1m', target: 200 },   // Stay at 200 users
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    errors: ['rate<0.1'],              // Error rate should be below 10%
  },
};

const BASE_URL = 'http://localhost:8080/api/v1';

// Sample products (from seed data)
const products = [1, 2, 3, 4, 5];

export default function () {
  // Generate random order
  const numItems = Math.floor(Math.random() * 3) + 1; // 1-3 items
  const items = [];
  
  for (let i = 0; i < numItems; i++) {
    items.push({
      product_id: products[Math.floor(Math.random() * products.length)],
      quantity: Math.floor(Math.random() * 3) + 1, // 1-3 quantity
    });
  }

  const payload = JSON.stringify({
    user_id: Math.floor(Math.random() * 1000) + 1,
    items: items,
    payment_method: 'mock',
    idempotency_key: `load-test-${Date.now()}-${Math.random()}`,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Create order
  const createRes = http.post(`${BASE_URL}/orders`, payload, params);
  
  const createSuccess = check(createRes, {
    'order created successfully': (r) => r.status === 201,
    'response has order_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.order_id !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (!createSuccess) {
    errorRate.add(1);
    console.log(`Failed to create order: ${createRes.status} - ${createRes.body}`);
  } else {
    errorRate.add(0);
    
    // Get order details
    const body = JSON.parse(createRes.body);
    const orderID = body.order_id;
    
    sleep(1); // Wait a bit before fetching
    
    const getRes = http.get(`${BASE_URL}/orders/${orderID}`, params);
    check(getRes, {
      'order retrieved successfully': (r) => r.status === 200,
    });
  }

  sleep(Math.random() * 2); // Random sleep between 0-2 seconds
}

// Summary report
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  let summary = '\n';
  summary += `${indent}Checks................: ${data.metrics.checks.values.passes / data.metrics.checks.values.count * 100}% passed\n`;
  summary += `${indent}Requests...............: ${data.metrics.http_reqs.values.count}\n`;
  summary += `${indent}Request duration.......: avg=${data.metrics.http_req_duration.values.avg}ms p95=${data.metrics.http_req_duration.values['p(95)']}ms\n`;
  summary += `${indent}Error rate.............: ${data.metrics.errors ? data.metrics.errors.values.rate * 100 : 0}%\n`;
  
  return summary;
}
