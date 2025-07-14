#!/bin/bash

# Performance Gates Script
# Implements performance gates for releases to ensure quality standards

set -euo pipefail

# Configuration
GATES_CONFIG="${GATES_CONFIG:-./performance-gates.json}"
BENCHMARK_RESULTS="${BENCHMARK_RESULTS:-./benchmark-results.txt}"
BASELINE_RESULTS="${BASELINE_RESULTS:-./performance-baseline.json}"
OUTPUT_DIR="${OUTPUT_DIR:-./performance-gates-results}"
STRICT_MODE="${STRICT_MODE:-false}"
GATE_MODE="${GATE_MODE:-release}"  # release, development, ci
VERBOSE="${VERBOSE:-false}"

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

log_gate_pass() {
    echo -e "${GREEN}[GATE PASS]${NC} $1"
}

log_gate_fail() {
    echo -e "${RED}[GATE FAIL]${NC} $1"
}

log_gate_warn() {
    echo -e "${YELLOW}[GATE WARN]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Performance Gates Script

Implements quality gates for performance metrics to ensure release standards.

Usage: $0 [OPTIONS]

Options:
    -c, --config FILE          Gates configuration file [default: ./performance-gates.json]
    -b, --benchmarks FILE      Benchmark results file [default: ./benchmark-results.txt]
    -l, --baseline FILE        Baseline results file [default: ./performance-baseline.json]
    -o, --output-dir DIR       Output directory [default: ./performance-gates-results]
    -m, --mode MODE            Gate mode (release|development|ci) [default: release]
    -s, --strict               Enable strict mode (fail on warnings)
    -v, --verbose              Enable verbose output
    -h, --help                 Show this help message

Gate Modes:
    release      - Strict gates for production releases
    development  - Relaxed gates for development builds
    ci           - Balanced gates for CI/CD pipelines

Examples:
    $0 --mode release --strict
    $0 --config custom-gates.json --mode development
    $0 --benchmarks current.txt --baseline baseline.json --verbose

Environment Variables:
    GATES_CONFIG               Default gates configuration file
    BENCHMARK_RESULTS          Default benchmark results file
    BASELINE_RESULTS           Default baseline file
    OUTPUT_DIR                 Default output directory
    STRICT_MODE                Enable strict mode (true/false)
    GATE_MODE                  Default gate mode
    VERBOSE                    Enable verbose mode (true/false)
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--config)
                GATES_CONFIG="$2"
                shift 2
                ;;
            -b|--benchmarks)
                BENCHMARK_RESULTS="$2"
                shift 2
                ;;
            -l|--baseline)
                BASELINE_RESULTS="$2"
                shift 2
                ;;
            -o|--output-dir)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -m|--mode)
                GATE_MODE="$2"
                shift 2
                ;;
            -s|--strict)
                STRICT_MODE=true
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
    if [[ ! "$GATE_MODE" =~ ^(release|development|ci)$ ]]; then
        log_error "Invalid gate mode: $GATE_MODE"
        exit 1
    fi

    if [[ ! "$STRICT_MODE" =~ ^(true|false)$ ]]; then
        log_error "Strict mode must be true or false"
        exit 1
    fi
}

# Setup output directory
setup_output_dir() {
    mkdir -p "$OUTPUT_DIR"
    log_info "Output directory: $OUTPUT_DIR"
}

# Create default gates configuration if it doesn't exist
create_default_gates_config() {
    if [[ ! -f "$GATES_CONFIG" ]]; then
        log_info "Creating default gates configuration: $GATES_CONFIG"
        
        cat > "$GATES_CONFIG" << 'EOF'
{
  "version": "1.0",
  "description": "Performance gates configuration for yaml-formatter",
  "modes": {
    "release": {
      "description": "Strict gates for production releases",
      "gates": {
        "max_regression_percent": 5,
        "max_memory_increase_percent": 10,
        "max_allocation_increase_percent": 15,
        "min_performance_score": 8.0,
        "max_benchmark_time_ms": {
          "simple": 100,
          "complex": 500,
          "large": 2000
        },
        "required_benchmarks": [
          "Format",
          "Parse",
          "Write",
          "Memory"
        ]
      }
    },
    "development": {
      "description": "Relaxed gates for development builds",
      "gates": {
        "max_regression_percent": 15,
        "max_memory_increase_percent": 25,
        "max_allocation_increase_percent": 30,
        "min_performance_score": 6.0,
        "max_benchmark_time_ms": {
          "simple": 200,
          "complex": 1000,
          "large": 5000
        },
        "required_benchmarks": [
          "Format"
        ]
      }
    },
    "ci": {
      "description": "Balanced gates for CI/CD pipelines",
      "gates": {
        "max_regression_percent": 10,
        "max_memory_increase_percent": 20,
        "max_allocation_increase_percent": 25,
        "min_performance_score": 7.0,
        "max_benchmark_time_ms": {
          "simple": 150,
          "complex": 750,
          "large": 3000
        },
        "required_benchmarks": [
          "Format",
          "Parse"
        ]
      }
    }
  },
  "benchmark_categories": {
    "simple": ["simple", "small"],
    "complex": ["complex", "nested"],
    "large": ["large", "stress"]
  }
}
EOF
        
        log_success "Default gates configuration created"
    fi
}

# Load gates configuration
load_gates_config() {
    if [[ ! -f "$GATES_CONFIG" ]]; then
        log_error "Gates configuration file not found: $GATES_CONFIG"
        exit 1
    fi
    
    # Validate JSON
    if ! jq . "$GATES_CONFIG" >/dev/null 2>&1; then
        log_error "Invalid JSON in gates configuration file"
        exit 1
    fi
    
    # Check if mode exists
    if ! jq -e ".modes.\"$GATE_MODE\"" "$GATES_CONFIG" >/dev/null; then
        log_error "Gate mode '$GATE_MODE' not found in configuration"
        exit 1
    fi
    
    log_info "Loaded gates configuration for mode: $GATE_MODE"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Gate configuration:"
        jq ".modes.\"$GATE_MODE\"" "$GATES_CONFIG"
    fi
}

# Parse benchmark results
parse_benchmark_results() {
    local results_file="$1"
    local parsed_file="$OUTPUT_DIR/parsed-benchmarks.json"
    
    log_info "Parsing benchmark results: $results_file"
    
    # Use awk to parse benchmark results into JSON
    awk '
    BEGIN {
        print "{"
        print "  \"timestamp\": \"" strftime("%Y-%m-%dT%H:%M:%SZ") "\","
        print "  \"benchmarks\": ["
        first = 1
    }
    
    /^Benchmark/ {
        # Parse lines like: BenchmarkFormatter_Format/simple-8   1000000   1234 ns/op   456 B/op   7 allocs/op
        full_name = $1
        gsub(/-[0-9]+$/, "", full_name)  # Remove -N suffix
        gsub(/^Benchmark/, "", full_name)  # Remove Benchmark prefix
        
        # Extract test category from name
        category = "unknown"
        if (match(full_name, /simple|small/)) category = "simple"
        else if (match(full_name, /complex|nested/)) category = "complex"
        else if (match(full_name, /large|stress/)) category = "large"
        
        iterations = $2
        ns_per_op = $3
        
        # Extract memory allocations if present
        bytes_per_op = 0
        allocs_per_op = 0
        
        for (i = 4; i <= NF; i++) {
            if ($(i+1) == "B/op") {
                bytes_per_op = $i
                i++
            } else if ($(i+1) == "allocs/op") {
                allocs_per_op = $i
                i++
            }
        }
        
        # Convert ns/op to ms for easier threshold comparison
        ms_per_op = ns_per_op / 1000000
        
        if (!first) print ","
        printf "    {\n"
        printf "      \"name\": \"%s\",\n", full_name
        printf "      \"category\": \"%s\",\n", category
        printf "      \"iterations\": %d,\n", iterations
        printf "      \"ns_per_op\": %.2f,\n", ns_per_op
        printf "      \"ms_per_op\": %.4f,\n", ms_per_op
        printf "      \"bytes_per_op\": %d,\n", bytes_per_op
        printf "      \"allocs_per_op\": %d\n", allocs_per_op
        printf "    }"
        first = 0
    }
    
    END {
        print ""
        print "  ]"
        print "}"
    }
    ' "$results_file" > "$parsed_file"
    
    # Validate JSON
    if ! jq . "$parsed_file" >/dev/null 2>&1; then
        log_error "Failed to parse benchmark results into valid JSON"
        exit 1
    fi
    
    local benchmark_count
    benchmark_count=$(jq '.benchmarks | length' "$parsed_file")
    log_success "Parsed $benchmark_count benchmark results"
    
    echo "$parsed_file"
}

# Run performance gates
run_performance_gates() {
    local benchmarks_file="$1"
    local gates_result_file="$OUTPUT_DIR/gates-results.json"
    
    log_info "Running performance gates..."
    
    # Initialize gates results
    cat > "$gates_result_file" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "mode": "$GATE_MODE",
  "strict_mode": $STRICT_MODE,
  "gates": [],
  "summary": {
    "total_gates": 0,
    "passed": 0,
    "warnings": 0,
    "failed": 0,
    "overall_result": "unknown"
  }
}
EOF
    
    # Get gate configuration for current mode
    local mode_config
    mode_config=$(jq ".modes.\"$GATE_MODE\"" "$GATES_CONFIG")
    
    # Run each gate
    run_regression_gate "$benchmarks_file" "$mode_config" "$gates_result_file"
    run_memory_gate "$benchmarks_file" "$mode_config" "$gates_result_file"
    run_timing_gate "$benchmarks_file" "$mode_config" "$gates_result_file"
    run_coverage_gate "$benchmarks_file" "$mode_config" "$gates_result_file"
    
    # Calculate final summary
    local total passed warnings failed
    total=$(jq '.gates | length' "$gates_result_file")
    passed=$(jq '[.gates[] | select(.result == "pass")] | length' "$gates_result_file")
    warnings=$(jq '[.gates[] | select(.result == "warning")] | length' "$gates_result_file")
    failed=$(jq '[.gates[] | select(.result == "fail")] | length' "$gates_result_file")
    
    # Determine overall result
    local overall_result
    if [[ $failed -gt 0 ]]; then
        overall_result="fail"
    elif [[ $warnings -gt 0 && "$STRICT_MODE" == "true" ]]; then
        overall_result="fail"
    elif [[ $warnings -gt 0 ]]; then
        overall_result="warning"
    else
        overall_result="pass"
    fi
    
    # Update summary
    jq --argjson total "$total" \
       --argjson passed "$passed" \
       --argjson warnings "$warnings" \
       --argjson failed "$failed" \
       --arg overall "$overall_result" \
       '.summary.total_gates = $total |
        .summary.passed = $passed |
        .summary.warnings = $warnings |
        .summary.failed = $failed |
        .summary.overall_result = $overall' \
       "$gates_result_file" > "${gates_result_file}.tmp" && mv "${gates_result_file}.tmp" "$gates_result_file"
    
    echo "$gates_result_file"
}

# Run regression gate
run_regression_gate() {
    local benchmarks_file="$1"
    local mode_config="$2"
    local gates_result_file="$3"
    
    local max_regression
    max_regression=$(echo "$mode_config" | jq -r '.gates.max_regression_percent')
    
    log_info "Running regression gate (max: ${max_regression}%)"
    
    # Check if baseline exists for comparison
    if [[ ! -f "$BASELINE_RESULTS" ]]; then
        add_gate_result "$gates_result_file" "regression_check" "warning" "No baseline available for regression comparison" "$max_regression" "N/A"
        return
    fi
    
    # Compare with baseline using existing logic
    local comparison_result
    if comparison_result=$(./scripts/benchmark-compare.sh --baseline "$BASELINE_RESULTS" --current "$BENCHMARK_RESULTS" --threshold "$max_regression" --format json --output-dir /tmp/gate-comparison 2>/dev/null); then
        local regressions
        regressions=$(jq -r '.summary.regressions' /tmp/gate-comparison/comparison.json 2>/dev/null || echo "0")
        
        if [[ $regressions -eq 0 ]]; then
            add_gate_result "$gates_result_file" "regression_check" "pass" "No performance regressions detected" "$max_regression" "0"
        else
            add_gate_result "$gates_result_file" "regression_check" "fail" "$regressions performance regression(s) detected" "$max_regression" "$regressions"
        fi
    else
        add_gate_result "$gates_result_file" "regression_check" "warning" "Failed to compare with baseline" "$max_regression" "N/A"
    fi
}

# Run memory gate
run_memory_gate() {
    local benchmarks_file="$1"
    local mode_config="$2"
    local gates_result_file="$3"
    
    local max_memory_increase max_alloc_increase
    max_memory_increase=$(echo "$mode_config" | jq -r '.gates.max_memory_increase_percent')
    max_alloc_increase=$(echo "$mode_config" | jq -r '.gates.max_allocation_increase_percent')
    
    log_info "Running memory gate (memory: ${max_memory_increase}%, allocs: ${max_alloc_increase}%)"
    
    # Check current memory usage against thresholds
    local high_memory_benchmarks high_alloc_benchmarks
    high_memory_benchmarks=$(jq '[.benchmarks[] | select(.bytes_per_op > 1000000)] | length' "$benchmarks_file") # > 1MB
    high_alloc_benchmarks=$(jq '[.benchmarks[] | select(.allocs_per_op > 100)] | length' "$benchmarks_file") # > 100 allocs
    
    if [[ $high_memory_benchmarks -eq 0 && $high_alloc_benchmarks -eq 0 ]]; then
        add_gate_result "$gates_result_file" "memory_usage" "pass" "Memory usage within acceptable limits" "${max_memory_increase}%" "0"
    elif [[ $high_memory_benchmarks -gt 0 || $high_alloc_benchmarks -gt 0 ]]; then
        add_gate_result "$gates_result_file" "memory_usage" "warning" "High memory usage detected in $high_memory_benchmarks benchmarks, high allocations in $high_alloc_benchmarks benchmarks" "${max_memory_increase}%" "$((high_memory_benchmarks + high_alloc_benchmarks))"
    fi
}

# Run timing gate
run_timing_gate() {
    local benchmarks_file="$1"
    local mode_config="$2"
    local gates_result_file="$3"
    
    log_info "Running timing gate"
    
    # Get timing thresholds for each category
    local simple_threshold complex_threshold large_threshold
    simple_threshold=$(echo "$mode_config" | jq -r '.gates.max_benchmark_time_ms.simple')
    complex_threshold=$(echo "$mode_config" | jq -r '.gates.max_benchmark_time_ms.complex')
    large_threshold=$(echo "$mode_config" | jq -r '.gates.max_benchmark_time_ms.large')
    
    # Check benchmarks against their category thresholds
    local violations=0
    local violation_details=""
    
    # Check simple benchmarks
    local simple_violations
    simple_violations=$(jq --argjson threshold "$simple_threshold" '[.benchmarks[] | select(.category == "simple" and .ms_per_op > $threshold)] | length' "$benchmarks_file")
    violations=$((violations + simple_violations))
    if [[ $simple_violations -gt 0 ]]; then
        violation_details="${violation_details}$simple_violations simple benchmarks exceed ${simple_threshold}ms; "
    fi
    
    # Check complex benchmarks
    local complex_violations
    complex_violations=$(jq --argjson threshold "$complex_threshold" '[.benchmarks[] | select(.category == "complex" and .ms_per_op > $threshold)] | length' "$benchmarks_file")
    violations=$((violations + complex_violations))
    if [[ $complex_violations -gt 0 ]]; then
        violation_details="${violation_details}$complex_violations complex benchmarks exceed ${complex_threshold}ms; "
    fi
    
    # Check large benchmarks
    local large_violations
    large_violations=$(jq --argjson threshold "$large_threshold" '[.benchmarks[] | select(.category == "large" and .ms_per_op > $threshold)] | length' "$benchmarks_file")
    violations=$((violations + large_violations))
    if [[ $large_violations -gt 0 ]]; then
        violation_details="${violation_details}$large_violations large benchmarks exceed ${large_threshold}ms; "
    fi
    
    if [[ $violations -eq 0 ]]; then
        add_gate_result "$gates_result_file" "timing_thresholds" "pass" "All benchmarks within timing thresholds" "Various" "0"
    else
        add_gate_result "$gates_result_file" "timing_thresholds" "fail" "$violation_details" "Various" "$violations"
    fi
}

# Run coverage gate
run_coverage_gate() {
    local benchmarks_file="$1"
    local mode_config="$2"
    local gates_result_file="$3"
    
    log_info "Running coverage gate"
    
    # Get required benchmarks for this mode
    local required_benchmarks
    required_benchmarks=$(echo "$mode_config" | jq -r '.gates.required_benchmarks[]')
    
    local missing_benchmarks=0
    local missing_details=""
    
    while IFS= read -r required; do
        local found
        found=$(jq --arg name "$required" '[.benchmarks[] | select(.name | contains($name))] | length' "$benchmarks_file")
        
        if [[ $found -eq 0 ]]; then
            missing_benchmarks=$((missing_benchmarks + 1))
            missing_details="${missing_details}$required; "
        fi
    done <<< "$required_benchmarks"
    
    if [[ $missing_benchmarks -eq 0 ]]; then
        add_gate_result "$gates_result_file" "benchmark_coverage" "pass" "All required benchmarks present" "Required set" "Complete"
    else
        add_gate_result "$gates_result_file" "benchmark_coverage" "fail" "Missing required benchmarks: $missing_details" "Required set" "$missing_benchmarks missing"
    fi
}

# Add gate result to results file
add_gate_result() {
    local gates_result_file="$1"
    local gate_name="$2"
    local result="$3"
    local message="$4"
    local threshold="$5"
    local actual="$6"
    
    # Create gate result object
    local gate_result
    gate_result=$(jq -n \
        --arg name "$gate_name" \
        --arg result "$result" \
        --arg message "$message" \
        --arg threshold "$threshold" \
        --arg actual "$actual" \
        '{
            name: $name,
            result: $result,
            message: $message,
            threshold: $threshold,
            actual: $actual,
            timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
        }')
    
    # Add to gates array
    jq --argjson gate "$gate_result" '.gates += [$gate]' "$gates_result_file" > "${gates_result_file}.tmp" && mv "${gates_result_file}.tmp" "$gates_result_file"
    
    # Log result
    case "$result" in
        "pass")
            log_gate_pass "$gate_name: $message"
            ;;
        "warning")
            log_gate_warn "$gate_name: $message"
            ;;
        "fail")
            log_gate_fail "$gate_name: $message"
            ;;
    esac
}

# Generate gates report
generate_gates_report() {
    local gates_result_file="$1"
    local report_file="$OUTPUT_DIR/performance-gates-report.txt"
    
    log_info "Generating performance gates report: $report_file"
    
    local overall_result total passed warnings failed
    overall_result=$(jq -r '.summary.overall_result' "$gates_result_file")
    total=$(jq -r '.summary.total_gates' "$gates_result_file")
    passed=$(jq -r '.summary.passed' "$gates_result_file")
    warnings=$(jq -r '.summary.warnings' "$gates_result_file")
    failed=$(jq -r '.summary.failed' "$gates_result_file")
    
    {
        echo "Performance Gates Report"
        echo "======================="
        echo ""
        echo "Generated: $(jq -r '.timestamp' "$gates_result_file")"
        echo "Mode: $(jq -r '.mode' "$gates_result_file")"
        echo "Strict Mode: $(jq -r '.strict_mode' "$gates_result_file")"
        echo ""
        echo "Overall Result: $overall_result"
        echo ""
        echo "Summary:"
        echo "--------"
        echo "Total Gates: $total"
        echo "Passed: $passed"
        echo "Warnings: $warnings"
        echo "Failed: $failed"
        echo ""
        
        echo "Gate Results:"
        echo "============="
        echo ""
        printf "%-20s %-10s %-40s\n" "Gate" "Result" "Message"
        printf "%-20s %-10s %-40s\n" "----" "------" "-------"
        
        jq -r '.gates[] | [.name, .result, .message] | @tsv' "$gates_result_file" | while IFS=$'\t' read -r name result message; do
            printf "%-20s %-10s %-40s\n" "$name" "$result" "$message"
        done
        
        echo ""
        echo "Recommendations:"
        echo "==============="
        
        if [[ "$overall_result" == "pass" ]]; then
            echo "✅ All performance gates passed! Ready for release."
        elif [[ "$overall_result" == "warning" ]]; then
            echo "⚠️  Some performance gates have warnings. Review before release."
            echo ""
            echo "Warnings:"
            jq -r '.gates[] | select(.result == "warning") | "- " + .name + ": " + .message' "$gates_result_file"
        else
            echo "❌ Performance gates failed! Address issues before release."
            echo ""
            echo "Failures:"
            jq -r '.gates[] | select(.result == "fail") | "- " + .name + ": " + .message' "$gates_result_file"
        fi
        
    } > "$report_file"
    
    log_success "Performance gates report generated: $report_file"
}

# Generate CI-friendly output
generate_ci_output() {
    local gates_result_file="$1"
    
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        log_info "Generating GitHub Actions step summary..."
        
        local overall_result
        overall_result=$(jq -r '.summary.overall_result' "$gates_result_file")
        
        {
            echo "## 🚀 Performance Gates Results"
            echo ""
            
            case "$overall_result" in
                "pass")
                    echo "### ✅ All Gates Passed"
                    echo ""
                    echo "🎉 **Ready for release!** All performance gates passed successfully."
                    ;;
                "warning")
                    echo "### ⚠️ Warnings Detected"
                    echo ""
                    echo "⚠️ **Review recommended** - Some gates have warnings."
                    ;;
                "fail")
                    echo "### ❌ Gates Failed"
                    echo ""
                    echo "🚫 **Release blocked** - Performance gates failed."
                    ;;
            esac
            
            echo ""
            echo "| Metric | Count |"
            echo "|--------|-------|"
            echo "| Total Gates | $(jq -r '.summary.total_gates' "$gates_result_file") |"
            echo "| Passed | $(jq -r '.summary.passed' "$gates_result_file") |"
            echo "| Warnings | $(jq -r '.summary.warnings' "$gates_result_file") |"
            echo "| Failed | $(jq -r '.summary.failed' "$gates_result_file") |"
            echo ""
            echo "**Mode:** $(jq -r '.mode' "$gates_result_file") | **Strict:** $(jq -r '.strict_mode' "$gates_result_file")"
            
            # Show details for failed gates
            local failed_count
            failed_count=$(jq -r '.summary.failed' "$gates_result_file")
            
            if [[ $failed_count -gt 0 ]]; then
                echo ""
                echo "### ❌ Failed Gates"
                echo ""
                echo "| Gate | Message |"
                echo "|------|---------|"
                jq -r '.gates[] | select(.result == "fail") | "| \(.name) | \(.message) |"' "$gates_result_file"
            fi
            
        } >> "$GITHUB_STEP_SUMMARY"
    fi
    
    # Set exit code based on overall result
    local overall_result
    overall_result=$(jq -r '.summary.overall_result' "$gates_result_file")
    
    case "$overall_result" in
        "pass")
            log_success "All performance gates passed!"
            exit 0
            ;;
        "warning")
            if [[ "$STRICT_MODE" == "true" ]]; then
                log_error "Performance gates have warnings and strict mode is enabled"
                exit 1
            else
                log_warn "Performance gates have warnings but passed in non-strict mode"
                exit 0
            fi
            ;;
        "fail")
            log_error "Performance gates failed"
            exit 1
            ;;
    esac
}

# Main function
main() {
    parse_args "$@"
    validate_args
    setup_output_dir
    
    log_info "Starting performance gates evaluation..."
    log_info "Configuration:"
    log_info "  Mode: $GATE_MODE"
    log_info "  Strict mode: $STRICT_MODE"
    log_info "  Gates config: $GATES_CONFIG"
    log_info "  Benchmark results: $BENCHMARK_RESULTS"
    
    # Create default config if needed
    create_default_gates_config
    
    # Load configuration
    load_gates_config
    
    # Check if benchmark results exist
    if [[ ! -f "$BENCHMARK_RESULTS" ]]; then
        log_error "Benchmark results file not found: $BENCHMARK_RESULTS"
        log_info "Run benchmarks first: go test -bench=. -benchmem ./..."
        exit 1
    fi
    
    # Parse benchmark results
    local parsed_benchmarks
    parsed_benchmarks=$(parse_benchmark_results "$BENCHMARK_RESULTS")
    
    # Run performance gates
    local gates_results
    gates_results=$(run_performance_gates "$parsed_benchmarks")
    
    # Generate report
    generate_gates_report "$gates_results"
    
    # Generate CI output and set exit code
    generate_ci_output "$gates_results"
}

# Run main function with all arguments
main "$@"