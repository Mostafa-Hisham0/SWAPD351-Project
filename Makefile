.PHONY: build run test clean migrate test-page

build:
	go build -o messaging-service ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

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