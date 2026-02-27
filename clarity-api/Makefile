.PHONY: run build test lint tidy

run:
	go run ./cmd/api/.

build:
	go build -o bin/clarity-api ./cmd/api/.

test:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

# Docker targets
docker-build:
	docker build -t clarity-api:latest .

docker-run:
	docker run -p 8080:8080 --env-file .env clarity-api:latest
