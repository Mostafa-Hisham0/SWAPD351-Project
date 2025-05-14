# Real-Time Chat System (RTCS)

<div align="center">

![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat&logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)
![License](https://img.shields.io/badge/License-MIT-yellow.svg)

</div>

A robust, high-performance real-time chat system built with Go, featuring WebSocket communication, microservices architecture, and containerized deployment. This project demonstrates enterprise-level design patterns, clean architecture, and best practices in building scalable real-time applications.

## Key Features

- **Real-time Messaging** - Instant message delivery using WebSockets with rate limiting
- **User Authentication** - Secure JWT-based authentication system
- **Chat Management** - Create, join, and leave chat rooms
- **Message Persistence** - Database storage of messages with caching layer
- **Cross-Device Compatibility** - Works across devices on local networks
- **Security** - CORS protection, input validation, and origin checking
- **Containerization** - Complete Docker setup for easy deployment
- **Testing Tools** - Interactive WebSocket test page

## Technical Architecture

### Project Structure

```
.
├── cmd/                  # Application entry points
│   └── server/          # Main server application
├── internal/            # Application code (protected by Go's build system)
│   ├── transport/      # HTTP and WebSocket handlers
│   ├── service/        # Business logic layer
│   ├── repository/     # Data access layer
│   ├── cache/          # Redis caching implementation
│   ├── middleware/     # HTTP middleware components
│   ├── config/         # Application configuration
│   └── model/          # Data models and entities
├── public/             # Static assets and test pages
├── migrations/         # Database migration scripts
├── .env and .env.example # Environment configuration
├── docker-compose.yml  # Multi-container Docker setup
├── Dockerfile          # Container build instructions
└── Makefile            # Development and build automation
```

### Architecture Pattern

The application follows clean architecture principles with clear separation of concerns:

- **Transport Layer** - Handles HTTP requests and WebSocket connections
- **Service Layer** - Contains business logic and orchestrates operations
- **Repository Layer** - Abstracts data access and storage
- **Model Layer** - Defines data structures and entities

### Technology Stack

- **Backend** - Go 1.23 with standard library and minimal dependencies
- **Database** - PostgreSQL 15 for persistent storage
- **Caching** - Redis 7 for high-performance data caching
- **API** - RESTful HTTP endpoints + WebSocket for real-time communication
- **Authentication** - JWT (JSON Web Tokens)
- **Containerization** - Docker with multi-stage builds
- **ORM** - GORM for database operations

## Quick Start Guide

### Prerequisites

- Docker and Docker Compose
- Git

### Installation

1. Clone the repository:
```bash
git clone https://github.com/Mostafa-Hisham0/SWAPD351-Project
cd SWAPD351-Project
```

2. Start all services using Docker Compose:
```bash
docker-compose up -d
```

3. The application will be available at:
```
http://localhost:8080
```

4. Access the WebSocket test interface:
```
http://localhost:8080/test_websocket.html
```

### Local Network Access

To access from other devices on your local network:

1. Find your machine's IP address
2. Access the application at:
```
http://YOUR_IP_ADDRESS:8080/test_websocket.html
```

## Development Guide

### Manual Setup

For development without Docker:

1. Install Go 1.23 or higher
2. Install PostgreSQL 15 and Redis 7
3. Configure environment variables:
```bash
cp .env.example .env
# Edit .env with your local configuration
```

4. Install dependencies:
```bash
go mod download
```

5. Run database migrations:
```bash
make migrate-up
```

6. Start the application:
```bash
make run
```

### Testing

Run the automated test suite:
```bash
make test
```

For API testing:
1. Register a user at `/auth/register`
2. Login to get a JWT token at `/auth/login`
3. Use the token in the Authorization header for subsequent requests

## API Documentation

### Authentication Endpoints

- `POST /auth/register` - Register a new user
  - Request: `{"username": "string", "password": "string"}`
  - Response: `{"id": "uuid", "username": "string"}`

- `POST /auth/login` - Login and obtain JWT token
  - Request: `{"username": "string", "password": "string"}`
  - Response: `{"token": "string"}`

### Chat Endpoints

- `GET /chats` - Get user's chats
  - Auth: JWT token required
  - Response: `[{"id":"uuid", "name":"string", "created_at":"time", "updated_at":"time"}]`

- `POST /chats` - Create a new chat
  - Auth: JWT token required
  - Request: `{"name": "string"}`
  - Response: `{"id":"uuid", "name":"string", "created_at":"time", "updated_at":"time"}`

- `GET /chats/{id}` - Get chat details
  - Auth: JWT token required
  - Response: `{"id":"uuid", "name":"string", "created_at":"time", "updated_at":"time"}`

### Message Endpoints

- `GET /messages/chat/{id}` - Get chat messages
  - Auth: JWT token required
  - Response: `[{"id":"uuid", "chat_id":"uuid", "sender_id":"uuid", "text":"string", "created_at":"time"}]`

- `POST /messages` - Send a message
  - Auth: JWT token required
  - Request: `{"chat_id": "uuid", "text": "string"}`
  - Response: Message object

- `DELETE /messages/{id}` - Delete a message
  - Auth: JWT token required
  - Response: Status 204 No Content

### WebSocket Interface

Connect to the WebSocket endpoint at `/ws` with a valid JWT token for real-time communication.

#### Message Types

- User Join: `{"type": "user_join", "userId": "string"}`
- User Leave: `{"type": "user_leave", "userId": "string"}`
- Chat Message: `{"type": "message", "text": "string"}`
- User List: `{"type": "user_list", "users": ["string"]}`

## Security Features

- JWT-based authentication with proper token validation
- Password hashing using bcrypt
- Rate limiting on WebSocket connections (5 messages/second)
- Input validation and sanitization
- CORS protection for API endpoints
- Origin checking for WebSocket connections
- Database query protection against SQL injection via ORM

## Performance Optimizations

- Redis caching for frequently accessed data
- Database connection pooling
- Efficient WebSocket broadcasting with client tracking
- Docker multi-stage builds for smaller image size
- Proper error handling and graceful degradation

## Load Testing Results

The system has been tested under significant load with both HTTP and WebSocket connections:

### HTTP Performance (200 Concurrent Users)
- Average response time: 90.61ms
- 95th percentile response time: 245.77ms
- Request rate: 157 requests/second
- Error rate: 0.36%
- Success rate: 100%
- Total requests: 54,196

### WebSocket Performance (200 Concurrent Connections)
- Connection success rate: 100%
- Message success rate: 100%
- Average connection time: 8.98ms
- Message throughput: 1,275 messages/second
- Total messages received: 455,623
- Total messages sent: 3,271
- Zero connection drops or timeouts

### Load Test Configuration
The system was tested using k6 with the following stages:
1. Ramp up to 50 concurrent users over 30 seconds
2. Maintain 50 users for 1 minute
3. Ramp up to 100 users over 30 seconds
4. Maintain 100 users for 1 minute
5. Ramp up to 200 users over 30 seconds
6. Maintain 200 users for 1 minute
7. Ramp down to 0 over 30 seconds

### Test Scripts
Load test scripts are available in the `load-tests` directory:
- `http_test.js` - HTTP API load testing
- `websocket_test.js` - WebSocket connection and message testing

To run the load tests:
```bash
cd load-tests
k6 run http_test.js    # For HTTP testing
k6 run websocket_test.js  # For WebSocket testing
```

## Recent Improvements

1. **Database Schema Enhancement**
   - Added `updated_at` column to messages table
   - Improved message tracking and history

2. **Load Testing Infrastructure**
   - Implemented comprehensive load testing suite
   - Added custom metrics for error tracking
   - Improved test user management with unique IDs

3. **Authentication Improvements**
   - Enhanced token handling in WebSocket connections
   - Improved user registration and login flow
   - Better error handling for authentication failures

4. **Performance Optimizations**
   - Optimized WebSocket connection handling
   - Improved message broadcasting efficiency
   - Enhanced database query performance

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Commit your changes: `git commit -m 'Add feature'`
4. Push to the branch: `git push origin feature-name`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Project Team

- [Mostafa-Hisham0](https://github.com/Mostafa-Hisham0) 

---

*This Real-Time Chat System was developed as part of SWAPD351 course project.*

## Testing

The project includes comprehensive testing across multiple levels:

### Unit Tests

Unit tests focus on testing individual components in isolation. They are located in the `tests/unit` directory and use mocks to simulate dependencies.

To run unit tests:
```bash
make test-unit
```

### Integration Tests

Integration tests verify that different components work together correctly. They are located in the `tests/integration` directory and use test containers to spin up real dependencies.

To run integration tests:
```bash
make test-integration
```

### Contract Tests

Contract tests ensure that the API contracts between the client and server are maintained. They use Pact to define and verify these contracts. The tests are located in:

- Consumer tests: `tests/contract/consumer`
- Provider tests: `tests/contract/provider`

To run contract tests:
```bash
make test-contract
```

### Load Tests

Load tests simulate real-world usage patterns and verify system performance under load. They use k6 to generate load and measure performance metrics. The test scenarios are located in `tests/load/scenarios`.

To run load tests:
```bash
make test-load
```

## Test Dependencies

The project uses several testing tools:

- `testify`: For assertions and test utilities
- `gomock`: For generating and using mocks
- `testcontainers-go`: For managing test containers
- `pact-go`: For contract testing
- `k6`: For load testing

To install all test dependencies:
```bash
make deps
```

## Generating Mocks

The project uses mocks for unit testing. To generate mocks:
```bash
make generate
```

## Cleaning Up

To clean up test artifacts:
```bash
make clean
```

## Running All Tests

To run all tests:
```bash
make test
```

## Test Coverage

The project aims for high test coverage. You can generate a coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Continuous Integration

The tests are integrated into the CI pipeline and run automatically on every push to the main branch. The pipeline includes:

1. Running unit tests
2. Running integration tests
3. Running contract tests
4. Running load tests
5. Generating and uploading coverage reports

## Best Practices

When writing tests, follow these best practices:

1. Use descriptive test names that explain the behavior being tested
2. Follow the Arrange-Act-Assert pattern
3. Keep tests focused and test one thing at a time
4. Use table-driven tests for testing multiple scenarios
5. Mock external dependencies to keep tests fast and reliable
6. Use test containers for integration tests
7. Write contract tests for all API endpoints
8. Include load tests for performance-critical paths

## Troubleshooting

If you encounter issues with the tests:

1. Make sure all dependencies are installed: `make deps`
2. Check that Docker is running (required for test containers)
3. Verify that the database is accessible
4. Check the test logs for detailed error messages
5. Run tests with verbose output: `go test -v ./...`
