import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '3m', target: 100 },  // Stay at 100 users
    { duration: '1m', target: 500 },  // Ramp up to 500 users
    { duration: '3m', target: 500 },  // Stay at 500 users
    { duration: '1m', target: 1000 }, // Ramp up to 1000 users
    { duration: '3m', target: 1000 }, // Stay at 1000 users
    { duration: '1m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    errors: ['rate<0.1'],             // Error rate should be below 10%
  },
};

// Test data
const BASE_URL = 'http://localhost:8080';
let authToken = '';

// Helper functions
function registerUser(email, password) {
  const payload = JSON.stringify({
    email: email,
    password: password,
  });

  const response = http.post(`${BASE_URL}/auth/register`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(response, {
    'registration successful': (r) => r.status === 201,
  });

  return response;
}

function loginUser(email, password) {
  const payload = JSON.stringify({
    email: email,
    password: password,
  });

  const response = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(response, {
    'login successful': (r) => r.status === 200,
  });

  if (response.status === 200) {
    const data = JSON.parse(response.body);
    return data.token;
  }

  return null;
}

function createChat(token, name) {
  const payload = JSON.stringify({
    name: name,
  });

  const response = http.post(`${BASE_URL}/chats`, payload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  });

  check(response, {
    'chat creation successful': (r) => r.status === 201,
  });

  if (response.status === 201) {
    const data = JSON.parse(response.body);
    return data.id;
  }

  return null;
}

function sendMessage(token, chatId, content) {
  const payload = JSON.stringify({
    chatId: chatId,
    content: content,
  });

  const response = http.post(`${BASE_URL}/messages`, payload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  });

  check(response, {
    'message sent successfully': (r) => r.status === 201,
  });

  return response;
}

function getChatHistory(token, chatId) {
  const response = http.get(`${BASE_URL}/messages/chat/${chatId}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  check(response, {
    'chat history retrieved successfully': (r) => r.status === 200,
  });

  return response;
}

// Test setup
export function setup() {
  // Register and login a test user
  const email = `test${__VU}@example.com`;
  const password = 'password123';

  registerUser(email, password);
  const token = loginUser(email, password);

  if (!token) {
    throw new Error('Failed to login test user');
  }

  return { token };
}

// Main test function
export default function(data) {
  const { token } = data;

  // Create a new chat
  const chatId = createChat(token, `Test Chat ${__VU}`);
  if (!chatId) {
    errorRate.add(1);
    return;
  }

  // Send a message
  const messageResponse = sendMessage(token, chatId, `Test message from VU ${__VU}`);
  if (messageResponse.status !== 201) {
    errorRate.add(1);
  }

  // Get chat history
  const historyResponse = getChatHistory(token, chatId);
  if (historyResponse.status !== 200) {
    errorRate.add(1);
  }

  // Sleep between iterations
  sleep(1);
}

// Test teardown
export function teardown(data) {
  // Cleanup if needed
} 