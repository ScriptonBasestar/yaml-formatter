#!/bin/bash
# Simple parallel test execution for yaml-formatter

set -e

# Run tests in parallel based on CPU count
PARALLEL_JOBS=${PARALLEL_JOBS:-$(nproc)}

echo "Running tests with $PARALLEL_JOBS parallel jobs..."

# Run unit tests
go test -parallel $PARALLEL_JOBS ./...

echo "All tests completed successfully"