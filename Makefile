.PHONY: run build fmt lint test swagger migrate-up migrate-down verify ci docker-build docker-up docker-down

# Local run
run:
	go run ./cmd/api

# Compile binary
build:
	go build -o bin/linkpulse ./cmd/api

# Format Go files
fmt:
	go fmt ./...

# Static code analysis
lint:
	go vet ./...

# Run test suite
test:
	go test -v -race -cover ./...

# Compile Swagger specs (requires github.com/swaggo/swag/cmd/swag)
swagger:
	swag init -g cmd/api/main.go -o docs

# Run db migrations up (Requires golang-migrate CLI)
migrate-up:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/linkpulse_db?sslmode=disable" up

# Run db migrations down
migrate-down:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/linkpulse_db?sslmode=disable" down

# Verify local code quality (tidy check, formatting, vet, test, build)
verify:
	go mod tidy
	go fmt ./...
	go vet ./...
	go test -race ./...
	go build ./...

# CI alias command
ci: verify

# Build docker image locally
docker-build:
	docker build -t linkpulse:latest .

# Spin up Docker containers
docker-up:
	docker-compose up -d --build

# Tear down Docker containers
docker-down:
	docker-compose down
