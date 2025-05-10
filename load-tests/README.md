# Load Testing Suite

This directory contains load testing scripts for the RTCS application using k6. The tests are designed to evaluate the system's performance under high concurrency for both HTTP endpoints and WebSocket connections.

## Prerequisites

1. Install k6:
   - Windows: `choco install k6`
   - macOS: `brew install k6`
   - Linux: `sudo gpg -k && sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69 && echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list && sudo apt-get update && sudo apt-get install k6`

2. Make sure the RTCS application is running:
   ```bash
   docker-compose up -d
   ```

## Test Scripts

1. `http_test.js`: Tests HTTP endpoints including:
   - User registration and authentication
   - Chat creation and listing
   - Message sending and retrieval
   - Health check endpoint

2. `websocket_test.js`: Tests WebSocket functionality including:
   - Connection establishment
   - User join/leave events
   - Real-time message sending and receiving
   - Connection stability under load

## Running the Tests

### HTTP Load Test

```bash
k6 run http_test.js
```

This will:
- Ramp up to 50 virtual users over 1 minute
- Maintain 50 users for 3 minutes
- Ramp up to 100 users over 1 minute
- Maintain 100 users for 3 minutes
- Ramp down to 0 users over 1 minute

### WebSocket Load Test

```bash
k6 run websocket_test.js
```

This will:
- Ramp up to 100 concurrent WebSocket connections over 1 minute
- Maintain 100 connections for 3 minutes
- Ramp up to 200 connections over 1 minute
- Maintain 200 connections for 3 minutes
- Ramp down to 0 connections over 1 minute

## Test Metrics

The tests collect and report the following metrics:

### HTTP Test Metrics
- Request duration (p95 < 500ms)
- Error rate (< 10%)
- HTTP request success rate
- Response time distribution

### WebSocket Test Metrics
- Connection success rate
- Message delivery rate
- Error rate
- Connection stability
- Message latency

## Interpreting Results

1. Check the console output for real-time metrics
2. Look for any failed checks or errors
3. Monitor the following key indicators:
   - Error rates should be below thresholds
   - Response times should be within acceptable ranges
   - Connection success rates should be high
   - Message delivery should be reliable

## Customizing Tests

You can modify the test parameters in each script:

1. Adjust the number of virtual users in the `stages` configuration
2. Modify the test duration in each stage
3. Change the thresholds for success criteria
4. Add or remove test scenarios as needed

## Best Practices

1. Run tests in a staging environment first
2. Monitor system resources during tests
3. Start with lower concurrency and gradually increase
4. Keep test data separate from production data
5. Clean up test data after running tests 