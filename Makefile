.PHONY: build run run-server test test-cover tidy docker-up docker-down docker-logs docker-db fmt vet openapi

APP_NAME := atlas
BIN_DIR := bin

build:
	go build -o $(BIN_DIR)/$(APP_NAME) .

run: build
	set -a && [ -f .env ] && . ./.env; set +a; ./$(BIN_DIR)/$(APP_NAME) worker

run-server: build
	set -a && [ -f .env ] && . ./.env; set +a; ./$(BIN_DIR)/$(APP_NAME) server

openapi: build
	set -a && [ -f .env ] && . ./.env; set +a; ./$(BIN_DIR)/$(APP_NAME) openapi

test:
	go test ./...

test-cover:
	@chmod +x scripts/check-coverage.sh
	@./scripts/check-coverage.sh

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

docker-db:
	docker compose up -d postgres clickhouse

docker-logs:
	docker compose logs -f atlas server
