.PHONY: dev build web-install web-build test lint docker-up docker-down seed migrate-up migrate-down migrate-version migrate-create

dev:
	docker compose -f deployments/docker-compose.yml up -d postgres redis nats
	cd web && npm run build
	go run ./cmd/server

build: web-build
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

web-install:
	cd web && npm install

web-build:
	cd web && npm run build

test:
	go test ./...

seed:
	go run ./scripts/seed.go

migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down 1

migrate-version:
	./scripts/migrate.sh version

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

lint:
	golangci-lint run ./...

docker-up:
	docker compose -f deployments/docker-compose.yml up --build

docker-down:
	docker compose -f deployments/docker-compose.yml down
