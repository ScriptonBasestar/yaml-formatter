#!/bin/bash
# Simple test execution for yaml-formatter

set -e

MODE="${1:-all}"

case "$MODE" in
    "unit")
        echo "Running unit tests..."
        go test ./internal/...
        ;;
    "all"|*)
        echo "Running all tests..."
        go test ./...
        ;;
esac

echo "Tests completed successfully"