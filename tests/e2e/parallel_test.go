package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestResourcePool manages a pool of reusable test harnesses for parallel execution
type TestResourcePool struct {
	pool    chan *E2ETestHarness
	size    int
	created int
	mu      sync.Mutex
}

// NewTestResourcePool creates a new resource pool with the specified size
func NewTestResourcePool(size int) *TestResourcePool {
	return &TestResourcePool{
		pool: make(chan *E2ETestHarness, size),
		size: size,
	}
}

// Get retrieves a test harness from the pool or creates a new one if none available
func (p *TestResourcePool) Get(t *testing.T) *E2ETestHarness {
	select {
	case harness := <-p.pool:
		// Reset the harness for reuse
		if err := harness.Reset(); err != nil {
			t.Fatalf("Failed to reset harness: %v", err)
		}
		return harness
	default:
		p.mu.Lock()
		defer p.mu.Unlock()

		if p.created < p.size {
			p.created++
			return NewE2ETestHarness(t)
		}

		// If pool is full, wait for an available harness
		harness := <-p.pool
		if err := harness.Reset(); err != nil {
			t.Fatalf("Failed to reset harness: %v", err)
		}
		return harness
	}
}

// Put returns a test harness to the pool for reuse
func (p *TestResourcePool) Put(harness *E2ETestHarness) {
	if harness == nil {
		return
	}

	select {
	case p.pool <- harness:
		// Successfully returned to pool
	default:
		// Pool is full, clean up the harness
		harness.cleanup()
	}
}

// Close shuts down the resource pool and cleans up all harnesses
func (p *TestResourcePool) Close() {
	close(p.pool)
	for harness := range p.pool {
		harness.cleanup()
	}
}

// ParallelTestSuite manages parallel test execution with resource pooling
type ParallelTestSuite struct {
	pool    *TestResourcePool
	timeout time.Duration
}

// NewParallelTestSuite creates a new parallel test suite
func NewParallelTestSuite(poolSize int, timeout time.Duration) *ParallelTestSuite {
	return &ParallelTestSuite{
		pool:    NewTestResourcePool(poolSize),
		timeout: timeout,
	}
}

// RunTest executes a test function with a pooled harness
func (s *ParallelTestSuite) RunTest(t *testing.T, testFunc func(*testing.T, *E2ETestHarness)) {
	t.Parallel()

	// Get harness from pool
	harness := s.pool.Get(t)
	defer s.pool.Put(harness)

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Run test in goroutine with timeout
	done := make(chan bool)
	go func() {
		testFunc(t, harness)
		done <- true
	}()

	select {
	case <-done:
		// Test completed successfully
	case <-ctx.Done():
		t.Fatalf("Test timed out after %v", s.timeout)
	}
}

// Close shuts down the parallel test suite
func (s *ParallelTestSuite) Close() {
	s.pool.Close()
}

// TestParallelExecution tests the parallel execution capabilities
func TestParallelExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel execution test in short mode")
	}

	// Test cases to run in parallel
	testCases := []struct {
		name string
		test func(*testing.T, *E2ETestHarness)
	}{
		{
			name: "ParallelFormat1",
			test: func(t *testing.T, h *E2ETestHarness) {
				if err := h.ChangeToTempDir(); err != nil {
					t.Fatal(err)
				}

				content := `name: parallel-test-1
version: 1.0.0`
				if err := h.CreateTestFile("test1.yml", content); err != nil {
					t.Fatal(err)
				}

				schema := `version:
name:`
				if err := h.CreateSchemaFile("test1", schema); err != nil {
					t.Fatal(err)
				}

				_, _, err := h.ExecuteCommand("format", "test1", "test1.yml")
				if err != nil {
					t.Errorf("Format failed: %v", err)
				}
			},
		},
		{
			name: "ParallelFormat2",
			test: func(t *testing.T, h *E2ETestHarness) {
				if err := h.ChangeToTempDir(); err != nil {
					t.Fatal(err)
				}

				content := `config:
  debug: true
app: parallel-test-2`
				if err := h.CreateTestFile("test2.yml", content); err != nil {
					t.Fatal(err)
				}

				schema := `app:
config:
  debug:`
				if err := h.CreateSchemaFile("test2", schema); err != nil {
					t.Fatal(err)
				}

				_, _, err := h.ExecuteCommand("format", "test2", "test2.yml")
				if err != nil {
					t.Errorf("Format failed: %v", err)
				}
			},
		},
		{
			name: "ParallelFormat3",
			test: func(t *testing.T, h *E2ETestHarness) {
				if err := h.ChangeToTempDir(); err != nil {
					t.Fatal(err)
				}

				content := `services:
  web:
    image: nginx
database:
  host: localhost`
				if err := h.CreateTestFile("test3.yml", content); err != nil {
					t.Fatal(err)
				}

				schema := `database:
  host:
services:
  web:
    image:`
				if err := h.CreateSchemaFile("test3", schema); err != nil {
					t.Fatal(err)
				}

				_, _, err := h.ExecuteCommand("format", "test3", "test3.yml")
				if err != nil {
					t.Errorf("Format failed: %v", err)
				}
			},
		},
		{
			name: "ParallelSchemaGen",
			test: func(t *testing.T, h *E2ETestHarness) {
				if err := h.ChangeToTempDir(); err != nil {
					t.Fatal(err)
				}

				content := `metadata:
  name: schema-gen-test
  labels:
    app: test
spec:
  replicas: 1`
				if err := h.CreateTestFile("schema-gen.yml", content); err != nil {
					t.Fatal(err)
				}

				stdout, _, err := h.ExecuteCommand("schema", "gen", "k8s-gen", "schema-gen.yml")
				if err != nil {
					t.Errorf("Schema generation failed: %v", err)
				}

				if len(stdout) == 0 {
					t.Error("Schema generation produced no output")
				}
			},
		},
		{
			name: "ParallelFileOperations",
			test: func(t *testing.T, h *E2ETestHarness) {
				if err := h.ChangeToTempDir(); err != nil {
					t.Fatal(err)
				}

				// Create multiple files rapidly
				for i := 0; i < 5; i++ {
					filename := fmt.Sprintf("file%d.yml", i)
					content := fmt.Sprintf("id: %d\ndata: test%d", i, i)
					if err := h.CreateTestFile(filename, content); err != nil {
						t.Errorf("Failed to create file %s: %v", filename, err)
					}
				}

				// Verify all files exist
				for i := 0; i < 5; i++ {
					filename := fmt.Sprintf("file%d.yml", i)
					if !h.FileExists(filename) {
						t.Errorf("File %s was not created", filename)
					}
				}
			},
		},
	}

	// Run all test cases in parallel
	for _, tc := range testCases {
		tc := tc // Capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Each test gets its own harness
			h := NewE2ETestHarness(t)
			defer h.cleanup()

			tc.test(t, h)
		})
	}
}

// TestConcurrentResourceAccess tests concurrent access to shared resources
func TestConcurrentResourceAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent resource access test in short mode")
	}

	const numWorkers = 5
	const numOperations = 10

	var wg sync.WaitGroup
	errorChan := make(chan error, numWorkers*numOperations)

	// Create multiple workers that perform concurrent operations
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker gets its own test harness
			h := NewE2ETestHarness(t)
			defer h.cleanup()

			if err := h.ChangeToTempDir(); err != nil {
				errorChan <- fmt.Errorf("worker %d: failed to change dir: %v", workerID, err)
				return
			}

			// Perform multiple operations
			for j := 0; j < numOperations; j++ {
				filename := fmt.Sprintf("worker%d_file%d.yml", workerID, j)
				content := fmt.Sprintf("worker: %d\noperation: %d\ndata: test", workerID, j)

				if err := h.CreateTestFile(filename, content); err != nil {
					errorChan <- fmt.Errorf("worker %d: failed to create file %s: %v", workerID, filename, err)
					continue
				}

				if !h.FileExists(filename) {
					errorChan <- fmt.Errorf("worker %d: file %s was not created", workerID, filename)
					continue
				}

				readContent, err := h.ReadTestFile(filename)
				if err != nil {
					errorChan <- fmt.Errorf("worker %d: failed to read file %s: %v", workerID, filename, err)
					continue
				}

				if readContent != content {
					errorChan <- fmt.Errorf("worker %d: content mismatch for file %s", workerID, filename)
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(errorChan)

	// Check for errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			t.Error(err)
		}
		t.Fatalf("Concurrent resource access test failed with %d errors", len(errors))
	}
}

// TestRaceConditionDetection tests for race conditions in parallel execution
func TestRaceConditionDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition detection test in short mode")
	}

	// This test should be run with -race flag to detect race conditions
	const numGoroutines = 10
	const numIterations = 100

	var wg sync.WaitGroup

	// Shared counter to detect race conditions
	var counter int
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Simulate work with test harness creation
				h := NewE2ETestHarness(t)
				defer h.cleanup()

				// Increment counter with proper synchronization
				mu.Lock()
				counter++
				mu.Unlock()

				// Perform a simple operation
				if err := h.ChangeToTempDir(); err != nil {
					t.Errorf("Failed to change directory: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	expectedCounter := numGoroutines * numIterations
	if counter != expectedCounter {
		t.Errorf("Race condition detected: expected counter %d, got %d", expectedCounter, counter)
	}
}
