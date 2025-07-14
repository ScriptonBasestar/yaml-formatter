#!/bin/bash

# Benchmark Comparison Tool
# Compares benchmark results between different versions, commits, or configurations

set -euo pipefail

# Configuration
BASELINE_FILE="${BASELINE_FILE:-./benchmarks/baseline.txt}"
CURRENT_FILE="${CURRENT_FILE:-./benchmarks/current.txt}"
OUTPUT_DIR="${OUTPUT_DIR:-./benchmark-comparison}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
THRESHOLD="${THRESHOLD:-10}"  # Percentage threshold for significant changes
BENCHMARK_PATTERN="${BENCHMARK_PATTERN:-./...}"
VERBOSE="${VERBOSE:-false}"
GENERATE_BASELINE="${GENERATE_BASELINE:-false}"

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
Benchmark Comparison Tool

Compares benchmark results to detect performance changes between versions.

Usage: $0 [OPTIONS]

Options:
    -b, --baseline FILE        Baseline benchmark file [default: ./benchmarks/baseline.txt]
    -c, --current FILE         Current benchmark file [default: ./benchmarks/current.txt]
    -o, --output-dir DIR       Output directory [default: ./benchmark-comparison]
    -f, --format FORMAT        Output format (text|json|html|csv) [default: text]
    -t, --threshold PERCENT    Significance threshold percentage [default: 10]
    -p, --pattern PATTERN      Benchmark pattern to run [default: ./...]
    -g, --generate-baseline    Generate new baseline from current benchmarks
    -v, --verbose              Enable verbose output
    -h, --help                 Show this help message

Examples:
    $0 --baseline old.txt --current new.txt --threshold 15
    $0 --generate-baseline --pattern "./internal/formatter/..."
    $0 --format json --output-dir /tmp/benchmark-results

Environment Variables:
    BASELINE_FILE              Default baseline file
    CURRENT_FILE               Default current file
    OUTPUT_DIR                 Default output directory
    OUTPUT_FORMAT              Default output format
    THRESHOLD                  Default threshold percentage
    BENCHMARK_PATTERN          Default benchmark pattern
    VERBOSE                    Enable verbose mode (true/false)
    GENERATE_BASELINE          Generate baseline (true/false)

Benchmark File Format:
    The tool expects Go benchmark output format:
    BenchmarkName-X    N    ns/op    B/op    allocs/op
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
            -c|--current)
                CURRENT_FILE="$2"
                shift 2
                ;;
            -o|--output-dir)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -f|--format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            -t|--threshold)
                THRESHOLD="$2"
                shift 2
                ;;
            -p|--pattern)
                BENCHMARK_PATTERN="$2"
                shift 2
                ;;
            -g|--generate-baseline)
                GENERATE_BASELINE=true
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
    if ! [[ "$THRESHOLD" =~ ^[0-9]+$ ]] || [[ "$THRESHOLD" -lt 1 ]]; then
        log_error "Threshold must be a positive integer"
        exit 1
    fi

    if [[ ! "$OUTPUT_FORMAT" =~ ^(text|json|html|csv)$ ]]; then
        log_error "Invalid output format: $OUTPUT_FORMAT"
        exit 1
    fi
}

# Setup output directory
setup_output_dir() {
    mkdir -p "$OUTPUT_DIR"
    mkdir -p "$(dirname "$BASELINE_FILE")" 2>/dev/null || true
    mkdir -p "$(dirname "$CURRENT_FILE")" 2>/dev/null || true
    log_info "Output directory: $OUTPUT_DIR"
}

# Run benchmarks and save results
run_benchmarks() {
    local output_file="$1"
    
    log_info "Running benchmarks with pattern: $BENCHMARK_PATTERN"
    log_info "Saving results to: $output_file"
    
    # Run benchmarks with proper formatting
    if go test -bench="$BENCHMARK_PATTERN" -benchmem -run=^$ ./... > "$output_file" 2>&1; then
        log_success "Benchmarks completed successfully"
    else
        log_error "Benchmark execution failed"
        cat "$output_file"
        exit 1
    fi
    
    # Validate that we got benchmark results
    local benchmark_count
    benchmark_count=$(grep -c "^Benchmark" "$output_file" 2>/dev/null || echo 0)
    
    if [[ $benchmark_count -eq 0 ]]; then
        log_error "No benchmark results found in output"
        exit 1
    fi
    
    log_info "Captured $benchmark_count benchmark results"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Benchmark results preview:"
        grep "^Benchmark" "$output_file" | head -5
        if [[ $benchmark_count -gt 5 ]]; then
            log_info "... and $((benchmark_count - 5)) more"
        fi
    fi
}

# Parse benchmark results into structured data
parse_benchmark_file() {
    local input_file="$1"
    local output_file="$2"
    
    log_info "Parsing benchmark file: $input_file"
    
    # Use awk to parse the benchmark results
    awk '
    BEGIN {
        print "{"
        print "  \"timestamp\": \"" strftime("%Y-%m-%dT%H:%M:%SZ") "\","
        print "  \"benchmarks\": ["
        first = 1
    }
    
    /^Benchmark/ {
        # Parse lines like: BenchmarkFunction-8   1000000   1234 ns/op   456 B/op   7 allocs/op
        benchmark_name = $1
        gsub(/-[0-9]+$/, "", benchmark_name)  # Remove -N suffix
        gsub(/^Benchmark/, "", benchmark_name)  # Remove Benchmark prefix
        
        iterations = $2
        ns_per_op = $3
        
        # Extract memory allocations if present
        bytes_per_op = 0
        allocs_per_op = 0
        
        for (i = 4; i <= NF; i++) {
            if ($(i+1) == "B/op") {
                bytes_per_op = $i
                i++  # Skip the unit
            } else if ($(i+1) == "allocs/op") {
                allocs_per_op = $i
                i++  # Skip the unit
            }
        }
        
        if (!first) print ","
        printf "    {\n"
        printf "      \"name\": \"%s\",\n", benchmark_name
        printf "      \"iterations\": %d,\n", iterations
        printf "      \"ns_per_op\": %.2f,\n", ns_per_op
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
    ' "$input_file" > "$output_file"
    
    # Validate JSON
    if ! jq . "$output_file" >/dev/null 2>&1; then
        log_error "Failed to generate valid JSON from benchmark results"
        exit 1
    fi
    
    local benchmark_count
    benchmark_count=$(jq '.benchmarks | length' "$output_file")
    log_success "Parsed $benchmark_count benchmarks into JSON format"
}

# Compare benchmark results
compare_benchmarks() {
    local baseline_json="$1"
    local current_json="$2"
    local comparison_file="$3"
    
    log_info "Comparing benchmarks..."
    log_info "Baseline: $baseline_json"
    log_info "Current: $current_json"
    
    # Use jq to compare the benchmark results
    jq --argjson baseline "$(cat "$baseline_json")" \
       --argjson current "$(cat "$current_json")" \
       --argjson threshold "$THRESHOLD" \
       '
    {
        "comparison_timestamp": now | strftime("%Y-%m-%dT%H:%M:%SZ"),
        "baseline_timestamp": $baseline.timestamp,
        "current_timestamp": $current.timestamp,
        "threshold_percent": $threshold,
        "summary": {
            "total_benchmarks": 0,
            "matched_benchmarks": 0,
            "regressions": 0,
            "improvements": 0,
            "stable": 0,
            "new_benchmarks": 0,
            "removed_benchmarks": 0
        },
        "results": []
    } as $template |
    
    # Create lookup maps
    ($baseline.benchmarks | map({(.name): .}) | add) as $baseline_map |
    ($current.benchmarks | map({(.name): .}) | add) as $current_map |
    
    # Compare current benchmarks against baseline
    $template |
    .results = [
        $current.benchmarks[] |
        . as $current_bench |
        ($baseline_map[.name] // null) as $baseline_bench |
        
        if $baseline_bench then
            {
                "name": .name,
                "baseline": {
                    "ns_per_op": $baseline_bench.ns_per_op,
                    "bytes_per_op": $baseline_bench.bytes_per_op,
                    "allocs_per_op": $baseline_bench.allocs_per_op,
                    "iterations": $baseline_bench.iterations
                },
                "current": {
                    "ns_per_op": .ns_per_op,
                    "bytes_per_op": .bytes_per_op,
                    "allocs_per_op": .allocs_per_op,
                    "iterations": .iterations
                },
                "changes": {
                    "ns_per_op_percent": ((.ns_per_op - $baseline_bench.ns_per_op) / $baseline_bench.ns_per_op * 100),
                    "bytes_per_op_percent": (if $baseline_bench.bytes_per_op > 0 then ((.bytes_per_op - $baseline_bench.bytes_per_op) / $baseline_bench.bytes_per_op * 100) else 0 end),
                    "allocs_per_op_percent": (if $baseline_bench.allocs_per_op > 0 then ((.allocs_per_op - $baseline_bench.allocs_per_op) / $baseline_bench.allocs_per_op * 100) else 0 end)
                },
                "status": (
                    if ((.ns_per_op - $baseline_bench.ns_per_op) / $baseline_bench.ns_per_op * 100) > $threshold then "regression"
                    elif (($baseline_bench.ns_per_op - .ns_per_op) / $baseline_bench.ns_per_op * 100) > $threshold then "improvement"
                    else "stable"
                    end
                ),
                "type": "matched"
            }
        else
            {
                "name": .name,
                "baseline": null,
                "current": {
                    "ns_per_op": .ns_per_op,
                    "bytes_per_op": .bytes_per_op,
                    "allocs_per_op": .allocs_per_op,
                    "iterations": .iterations
                },
                "changes": null,
                "status": "new",
                "type": "new"
            }
        end
    ] +
    # Add removed benchmarks
    [
        $baseline.benchmarks[] |
        select(($current_map[.name] // null) == null) |
        {
            "name": .name,
            "baseline": {
                "ns_per_op": .ns_per_op,
                "bytes_per_op": .bytes_per_op,
                "allocs_per_op": .allocs_per_op,
                "iterations": .iterations
            },
            "current": null,
            "changes": null,
            "status": "removed",
            "type": "removed"
        }
    ] |
    
    # Update summary counts
    .summary.total_benchmarks = ($current.benchmarks | length) |
    .summary.matched_benchmarks = ([.results[] | select(.type == "matched")] | length) |
    .summary.regressions = ([.results[] | select(.status == "regression")] | length) |
    .summary.improvements = ([.results[] | select(.status == "improvement")] | length) |
    .summary.stable = ([.results[] | select(.status == "stable")] | length) |
    .summary.new_benchmarks = ([.results[] | select(.status == "new")] | length) |
    .summary.removed_benchmarks = ([.results[] | select(.status == "removed")] | length)
    ' > "$comparison_file"
    
    log_success "Benchmark comparison completed: $comparison_file"
}

# Generate comparison report
generate_report() {
    local comparison_file="$1"
    
    if [[ ! -f "$comparison_file" ]]; then
        log_error "Comparison file not found: $comparison_file"
        exit 1
    fi
    
    case "$OUTPUT_FORMAT" in
        "text")
            generate_text_report "$comparison_file"
            ;;
        "json")
            generate_json_report "$comparison_file"
            ;;
        "html")
            generate_html_report "$comparison_file"
            ;;
        "csv")
            generate_csv_report "$comparison_file"
            ;;
    esac
}

# Generate text report
generate_text_report() {
    local comparison_file="$1"
    local report_file="$OUTPUT_DIR/benchmark-comparison.txt"
    
    log_info "Generating text report: $report_file"
    
    # Extract summary data
    local total regressions improvements stable new removed
    total=$(jq -r '.summary.total_benchmarks' "$comparison_file")
    regressions=$(jq -r '.summary.regressions' "$comparison_file")
    improvements=$(jq -r '.summary.improvements' "$comparison_file")
    stable=$(jq -r '.summary.stable' "$comparison_file")
    new=$(jq -r '.summary.new_benchmarks' "$comparison_file")
    removed=$(jq -r '.summary.removed_benchmarks' "$comparison_file")
    
    {
        echo "Benchmark Comparison Report"
        echo "=========================="
        echo ""
        echo "Generated: $(jq -r '.comparison_timestamp' "$comparison_file")"
        echo "Threshold: ${THRESHOLD}%"
        echo ""
        echo "Summary:"
        echo "--------"
        echo "Total Benchmarks: $total"
        echo "Regressions: $regressions"
        echo "Improvements: $improvements"
        echo "Stable: $stable"
        echo "New Benchmarks: $new"
        echo "Removed Benchmarks: $removed"
        echo ""
        
        if [[ $regressions -gt 0 ]]; then
            echo "⚠️  PERFORMANCE REGRESSIONS"
            echo "============================="
            echo ""
            jq -r '.results[] | select(.status == "regression") | "❌ " + .name + ": " + (.changes.ns_per_op_percent | tostring) + "% slower (" + (.current.ns_per_op | tostring) + " vs " + (.baseline.ns_per_op | tostring) + " ns/op)"' "$comparison_file"
            echo ""
        fi
        
        if [[ $improvements -gt 0 ]]; then
            echo "🚀 PERFORMANCE IMPROVEMENTS"
            echo "============================"
            echo ""
            jq -r '.results[] | select(.status == "improvement") | "✅ " + .name + ": " + ((.changes.ns_per_op_percent * -1) | tostring) + "% faster (" + (.current.ns_per_op | tostring) + " vs " + (.baseline.ns_per_op | tostring) + " ns/op)"' "$comparison_file"
            echo ""
        fi
        
        if [[ $new -gt 0 ]]; then
            echo "New Benchmarks:"
            echo "==============="
            echo ""
            jq -r '.results[] | select(.status == "new") | "🆕 " + .name + ": " + (.current.ns_per_op | tostring) + " ns/op"' "$comparison_file"
            echo ""
        fi
        
        if [[ $removed -gt 0 ]]; then
            echo "Removed Benchmarks:"
            echo "==================="
            echo ""
            jq -r '.results[] | select(.status == "removed") | "🗑️  " + .name + ": was " + (.baseline.ns_per_op | tostring) + " ns/op"' "$comparison_file"
            echo ""
        fi
        
        echo "Detailed Results:"
        echo "================="
        echo ""
        printf "%-40s %15s %15s %15s %10s\n" "Benchmark" "Current (ns/op)" "Baseline (ns/op)" "Change %" "Status"
        printf "%-40s %15s %15s %15s %10s\n" "---------" "---------------" "----------------" "---------" "------"
        
        jq -r '.results[] | select(.type == "matched") | [.name, (.current.ns_per_op | tostring), (.baseline.ns_per_op | tostring), ((.changes.ns_per_op_percent * 100 | floor) / 100 | tostring), .status] | @tsv' "$comparison_file" | while IFS=$'\t' read -r name current baseline change status; do
            printf "%-40s %15s %15s %15s %10s\n" "$name" "$current" "$baseline" "${change}%" "$status"
        done
        
    } > "$report_file"
    
    log_success "Text report generated: $report_file"
    
    # Print summary to console
    if [[ $regressions -gt 0 ]]; then
        log_regression "$regressions performance regression(s) detected!"
    fi
    
    if [[ $improvements -gt 0 ]]; then
        log_improvement "$improvements performance improvement(s) detected!"
    fi
    
    if [[ $regressions -eq 0 && $improvements -eq 0 && $stable -gt 0 ]]; then
        log_success "All benchmarks are stable - no significant performance changes"
    fi
}

# Generate JSON report
generate_json_report() {
    local comparison_file="$1"
    local report_file="$OUTPUT_DIR/benchmark-comparison.json"
    
    log_info "Generating JSON report: $report_file"
    
    # Add metadata
    jq '. + {
        "report_metadata": {
            "format": "json",
            "threshold_percent": '$THRESHOLD',
            "generated_by": "benchmark-compare.sh"
        }
    }' "$comparison_file" > "$report_file"
    
    log_success "JSON report generated: $report_file"
}

# Generate HTML report
generate_html_report() {
    local comparison_file="$1"
    local report_file="$OUTPUT_DIR/benchmark-comparison.html"
    
    log_info "Generating HTML report: $report_file"
    
    # Create a comprehensive HTML report
    cat > "$report_file" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Benchmark Comparison Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { border-bottom: 3px solid #007acc; margin-bottom: 30px; padding-bottom: 20px; }
        .header h1 { color: #007acc; margin: 0; font-size: 2.5em; }
        .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin-bottom: 30px; }
        .metric-card { background: linear-gradient(135deg, #f8f9fa, #e9ecef); padding: 15px; border-radius: 8px; text-align: center; }
        .metric-card.regression { border-left: 4px solid #dc3545; }
        .metric-card.improvement { border-left: 4px solid #28a745; }
        .metric-card.stable { border-left: 4px solid #007acc; }
        .metric-card.new { border-left: 4px solid #ffc107; }
        .metric-value { font-size: 1.8em; font-weight: bold; color: #333; }
        .metric-label { color: #666; margin-top: 5px; font-size: 0.9em; }
        .section { margin-bottom: 30px; }
        .section h2 { color: #333; border-bottom: 2px solid #eee; padding-bottom: 10px; }
        .benchmark-table { width: 100%; border-collapse: collapse; margin-top: 15px; font-size: 0.9em; }
        .benchmark-table th, .benchmark-table td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #dee2e6; }
        .benchmark-table th { background: #f8f9fa; font-weight: 600; position: sticky; top: 0; }
        .status-regression { color: #dc3545; font-weight: bold; }
        .status-improvement { color: #28a745; font-weight: bold; }
        .status-stable { color: #6c757d; }
        .status-new { color: #ffc107; font-weight: bold; }
        .change-positive { color: #dc3545; }
        .change-negative { color: #28a745; }
        .table-container { max-height: 500px; overflow-y: auto; border: 1px solid #dee2e6; border-radius: 4px; }
        .timestamp { color: #666; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Benchmark Comparison Report</h1>
            <div class="timestamp">Generated: <span id="timestamp"></span></div>
            <div class="timestamp">Threshold: <span id="threshold"></span>%</div>
        </div>

        <div class="summary-grid">
            <div class="metric-card">
                <div class="metric-value" id="total">-</div>
                <div class="metric-label">Total Benchmarks</div>
            </div>
            <div class="metric-card regression">
                <div class="metric-value" id="regressions">-</div>
                <div class="metric-label">Regressions</div>
            </div>
            <div class="metric-card improvement">
                <div class="metric-value" id="improvements">-</div>
                <div class="metric-label">Improvements</div>
            </div>
            <div class="metric-card stable">
                <div class="metric-value" id="stable">-</div>
                <div class="metric-label">Stable</div>
            </div>
            <div class="metric-card new">
                <div class="metric-value" id="new">-</div>
                <div class="metric-label">New</div>
            </div>
        </div>

        <div class="section">
            <h2>📈 Detailed Comparison</h2>
            <div class="table-container">
                <table class="benchmark-table">
                    <thead>
                        <tr>
                            <th>Benchmark Name</th>
                            <th>Current (ns/op)</th>
                            <th>Baseline (ns/op)</th>
                            <th>Change %</th>
                            <th>Status</th>
                            <th>Memory (B/op)</th>
                            <th>Allocs/op</th>
                        </tr>
                    </thead>
                    <tbody id="benchmark-results">
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <script>
        const data = DATA_PLACEHOLDER;
        
        function formatNumber(num) {
            return num.toLocaleString();
        }
        
        function formatPercent(num) {
            const sign = num >= 0 ? '+' : '';
            return sign + num.toFixed(1) + '%';
        }
        
        function renderSummary() {
            document.getElementById('timestamp').textContent = new Date(data.comparison_timestamp).toLocaleString();
            document.getElementById('threshold').textContent = data.threshold_percent;
            document.getElementById('total').textContent = data.summary.total_benchmarks;
            document.getElementById('regressions').textContent = data.summary.regressions;
            document.getElementById('improvements').textContent = data.summary.improvements;
            document.getElementById('stable').textContent = data.summary.stable;
            document.getElementById('new').textContent = data.summary.new_benchmarks;
        }
        
        function renderResults() {
            const tbody = document.getElementById('benchmark-results');
            tbody.innerHTML = '';
            
            data.results.forEach(result => {
                const row = tbody.insertRow();
                
                // Name
                row.insertCell().textContent = result.name;
                
                // Current
                const currentCell = row.insertCell();
                currentCell.textContent = result.current ? formatNumber(result.current.ns_per_op) : 'N/A';
                
                // Baseline
                const baselineCell = row.insertCell();
                baselineCell.textContent = result.baseline ? formatNumber(result.baseline.ns_per_op) : 'N/A';
                
                // Change %
                const changeCell = row.insertCell();
                if (result.changes && result.changes.ns_per_op_percent !== null) {
                    changeCell.textContent = formatPercent(result.changes.ns_per_op_percent);
                    changeCell.className = result.changes.ns_per_op_percent >= 0 ? 'change-positive' : 'change-negative';
                } else {
                    changeCell.textContent = 'N/A';
                }
                
                // Status
                const statusCell = row.insertCell();
                statusCell.textContent = result.status.toUpperCase();
                statusCell.className = 'status-' + result.status;
                
                // Memory
                const memoryCell = row.insertCell();
                memoryCell.textContent = result.current ? formatNumber(result.current.bytes_per_op) : 'N/A';
                
                // Allocs
                const allocsCell = row.insertCell();
                allocsCell.textContent = result.current ? formatNumber(result.current.allocs_per_op) : 'N/A';
            });
        }
        
        renderSummary();
        renderResults();
    </script>
</body>
</html>
EOF
    
    # Inject data into HTML
    local escaped_data
    escaped_data=$(jq -c . "$comparison_file" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g')
    sed -i "s/DATA_PLACEHOLDER/\"$escaped_data\"/" "$report_file"
    
    log_success "HTML report generated: $report_file"
}

# Generate CSV report
generate_csv_report() {
    local comparison_file="$1"
    local report_file="$OUTPUT_DIR/benchmark-comparison.csv"
    
    log_info "Generating CSV report: $report_file"
    
    # Create CSV header
    echo "Benchmark,Current_ns_per_op,Baseline_ns_per_op,Change_percent,Status,Current_bytes_per_op,Current_allocs_per_op" > "$report_file"
    
    # Extract data and append to CSV
    jq -r '.results[] | [
        .name,
        (.current.ns_per_op // "N/A"),
        (.baseline.ns_per_op // "N/A"),
        (.changes.ns_per_op_percent // "N/A"),
        .status,
        (.current.bytes_per_op // "N/A"),
        (.current.allocs_per_op // "N/A")
    ] | @csv' "$comparison_file" >> "$report_file"
    
    log_success "CSV report generated: $report_file"
}

# Generate CI-friendly output
generate_ci_output() {
    local comparison_file="$1"
    
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        log_info "Generating GitHub Actions step summary..."
        
        local regressions improvements
        regressions=$(jq -r '.summary.regressions' "$comparison_file")
        improvements=$(jq -r '.summary.improvements' "$comparison_file")
        
        {
            echo "## 📊 Benchmark Comparison Results"
            echo ""
            echo "| Metric | Count |"
            echo "|--------|-------|"
            echo "| Total Benchmarks | $(jq -r '.summary.total_benchmarks' "$comparison_file") |"
            echo "| Regressions | $(jq -r '.summary.regressions' "$comparison_file") |"
            echo "| Improvements | $(jq -r '.summary.improvements' "$comparison_file") |"
            echo "| Stable | $(jq -r '.summary.stable' "$comparison_file") |"
            echo "| New | $(jq -r '.summary.new_benchmarks' "$comparison_file") |"
            echo ""
            echo "**Threshold:** ${THRESHOLD}%"
            echo ""
            
            if [[ $regressions -gt 0 ]]; then
                echo "### ⚠️ Performance Regressions"
                echo ""
                echo "| Benchmark | Change | Current | Baseline |"
                echo "|-----------|--------|---------|----------|"
                jq -r '.results[] | select(.status == "regression") | "| \(.name) | \(.changes.ns_per_op_percent | . * 10 | round / 10)% | \(.current.ns_per_op) ns/op | \(.baseline.ns_per_op) ns/op |"' "$comparison_file"
                echo ""
            fi
            
            if [[ $improvements -gt 0 ]]; then
                echo "### 🚀 Performance Improvements"
                echo ""
                echo "| Benchmark | Change | Current | Baseline |"
                echo "|-----------|--------|---------|----------|"
                jq -r '.results[] | select(.status == "improvement") | "| \(.name) | \((.changes.ns_per_op_percent * -1) | . * 10 | round / 10)% | \(.current.ns_per_op) ns/op | \(.baseline.ns_per_op) ns/op |"' "$comparison_file"
                echo ""
            fi
            
        } >> "$GITHUB_STEP_SUMMARY"
    fi
    
    # Set exit code for CI if regressions found
    if [[ $regressions -gt 0 ]]; then
        log_error "Performance regressions detected - check results"
        exit 1
    fi
}

# Main function
main() {
    parse_args "$@"
    validate_args
    setup_output_dir
    
    log_info "Starting benchmark comparison..."
    log_info "Configuration:"
    log_info "  Baseline file: $BASELINE_FILE"
    log_info "  Current file: $CURRENT_FILE"
    log_info "  Output format: $OUTPUT_FORMAT"
    log_info "  Threshold: ${THRESHOLD}%"
    log_info "  Generate baseline: $GENERATE_BASELINE"
    
    # Generate baseline if requested
    if [[ "$GENERATE_BASELINE" == "true" ]]; then
        log_info "Generating new baseline..."
        run_benchmarks "$BASELINE_FILE"
        log_success "Baseline generated: $BASELINE_FILE"
        return 0
    fi
    
    # Check if we need to run current benchmarks
    if [[ ! -f "$CURRENT_FILE" ]]; then
        log_info "Current benchmark file not found, running benchmarks..."
        run_benchmarks "$CURRENT_FILE"
    fi
    
    # Check if baseline exists
    if [[ ! -f "$BASELINE_FILE" ]]; then
        log_error "Baseline file not found: $BASELINE_FILE"
        log_info "Run with --generate-baseline to create a baseline"
        exit 1
    fi
    
    # Parse benchmark files to JSON
    local baseline_json="$OUTPUT_DIR/baseline.json"
    local current_json="$OUTPUT_DIR/current.json"
    
    parse_benchmark_file "$BASELINE_FILE" "$baseline_json"
    parse_benchmark_file "$CURRENT_FILE" "$current_json"
    
    # Compare benchmarks
    local comparison_file="$OUTPUT_DIR/comparison.json"
    compare_benchmarks "$baseline_json" "$current_json" "$comparison_file"
    
    # Generate report
    generate_report "$comparison_file"
    
    # Generate CI output
    generate_ci_output "$comparison_file"
    
    log_success "Benchmark comparison completed!"
    log_info "Results available in: $OUTPUT_DIR"
}

# Run main function with all arguments
main "$@"