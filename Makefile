.PHONY: dev backend frontend test lint build docker-build docker-up

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

backend:
	go run ./cmd/pgsentinel

frontend:
	npm --prefix frontend run dev

test:
	go test ./...
	npm --prefix frontend test -- --run

lint:
	test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './frontend/*'))"
	go vet ./...
	golangci-lint run ./...
	npm --prefix frontend run lint
	npm --prefix frontend run typecheck

build:
	npm --prefix frontend run build
	go build -o bin/pgsentinel ./cmd/pgsentinel

docker-build:
	docker build -t pgsentinel:local .

docker-up:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
