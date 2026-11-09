.PHONY: build run test test-integration lint migrate-up migrate-down docker-up docker-down load-test

build:
	go build ./cmd/api

run:
	go run ./cmd/api

test:
	go test -race ./internal/...

test-integration:
	go test -race -tags=integration ./internal/postgres

lint:
	go vet ./...
	cd web && npm run lint

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

load-test:
	k6 run loadtest/links.js
