.PHONY: build run test tidy docker-up docker-down docker-logs fmt vet

APP_NAME := atlas
BIN_DIR := bin

build:
	go build -o $(BIN_DIR)/$(APP_NAME) .

run: build
	set -a && [ -f .env ] && . ./.env; set +a; ./$(BIN_DIR)/$(APP_NAME) worker

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f atlas
