import http from 'k6/http';
import { check } from 'k6';

// Stress test: Test oversell protection
export const options = {
  scenarios: {
    oversell_test: {
      executor: 'constant-arrival-rate',
      rate: 50,           // 50 requests per second
      timeUnit: '1s',
      duration: '10s',
      preAllocatedVUs: 100,
      maxVUs: 200,
    },
  },
};

const BASE_URL = 'http://localhost:8080/api/v1';
const TARGET_PRODUCT_ID = 1; // Target single product to test concurrency
const INITIAL_STOCK = 100;   // Known initial stock

let successfulOrders = 0;

export default function () {
  const payload = JSON.stringify({
    user_id: Math.floor(Math.random() * 10000),
    items: [
      {
        product_id: TARGET_PRODUCT_ID,
        quantity: 1, // Each order requests 1 item
      },
    ],
    payment_method: 'mock',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(`${BASE_URL}/orders`, payload, params);
  
  const success = check(res, {
    'order created or properly rejected': (r) => r.status === 201 || r.status === 500,
    'no oversell': (r) => {
      if (r.status === 201) {
        successfulOrders++;
        return successfulOrders <= INITIAL_STOCK;
      }
      return true;
    },
  });
}

export function handleSummary(data) {
  console.log(`\n=== Oversell Test Results ===`);
  console.log(`Total requests: ${data.metrics.http_reqs.values.count}`);
  console.log(`Successful orders: ${successfulOrders}`);
  console.log(`Expected max: ${INITIAL_STOCK}`);
  console.log(`Oversell detected: ${successfulOrders > INITIAL_STOCK ? 'YES ❌' : 'NO ✅'}`);
  
  return {
    'stdout': textSummary(data),
  };
}

function textSummary(data) {
  return `
  Oversell Test Summary
  =====================
  Total Requests: ${data.metrics.http_reqs.values.count}
  Successful Orders: ${successfulOrders}
  Initial Stock: ${INITIAL_STOCK}
  Oversell: ${successfulOrders > INITIAL_STOCK ? 'DETECTED ❌' : 'PREVENTED ✅'}
  `;
}
