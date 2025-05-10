import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const httpErrorRate = new Rate('http_errors');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp up to 50 connections
    { duration: '1m', target: 50 },   // Stay at 50 connections
    { duration: '30s', target: 100 }, // Ramp up to 100 connections
    { duration: '1m', target: 100 },  // Stay at 100 connections
    { duration: '30s', target: 200 }, // Ramp up to 200 connections
    { duration: '1m', target: 200 },  // Stay at 200 connections
    { duration: '30s', target: 0 },   // Ramp down to 0
  ],
  thresholds: {
    'http_req_duration': ['p(95)<2000'], // 95% of requests should be below 2s
    'http_req_failed': ['rate<0.1'],     // Error rate should be below 10%
    'errors': ['rate<0.1'],              // Custom error rate should be below 10%
    'http_errors': ['rate<0.1'],         // HTTP error rate should be below 10%
  },
  setupTimeout: '2m',  // Increase setup timeout to 2 minutes
};

const BASE_URL = 'http://localhost:8080';

// Setup function to register and login users
export function setup() {
  const testId = uuidv4().substring(0, 8);  // Generate a unique test ID
  const users = [];
  
  for (let i = 0; i < 200; i++) {  // Pre-generate users for all 200 VUs
    const username = `testuser_${testId}_${i + 1}`;  // Add test ID to username
    const password = 'testpass123';

    // Try to login first (in case user exists)
    let token = null;
    const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
      username: username,
      password: password,
    }), {
      headers: { 'Content-Type': 'application/json' },
      timeout: '10s',
    });

    if (loginRes.status === 200) {
      try {
        const response = JSON.parse(loginRes.body);
        if (response.token) {
          token = response.token;
        }
      } catch (e) {
        // Ignore parse error, we'll try to register
      }
    }

    // If login failed, try to register
    if (!token) {
      const registerRes = http.post(`${BASE_URL}/auth/register`, JSON.stringify({
        username: username,
        password: password,
      }), {
        headers: { 'Content-Type': 'application/json' },
        timeout: '10s',
      });

      if (registerRes.status === 200) {
        // After successful registration, try to login
        const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
          username: username,
          password: password,
        }), {
          headers: { 'Content-Type': 'application/json' },
          timeout: '10s',
        });

        if (loginRes.status === 200) {
          try {
            const response = JSON.parse(loginRes.body);
            if (response.token) {
              token = response.token;
            }
          } catch (e) {
            console.error(`Failed to parse login response for user ${username}:`, e);
            httpErrorRate.add(1);
          }
        } else {
          console.error(`Failed to login user ${username} after registration:`, loginRes.status, loginRes.body);
          httpErrorRate.add(1);
        }
      } else {
        console.error(`Failed to register user ${username}:`, registerRes.status, registerRes.body);
        httpErrorRate.add(1);
      }
    }

    if (token) {
      users.push({ username, token });
    }
  }

  // Log the number of successful users
  console.log(`Setup complete. Got ${users.length} valid users out of 200`);

  return { users };
}

// Main test function
export default function(data) {
  const users = data.users;
  if (!users || users.length === 0) {
    console.error('No users available');
    errorRate.add(1);
    return;
  }

  // Select a random user from the pool
  const user = users[Math.floor(Math.random() * users.length)];
  const token = user.token;

  if (!token) {
    console.error('No auth token available');
    errorRate.add(1);
    return;
  }

  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  try {
    // Test health endpoint
    const healthRes = http.get(`${BASE_URL}/health`, {
      headers: headers,
      timeout: '5s',
    });
    check(healthRes, {
      'health status is 200': (r) => r.status === 200,
    });

    if (healthRes.status !== 200) {
      console.error('Health check failed:', healthRes.body);
      httpErrorRate.add(1);
      return;
    }

    // Create a new chat
    const createChatRes = http.post(`${BASE_URL}/chats`, JSON.stringify({
      name: `Test Chat ${__VU}`,
    }), {
      headers: headers,
      timeout: '10s',
    });

    check(createChatRes, {
      'create chat status is 200': (r) => r.status === 200,
    });

    if (createChatRes.status !== 200) {
      console.error('Failed to create chat:', createChatRes.body);
      httpErrorRate.add(1);
      return;
    }

    const chatData = JSON.parse(createChatRes.body);
    const chatId = chatData.id;
    sleep(1);

    // Send a message
    const messageRes = http.post(`${BASE_URL}/messages`, JSON.stringify({
      chat_id: chatId,
      text: `Test message from VU ${__VU}`,
    }), {
      headers: headers,
      timeout: '10s',
    });

    check(messageRes, {
      'send message status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    });

    if (messageRes.status !== 200 && messageRes.status !== 201) {
      console.error('Failed to send message:', messageRes.body);
      httpErrorRate.add(1);
      return;
    }

    // Get chat history
    const historyRes = http.get(`${BASE_URL}/messages/chat/${chatId}`, {
      headers: headers,
      timeout: '10s',
    });

    check(historyRes, {
      'get history status is 200': (r) => r.status === 200,
    });

    if (historyRes.status !== 200) {
      console.error('Failed to get chat history:', historyRes.body);
      httpErrorRate.add(1);
      return;
    }

  } catch (error) {
    console.error('Test iteration failed:', error);
    errorRate.add(1);
  }

  // Add a small sleep between iterations to prevent overwhelming the server
  sleep(1);
} 