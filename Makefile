.PHONY: build run test test-cover lint migrate-up migrate-down tidy docker-up docker-down docs docs-check

BIN=bin/api
MAIN=src/cmd/api/main.go
COVERAGE_OUT=coverage.out
COVERAGE_HTML=coverage.html

build:
	go build -o $(BIN) ./$(MAIN)

run:
	./$(BIN)

dev:
	go run ./$(MAIN)

test:
	go test ./... -race -count=1

test-cover:
	go test ./... -race -count=1 -coverprofile=$(COVERAGE_OUT) ./...
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/migrate/main.go up

migrate-down:
	go run ./cmd/migrate/main.go down

docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

generate-mocks:
	go generate ./...

docs:
	swag init -g src/cmd/api/main.go -o src/docs --parseDependency --parseInternal

docs-check:
	@$(MAKE) docs
	@git diff --exit-code src/docs || (echo "ERROR: API docs are stale. Run 'make docs' and commit the changes." && exit 1)

.DEFAULT_GOAL := build
