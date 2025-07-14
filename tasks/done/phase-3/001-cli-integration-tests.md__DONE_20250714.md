# Phase 3.1: CLI Integration Tests

**Status**: 📋 PENDING
**Order**: 5
**Estimated Time**: 10 hours

## Description
Build stable integration tests for CLI commands and End-to-End scenarios with proper test isolation.

## Tasks to Complete

### Task 5.1: CLI Test Infrastructure (4 hours)
- [x] Create CLI test harness utility
- [x] Implement environment isolation
- [x] Add temporary directory management
- [x] Build command execution wrapper

**Files to Create/Modify**:
- `cmd/testing_utils.go` - CLI test harness
- `cmd/format_test.go` - Enhanced format command tests
- `cmd/schema_test.go` - Enhanced schema command tests

**Implementation Details**:
```go
type CLITestHarness struct {
    tempDir    string
    schemaDir  string
    stdout     *bytes.Buffer
    stderr     *bytes.Buffer
    originalEnv map[string]string
}

func NewCLITestHarness(t *testing.T) *CLITestHarness
func (h *CLITestHarness) ExecuteCommand(args ...string) error
func (h *CLITestHarness) CreateTestFile(path string, content string) error
```

### Task 5.2: E2E Test Stabilization (3 hours)  
- [x] Implement automatic binary building
- [x] Create isolated test environments
- [x] Add comprehensive workflow tests
- [x] Handle filesystem dependencies

**Files to Create/Modify**:
- `tests/e2e/setup_test.go` - Test setup and binary building
- `tests/e2e/harness.go` - E2E test environment
- `tests/e2e/workflow_test.go` - Complete workflow tests

### Task 5.3: Parallel Test Support (2 hours)
- [x] Ensure test isolation for parallel execution
- [x] Implement resource pooling
- [x] Add test environment cleanup
- [x] Handle concurrent file access

### Task 5.4: Error Scenario Testing (1 hour)
- [x] Test invalid input handling
- [x] Test missing file scenarios
- [x] Test permission errors
- [x] Test malformed command arguments

## Commands to Run
```bash
# Build binary for testing
go build -o sb-yaml-test .

# Run CLI tests
go test ./cmd/... -v

# Run E2E tests  
go test ./tests/e2e/... -v

# Run parallel tests
go test -race -parallel=4 ./cmd/... ./tests/e2e/...

# Expected: All integration tests should pass with proper isolation
```

## Success Criteria
- [x] All CLI tests run in isolated environments
- [x] E2E tests automatically build and use binaries
- [x] No conflicts during parallel test execution
- [x] Comprehensive error scenario coverage
- [x] Test execution time under 15 minutes