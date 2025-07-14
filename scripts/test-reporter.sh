#!/bin/bash

# Test Execution Reporting Script
# Generates comprehensive test execution reports with metrics and insights

set -euo pipefail

# Configuration
REPORT_DIR="${REPORT_DIR:-./test-reports}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-html}"
TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-./test-results}"
INCLUDE_COVERAGE="${INCLUDE_COVERAGE:-true}"
INCLUDE_BENCHMARKS="${INCLUDE_BENCHMARKS:-true}"
INCLUDE_FLAKY="${INCLUDE_FLAKY:-true}"
VERBOSE="${VERBOSE:-false}"
TEMPLATE_DIR="${TEMPLATE_DIR:-./templates}"

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
Test Execution Reporting Script

Generates comprehensive test execution reports combining metrics, coverage, benchmarks, and flaky test detection.

Usage: $0 [OPTIONS]

Options:
    -d, --report-dir DIR       Report output directory [default: ./test-reports]
    -f, --format FORMAT        Output format (html|json|text|markdown) [default: html]
    -r, --results-dir DIR      Test results directory [default: ./test-results]
    -c, --coverage BOOL        Include coverage analysis [default: true]
    -b, --benchmarks BOOL      Include benchmark results [default: true]
    -k, --flaky BOOL           Include flaky test analysis [default: true]
    -t, --template-dir DIR     Templates directory [default: ./templates]
    -v, --verbose              Enable verbose output
    -h, --help                 Show this help message

Examples:
    $0 --format markdown --report-dir /tmp/reports
    $0 --no-coverage --no-benchmarks
    $0 --results-dir ./ci-results --format json --verbose

Environment Variables:
    REPORT_DIR                 Default report directory
    OUTPUT_FORMAT              Default output format
    TEST_RESULTS_DIR           Default test results directory
    INCLUDE_COVERAGE           Include coverage (true/false)
    INCLUDE_BENCHMARKS         Include benchmarks (true/false)
    INCLUDE_FLAKY              Include flaky analysis (true/false)
    VERBOSE                    Enable verbose mode (true/false)

Input Data Sources:
    The script looks for the following files in TEST_RESULTS_DIR:
    - test-metrics/aggregate-metrics.json (from collect-metrics.sh)
    - flaky-test-results/flaky-summary.json (from flaky-test-detector.sh)
    - performance-results/performance-summary.json (from performance-monitor.sh)
    - coverage.out (Go test coverage)
    - benchmark-results.txt (Go benchmark output)
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--report-dir)
                REPORT_DIR="$2"
                shift 2
                ;;
            -f|--format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            -r|--results-dir)
                TEST_RESULTS_DIR="$2"
                shift 2
                ;;
            -c|--coverage)
                INCLUDE_COVERAGE="$2"
                shift 2
                ;;
            -b|--benchmarks)
                INCLUDE_BENCHMARKS="$2"
                shift 2
                ;;
            -k|--flaky)
                INCLUDE_FLAKY="$2"
                shift 2
                ;;
            -t|--template-dir)
                TEMPLATE_DIR="$2"
                shift 2
                ;;
            --no-coverage)
                INCLUDE_COVERAGE=false
                shift
                ;;
            --no-benchmarks)
                INCLUDE_BENCHMARKS=false
                shift
                ;;
            --no-flaky)
                INCLUDE_FLAKY=false
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
    if [[ ! "$OUTPUT_FORMAT" =~ ^(html|json|text|markdown)$ ]]; then
        log_error "Invalid output format: $OUTPUT_FORMAT"
        exit 1
    fi

    if [[ ! "$INCLUDE_COVERAGE" =~ ^(true|false)$ ]]; then
        log_error "Include coverage must be true or false"
        exit 1
    fi

    if [[ ! "$INCLUDE_BENCHMARKS" =~ ^(true|false)$ ]]; then
        log_error "Include benchmarks must be true or false"
        exit 1
    fi

    if [[ ! "$INCLUDE_FLAKY" =~ ^(true|false)$ ]]; then
        log_error "Include flaky must be true or false"
        exit 1
    fi
}

# Setup report directory
setup_report_dir() {
    mkdir -p "$REPORT_DIR"
    log_info "Report directory: $REPORT_DIR"
}

# Collect all available test data
collect_test_data() {
    local data_file="$REPORT_DIR/collected-data.json"
    
    log_info "Collecting test data from: $TEST_RESULTS_DIR"
    
    # Initialize the data structure
    cat > "$data_file" << 'EOF'
{
  "timestamp": "",
  "sources": {
    "metrics": null,
    "flaky": null,
    "performance": null,
    "coverage": null,
    "benchmarks": null
  },
  "summary": {
    "total_tests": 0,
    "success_rate": 0,
    "coverage_percent": 0,
    "flaky_tests": 0,
    "performance_regressions": 0
  }
}
EOF
    
    # Update timestamp
    jq --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.timestamp = $timestamp' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
    
    # Collect metrics data
    local metrics_file="$TEST_RESULTS_DIR/test-metrics/aggregate-metrics.json"
    if [[ -f "$metrics_file" ]]; then
        log_info "Found test metrics: $metrics_file"
        jq --argjson metrics "$(cat "$metrics_file")" '.sources.metrics = $metrics' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
        
        # Update summary from metrics
        jq '.summary.total_tests = .sources.metrics.overall_metrics.total_tests' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
        jq '.summary.success_rate = .sources.metrics.overall_metrics.success_rate_percent' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
    else
        log_warn "Test metrics not found: $metrics_file"
    fi
    
    # Collect flaky test data
    if [[ "$INCLUDE_FLAKY" == "true" ]]; then
        local flaky_file="$TEST_RESULTS_DIR/flaky-test-results/flaky-summary.json"
        if [[ -f "$flaky_file" ]]; then
            log_info "Found flaky test data: $flaky_file"
            jq --argjson flaky "$(cat "$flaky_file")" '.sources.flaky = $flaky' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
            
            # Update summary from flaky data
            jq '.summary.flaky_tests = .sources.flaky.flaky_tests' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
        else
            log_warn "Flaky test data not found: $flaky_file"
        fi
    fi
    
    # Collect performance data
    if [[ "$INCLUDE_BENCHMARKS" == "true" ]]; then
        local perf_file="$TEST_RESULTS_DIR/performance-results/performance-summary.json"
        if [[ -f "$perf_file" ]]; then
            log_info "Found performance data: $perf_file"
            jq --argjson perf "$(cat "$perf_file")" '.sources.performance = $perf' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
            
            # Update summary from performance data
            jq '.summary.performance_regressions = (.sources.performance.summary.regressions // 0)' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
        else
            log_warn "Performance data not found: $perf_file"
        fi
    fi
    
    # Collect coverage data
    if [[ "$INCLUDE_COVERAGE" == "true" ]]; then
        local coverage_file="$TEST_RESULTS_DIR/coverage.out"
        if [[ -f "$coverage_file" ]]; then
            log_info "Found coverage data: $coverage_file"
            
            # Extract coverage percentage
            local coverage_percent
            if command -v go >/dev/null 2>&1; then
                coverage_percent=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}' | sed 's/%//')
                jq --argjson coverage "$coverage_percent" '.sources.coverage = {"percent": $coverage, "file": "coverage.out"}' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
                jq --argjson coverage "$coverage_percent" '.summary.coverage_percent = $coverage' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
            else
                log_warn "Go toolchain not available for coverage analysis"
            fi
        else
            log_warn "Coverage data not found: $coverage_file"
        fi
    fi
    
    # Collect benchmark data
    if [[ "$INCLUDE_BENCHMARKS" == "true" ]]; then
        local benchmark_file="$TEST_RESULTS_DIR/benchmark-results.txt"
        if [[ -f "$benchmark_file" ]]; then
            log_info "Found benchmark data: $benchmark_file"
            local benchmark_count
            benchmark_count=$(grep -c "^Benchmark" "$benchmark_file" 2>/dev/null || echo 0)
            jq --argjson count "$benchmark_count" '.sources.benchmarks = {"count": $count, "file": "benchmark-results.txt"}' "$data_file" > "${data_file}.tmp" && mv "${data_file}.tmp" "$data_file"
        else
            log_warn "Benchmark data not found: $benchmark_file"
        fi
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Collected data summary:"
        jq '.summary' "$data_file"
    fi
    
    echo "$data_file"
}

# Generate HTML report
generate_html_report() {
    local data_file="$1"
    local html_file="$REPORT_DIR/test-report.html"
    
    log_info "Generating HTML report: $html_file"
    
    # Create HTML report
    cat > "$html_file" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Test Execution Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { border-bottom: 3px solid #007acc; margin-bottom: 30px; padding-bottom: 20px; }
        .header h1 { color: #007acc; margin: 0; font-size: 2.5em; }
        .header .subtitle { color: #666; margin-top: 5px; }
        .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .metric-card { background: linear-gradient(135deg, #f8f9fa, #e9ecef); padding: 20px; border-radius: 8px; text-align: center; border-left: 4px solid #007acc; }
        .metric-card.success { border-left-color: #28a745; }
        .metric-card.warning { border-left-color: #ffc107; }
        .metric-card.danger { border-left-color: #dc3545; }
        .metric-value { font-size: 2em; font-weight: bold; color: #333; }
        .metric-label { color: #666; margin-top: 5px; }
        .section { margin-bottom: 40px; }
        .section h2 { color: #333; border-bottom: 2px solid #eee; padding-bottom: 10px; }
        .status-badge { padding: 4px 8px; border-radius: 4px; font-size: 0.8em; font-weight: bold; }
        .status-success { background: #d4edda; color: #155724; }
        .status-warning { background: #fff3cd; color: #856404; }
        .status-danger { background: #f8d7da; color: #721c24; }
        .data-table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        .data-table th, .data-table td { padding: 12px; text-align: left; border-bottom: 1px solid #dee2e6; }
        .data-table th { background: #f8f9fa; font-weight: 600; }
        .progress-bar { width: 100%; height: 20px; background: #e9ecef; border-radius: 10px; overflow: hidden; }
        .progress-fill { height: 100%; background: #007acc; transition: width 0.3s ease; }
        .timestamp { color: #666; font-size: 0.9em; }
        .no-data { text-align: center; color: #999; padding: 40px; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Test Execution Report</h1>
            <div class="subtitle">Comprehensive analysis of test execution results</div>
            <div class="timestamp">Generated: <span id="timestamp"></span></div>
        </div>

        <div class="summary-grid">
            <div class="metric-card success">
                <div class="metric-value" id="total-tests">-</div>
                <div class="metric-label">Total Tests</div>
            </div>
            <div class="metric-card" id="success-rate-card">
                <div class="metric-value" id="success-rate">-</div>
                <div class="metric-label">Success Rate</div>
            </div>
            <div class="metric-card" id="coverage-card">
                <div class="metric-value" id="coverage">-</div>
                <div class="metric-label">Code Coverage</div>
            </div>
            <div class="metric-card" id="flaky-card">
                <div class="metric-value" id="flaky-tests">-</div>
                <div class="metric-label">Flaky Tests</div>
            </div>
        </div>

        <div class="section">
            <h2>📊 Test Metrics Overview</h2>
            <div id="metrics-content"></div>
        </div>

        <div class="section">
            <h2>🔍 Test Quality Analysis</h2>
            <div id="quality-content"></div>
        </div>

        <div class="section">
            <h2>📈 Performance Analysis</h2>
            <div id="performance-content"></div>
        </div>

        <div class="section">
            <h2>📋 Detailed Results</h2>
            <div id="details-content"></div>
        </div>
    </div>

    <script>
        // Data will be injected here
        const testData = DATA_PLACEHOLDER;
        
        function formatPercent(value) {
            return value ? value.toFixed(1) + '%' : 'N/A';
        }
        
        function formatNumber(value) {
            return value || 0;
        }
        
        function getStatusClass(value, thresholds) {
            if (value >= thresholds.good) return 'success';
            if (value >= thresholds.warning) return 'warning';
            return 'danger';
        }
        
        function renderSummary() {
            document.getElementById('timestamp').textContent = new Date(testData.timestamp).toLocaleString();
            document.getElementById('total-tests').textContent = formatNumber(testData.summary.total_tests);
            document.getElementById('success-rate').textContent = formatPercent(testData.summary.success_rate);
            document.getElementById('coverage').textContent = formatPercent(testData.summary.coverage_percent);
            document.getElementById('flaky-tests').textContent = formatNumber(testData.summary.flaky_tests);
            
            // Apply status colors
            const successRate = testData.summary.success_rate || 0;
            const successCard = document.getElementById('success-rate-card');
            successCard.className = 'metric-card ' + getStatusClass(successRate, {good: 95, warning: 80});
            
            const coverage = testData.summary.coverage_percent || 0;
            const coverageCard = document.getElementById('coverage-card');
            coverageCard.className = 'metric-card ' + getStatusClass(coverage, {good: 80, warning: 60});
            
            const flakyTests = testData.summary.flaky_tests || 0;
            const flakyCard = document.getElementById('flaky-card');
            flakyCard.className = 'metric-card ' + (flakyTests > 0 ? 'danger' : 'success');
        }
        
        function renderMetrics() {
            const metricsEl = document.getElementById('metrics-content');
            const metrics = testData.sources.metrics;
            
            if (!metrics) {
                metricsEl.innerHTML = '<div class="no-data">No test metrics data available</div>';
                return;
            }
            
            let html = '<table class="data-table">';
            html += '<tr><th>Category</th><th>Tests</th><th>Passed</th><th>Failed</th><th>Success Rate</th><th>Duration</th></tr>';
            
            metrics.categories.forEach(cat => {
                const successRate = cat.test_counts.total > 0 ? (cat.test_counts.passed / cat.test_counts.total * 100) : 0;
                html += `<tr>
                    <td>${cat.category}</td>
                    <td>${cat.test_counts.total}</td>
                    <td>${cat.test_counts.passed}</td>
                    <td>${cat.test_counts.failed}</td>
                    <td>${successRate.toFixed(1)}%</td>
                    <td>${cat.duration_seconds.toFixed(2)}s</td>
                </tr>`;
            });
            
            html += '</table>';
            metricsEl.innerHTML = html;
        }
        
        function renderQuality() {
            const qualityEl = document.getElementById('quality-content');
            let html = '';
            
            // Coverage analysis
            if (testData.sources.coverage) {
                const coverage = testData.sources.coverage.percent;
                html += `<div style="margin-bottom: 20px;">
                    <h3>Code Coverage</h3>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${coverage}%"></div>
                    </div>
                    <p>${coverage}% of code is covered by tests</p>
                </div>`;
            }
            
            // Flaky test analysis
            if (testData.sources.flaky) {
                const flaky = testData.sources.flaky;
                html += `<div style="margin-bottom: 20px;">
                    <h3>Test Stability</h3>
                    <p><strong>${flaky.flaky_tests}</strong> flaky tests detected out of <strong>${flaky.total_tests}</strong> total tests</p>
                    <p>Flaky percentage: <strong>${flaky.flaky_percentage}%</strong></p>
                </div>`;
            }
            
            qualityEl.innerHTML = html || '<div class="no-data">No quality analysis data available</div>';
        }
        
        function renderPerformance() {
            const perfEl = document.getElementById('performance-content');
            const perf = testData.sources.performance;
            
            if (!perf) {
                perfEl.innerHTML = '<div class="no-data">No performance data available</div>';
                return;
            }
            
            let html = `<div style="margin-bottom: 20px;">
                <h3>Performance Summary</h3>
                <p><strong>Regressions:</strong> ${perf.summary.regressions}</p>
                <p><strong>Improvements:</strong> ${perf.summary.improvements}</p>
                <p><strong>Overall Status:</strong> <span class="status-badge status-${perf.has_regressions ? 'danger' : 'success'}">${perf.overall_status}</span></p>
            </div>`;
            
            perfEl.innerHTML = html;
        }
        
        function renderDetails() {
            const detailsEl = document.getElementById('details-content');
            let html = '<h3>Configuration</h3>';
            html += '<ul>';
            html += `<li><strong>Include Coverage:</strong> ${INCLUDE_COVERAGE}</li>`;
            html += `<li><strong>Include Benchmarks:</strong> ${INCLUDE_BENCHMARKS}</li>`;
            html += `<li><strong>Include Flaky Analysis:</strong> ${INCLUDE_FLAKY}</li>`;
            html += '</ul>';
            
            detailsEl.innerHTML = html;
        }
        
        // Initialize report
        renderSummary();
        renderMetrics();
        renderQuality();
        renderPerformance();
        renderDetails();
    </script>
</body>
</html>
EOF
    
    # Inject data into HTML
    local escaped_data
    escaped_data=$(jq -c . "$data_file" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g')
    sed -i "s/DATA_PLACEHOLDER/\"$escaped_data\"/" "$html_file"
    sed -i "s/INCLUDE_COVERAGE/$INCLUDE_COVERAGE/" "$html_file"
    sed -i "s/INCLUDE_BENCHMARKS/$INCLUDE_BENCHMARKS/" "$html_file"
    sed -i "s/INCLUDE_FLAKY/$INCLUDE_FLAKY/" "$html_file"
    
    echo "$html_file"
}

# Generate JSON report
generate_json_report() {
    local data_file="$1"
    local json_file="$REPORT_DIR/test-report.json"
    
    log_info "Generating JSON report: $json_file"
    
    # Add metadata to the data
    jq --arg format "json" \
       --arg include_coverage "$INCLUDE_COVERAGE" \
       --arg include_benchmarks "$INCLUDE_BENCHMARKS" \
       --arg include_flaky "$INCLUDE_FLAKY" \
       '. + {
         "report_metadata": {
           "format": $format,
           "include_coverage": ($include_coverage == "true"),
           "include_benchmarks": ($include_benchmarks == "true"),
           "include_flaky": ($include_flaky == "true"),
           "generated_by": "test-reporter.sh"
         }
       }' "$data_file" > "$json_file"
    
    echo "$json_file"
}

# Generate Markdown report
generate_markdown_report() {
    local data_file="$1"
    local md_file="$REPORT_DIR/test-report.md"
    
    log_info "Generating Markdown report: $md_file"
    
    # Generate markdown using jq
    jq -r '
    "# Test Execution Report\n",
    "Generated: " + .timestamp + "\n",
    "## Summary\n",
    "| Metric | Value |",
    "|--------|-------|",
    "| Total Tests | " + (.summary.total_tests | tostring) + " |",
    "| Success Rate | " + ((.summary.success_rate // 0) | tostring) + "% |",
    "| Code Coverage | " + ((.summary.coverage_percent // 0) | tostring) + "% |",
    "| Flaky Tests | " + (.summary.flaky_tests | tostring) + " |\n",
    
    if .sources.metrics then
        "## Test Metrics by Category\n",
        "| Category | Tests | Passed | Failed | Success Rate | Duration |",
        "|----------|-------|--------|--------|--------------|----------|",
        (.sources.metrics.categories[] | "| " + .category + " | " + (.test_counts.total | tostring) + " | " + (.test_counts.passed | tostring) + " | " + (.test_counts.failed | tostring) + " | " + (.success_rate_percent | tostring) + "% | " + (.duration_seconds | tostring) + "s |"),
        ""
    else "" end,
    
    if .sources.flaky then
        "## Test Stability Analysis\n",
        "- **Total Tests Analyzed:** " + (.sources.flaky.total_tests | tostring),
        "- **Flaky Tests Detected:** " + (.sources.flaky.flaky_tests | tostring),
        "- **Flaky Percentage:** " + (.sources.flaky.flaky_percentage | tostring) + "%\n"
    else "" end,
    
    if .sources.performance then
        "## Performance Analysis\n",
        "- **Regressions:** " + (.sources.performance.summary.regressions | tostring),
        "- **Improvements:** " + (.sources.performance.summary.improvements | tostring),
        "- **Overall Status:** " + .sources.performance.overall_status + "\n"
    else "" end,
    
    "## Configuration\n",
    "- Include Coverage: " + (if .report_metadata.include_coverage then "Yes" else "No" end),
    "- Include Benchmarks: " + (if .report_metadata.include_benchmarks then "Yes" else "No" end),
    "- Include Flaky Analysis: " + (if .report_metadata.include_flaky then "Yes" else "No" end)
    ' "$data_file" > "$md_file"
    
    echo "$md_file"
}

# Generate text report
generate_text_report() {
    local data_file="$1"
    local txt_file="$REPORT_DIR/test-report.txt"
    
    log_info "Generating text report: $txt_file"
    
    {
        echo "Test Execution Report"
        echo "===================="
        echo ""
        echo "Generated: $(jq -r '.timestamp' "$data_file")"
        echo ""
        echo "Summary:"
        echo "--------"
        echo "Total Tests: $(jq -r '.summary.total_tests' "$data_file")"
        echo "Success Rate: $(jq -r '.summary.success_rate // 0' "$data_file")%"
        echo "Code Coverage: $(jq -r '.summary.coverage_percent // 0' "$data_file")%"
        echo "Flaky Tests: $(jq -r '.summary.flaky_tests' "$data_file")"
        echo ""
        
        if jq -e '.sources.metrics' "$data_file" >/dev/null; then
            echo "Test Metrics by Category:"
            echo "------------------------"
            printf "%-15s %8s %8s %8s %12s %10s\n" "Category" "Tests" "Passed" "Failed" "Success %" "Duration"
            printf "%-15s %8s %8s %8s %12s %10s\n" "--------" "-----" "------" "------" "----------" "--------"
            jq -r '.sources.metrics.categories[] | [.category, .test_counts.total, .test_counts.passed, .test_counts.failed, (.success_rate_percent | tostring + "%"), (.duration_seconds | tostring + "s")] | @tsv' "$data_file" | while IFS=$'\t' read -r category tests passed failed success duration; do
                printf "%-15s %8s %8s %8s %12s %10s\n" "$category" "$tests" "$passed" "$failed" "$success" "$duration"
            done
            echo ""
        fi
        
        if jq -e '.sources.flaky' "$data_file" >/dev/null; then
            echo "Test Stability Analysis:"
            echo "-----------------------"
            echo "Total Tests Analyzed: $(jq -r '.sources.flaky.total_tests' "$data_file")"
            echo "Flaky Tests Detected: $(jq -r '.sources.flaky.flaky_tests' "$data_file")"
            echo "Flaky Percentage: $(jq -r '.sources.flaky.flaky_percentage' "$data_file")%"
            echo ""
        fi
        
        if jq -e '.sources.performance' "$data_file" >/dev/null; then
            echo "Performance Analysis:"
            echo "--------------------"
            echo "Regressions: $(jq -r '.sources.performance.summary.regressions' "$data_file")"
            echo "Improvements: $(jq -r '.sources.performance.summary.improvements' "$data_file")"
            echo "Overall Status: $(jq -r '.sources.performance.overall_status' "$data_file")"
            echo ""
        fi
        
        echo "Configuration:"
        echo "-------------"
        echo "Include Coverage: $INCLUDE_COVERAGE"
        echo "Include Benchmarks: $INCLUDE_BENCHMARKS"
        echo "Include Flaky Analysis: $INCLUDE_FLAKY"
        
    } > "$txt_file"
    
    echo "$txt_file"
}

# Generate CI-friendly output
generate_ci_output() {
    local data_file="$1"
    
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        log_info "Generating GitHub Actions step summary..."
        
        {
            echo "## 📋 Test Execution Report"
            echo ""
            echo "| Metric | Value |"
            echo "|--------|-------|"
            echo "| Total Tests | $(jq -r '.summary.total_tests' "$data_file") |"
            echo "| Success Rate | $(jq -r '.summary.success_rate // 0' "$data_file")% |"
            echo "| Code Coverage | $(jq -r '.summary.coverage_percent // 0' "$data_file")% |"
            echo "| Flaky Tests | $(jq -r '.summary.flaky_tests' "$data_file") |"
            echo ""
            
            if jq -e '.sources.metrics' "$data_file" >/dev/null; then
                echo "### Test Results by Category"
                echo ""
                echo "| Category | Tests | Success Rate | Duration |"
                echo "|----------|-------|--------------|----------|"
                jq -r '.sources.metrics.categories[] | "| \(.category) | \(.test_counts.total) | \(.success_rate_percent)% | \(.duration_seconds)s |"' "$data_file"
                echo ""
            fi
            
            # Add quality indicators
            local success_rate flaky_tests coverage
            success_rate=$(jq -r '.summary.success_rate // 0' "$data_file")
            flaky_tests=$(jq -r '.summary.flaky_tests' "$data_file")
            coverage=$(jq -r '.summary.coverage_percent // 0' "$data_file")
            
            echo "### Quality Indicators"
            echo ""
            if (( $(echo "$success_rate >= 95" | bc -l) )); then
                echo "✅ **Test Success Rate**: Excellent (${success_rate}%)"
            elif (( $(echo "$success_rate >= 80" | bc -l) )); then
                echo "⚠️ **Test Success Rate**: Good (${success_rate}%)"
            else
                echo "❌ **Test Success Rate**: Needs Improvement (${success_rate}%)"
            fi
            
            if [[ "$flaky_tests" == "0" ]]; then
                echo "✅ **Test Stability**: No flaky tests detected"
            else
                echo "⚠️ **Test Stability**: ${flaky_tests} flaky test(s) detected"
            fi
            
            if (( $(echo "$coverage >= 80" | bc -l) )); then
                echo "✅ **Code Coverage**: Good (${coverage}%)"
            elif (( $(echo "$coverage >= 60" | bc -l) )); then
                echo "⚠️ **Code Coverage**: Moderate (${coverage}%)"
            else
                echo "❌ **Code Coverage**: Low (${coverage}%)"
            fi
            
        } >> "$GITHUB_STEP_SUMMARY"
    fi
}

# Main function
main() {
    parse_args "$@"
    validate_args
    setup_report_dir
    
    log_info "Starting test report generation..."
    log_info "Configuration:"
    log_info "  Output format: $OUTPUT_FORMAT"
    log_info "  Test results directory: $TEST_RESULTS_DIR"
    log_info "  Include coverage: $INCLUDE_COVERAGE"
    log_info "  Include benchmarks: $INCLUDE_BENCHMARKS"
    log_info "  Include flaky analysis: $INCLUDE_FLAKY"
    
    # Collect all test data
    local data_file
    data_file=$(collect_test_data)
    
    # Generate report in requested format
    local report_file
    case "$OUTPUT_FORMAT" in
        "html")
            report_file=$(generate_html_report "$data_file")
            ;;
        "json")
            report_file=$(generate_json_report "$data_file")
            ;;
        "markdown")
            report_file=$(generate_markdown_report "$data_file")
            ;;
        "text")
            report_file=$(generate_text_report "$data_file")
            ;;
    esac
    
    # Generate CI output
    generate_ci_output "$data_file"
    
    log_success "Test report generated successfully!"
    log_info "Report file: $report_file"
    
    # Print summary to console
    local total_tests success_rate flaky_tests
    total_tests=$(jq -r '.summary.total_tests' "$data_file")
    success_rate=$(jq -r '.summary.success_rate // 0' "$data_file")
    flaky_tests=$(jq -r '.summary.flaky_tests' "$data_file")
    
    log_info "Report Summary:"
    log_info "  Total tests: $total_tests"
    log_info "  Success rate: ${success_rate}%"
    log_info "  Flaky tests: $flaky_tests"
}

# Run main function with all arguments
main "$@"