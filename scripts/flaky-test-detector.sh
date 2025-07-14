#!/bin/bash

# Flaky Test Detection Script
# Detects and reports flaky tests by running tests multiple times and analyzing failure patterns

set -euo pipefail

# Configuration
RUNS="${RUNS:-10}"
PARALLEL="${PARALLEL:-4}"
TEST_PATTERN="${TEST_PATTERN:-./...}"
OUTPUT_DIR="${OUTPUT_DIR:-./flaky-test-results}"
CONFIDENCE_THRESHOLD="${CONFIDENCE_THRESHOLD:-0.8}"
MIN_RUNS="${MIN_RUNS:-5}"
VERBOSE="${VERBOSE:-false}"
DRY_RUN="${DRY_RUN:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

log_flaky() {
    echo -e "${PURPLE}[FLAKY]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Flaky Test Detection Script

Runs tests multiple times to detect flaky (non-deterministic) tests that sometimes pass and sometimes fail.

Usage: $0 [OPTIONS]

Options:
    -r, --runs NUMBER          Number of test runs per test [default: 10]
    -p, --parallel NUMBER      Number of parallel test executions [default: 4]
    -t, --test-pattern PATTERN Go test pattern to run [default: ./...]
    -o, --output-dir DIR       Output directory for results [default: ./flaky-test-results]
    -c, --confidence FLOAT     Confidence threshold for flaky detection (0.0-1.0) [default: 0.8]
    -m, --min-runs NUMBER      Minimum runs required for flaky detection [default: 5]
    -v, --verbose              Enable verbose output
    -d, --dry-run              Show what would be done without executing
    -h, --help                 Show this help message

Examples:
    $0 --runs 20 --parallel 8
    $0 --test-pattern "./internal/..." --confidence 0.9
    $0 --output-dir /tmp/flaky --verbose
    $0 --dry-run --runs 5

Environment Variables:
    RUNS                       Default number of test runs
    PARALLEL                   Default parallelism level  
    TEST_PATTERN               Default test pattern
    OUTPUT_DIR                 Default output directory
    CONFIDENCE_THRESHOLD       Default confidence threshold
    MIN_RUNS                   Default minimum runs
    VERBOSE                    Enable verbose mode (true/false)
    DRY_RUN                    Enable dry-run mode (true/false)

Confidence Threshold:
    The script calculates a flakiness score based on the pattern of passes/fails.
    A test is considered flaky if:
    - It has both passes and failures
    - The failure rate is between 10% and 90%
    - The confidence score exceeds the threshold
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -r|--runs)
                RUNS="$2"
                shift 2
                ;;
            -p|--parallel)
                PARALLEL="$2"
                shift 2
                ;;
            -t|--test-pattern)
                TEST_PATTERN="$2"
                shift 2
                ;;
            -o|--output-dir)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -c|--confidence)
                CONFIDENCE_THRESHOLD="$2"
                shift 2
                ;;
            -m|--min-runs)
                MIN_RUNS="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -d|--dry-run)
                DRY_RUN=true
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
    if ! [[ "$RUNS" =~ ^[0-9]+$ ]] || [[ "$RUNS" -lt 2 ]]; then
        log_error "Runs must be a positive integer >= 2"
        exit 1
    fi

    if ! [[ "$PARALLEL" =~ ^[0-9]+$ ]] || [[ "$PARALLEL" -lt 1 ]]; then
        log_error "Parallel must be a positive integer"
        exit 1
    fi

    if ! [[ "$MIN_RUNS" =~ ^[0-9]+$ ]] || [[ "$MIN_RUNS" -lt 2 ]]; then
        log_error "Min runs must be a positive integer >= 2"
        exit 1
    fi

    if [[ "$MIN_RUNS" -gt "$RUNS" ]]; then
        log_error "Min runs ($MIN_RUNS) cannot be greater than total runs ($RUNS)"
        exit 1
    fi

    # Validate confidence threshold is a float between 0 and 1
    if ! echo "$CONFIDENCE_THRESHOLD" | grep -E '^0\.[0-9]+$|^1\.0*$|^0*\.?0*$' >/dev/null; then
        log_error "Confidence threshold must be a float between 0.0 and 1.0"
        exit 1
    fi
}

# Setup output directory
setup_output_dir() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would create directory: $OUTPUT_DIR"
        return
    fi

    mkdir -p "$OUTPUT_DIR"
    log_info "Output directory: $OUTPUT_DIR"
}

# Discover all tests matching the pattern
discover_tests() {
    log_info "Discovering tests matching pattern: $TEST_PATTERN"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would discover tests with: go test -list=. $TEST_PATTERN"
        echo "ExampleTest1 ExampleTest2 ExampleTest3"
        return
    fi

    # Get list of all tests
    local test_list
    test_list=$(go test -list=. "$TEST_PATTERN" 2>/dev/null | grep "^Test" | sort | uniq)
    
    if [[ -z "$test_list" ]]; then
        log_warn "No tests found matching pattern: $TEST_PATTERN"
        exit 0
    fi

    local test_count
    test_count=$(echo "$test_list" | wc -l)
    log_info "Discovered $test_count tests"

    if [[ "$VERBOSE" == "true" ]]; then
        echo "$test_list" | head -10
        if [[ $test_count -gt 10 ]]; then
            log_info "... and $((test_count - 10)) more"
        fi
    fi

    echo "$test_list"
}

# Run a single test multiple times
run_test_multiple_times() {
    local test_name="$1"
    local results_file="$2"
    local run_number="$3"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Running $test_name (run $run_number/$RUNS)"
    fi

    local start_time end_time duration exit_code
    start_time=$(date +%s.%N)
    
    # Run the test and capture result
    if go test -v -run "^${test_name}$" "$TEST_PATTERN" >/dev/null 2>&1; then
        exit_code=0
    else
        exit_code=1
    fi
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Record result
    echo "${run_number},${test_name},${exit_code},${duration}" >> "$results_file"
}

# Run all tests multiple times
run_tests_multiple_times() {
    local test_list="$1"
    local results_file="$OUTPUT_DIR/test-results.csv"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would run each test $RUNS times with parallelism $PARALLEL"
        log_info "[DRY RUN] Results would be saved to: $results_file"
        return
    fi

    # Initialize results file
    echo "run,test_name,exit_code,duration" > "$results_file"
    
    log_info "Running tests $RUNS times each with parallelism $PARALLEL..."
    
    # Create a job queue
    local job_queue=()
    for run in $(seq 1 "$RUNS"); do
        while IFS= read -r test_name; do
            job_queue+=("$test_name:$run")
        done <<< "$test_list"
    done
    
    local total_jobs=${#job_queue[@]}
    log_info "Total test executions: $total_jobs"
    
    # Function to process a single job
    process_job() {
        local job="$1"
        local test_name="${job%:*}"
        local run_number="${job#*:}"
        run_test_multiple_times "$test_name" "$results_file" "$run_number"
    }
    
    # Export function for parallel execution
    export -f run_test_multiple_times process_job log_info log_warn VERBOSE TEST_PATTERN OUTPUT_DIR
    
    # Process jobs in parallel
    printf '%s\n' "${job_queue[@]}" | xargs -P "$PARALLEL" -I {} bash -c 'process_job "$@"' _ {}
    
    log_success "Completed all test runs"
}

# Analyze test results for flakiness
analyze_flakiness() {
    local results_file="$OUTPUT_DIR/test-results.csv"
    local analysis_file="$OUTPUT_DIR/flaky-analysis.json"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would analyze results from: $results_file"
        log_info "[DRY RUN] Analysis would be saved to: $analysis_file"
        return
    fi

    if [[ ! -f "$results_file" ]]; then
        log_error "Results file not found: $results_file"
        exit 1
    fi

    log_info "Analyzing test results for flakiness..."
    
    # Use awk to analyze the CSV data
    awk -F',' -v confidence_threshold="$CONFIDENCE_THRESHOLD" -v min_runs="$MIN_RUNS" '
    BEGIN {
        print "{"
        print "  \"analysis_timestamp\": \"" strftime("%Y-%m-%dT%H:%M:%SZ", systime()) "\","
        print "  \"configuration\": {"
        print "    \"total_runs\": " ENVIRON["RUNS"] ","
        print "    \"confidence_threshold\": " confidence_threshold ","
        print "    \"min_runs\": " min_runs
        print "  },"
        print "  \"tests\": ["
        first_test = 1
    }
    
    NR > 1 {  # Skip header
        test_counts[$2]++
        if ($3 == "0") {
            test_passes[$2]++
        } else {
            test_failures[$2]++
        }
        test_durations[$2] += $4
    }
    
    END {
        for (test in test_counts) {
            if (test_counts[test] < min_runs) continue
            
            passes = test_passes[test] + 0
            failures = test_failures[test] + 0
            total = test_counts[test]
            failure_rate = failures / total
            success_rate = passes / total
            avg_duration = test_durations[test] / total
            
            # Calculate flakiness score
            # A test is flaky if it has both passes and failures
            # and the failure rate is in the "flaky zone" (10%-90%)
            is_flaky = 0
            flaky_score = 0
            flaky_reason = "stable"
            
            if (passes > 0 && failures > 0) {
                if (failure_rate >= 0.1 && failure_rate <= 0.9) {
                    # Calculate confidence based on how close to 50/50 the split is
                    # The closer to 50/50, the more likely it is flaky
                    split_distance = abs(0.5 - failure_rate)
                    flaky_score = 1 - (split_distance / 0.4)  # Normalize to 0-1
                    
                    if (flaky_score >= confidence_threshold) {
                        is_flaky = 1
                        flaky_reason = "inconsistent_results"
                    }
                }
            } else if (failures == total) {
                flaky_reason = "always_fails"
            } else if (passes == total) {
                flaky_reason = "always_passes"
            }
            
            if (!first_test) print ","
            printf "    {\n"
            printf "      \"test_name\": \"%s\",\n", test
            printf "      \"runs\": %d,\n", total
            printf "      \"passes\": %d,\n", passes
            printf "      \"failures\": %d,\n", failures
            printf "      \"success_rate\": %.4f,\n", success_rate
            printf "      \"failure_rate\": %.4f,\n", failure_rate
            printf "      \"average_duration\": %.4f,\n", avg_duration
            printf "      \"is_flaky\": %s,\n", (is_flaky ? "true" : "false")
            printf "      \"flaky_score\": %.4f,\n", flaky_score
            printf "      \"flaky_reason\": \"%s\"\n", flaky_reason
            printf "    }"
            first_test = 0
        }
        print ""
        print "  ]"
        print "}"
    }
    
    function abs(x) { return x < 0 ? -x : x }
    ' "$results_file" > "$analysis_file"
    
    log_success "Flakiness analysis completed: $analysis_file"
}

# Generate flakiness report
generate_report() {
    local analysis_file="$OUTPUT_DIR/flaky-analysis.json"
    local report_file="$OUTPUT_DIR/flaky-report.txt"
    local summary_file="$OUTPUT_DIR/flaky-summary.json"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would generate reports from: $analysis_file"
        return
    fi

    if [[ ! -f "$analysis_file" ]]; then
        log_error "Analysis file not found: $analysis_file"
        exit 1
    fi

    log_info "Generating flaky test report..."
    
    # Extract flaky tests
    local flaky_tests
    flaky_tests=$(jq '.tests[] | select(.is_flaky == true)' "$analysis_file")
    
    local flaky_count
    flaky_count=$(echo "$flaky_tests" | jq -s 'length')
    
    local total_tests
    total_tests=$(jq '.tests | length' "$analysis_file")
    
    # Generate summary
    cat > "$summary_file" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "total_tests": $total_tests,
  "flaky_tests": $flaky_count,
  "flaky_percentage": $(echo "scale=2; $flaky_count * 100 / $total_tests" | bc -l 2>/dev/null || echo "0"),
  "configuration": $(jq '.configuration' "$analysis_file")
}
EOF
    
    # Generate text report
    {
        echo "Flaky Test Detection Report"
        echo "=========================="
        echo ""
        echo "Generated: $(date)"
        echo "Test Pattern: $TEST_PATTERN"
        echo "Runs per test: $RUNS"
        echo "Confidence threshold: $CONFIDENCE_THRESHOLD"
        echo ""
        echo "Summary:"
        echo "  Total tests analyzed: $total_tests"
        echo "  Flaky tests detected: $flaky_count"
        echo "  Flaky percentage: $(echo "scale=1; $flaky_count * 100 / $total_tests" | bc -l 2>/dev/null || echo "0")%"
        echo ""
        
        if [[ $flaky_count -gt 0 ]]; then
            echo "Flaky Tests Detected:"
            echo "===================="
            echo ""
            
            echo "$flaky_tests" | jq -r '"Test: " + .test_name + " (Score: " + (.flaky_score | tostring) + ")"'
            echo ""
            
            echo "Detailed Results:"
            echo "-----------------"
            echo ""
            echo "$flaky_tests" | jq -r '"Test: " + .test_name + "\n  Runs: " + (.runs | tostring) + "\n  Passes: " + (.passes | tostring) + " (" + ((.success_rate * 100) | tostring) + "%)\n  Failures: " + (.failures | tostring) + " (" + ((.failure_rate * 100) | tostring) + "%)\n  Average Duration: " + (.average_duration | tostring) + "s\n  Flaky Score: " + (.flaky_score | tostring) + "\n  Reason: " + .flaky_reason + "\n"'
        else
            echo "No flaky tests detected! 🎉"
            echo ""
            echo "All tests appear to be stable and deterministic."
        fi
        
        echo ""
        echo "Recommendations:"
        echo "==============="
        
        if [[ $flaky_count -gt 0 ]]; then
            echo "1. Investigate the flaky tests listed above"
            echo "2. Look for race conditions, timing dependencies, or external dependencies"
            echo "3. Consider adding proper synchronization or mocking"
            echo "4. Run flaky tests with increased verbosity: go test -v -count=10 -run TestName"
            echo "5. Use tools like go test -race to detect race conditions"
        else
            echo "1. Continue monitoring for flaky tests in future runs"
            echo "2. Consider running this detector periodically in CI"
            echo "3. Maintain good test practices to prevent flakiness"
        fi
        
        echo ""
        echo "Configuration used:"
        echo "  Runs: $RUNS"
        echo "  Parallel: $PARALLEL"
        echo "  Confidence threshold: $CONFIDENCE_THRESHOLD"
        echo "  Minimum runs: $MIN_RUNS"
        
    } > "$report_file"
    
    log_success "Reports generated:"
    log_info "  Text report: $report_file"
    log_info "  JSON summary: $summary_file"
    log_info "  Detailed analysis: $analysis_file"
    
    # Print summary to console
    if [[ $flaky_count -gt 0 ]]; then
        log_flaky "Found $flaky_count flaky test(s) out of $total_tests total tests"
        echo "$flaky_tests" | jq -r '"  - " + .test_name + " (Score: " + (.flaky_score | tostring) + ", " + ((.failure_rate * 100) | tostring) + "% failure rate)"'
    else
        log_success "No flaky tests detected! All $total_tests tests appear stable."
    fi
}

# Generate CI-friendly output
generate_ci_output() {
    local summary_file="$OUTPUT_DIR/flaky-summary.json"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would generate CI output"
        return
    fi

    if [[ ! -f "$summary_file" ]]; then
        log_warn "Summary file not found, skipping CI output"
        return
    fi
    
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        log_info "Generating GitHub Actions step summary..."
        
        local flaky_count total_tests flaky_percentage
        flaky_count=$(jq -r '.flaky_tests' "$summary_file")
        total_tests=$(jq -r '.total_tests' "$summary_file")
        flaky_percentage=$(jq -r '.flaky_percentage' "$summary_file")
        
        {
            echo "## 🔍 Flaky Test Detection Results"
            echo ""
            echo "| Metric | Value |"
            echo "|--------|-------|"
            echo "| Total Tests | $total_tests |"
            echo "| Flaky Tests | $flaky_count |"
            echo "| Flaky Percentage | ${flaky_percentage}% |"
            echo "| Runs per Test | $RUNS |"
            echo "| Confidence Threshold | $CONFIDENCE_THRESHOLD |"
            echo ""
            
            if [[ $flaky_count -gt 0 ]]; then
                echo "### ⚠️ Flaky Tests Detected"
                echo ""
                local analysis_file="$OUTPUT_DIR/flaky-analysis.json"
                if [[ -f "$analysis_file" ]]; then
                    echo "| Test Name | Failure Rate | Flaky Score | Reason |"
                    echo "|-----------|--------------|-------------|---------|"
                    jq -r '.tests[] | select(.is_flaky == true) | "| \(.test_name) | \((.failure_rate * 100) | round)% | \(.flaky_score | . * 100 | round)% | \(.flaky_reason) |"' "$analysis_file"
                fi
                echo ""
                echo "> 💡 **Recommendation**: Investigate these tests for race conditions, timing dependencies, or external dependencies."
            else
                echo "### ✅ No Flaky Tests Detected"
                echo ""
                echo "All tests appear to be stable and deterministic. Great job! 🎉"
            fi
        } >> "$GITHUB_STEP_SUMMARY"
    fi
    
    # Set exit code for CI
    if [[ $flaky_count -gt 0 ]]; then
        log_warn "Flaky tests detected. Consider investigating and fixing them."
        # Note: We don't exit with error code here to allow CI to continue
        # but the information is available for decision making
    fi
}

# Main function
main() {
    parse_args "$@"
    validate_args
    setup_output_dir
    
    log_info "Starting flaky test detection..."
    log_info "Configuration:"
    log_info "  Test pattern: $TEST_PATTERN"
    log_info "  Runs per test: $RUNS"
    log_info "  Parallel executions: $PARALLEL"
    log_info "  Confidence threshold: $CONFIDENCE_THRESHOLD"
    log_info "  Output directory: $OUTPUT_DIR"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "DRY RUN MODE - No tests will be executed"
    fi
    
    # Discover tests
    local test_list
    test_list=$(discover_tests)
    
    # Run tests multiple times
    run_tests_multiple_times "$test_list"
    
    # Analyze results
    analyze_flakiness
    
    # Generate reports
    generate_report
    
    # Generate CI output
    generate_ci_output
    
    log_success "Flaky test detection completed!"
    log_info "Results available in: $OUTPUT_DIR"
}

# Run main function with all arguments
main "$@"