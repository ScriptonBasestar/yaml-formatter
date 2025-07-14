package testing

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestResourcePool(t *testing.T) {
	pool := NewResourcePoolWithLimit(2)

	if pool.GetConcurrency() != 2 {
		t.Errorf("Expected concurrency 2, got %d", pool.GetConcurrency())
	}

	// Test basic acquire/release
	pool.Acquire()
	pool.Release()

	metrics := pool.GetMetrics()
	if metrics.totalJobs != 1 {
		t.Errorf("Expected 1 total job, got %d", metrics.totalJobs)
	}
	if metrics.completedJobs != 1 {
		t.Errorf("Expected 1 completed job, got %d", metrics.completedJobs)
	}
}

func TestResourcePoolConcurrency(t *testing.T) {
	pool := NewResourcePoolWithLimit(2)
	var activeJobs int64
	var maxActiveJobs int64

	// Start 5 goroutines, but only 2 should run concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.Acquire()
			defer pool.Release()

			active := atomic.AddInt64(&activeJobs, 1)
			for {
				current := atomic.LoadInt64(&maxActiveJobs)
				if active <= current || atomic.CompareAndSwapInt64(&maxActiveJobs, current, active) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond) // Simulate work
			atomic.AddInt64(&activeJobs, -1)
		}()
	}

	wg.Wait()

	if maxActive := atomic.LoadInt64(&maxActiveJobs); maxActive > 2 {
		t.Errorf("Expected max 2 concurrent jobs, got %d", maxActive)
	}

	metrics := pool.GetMetrics()
	if metrics.completedJobs != 5 {
		t.Errorf("Expected 5 completed jobs, got %d", metrics.completedJobs)
	}
	if metrics.peakJobs > 2 {
		t.Errorf("Expected peak jobs <= 2, got %d", metrics.peakJobs)
	}
}

func TestResourcePoolExecuteWithPool(t *testing.T) {
	pool := NewResourcePoolWithLimit(1)
	executed := false

	pool.ExecuteWithPool(func() {
		executed = true
	})

	if !executed {
		t.Error("Function was not executed")
	}

	metrics := pool.GetMetrics()
	if metrics.completedJobs != 1 {
		t.Errorf("Expected 1 completed job, got %d", metrics.completedJobs)
	}
}

func TestParallelExecutor(t *testing.T) {
	pool := NewResourcePoolWithLimit(2)
	executor := NewParallelExecutor(pool)

	executor.Start()
	defer executor.Stop()

	var executed int64
	var wg sync.WaitGroup

	// Submit 5 jobs
	for i := 0; i < 5; i++ {
		wg.Add(1)
		executor.Submit(Job{
			Name:     "test-job",
			Category: "test",
			Function: func() error {
				atomic.AddInt64(&executed, 1)
				wg.Done()
				return nil
			},
		})
	}

	wg.Wait()

	if atomic.LoadInt64(&executed) != 5 {
		t.Errorf("Expected 5 executed jobs, got %d", atomic.LoadInt64(&executed))
	}
}

func TestGetRecommendedParallelism(t *testing.T) {
	tests := []struct {
		category string
		expected int
	}{
		{"unit", FastTestParallelism},
		{"fast", FastTestParallelism},
		{"integration", OptimalParallelism},
		{"ci", OptimalParallelism},
		{"e2e", SlowTestParallelism},
		{"slow", SlowTestParallelism},
		{"unknown", OptimalParallelism},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := GetRecommendedParallelism(tt.category)
			if result != tt.expected {
				t.Errorf("Expected %d for category %s, got %d", tt.expected, tt.category, result)
			}
		})
	}
}

func TestGetPoolForCategory(t *testing.T) {
	tests := []struct {
		category     string
		expectedPool *ResourcePool
	}{
		{"unit", UnitTestPool},
		{"fast", UnitTestPool},
		{"integration", IntegrationTestPool},
		{"ci", IntegrationTestPool},
		{"e2e", E2ETestPool},
		{"slow", E2ETestPool},
		{"unknown", IntegrationTestPool},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := GetPoolForCategory(tt.category)
			if result != tt.expectedPool {
				t.Errorf("Expected specific pool for category %s", tt.category)
			}
		})
	}
}

func TestOptimalParallelismValues(t *testing.T) {
	// Test that computed values are reasonable
	if OptimalParallelism < 1 {
		t.Errorf("OptimalParallelism should be at least 1, got %d", OptimalParallelism)
	}
	if OptimalParallelism > 8 {
		t.Errorf("OptimalParallelism should be at most 8, got %d", OptimalParallelism)
	}

	if FastTestParallelism < 1 {
		t.Errorf("FastTestParallelism should be at least 1, got %d", FastTestParallelism)
	}

	if SlowTestParallelism < 1 {
		t.Errorf("SlowTestParallelism should be at least 1, got %d", SlowTestParallelism)
	}
	if SlowTestParallelism > 4 {
		t.Errorf("SlowTestParallelism should be at most 4, got %d", SlowTestParallelism)
	}
}

func TestResourcePoolMetrics(t *testing.T) {
	pool := NewResourcePoolWithLimit(1)

	// Execute multiple jobs to test metrics
	for i := 0; i < 3; i++ {
		pool.ExecuteWithPool(func() {
			time.Sleep(10 * time.Millisecond)
		})
	}

	metrics := pool.GetMetrics()
	if metrics.totalJobs != 3 {
		t.Errorf("Expected 3 total jobs, got %d", metrics.totalJobs)
	}
	if metrics.completedJobs != 3 {
		t.Errorf("Expected 3 completed jobs, got %d", metrics.completedJobs)
	}
	if metrics.activeJobs != 0 {
		t.Errorf("Expected 0 active jobs after completion, got %d", metrics.activeJobs)
	}
	if metrics.peakJobs != 1 {
		t.Errorf("Expected peak jobs of 1, got %d", metrics.peakJobs)
	}
}

func BenchmarkResourcePool(b *testing.B) {
	pool := NewResourcePoolWithLimit(4)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.ExecuteWithPool(func() {
				// Simulate minimal work
			})
		}
	})
}

func BenchmarkParallelExecutor(b *testing.B) {
	pool := NewResourcePoolWithLimit(4)
	executor := NewParallelExecutor(pool)
	executor.Start()
	defer executor.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(1)
		executor.Submit(Job{
			Name: "bench-job",
			Function: func() error {
				wg.Done()
				return nil
			},
		})
		wg.Wait()
	}
}
