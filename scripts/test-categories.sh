#!/bin/bash
# Test Category Classification System
# Implements selective test execution based on TEST_MODE environment variable

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
TEST_MODE="${TEST_MODE:-fast}"
VERBOSE="${VERBOSE:-false}"
DRY_RUN="${DRY_RUN:-false}"

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Test Category Classification System for yaml-formatter

OPTIONS:
    -m, --mode MODE     Test mode: fast, ci, full, smoke (default: fast)
    -v, --verbose       Enable verbose output
    -d, --dry-run       Show commands without executing
    -h, --help          Show this help message

TEST MODES:
    fast    - Unit tests only (for development)
    ci      - Unit + integration tests (for PR validation)
    full    - All tests (for releases)
    smoke   - Smoke tests only (for post-deployment validation)

ENVIRONMENT VARIABLES:
    TEST_MODE           Override test mode
    VERBOSE             Enable verbose output (true/false)
    DRY_RUN             Enable dry run mode (true/false)
    GO_TEST_FLAGS       Additional flags to pass to go test

EXAMPLES:
    $0                              # Run fast tests
    $0 -m ci                        # Run CI tests
    TEST_MODE=full $0               # Run full test suite
    $0 -m smoke -v                  # Run smoke tests with verbose output
    $0 -d -m full                   # Dry run of full test suite
EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--mode)
                TEST_MODE="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            -d|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

# Function to validate test mode
validate_mode() {
    case "$TEST_MODE" in
        fast|ci|full|smoke)
            print_info "Running tests in '$TEST_MODE' mode"
            ;;
        *)
            print_error "Invalid test mode: '$TEST_MODE'"
            print_error "Valid modes: fast, ci, full, smoke"
            exit 1
            ;;
    esac
}

# Function to check if binary exists for smoke tests
check_binary() {
    local binary_name="yaml-formatter-test"
    
    if [[ ! -f "$binary_name" ]]; then
        print_warning "Binary '$binary_name' not found, building..."
        if [[ "$DRY_RUN" == "true" ]]; then
            print_info "DRY RUN: Would execute: go build -o $binary_name ."
        else
            go build -o "$binary_name" .
            print_success "Binary built successfully"
        fi
    else
        print_info "Using existing binary: $binary_name"
    fi
}

# Function to execute command with optional dry run
execute_cmd() {
    local cmd="$1"
    local description="$2"
    
    print_info "$description"
    
    if [[ "$VERBOSE" == "true" ]]; then
        print_info "Command: $cmd"
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_info "DRY RUN: Would execute: $cmd"
        return 0
    fi
    
    eval "$cmd"
}

# Function to run fast tests (unit only)
run_fast_tests() {
    print_info "Running fast tests (unit only) - optimized for development"
    
    local base_flags="-short"
    local additional_flags="${GO_TEST_FLAGS:-}"
    
    if [[ "$VERBOSE" == "true" ]]; then
        base_flags="$base_flags -v"
    fi
    
    local cmd="go test $base_flags $additional_flags ./internal/..."
    execute_cmd "$cmd" "Executing unit tests"
}

# Function to run CI tests (unit + integration)
run_ci_tests() {
    print_info "Running CI tests (unit + integration) - optimized for PR validation"
    
    local base_flags="-short"
    local additional_flags="${GO_TEST_FLAGS:-}"
    
    if [[ "$VERBOSE" == "true" ]]; then
        base_flags="$base_flags -v"
    fi
    
    # Run unit tests
    local unit_cmd="go test $base_flags $additional_flags ./internal/..."
    execute_cmd "$unit_cmd" "Executing unit tests"
    
    # Run integration tests (requires integration tag)
    local integration_cmd="go test $base_flags -tags=integration $additional_flags ./cmd/..."
    execute_cmd "$integration_cmd" "Executing integration tests"
}

# Function to run full tests (all tests)
run_full_tests() {
    print_info "Running full test suite - optimized for releases"
    
    local base_flags=""
    local additional_flags="${GO_TEST_FLAGS:-}"
    
    if [[ "$VERBOSE" == "true" ]]; then
        base_flags="$base_flags -v"
    fi
    
    # Ensure binary exists for E2E tests
    check_binary
    
    # Run full test suite with all build tags
    local cmd="go test $base_flags -tags=\"integration,e2e,smoke\" $additional_flags ./..."
    execute_cmd "$cmd" "Executing full test suite"
}

# Function to run smoke tests
run_smoke_tests() {
    print_info "Running smoke tests - optimized for post-deployment validation"
    
    # Check if smoke test directory exists
    if [[ ! -d "./tests/smoke" ]]; then
        print_warning "Smoke test directory './tests/smoke' not found"
        print_info "Creating basic smoke test structure..."
        
        if [[ "$DRY_RUN" == "true" ]]; then
            print_info "DRY RUN: Would create smoke test directory and basic test"
        else
            mkdir -p "./tests/smoke"
            cat > "./tests/smoke/basic_test.go" << 'EOF'
//go:build smoke

package smoke

import (
    "os/exec"
    "testing"
)

// TestBinaryExists verifies the binary can be executed
func TestBinaryExists(t *testing.T) {
    cmd := exec.Command("./yaml-formatter-test", "--help")
    if err := cmd.Run(); err != nil {
        t.Fatalf("Binary execution failed: %v", err)
    }
}
EOF
            print_success "Created basic smoke test"
        fi
    fi
    
    local base_flags="-tags=smoke"
    local additional_flags="${GO_TEST_FLAGS:-}"
    
    if [[ "$VERBOSE" == "true" ]]; then
        base_flags="$base_flags -v"
    fi
    
    # Ensure binary exists for smoke tests
    check_binary
    
    local cmd="go test $base_flags $additional_flags ./tests/smoke/..."
    execute_cmd "$cmd" "Executing smoke tests"
}

# Function to display test summary
show_summary() {
    local start_time="$1"
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    print_success "Test execution completed"
    print_info "Mode: $TEST_MODE"
    print_info "Duration: ${duration}s"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_warning "This was a dry run - no tests were actually executed"
    fi
}

# Main execution function
main() {
    local start_time=$(date +%s)
    
    print_info "YAML Formatter Test Category Classification System"
    print_info "================================================="
    
    parse_args "$@"
    validate_mode
    
    case "$TEST_MODE" in
        fast)
            run_fast_tests
            ;;
        ci)
            run_ci_tests
            ;;
        full)
            run_full_tests
            ;;
        smoke)
            run_smoke_tests
            ;;
    esac
    
    show_summary "$start_time"
}

# Execute main function with all arguments
main "$@"