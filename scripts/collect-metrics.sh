#!/bin/bash

# Test Metrics Collection Script
# Collects and analyzes test execution metrics for CI/CD optimization

set -euo pipefail

# Configuration
METRICS_DIR="${METRICS_DIR:-./test-metrics}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-json}"
TEST_CATEGORY="${TEST_CATEGORY:-all}"
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Test Metrics Collection Script

Usage: $0 [OPTIONS]

Options:
    -c, --category CATEGORY     Test category to analyze (unit|integration|e2e|all) [default: all]
    -d, --dir DIRECTORY        Metrics output directory [default: ./test-metrics]
    -f, --format FORMAT        Output format (json|csv|text) [default: json]
    -v, --verbose              Enable verbose output
    -h, --help                 Show this help message

Examples:
    $0 --category unit --format json
    $0 --dir /tmp/metrics --verbose
    $0 --category integration --format csv

Environment Variables:
    METRICS_DIR                Default metrics directory
    OUTPUT_FORMAT              Default output format
    TEST_CATEGORY              Default test category
    VERBOSE                    Enable verbose mode (true/false)
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--category)
                TEST_CATEGORY="$2"
                shift 2
                ;;
            -d|--dir)
                METRICS_DIR="$2"
                shift 2
                ;;
            -f|--format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Validate arguments
validate_args() {
    if [[ ! "$TEST_CATEGORY" =~ ^(unit|integration|e2e|all)$ ]]; then
        log_error "Invalid test category: $TEST_CATEGORY"
        exit 1
    fi

    if [[ ! "$OUTPUT_FORMAT" =~ ^(json|csv|text)$ ]]; then
        log_error "Invalid output format: $OUTPUT_FORMAT"
        exit 1
    fi
}

# Create metrics directory
setup_metrics_dir() {
    mkdir -p "$METRICS_DIR"
    log_info "Metrics directory: $METRICS_DIR"
}

# Get test categories to run
get_test_categories() {
    if [[ "$TEST_CATEGORY" == "all" ]]; then
        echo "unit integration e2e"
    else
        echo "$TEST_CATEGORY"
    fi
}

# Run tests and collect metrics for a specific category
collect_category_metrics() {
    local category="$1"
    local start_time end_time duration
    local test_output metrics_file
    
    log_info "Collecting metrics for $category tests..."
    
    start_time=$(date +%s.%N)
    metrics_file="$METRICS_DIR/${category}-metrics.json"
    
    # Prepare test command based on category
    local test_cmd
    case "$category" in
        "unit")
            test_cmd="go test -v -json -short ./internal/..."
            ;;
        "integration")
            test_cmd="go test -v -json -tags=integration ./tests/integration/..."
            ;;
        "e2e")
            test_cmd="go test -v -json -tags=e2e ./tests/e2e/..."
            ;;
        *)
            log_error "Unknown test category: $category"
            return 1
            ;;
    esac
    
    # Run tests and capture output
    log_info "Running: $test_cmd"
    
    if test_output=$(eval "$test_cmd" 2>&1); then
        local exit_code=0
    else
        local exit_code=$?
    fi
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Parse test results
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    local skipped_tests=0
    local test_packages=()
    local failed_packages=()
    
    # Count test results from JSON output
    while IFS= read -r line; do
        if echo "$line" | jq -e . >/dev/null 2>&1; then
            local action package test
            action=$(echo "$line" | jq -r '.Action // empty')
            package=$(echo "$line" | jq -r '.Package // empty')
            test=$(echo "$line" | jq -r '.Test // empty')
            
            case "$action" in
                "run")
                    if [[ -n "$test" ]]; then
                        ((total_tests++))
                    fi
                    ;;
                "pass")
                    if [[ -n "$test" ]]; then
                        ((passed_tests++))
                    fi
                    ;;
                "fail")
                    if [[ -n "$test" ]]; then
                        ((failed_tests++))
                    elif [[ -n "$package" ]]; then
                        failed_packages+=("$package")
                    fi
                    ;;
                "skip")
                    if [[ -n "$test" ]]; then
                        ((skipped_tests++))
                    fi
                    ;;
            esac
            
            if [[ -n "$package" && "$action" == "run" && -z "$test" ]]; then
                test_packages+=("$package")
            fi
        fi
    done <<< "$test_output"
    
    # Calculate success rate
    local success_rate=0
    if [[ $total_tests -gt 0 ]]; then
        success_rate=$(echo "scale=2; $passed_tests * 100 / $total_tests" | bc -l)
    fi
    
    # Create metrics JSON
    cat > "$metrics_file" << EOF
{
  "category": "$category",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "duration_seconds": $duration,
  "exit_code": $exit_code,
  "test_counts": {
    "total": $total_tests,
    "passed": $passed_tests,
    "failed": $failed_tests,
    "skipped": $skipped_tests
  },
  "success_rate_percent": $success_rate,
  "packages": {
    "total": ${#test_packages[@]},
    "failed": ${#failed_packages[@]},
    "failed_list": $(printf '%s\n' "${failed_packages[@]}" | jq -R . | jq -s .)
  },
  "performance": {
    "average_test_time": $(echo "scale=4; $duration / $total_tests" | bc -l 2>/dev/null || echo "0"),
    "tests_per_second": $(echo "scale=2; $total_tests / $duration" | bc -l 2>/dev/null || echo "0")
  }
}
EOF
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Metrics for $category tests:"
        cat "$metrics_file" | jq .
    fi
    
    log_success "Collected metrics for $category tests (${total_tests} tests, ${success_rate}% success rate)"
}

# Aggregate metrics from all categories
aggregate_metrics() {
    local categories
    categories=$(get_test_categories)
    local aggregate_file="$METRICS_DIR/aggregate-metrics.json"
    
    log_info "Aggregating metrics..."
    
    # Initialize aggregate data
    local total_duration=0
    local total_tests=0
    local total_passed=0
    local total_failed=0
    local total_skipped=0
    local category_results=()
    
    for category in $categories; do
        local metrics_file="$METRICS_DIR/${category}-metrics.json"
        if [[ -f "$metrics_file" ]]; then
            local category_data
            category_data=$(cat "$metrics_file")
            category_results+=("$category_data")
            
            # Extract values for aggregation
            local duration tests passed failed skipped
            duration=$(echo "$category_data" | jq -r '.duration_seconds')
            tests=$(echo "$category_data" | jq -r '.test_counts.total')
            passed=$(echo "$category_data" | jq -r '.test_counts.passed')
            failed=$(echo "$category_data" | jq -r '.test_counts.failed')
            skipped=$(echo "$category_data" | jq -r '.test_counts.skipped')
            
            total_duration=$(echo "$total_duration + $duration" | bc -l)
            total_tests=$((total_tests + tests))
            total_passed=$((total_passed + passed))
            total_failed=$((total_failed + failed))
            total_skipped=$((total_skipped + skipped))
        fi
    done
    
    # Calculate overall success rate
    local overall_success_rate=0
    if [[ $total_tests -gt 0 ]]; then
        overall_success_rate=$(echo "scale=2; $total_passed * 100 / $total_tests" | bc -l)
    fi
    
    # Create aggregate metrics
    cat > "$aggregate_file" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "total_duration_seconds": $total_duration,
  "overall_metrics": {
    "total_tests": $total_tests,
    "passed_tests": $total_passed,
    "failed_tests": $total_failed,
    "skipped_tests": $total_skipped,
    "success_rate_percent": $overall_success_rate
  },
  "performance": {
    "average_test_time": $(echo "scale=4; $total_duration / $total_tests" | bc -l 2>/dev/null || echo "0"),
    "total_tests_per_second": $(echo "scale=2; $total_tests / $total_duration" | bc -l 2>/dev/null || echo "0")
  },
  "categories": [$(printf '%s\n' "${category_results[@]}" | paste -sd ',' -)]
}
EOF
    
    log_success "Aggregate metrics created: $aggregate_file"
}

# Export metrics in different formats
export_metrics() {
    local aggregate_file="$METRICS_DIR/aggregate-metrics.json"
    
    case "$OUTPUT_FORMAT" in
        "json")
            log_info "Metrics already in JSON format: $aggregate_file"
            ;;
        "csv")
            local csv_file="$METRICS_DIR/metrics.csv"
            log_info "Exporting to CSV: $csv_file"
            
            # Create CSV header
            echo "Category,Duration,Total Tests,Passed,Failed,Skipped,Success Rate %" > "$csv_file"
            
            # Extract category data and append to CSV
            jq -r '.categories[] | [.category, .duration_seconds, .test_counts.total, .test_counts.passed, .test_counts.failed, .test_counts.skipped, .success_rate_percent] | @csv' "$aggregate_file" >> "$csv_file"
            ;;
        "text")
            local text_file="$METRICS_DIR/metrics.txt"
            log_info "Exporting to text: $text_file"
            
            {
                echo "Test Metrics Summary"
                echo "==================="
                echo ""
                echo "Generated: $(jq -r '.timestamp' "$aggregate_file")"
                echo "Total Duration: $(jq -r '.total_duration_seconds' "$aggregate_file") seconds"
                echo ""
                echo "Overall Results:"
                echo "  Total Tests: $(jq -r '.overall_metrics.total_tests' "$aggregate_file")"
                echo "  Passed: $(jq -r '.overall_metrics.passed_tests' "$aggregate_file")"
                echo "  Failed: $(jq -r '.overall_metrics.failed_tests' "$aggregate_file")"
                echo "  Skipped: $(jq -r '.overall_metrics.skipped_tests' "$aggregate_file")"
                echo "  Success Rate: $(jq -r '.overall_metrics.success_rate_percent' "$aggregate_file")%"
                echo ""
                echo "Performance:"
                echo "  Average Test Time: $(jq -r '.performance.average_test_time' "$aggregate_file") seconds"
                echo "  Tests Per Second: $(jq -r '.performance.total_tests_per_second' "$aggregate_file")"
                echo ""
                echo "By Category:"
                jq -r '.categories[] | "  \(.category): \(.test_counts.total) tests, \(.success_rate_percent)% success, \(.duration_seconds)s"' "$aggregate_file"
            } > "$text_file"
            ;;
    esac
}

# Generate CI-friendly output
generate_ci_output() {
    local aggregate_file="$METRICS_DIR/aggregate-metrics.json"
    
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        log_info "Generating GitHub Actions step summary..."
        
        {
            echo "## 📊 Test Metrics Summary"
            echo ""
            echo "| Metric | Value |"
            echo "|--------|-------|"
            echo "| Total Tests | $(jq -r '.overall_metrics.total_tests' "$aggregate_file") |"
            echo "| Success Rate | $(jq -r '.overall_metrics.success_rate_percent' "$aggregate_file")% |"
            echo "| Total Duration | $(jq -r '.total_duration_seconds' "$aggregate_file")s |"
            echo "| Tests Per Second | $(jq -r '.performance.total_tests_per_second' "$aggregate_file") |"
            echo ""
            echo "### By Category"
            echo ""
            echo "| Category | Tests | Success Rate | Duration |"
            echo "|----------|-------|--------------|----------|"
            jq -r '.categories[] | "| \(.category) | \(.test_counts.total) | \(.success_rate_percent)% | \(.duration_seconds)s |"' "$aggregate_file"
        } >> "$GITHUB_STEP_SUMMARY"
    fi
}

# Main function
main() {
    parse_args "$@"
    validate_args
    setup_metrics_dir
    
    log_info "Starting test metrics collection..."
    log_info "Category: $TEST_CATEGORY"
    log_info "Output format: $OUTPUT_FORMAT"
    
    # Collect metrics for each category
    local categories
    categories=$(get_test_categories)
    
    for category in $categories; do
        collect_category_metrics "$category"
    done
    
    # Aggregate and export
    aggregate_metrics
    export_metrics
    generate_ci_output
    
    log_success "Test metrics collection completed!"
    log_info "Results available in: $METRICS_DIR"
}

# Run main function with all arguments
main "$@"