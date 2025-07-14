#!/bin/bash
# Parallel Test Execution Script
# Optimizes test execution using CPU-aware parallel settings and resource pooling

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Default values
TEST_MODE="${TEST_MODE:-auto}"
PARALLEL_JOBS="${PARALLEL_JOBS:-auto}"
VERBOSE="${VERBOSE:-false}"
DRY_RUN="${DRY_RUN:-false}"
TIMEOUT="${TIMEOUT:-300}" # 5 minutes default timeout
RACE_DETECTION="${RACE_DETECTION:-true}"

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

print_parallel() {
    echo -e "${PURPLE}[PARALLEL]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Parallel Test Execution Script for yaml-formatter

OPTIONS:
    -m, --mode MODE         Test mode: unit, integration, e2e, all (default: auto)
    -j, --jobs JOBS         Number of parallel jobs: auto, N (default: auto)
    -t, --timeout SECONDS   Test timeout in seconds (default: 300)
    -v, --verbose           Enable verbose output
    -d, --dry-run           Show commands without executing
    --no-race               Disable race detection
    -h, --help              Show this help message

TEST MODES:
    unit        - Unit tests only (high parallelism)
    integration - Integration tests (medium parallelism)
    e2e         - End-to-end tests (low parallelism)
    all         - All tests with optimized parallelism per category

PARALLEL JOBS:
    auto        - CPU-aware automatic selection
    N           - Specific number of parallel jobs (1-16)

ENVIRONMENT VARIABLES:
    TEST_MODE           Override test mode
    PARALLEL_JOBS       Override parallel jobs
    VERBOSE             Enable verbose output (true/false)
    DRY_RUN             Enable dry run mode (true/false)
    TIMEOUT             Test timeout in seconds
    RACE_DETECTION      Enable race detection (true/false)

EXAMPLES:
    $0                              # Auto-detect and run all tests
    $0 -m unit -j 8                 # Run unit tests with 8 parallel jobs
    $0 -m integration --no-race     # Run integration tests without race detection
    $0 -m e2e -t 600 -v             # Run E2E tests with 10min timeout, verbose
    $0 -d -m all                    # Dry run showing all test commands
EOF
}

# Function to detect optimal parallelism
detect_parallelism() {
    local test_mode="$1"
    local cpu_cores=$(nproc)
    local parallel_jobs
    
    case "$test_mode" in
        unit)
            # Unit tests can be more aggressive
            parallel_jobs=$((cpu_cores > 4 ? cpu_cores : cpu_cores + 1))
            ;;
        integration)
            # Moderate parallelism for integration tests
            parallel_jobs=$((cpu_cores * 3 / 4))
            if [[ $parallel_jobs -lt 1 ]]; then
                parallel_jobs=1
            elif [[ $parallel_jobs -gt 8 ]]; then
                parallel_jobs=8
            fi
            ;;
        e2e)
            # Conservative parallelism for E2E tests
            parallel_jobs=$((cpu_cores / 2))
            if [[ $parallel_jobs -lt 1 ]]; then
                parallel_jobs=1
            elif [[ $parallel_jobs -gt 4 ]]; then
                parallel_jobs=4
            fi
            ;;
        all)
            # Balanced approach for mixed tests
            parallel_jobs=$((cpu_cores * 2 / 3))
            if [[ $parallel_jobs -lt 1 ]]; then
                parallel_jobs=1
            elif [[ $parallel_jobs -gt 6 ]]; then
                parallel_jobs=6
            fi
            ;;
        *)
            # Default to moderate parallelism
            parallel_jobs=$((cpu_cores / 2))
            if [[ $parallel_jobs -lt 1 ]]; then
                parallel_jobs=1
            fi
            ;;
    esac
    
    echo "$parallel_jobs"
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--mode)
                TEST_MODE="$2"
                shift 2
                ;;
            -j|--jobs)
                PARALLEL_JOBS="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
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
            --no-race)
                RACE_DETECTION="false"
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

# Function to validate inputs
validate_inputs() {
    # Validate test mode
    case "$TEST_MODE" in
        unit|integration|e2e|all|auto)
            ;;
        *)
            print_error "Invalid test mode: '$TEST_MODE'"
            print_error "Valid modes: unit, integration, e2e, all, auto"
            exit 1
            ;;
    esac
    
    # Validate parallel jobs
    if [[ "$PARALLEL_JOBS" != "auto" ]]; then
        if ! [[ "$PARALLEL_JOBS" =~ ^[0-9]+$ ]] || [[ $PARALLEL_JOBS -lt 1 ]] || [[ $PARALLEL_JOBS -gt 16 ]]; then
            print_error "Invalid parallel jobs: '$PARALLEL_JOBS'"
            print_error "Must be 'auto' or a number between 1 and 16"
            exit 1
        fi
    fi
    
    # Validate timeout
    if ! [[ "$TIMEOUT" =~ ^[0-9]+$ ]] || [[ $TIMEOUT -lt 1 ]]; then
        print_error "Invalid timeout: '$TIMEOUT'"
        print_error "Must be a positive number"
        exit 1
    fi
}

# Function to execute command with optional dry run
execute_cmd() {
    local cmd="$1"
    local description="$2"
    
    print_parallel "$description"
    
    if [[ "$VERBOSE" == "true" ]]; then
        print_info "Command: $cmd"
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_info "DRY RUN: Would execute: $cmd"
        return 0
    fi
    
    eval "timeout ${TIMEOUT}s $cmd"
}

# Function to build test flags
build_test_flags() {
    local test_mode="$1"
    local parallel_jobs="$2"
    local flags=""
    
    # Add parallel flag
    flags="$flags -parallel=$parallel_jobs"
    
    # Add race detection if enabled
    if [[ "$RACE_DETECTION" == "true" ]]; then
        flags="$flags -race"
    fi
    
    # Add verbose flag if enabled
    if [[ "$VERBOSE" == "true" ]]; then
        flags="$flags -v"
    fi
    
    # Add timeout flag
    flags="$flags -timeout=${TIMEOUT}s"
    
    # Add short flag for faster feedback on unit tests
    if [[ "$test_mode" == "unit" ]]; then
        flags="$flags -short"
    fi
    
    echo "$flags"
}

# Function to run unit tests in parallel
run_unit_tests() {
    local parallel_jobs="$1"
    print_info "Running unit tests with $parallel_jobs parallel jobs"
    
    local flags
    flags=$(build_test_flags "unit" "$parallel_jobs")
    
    local cmd="go test $flags ./internal/..."
    execute_cmd "$cmd" "Executing parallel unit tests"
}

# Function to run integration tests in parallel
run_integration_tests() {
    local parallel_jobs="$1"
    print_info "Running integration tests with $parallel_jobs parallel jobs"
    
    local flags
    flags=$(build_test_flags "integration" "$parallel_jobs")
    
    local cmd="go test $flags -tags=integration ./cmd/..."
    execute_cmd "$cmd" "Executing parallel integration tests"
}

# Function to run E2E tests in parallel
run_e2e_tests() {
    local parallel_jobs="$1"
    print_info "Running E2E tests with $parallel_jobs parallel jobs"
    
    # Ensure binary exists for E2E tests
    if [[ ! -f "yaml-formatter-test" ]]; then
        print_info "Building test binary for E2E tests..."
        if [[ "$DRY_RUN" == "true" ]]; then
            print_info "DRY RUN: Would execute: go build -o yaml-formatter-test ."
        else
            go build -o yaml-formatter-test .
        fi
    fi
    
    local flags
    flags=$(build_test_flags "e2e" "$parallel_jobs")
    
    local cmd="go test $flags -tags=e2e ./tests/e2e/..."
    execute_cmd "$cmd" "Executing parallel E2E tests"
}

# Function to run all tests with optimized parallelism
run_all_tests() {
    print_info "Running all tests with category-specific parallelism"
    
    # Different parallelism for each category
    local unit_jobs
    local integration_jobs  
    local e2e_jobs
    
    if [[ "$PARALLEL_JOBS" == "auto" ]]; then
        unit_jobs=$(detect_parallelism "unit")
        integration_jobs=$(detect_parallelism "integration")
        e2e_jobs=$(detect_parallelism "e2e")
    else
        # Use specified jobs but adjust for test type
        unit_jobs="$PARALLEL_JOBS"
        integration_jobs=$(($PARALLEL_JOBS * 3 / 4))
        e2e_jobs=$(($PARALLEL_JOBS / 2))
        
        # Ensure minimums
        if [[ $integration_jobs -lt 1 ]]; then integration_jobs=1; fi
        if [[ $e2e_jobs -lt 1 ]]; then e2e_jobs=1; fi
    fi
    
    print_parallel "Unit tests: $unit_jobs jobs | Integration tests: $integration_jobs jobs | E2E tests: $e2e_jobs jobs"
    
    # Run tests sequentially by category but parallel within category
    run_unit_tests "$unit_jobs"
    run_integration_tests "$integration_jobs"
    run_e2e_tests "$e2e_jobs"
}

# Function to auto-detect test mode
auto_detect_mode() {
    # Simple heuristic: if running in CI or with specific flags, use 'all'
    if [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]]; then
        echo "all"
    else
        # For local development, default to unit tests for faster feedback
        echo "unit"
    fi
}

# Function to display test summary
show_summary() {
    local start_time="$1"
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local cpu_cores=$(nproc)
    
    print_success "Parallel test execution completed"
    print_info "Test mode: $TEST_MODE"
    print_info "Parallel jobs: $PARALLEL_JOBS"
    print_info "CPU cores: $cpu_cores"
    print_info "Duration: ${duration}s"
    print_info "Race detection: $RACE_DETECTION"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_warning "This was a dry run - no tests were actually executed"
    fi
}

# Main execution function
main() {
    local start_time=$(date +%s)
    
    print_info "YAML Formatter Parallel Test Execution"
    print_info "======================================"
    
    parse_args "$@"
    validate_inputs
    
    # Auto-detect test mode if needed
    if [[ "$TEST_MODE" == "auto" ]]; then
        TEST_MODE=$(auto_detect_mode)
        print_info "Auto-detected test mode: $TEST_MODE"
    fi
    
    # Calculate parallel jobs if auto
    if [[ "$PARALLEL_JOBS" == "auto" ]]; then
        PARALLEL_JOBS=$(detect_parallelism "$TEST_MODE")
        print_info "Auto-detected parallel jobs: $PARALLEL_JOBS"
    fi
    
    print_info "Configuration:"
    print_info "  Mode: $TEST_MODE"
    print_info "  Parallel jobs: $PARALLEL_JOBS"
    print_info "  Timeout: ${TIMEOUT}s"
    print_info "  Race detection: $RACE_DETECTION"
    print_info "  CPU cores: $(nproc)"
    
    case "$TEST_MODE" in
        unit)
            run_unit_tests "$PARALLEL_JOBS"
            ;;
        integration)
            run_integration_tests "$PARALLEL_JOBS"
            ;;
        e2e)
            run_e2e_tests "$PARALLEL_JOBS"
            ;;
        all)
            run_all_tests
            ;;
    esac
    
    show_summary "$start_time"
}

# Execute main function with all arguments
main "$@"