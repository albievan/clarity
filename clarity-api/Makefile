.PHONY: run build test lint tidy docker-build docker-run env-run

# Run directly — auto-loads .env from the working directory if present
run:
	go run ./cmd/api/.

# Build the binary
build:
	go build -o bin/clarity-api ./cmd/api/.

# Run the pre-built binary (also auto-loads .env)
start: build
	./bin/clarity-api

# Run tests with race detection and coverage report
test:
	go test -race -cover ./...

# Lint (requires golangci-lint: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

# Tidy dependencies
tidy:
	go mod tidy

# ── Docker targets ─────────────────────────────────────────────

# Build Docker image
docker-build:
	docker build -t clarity-api:latest .

# Run Docker container — passes .env file directly to the container
docker-run:
	docker run --rm -p 8080:8080 --env-file .env clarity-api:latest

# ── Alternative: explicit env passing without .env auto-load ──

# Export all vars from .env and run (works in bash/zsh)
# Usage: make env-run
env-run: build
	set -a && . ./.env && set +a && ./bin/clarity-api
