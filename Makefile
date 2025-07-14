.PHONY: build test test-unit test-integration test-e2e test-fast test-ci test-full test-smoke coverage clean

# Binary name
BINARY_NAME=sb-yaml

# Build the binary
build:
	go build -o $(BINARY_NAME) .

# Run all tests (default to CI mode for comprehensive testing)
test: test-ci

# Run unit tests
test-unit:
	go test -short ./internal/...

# Run integration tests (requires integration tag)
test-integration:
	go test -short -tags=integration ./cmd/...

# Run end-to-end tests (requires e2e tag and built binary)
test-e2e: build-test
	go test -tags=e2e ./tests/e2e/...

# Categorized test targets using test-categories.sh script

# Fast tests (unit only) - for development
test-fast:
	./scripts/test-categories.sh -m fast

# CI tests (unit + integration) - for PR validation
test-ci:
	./scripts/test-categories.sh -m ci

# Full tests (all tests) - for releases
test-full:
	./scripts/test-categories.sh -m full

# Smoke tests - for post-deployment validation
test-smoke: build-test
	./scripts/test-categories.sh -m smoke

# Build test binary for smoke tests
build-test:
	go build -o yaml-formatter-test .

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