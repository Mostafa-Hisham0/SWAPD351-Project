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

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Commit your changes: `git commit -m 'Add feature'`
4. Push to the branch: `git push origin feature-name`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Project Team

- [Mostafa-Hisham0](https://github.com/Mostafa-Hisham0) - Lead Developer

---

*This Real-Time Chat System was developed as part of SWAPD351 course project.*
