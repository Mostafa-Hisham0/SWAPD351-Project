.PHONY: build run test clean migrate test-page test-unit test-integration test-contract test-load deps generate

build:
	go build -o messaging-service ./cmd/server

run:
	go run ./cmd/server

test: test-unit test-integration test-contract test-load

clean:
	rm -f messaging-service

migrate:
	docker-compose up -d postgres
	./scripts/migrate.sh

docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

test-page:
	@echo "Opening WebSocket test page..."
	@start http://localhost:8080/test_websocket.html

# Run unit tests
test-unit:
	go test -v ./tests/unit/...

# Run integration tests
test-integration:
	go test -v ./tests/integration/...

# Run contract tests
test-contract:
	# Run consumer tests
	go test -v ./tests/contract/consumer/...
	# Run provider tests
	go test -v ./tests/contract/provider/...

# Run load tests
test-load:
	k6 run ./tests/load/scenarios/chat_load.js

# Install dependencies
deps:
	go mod download
	go install github.com/golang/mock/mockgen@latest
	go install github.com/pact-foundation/pact-go@latest
	go install github.com/k6io/k6@latest

# Generate mocks
generate:
	mockgen -source=internal/repository/user.go -destination=tests/unit/mocks/user_repository_mock.go
	mockgen -source=internal/repository/chat.go -destination=tests/unit/mocks/chat_repository_mock.go
	mockgen -source=internal/repository/message.go -destination=tests/unit/mocks/message_repository_mock.go

# Clean up test artifacts
clean:
	rm -rf tests/unit/mocks/*_mock.go
	rm -rf tests/contract/consumer/pacts/*
	rm -rf tests/contract/provider/pacts/* 