.PHONY: build test test-unit test-integration test-e2e coverage clean

# Binary name
BINARY_NAME=sb-yaml

# Build the binary
build:
	go build -o $(BINARY_NAME) .

# Run all tests
test: test-unit test-integration test-e2e

# Run unit tests
test-unit:
	go test ./internal/...

# Run integration tests
test-integration:
	go test ./cmd/...

# Run end-to-end tests (requires built binary)
test-e2e: build
	go test ./tests/e2e/...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run specific package tests
test-formatter:
	go test -v ./internal/formatter/...

test-schema:
	go test -v ./internal/schema/...

test-utils:
	go test -v ./internal/utils/...

test-config:
	go test -v ./internal/config/...

# Benchmark tests
bench:
	go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...