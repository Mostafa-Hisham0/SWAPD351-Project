import { Rate } from 'k6/metrics';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import http from 'k6/http';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Test data
const BASE_URL = 'http://localhost:8080';

// Custom metrics
const errorRate = new Rate('errors');
const messageRate = new Rate('messages_sent');
const connectionRate = new Rate('connections_successful');
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
    'errors': ['rate<0.1'],   // Error rate should be below 10%
    'messages_sent': ['rate>0.9'],  // Message success rate should be above 90%
    'connections_successful': ['rate>0.95'],  // Connection success rate should be above 95%
    'http_errors': ['rate<0.1'],  // HTTP error rate should be below 10%
  },
  setupTimeout: '2m',  // Increase setup timeout to 2 minutes
};

// Setup function to register and login users
export function setup() {
  const testId = uuidv4().substring(0, 8);  // Generate a unique test ID
  const tokens = [];
  
  for (let i = 0; i < 200; i++) {  // Pre-generate tokens for all 200 VUs
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

    tokens.push(token);
  }

  // Log the number of successful tokens
  const validTokens = tokens.filter(t => t !== null).length;
  console.log(`Setup complete. Got ${validTokens} valid tokens out of ${tokens.length}`);

  return { tokens };
}

// Main test function
export default function(data) {
  const token = data.tokens[__VU - 1];

  if (!token) {
    console.error('No auth token available');
    errorRate.add(1);
    return;
  }

  const url = `ws://localhost:8080/ws?token=${token}`;
  const params = {
    headers: {
      'User-Agent': 'k6/0.0.0',
    },
    timeout: '30s',
  };

  let wsError = false;
  const res = ws.connect(url, params, function(socket) {
    socket.on('open', function() {
      console.log('WebSocket connection established');
      connectionRate.add(1);
    });

    socket.on('message', function(message) {
      try {
        const data = JSON.parse(message);
        console.log('Received message:', data);
      } catch (e) {
        console.error('Failed to parse message:', e);
      }
    });

    socket.on('error', function(e) {
      console.error('WebSocket error:', e);
      wsError = true;
    });

    socket.on('close', function() {
      console.log('WebSocket connection closed');
      if (wsError) {
        errorRate.add(1);
      }
    });

    // Send a message every 10 seconds
    socket.setInterval(function() {
      try {
        const message = {
          text: `Test message from VU ${__VU}`,
        };

        socket.send(JSON.stringify(message));
        messageRate.add(1);
      } catch (e) {
        console.error('Failed to send message:', e);
        errorRate.add(1);
      }
    }, 10000);

    // Close the connection after 1 minute
    socket.setTimeout(function() {
      socket.close();
    }, 60000);
  });

  check(res, {
    'WebSocket connection established': (r) => r && r.status === 101,
  });

  sleep(1);
} 