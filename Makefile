.PHONY: dev build test lint docker-up docker-down seed

dev:
	docker compose -f deployments/docker-compose.yml up -d postgres redis nats
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test ./...

seed:
	go run ./scripts/seed.go

lint:
	golangci-lint run ./...

docker-up:
	docker compose -f deployments/docker-compose.yml up --build

docker-down:
	docker compose -f deployments/docker-compose.yml down
