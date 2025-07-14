#!/bin/bash

# Performance Regression Monitoring Script
# Monitors performance benchmarks over time and detects regressions

set -euo pipefail

# Configuration
BASELINE_FILE="${BASELINE_FILE:-./performance-baseline.json}"
RESULTS_DIR="${RESULTS_DIR:-./performance-results}"
REGRESSION_THRESHOLD="${REGRESSION_THRESHOLD:-15}"  # Percentage threshold for regression
IMPROVEMENT_THRESHOLD="${IMPROVEMENT_THRESHOLD:-10}"  # Percentage threshold for improvement
OUTPUT_FORMAT="${OUTPUT_FORMAT:-json}"
BENCHMARK_PATTERN="${BENCHMARK_PATTERN:-./...}"
VERBOSE="${VERBOSE:-false}"
SAVE_BASELINE="${SAVE_BASELINE:-false}"

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

log_regression() {
    echo -e "${RED}[REGRESSION]${NC} $1"
}

log_improvement() {
    echo -e "${GREEN}[IMPROVEMENT]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Performance Regression Monitoring Script

Runs benchmarks and compares results against a baseline to detect performance regressions or improvements.

Usage: $0 [OPTIONS]

Options:
    -b, --baseline FILE        Baseline performance file [default: ./performance-baseline.json]
    -r, --results-dir DIR      Results directory [default: ./performance-results]
    -t, --threshold PERCENT    Regression threshold percentage [default: 15]
    -i, --improvement PERCENT  Improvement threshold percentage [default: 10]
    -p, --pattern PATTERN      Benchmark pattern [default: ./...]
    -f, --format FORMAT        Output format (json|text|csv) [default: json]
    -s, --save-baseline        Save current results as new baseline
    -v, --verbose              Enable verbose output
    -h, --help                 Show this help message

Examples:
    $0 --threshold 20 --pattern "./internal/..."
    $0 --save-baseline --baseline custom-baseline.json
    $0 --results-dir /tmp/perf --format text --verbose

Environment Variables:
    BASELINE_FILE              Default baseline file
    RESULTS_DIR                Default results directory
    REGRESSION_THRESHOLD       Default regression threshold
    IMPROVEMENT_THRESHOLD      Default improvement threshold
    BENCHMARK_PATTERN          Default benchmark pattern
    OUTPUT_FORMAT              Default output format
    VERBOSE                    Enable verbose mode (true/false)
    SAVE_BASELINE              Save as baseline (true/false)

Threshold Values:
    The script compares current benchmark results with baseline results.
    - Regression: Performance degraded by more than threshold percentage
    - Improvement: Performance improved by more than improvement threshold
    - Stable: Performance changed within acceptable bounds
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--baseline)
                BASELINE_FILE="$2"
                shift 2
                ;;
            -r|--results-dir)
                RESULTS_DIR="$2"
                shift 2
                ;;
            -t|--threshold)
                REGRESSION_THRESHOLD="$2"
                shift 2
                ;;
            -i|--improvement)
                IMPROVEMENT_THRESHOLD="$2"
                shift 2
                ;;
            -p|--pattern)
                BENCHMARK_PATTERN="$2"
                shift 2
                ;;
            -f|--format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            -s|--save-baseline)
                SAVE_BASELINE=true
                shift
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
    if ! [[ "$REGRESSION_THRESHOLD" =~ ^[0-9]+$ ]] || [[ "$REGRESSION_THRESHOLD" -lt 1 ]]; then
        log_error "Regression threshold must be a positive integer"
        exit 1
    fi

    if ! [[ "$IMPROVEMENT_THRESHOLD" =~ ^[0-9]+$ ]] || [[ "$IMPROVEMENT_THRESHOLD" -lt 1 ]]; then
        log_error "Improvement threshold must be a positive integer"
        exit 1
    fi

    if [[ ! "$OUTPUT_FORMAT" =~ ^(json|text|csv)$ ]]; then
        log_error "Invalid output format: $OUTPUT_FORMAT"
        exit 1
    fi
}

# Setup results directory
setup_results_dir() {
    mkdir -p "$RESULTS_DIR"
    log_info "Results directory: $RESULTS_DIR"
}

# Run benchmarks and capture results
run_benchmarks() {
    local timestamp
    timestamp=$(date -u +%Y%m%d_%H%M%S)
    local raw_results_file="$RESULTS_DIR/benchmark-raw-${timestamp}.txt"
    local parsed_results_file="$RESULTS_DIR/benchmark-parsed-${timestamp}.json"
    
    log_info "Running benchmarks with pattern: $BENCHMARK_PATTERN"
    
    # Run benchmarks and capture output
    if go test -bench="$BENCHMARK_PATTERN" -benchmem -run=^$ ./... > "$raw_results_file" 2>&1; then
        log_success "Benchmarks completed successfully"
    else
        log_error "Benchmark execution failed"
        cat "$raw_results_file"
        exit 1
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Raw benchmark results:"
        cat "$raw_results_file"
    fi
    
    # Parse benchmark results
    parse_benchmark_results "$raw_results_file" "$parsed_results_file"
    
    echo "$parsed_results_file"
}

# Parse Go benchmark results into structured JSON
parse_benchmark_results() {
    local input_file="$1"
    local output_file="$2"
    
    log_info "Parsing benchmark results..."
    
    # Use awk to parse the benchmark output
    awk '
    BEGIN {
        print "{"
        print "  \"timestamp\": \"" strftime("%Y-%m-%dT%H:%M:%SZ") "\","
        print "  \"benchmarks\": ["
        first = 1
    }
    
    /^Benchmark/ {
        # Parse lines like: BenchmarkFunction-8   1000000   1234 ns/op   456 B/op   7 allocs/op
        if (match($0, /^(Benchmark[^-\s]+)(-[0-9]+)?\s+([0-9]+)\s+([0-9.]+)\s+ns\/op(\s+([0-9]+)\s+B\/op)?(\s+([0-9]+)\s+allocs\/op)?/)) {
            name = substr($0, RSTART, RLENGTH)
            gsub(/^Benchmark/, "", name)
            gsub(/-[0-9]+.*$/, "", name)
            
            # Extract values
            iterations = $2
            ns_per_op = $3
            
            # Extract memory usage if present
            bytes_per_op = 0
            allocs_per_op = 0
            if (match($0, /([0-9]+)\s+B\/op/)) {
                bytes_per_op = substr($0, RSTART, RLENGTH)
                gsub(/[^0-9]/, "", bytes_per_op)
            }
            if (match($0, /([0-9]+)\s+allocs\/op/)) {
                allocs_per_op = substr($0, RSTART, RLENGTH)
                gsub(/[^0-9]/, "", allocs_per_op)
            }
            
            if (!first) print ","
            printf "    {\n"
            printf "      \"name\": \"%s\",\n", name
            printf "      \"iterations\": %d,\n", iterations
            printf "      \"ns_per_op\": %.2f,\n", ns_per_op
            printf "      \"bytes_per_op\": %d,\n", bytes_per_op
            printf "      \"allocs_per_op\": %d\n", allocs_per_op
            printf "    }"
            first = 0
        }
    }
    
    END {
        print ""
        print "  ]"
        print "}"
    }
    ' "$input_file" > "$output_file"
    
    # Validate JSON
    if ! jq . "$output_file" >/dev/null 2>&1; then
        log_error "Failed to generate valid JSON benchmark results"
        exit 1
    fi
    
    local benchmark_count
    benchmark_count=$(jq '.benchmarks | length' "$output_file")
    log_success "Parsed $benchmark_count benchmark results"
}

# Compare current results with baseline
compare_with_baseline() {
    local current_results_file="$1"
    local comparison_file="$RESULTS_DIR/comparison-$(date -u +%Y%m%d_%H%M%S).json"
    
    if [[ ! -f "$BASELINE_FILE" ]]; then
        log_warn "No baseline file found: $BASELINE_FILE"
        log_info "Run with --save-baseline to create a baseline"
        return 1
    fi
    
    log_info "Comparing with baseline: $BASELINE_FILE"
    
    # Create comparison using jq
    jq --argjson current "$(cat "$current_results_file")" \
       --argjson baseline "$(cat "$BASELINE_FILE")" \
       --argjson regression_threshold "$REGRESSION_THRESHOLD" \
       --argjson improvement_threshold "$IMPROVEMENT_THRESHOLD" \
       '
    {
        "comparison_timestamp": now | strftime("%Y-%m-%dT%H:%M:%SZ"),
        "baseline_timestamp": $baseline.timestamp,
        "current_timestamp": $current.timestamp,
        "thresholds": {
            "regression_percent": $regression_threshold,
            "improvement_percent": $improvement_threshold
        },
        "summary": {
            "total_benchmarks": ($current.benchmarks | length),
            "regressions": 0,
            "improvements": 0,
            "stable": 0,
            "new_benchmarks": 0,
            "removed_benchmarks": 0
        },
        "results": []
    } as $template |
    
    # Create lookup tables
    ($baseline.benchmarks | map({(.name): .}) | add) as $baseline_map |
    ($current.benchmarks | map({(.name): .}) | add) as $current_map |
    
    # Compare benchmarks
    $template | 
    .results = [
        $current.benchmarks[] |
        . as $current_bench |
        ($baseline_map[.name] // null) as $baseline_bench |
        
        if $baseline_bench then
            {
                "name": .name,
                "current": {
                    "ns_per_op": .ns_per_op,
                    "bytes_per_op": .bytes_per_op,
                    "allocs_per_op": .allocs_per_op
                },
                "baseline": {
                    "ns_per_op": $baseline_bench.ns_per_op,
                    "bytes_per_op": $baseline_bench.bytes_per_op,
                    "allocs_per_op": $baseline_bench.allocs_per_op
                },
                "changes": {
                    "ns_per_op_percent": ((.ns_per_op - $baseline_bench.ns_per_op) / $baseline_bench.ns_per_op * 100),
                    "bytes_per_op_percent": (if $baseline_bench.bytes_per_op > 0 then ((.bytes_per_op - $baseline_bench.bytes_per_op) / $baseline_bench.bytes_per_op * 100) else 0 end),
                    "allocs_per_op_percent": (if $baseline_bench.allocs_per_op > 0 then ((.allocs_per_op - $baseline_bench.allocs_per_op) / $baseline_bench.allocs_per_op * 100) else 0 end)
                },
                "status": (
                    if ((.ns_per_op - $baseline_bench.ns_per_op) / $baseline_bench.ns_per_op * 100) > $regression_threshold then "regression"
                    elif (($baseline_bench.ns_per_op - .ns_per_op) / $baseline_bench.ns_per_op * 100) > $improvement_threshold then "improvement"
                    else "stable"
                    end
                ),
                "type": "existing"
            }
        else
            {
                "name": .name,
                "current": {
                    "ns_per_op": .ns_per_op,
                    "bytes_per_op": .bytes_per_op,
                    "allocs_per_op": .allocs_per_op
                },
                "baseline": null,
                "changes": null,
                "status": "new",
                "type": "new"
            }
        end
    ] |
    
    # Update summary counts
    .summary.regressions = ([.results[] | select(.status == "regression")] | length) |
    .summary.improvements = ([.results[] | select(.status == "improvement")] | length) |
    .summary.stable = ([.results[] | select(.status == "stable")] | length) |
    .summary.new_benchmarks = ([.results[] | select(.status == "new")] | length)
    ' > "$comparison_file"
    
    echo "$comparison_file"
}

# Generate performance report
generate_report() {
    local comparison_file="$1"
    local report_file="$RESULTS_DIR/performance-report.txt"
    local summary_file="$RESULTS_DIR/performance-summary.json"
    
    if [[ ! -f "$comparison_file" ]]; then
        log_error "Comparison file not found: $comparison_file"
        exit 1
    fi
    
    log_info "Generating performance report..."
    
    # Extract summary data
    local total_benchmarks regressions improvements stable new_benchmarks
    total_benchmarks=$(jq -r '.summary.total_benchmarks' "$comparison_file")
    regressions=$(jq -r '.summary.regressions' "$comparison_file")
    improvements=$(jq -r '.summary.improvements' "$comparison_file")
    stable=$(jq -r '.summary.stable' "$comparison_file")
    new_benchmarks=$(jq -r '.summary.new_benchmarks' "$comparison_file")
    
    # Create summary
    jq '{
        "timestamp": .comparison_timestamp,
        "summary": .summary,
        "thresholds": .thresholds,
        "has_regressions": (.summary.regressions > 0),
        "has_improvements": (.summary.improvements > 0),
        "overall_status": (
            if .summary.regressions > 0 then "regression_detected"
            elif .summary.improvements > 0 then "improvement_detected"
            else "stable"
        )
    }' "$comparison_file" > "$summary_file"
    
    # Generate text report
    {
        echo "Performance Monitoring Report"
        echo "============================"
        echo ""
        echo "Generated: $(date)"
        echo "Benchmark Pattern: $BENCHMARK_PATTERN"
        echo "Regression Threshold: ${REGRESSION_THRESHOLD}%"
        echo "Improvement Threshold: ${IMPROVEMENT_THRESHOLD}%"
        echo ""
        echo "Summary:"
        echo "  Total Benchmarks: $total_benchmarks"
        echo "  Regressions: $regressions"
        echo "  Improvements: $improvements"
        echo "  Stable: $stable"
        echo "  New Benchmarks: $new_benchmarks"
        echo ""
        
        if [[ $regressions -gt 0 ]]; then
            echo "⚠️  PERFORMANCE REGRESSIONS DETECTED"
            echo "====================================="
            echo ""
            jq -r '.results[] | select(.status == "regression") | "❌ " + .name + ": " + (.changes.ns_per_op_percent | tostring) + "% slower (" + (.current.ns_per_op | tostring) + " ns/op vs " + (.baseline.ns_per_op | tostring) + " ns/op)"' "$comparison_file"
            echo ""
        fi
        
        if [[ $improvements -gt 0 ]]; then
            echo "🚀 PERFORMANCE IMPROVEMENTS DETECTED"
            echo "====================================="
            echo ""
            jq -r '.results[] | select(.status == "improvement") | "✅ " + .name + ": " + ((.changes.ns_per_op_percent * -1) | tostring) + "% faster (" + (.current.ns_per_op | tostring) + " ns/op vs " + (.baseline.ns_per_op | tostring) + " ns/op)"' "$comparison_file"
            echo ""
        fi
        
        if [[ $stable -gt 0 ]]; then
            echo "Stable Benchmarks:"
            echo "=================="
            echo ""
            jq -r '.results[] | select(.status == "stable") | "✓ " + .name + ": " + (.changes.ns_per_op_percent | tostring) + "% change (" + (.current.ns_per_op | tostring) + " ns/op)"' "$comparison_file"
            echo ""
        fi
        
        if [[ $new_benchmarks -gt 0 ]]; then
            echo "New Benchmarks:"
            echo "==============="
            echo ""
            jq -r '.results[] | select(.status == "new") | "🆕 " + .name + ": " + (.current.ns_per_op | tostring) + " ns/op"' "$comparison_file"
            echo ""
        fi
        
        echo "Detailed Results:"
        echo "================="
        echo ""
        printf "%-30s %15s %15s %15s %10s\n" "Benchmark" "Current (ns/op)" "Baseline (ns/op)" "Change %" "Status"
        printf "%-30s %15s %15s %15s %10s\n" "----------" "---------------" "----------------" "---------" "------"
        jq -r '.results[] | select(.type == "existing") | [.name, (.current.ns_per_op | tostring), (.baseline.ns_per_op | tostring), ((.changes.ns_per_op_percent * 100 | floor) / 100 | tostring), .status] | @tsv' "$comparison_file" | while IFS=$'\t' read -r name current baseline change status; do
            printf "%-30s %15s %15s %15s %10s\n" "$name" "$current" "$baseline" "${change}%" "$status"
        done
        
    } > "$report_file"
    
    log_success "Reports generated:"
    log_info "  Text report: $report_file"
    log_info "  JSON summary: $summary_file"
    log_info "  Detailed comparison: $comparison_file"
    
    # Print summary to console
    if [[ $regressions -gt 0 ]]; then
        log_regression "$regressions performance regression(s) detected!"
        jq -r '.results[] | select(.status == "regression") | "  - " + .name + ": " + (.changes.ns_per_op_percent | tostring) + "% slower"' "$comparison_file"
    fi
    
    if [[ $improvements -gt 0 ]]; then
        log_improvement "$improvements performance improvement(s) detected!"
        jq -r '.results[] | select(.status == "improvement") | "  - " + .name + ": " + ((.changes.ns_per_op_percent * -1) | tostring) + "% faster"' "$comparison_file"
    fi
    
    if [[ $regressions -eq 0 && $improvements -eq 0 ]]; then
        log_success "All benchmarks are stable - no significant performance changes detected"
    fi
}

# Save current results as baseline
save_baseline() {
    local current_results_file="$1"
    
    if [[ "$SAVE_BASELINE" == "true" ]]; then
        log_info "Saving current results as baseline: $BASELINE_FILE"
        cp "$current_results_file" "$BASELINE_FILE"
        log_success "Baseline updated"
    fi
}

# Generate CI-friendly output
generate_ci_output() {
    local summary_file="$RESULTS_DIR/performance-summary.json"
    
    if [[ ! -f "$summary_file" ]]; then
        log_warn "Summary file not found, skipping CI output"
        return
    fi
    
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        log_info "Generating GitHub Actions step summary..."
        
        local has_regressions has_improvements overall_status
        has_regressions=$(jq -r '.has_regressions' "$summary_file")
        has_improvements=$(jq -r '.has_improvements' "$summary_file")
        overall_status=$(jq -r '.overall_status' "$summary_file")
        
        {
            echo "## 📈 Performance Monitoring Results"
            echo ""
            
            case "$overall_status" in
                "regression_detected")
                    echo "### ⚠️ Performance Regressions Detected"
                    ;;
                "improvement_detected")
                    echo "### 🚀 Performance Improvements Detected"
                    ;;
                "stable")
                    echo "### ✅ Performance is Stable"
                    ;;
            esac
            
            echo ""
            echo "| Metric | Count |"
            echo "|--------|-------|"
            echo "| Total Benchmarks | $(jq -r '.summary.total_benchmarks' "$summary_file") |"
            echo "| Regressions | $(jq -r '.summary.regressions' "$summary_file") |"
            echo "| Improvements | $(jq -r '.summary.improvements' "$summary_file") |"
            echo "| Stable | $(jq -r '.summary.stable' "$summary_file") |"
            echo "| New Benchmarks | $(jq -r '.summary.new_benchmarks' "$summary_file") |"
            echo ""
            echo "**Thresholds:** Regression >${REGRESSION_THRESHOLD}%, Improvement >${IMPROVEMENT_THRESHOLD}%"
            
        } >> "$GITHUB_STEP_SUMMARY"
    fi
    
    # Set appropriate exit code for CI
    local has_regressions
    has_regressions=$(jq -r '.has_regressions' "$summary_file")
    
    if [[ "$has_regressions" == "true" ]]; then
        log_error "Performance regressions detected - failing CI"
        exit 1
    fi
}

# Export results in different formats
export_results() {
    local comparison_file="$1"
    
    case "$OUTPUT_FORMAT" in
        "json")
            log_info "Results already in JSON format: $comparison_file"
            ;;
        "csv")
            local csv_file="$RESULTS_DIR/performance-comparison.csv"
            log_info "Exporting to CSV: $csv_file"
            
            # Create CSV header
            echo "Benchmark,Current_ns_per_op,Baseline_ns_per_op,Change_percent,Status" > "$csv_file"
            
            # Extract data and append to CSV
            jq -r '.results[] | select(.type == "existing") | [.name, .current.ns_per_op, .baseline.ns_per_op, .changes.ns_per_op_percent, .status] | @csv' "$comparison_file" >> "$csv_file"
            ;;
        "text")
            log_info "Text format already generated in report file"
            ;;
    esac
}

# Main function
main() {
    parse_args "$@"
    validate_args
    setup_results_dir
    
    log_info "Starting performance monitoring..."
    log_info "Configuration:"
    log_info "  Benchmark pattern: $BENCHMARK_PATTERN"
    log_info "  Baseline file: $BASELINE_FILE"
    log_info "  Regression threshold: ${REGRESSION_THRESHOLD}%"
    log_info "  Improvement threshold: ${IMPROVEMENT_THRESHOLD}%"
    log_info "  Output format: $OUTPUT_FORMAT"
    
    # Run benchmarks
    local current_results_file
    current_results_file=$(run_benchmarks)
    
    # Compare with baseline if it exists
    if [[ -f "$BASELINE_FILE" ]]; then
        local comparison_file
        comparison_file=$(compare_with_baseline "$current_results_file")
        
        # Generate reports
        generate_report "$comparison_file"
        
        # Export in requested format
        export_results "$comparison_file"
        
        # Generate CI output
        generate_ci_output
    else
        log_warn "No baseline found - this will be the first run"
        log_info "Use --save-baseline to save current results as baseline"
    fi
    
    # Save baseline if requested
    save_baseline "$current_results_file"
    
    log_success "Performance monitoring completed!"
    log_info "Results available in: $RESULTS_DIR"
}

# Run main function with all arguments
main "$@"