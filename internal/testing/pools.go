package testing

import (
	"runtime"
	"sync"
	"time"
)

// ResourcePool manages shared resources for parallel test execution
type ResourcePool struct {
	maxConcurrent int
	semaphore     chan struct{}
	wg            sync.WaitGroup
	metrics       *PoolMetrics
}

// PoolMetrics tracks resource pool usage statistics
type PoolMetrics struct {
	mu            sync.Mutex
	totalJobs     int64
	activeJobs    int64
	peakJobs      int64
	completedJobs int64
	avgWaitTime   time.Duration
	totalWaitTime time.Duration
}

// NewResourcePool creates a new resource pool with optimal concurrency settings
func NewResourcePool() *ResourcePool {
	// Use 75% of available CPU cores, with minimum 1 and maximum 8 for tests
	maxConcurrent := runtime.GOMAXPROCS(0) * 3 / 4
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if maxConcurrent > 8 {
		maxConcurrent = 8 // Limit to avoid overwhelming CI environments
	}

	return &ResourcePool{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
		metrics:       &PoolMetrics{},
	}
}

// NewResourcePoolWithLimit creates a resource pool with specific concurrency limit
func NewResourcePoolWithLimit(limit int) *ResourcePool {
	if limit < 1 {
		limit = 1
	}

	return &ResourcePool{
		maxConcurrent: limit,
		semaphore:     make(chan struct{}, limit),
		metrics:       &PoolMetrics{},
	}
}

// Acquire acquires a resource from the pool (blocking if at limit)
func (p *ResourcePool) Acquire() {
	start := time.Now()
	p.semaphore <- struct{}{}
	waitTime := time.Since(start)

	p.wg.Add(1)

	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.totalJobs++
	p.metrics.activeJobs++
	if p.metrics.activeJobs > p.metrics.peakJobs {
		p.metrics.peakJobs = p.metrics.activeJobs
	}
	p.metrics.totalWaitTime += waitTime
	p.metrics.avgWaitTime = p.metrics.totalWaitTime / time.Duration(p.metrics.totalJobs)
	p.metrics.mu.Unlock()
}

// Release releases a resource back to the pool
func (p *ResourcePool) Release() {
	<-p.semaphore
	p.wg.Done()

	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.activeJobs--
	p.metrics.completedJobs++
	p.metrics.mu.Unlock()
}

// Wait waits for all active jobs to complete
func (p *ResourcePool) Wait() {
	p.wg.Wait()
}

// GetMetrics returns current pool metrics
func (p *ResourcePool) GetMetrics() PoolMetrics {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	return *p.metrics
}

// GetConcurrency returns the maximum concurrency level
func (p *ResourcePool) GetConcurrency() int {
	return p.maxConcurrent
}

// ExecuteWithPool executes a function with resource pool management
func (p *ResourcePool) ExecuteWithPool(fn func()) {
	p.Acquire()
	defer p.Release()
	fn()
}

// ParallelExecutor manages parallel execution of test jobs
type ParallelExecutor struct {
	pool        *ResourcePool
	jobQueue    chan Job
	workerCount int
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// Job represents a test job to be executed
type Job struct {
	Name     string
	Function func() error
	Category string
	Priority int
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(pool *ResourcePool) *ParallelExecutor {
	if pool == nil {
		pool = NewResourcePool()
	}

	return &ParallelExecutor{
		pool:        pool,
		jobQueue:    make(chan Job, pool.GetConcurrency()*2), // Buffer for jobs
		workerCount: pool.GetConcurrency(),
		stopCh:      make(chan struct{}),
	}
}

// Start starts the parallel executor workers
func (pe *ParallelExecutor) Start() {
	for i := 0; i < pe.workerCount; i++ {
		pe.wg.Add(1)
		go pe.worker()
	}
}

// Stop stops the parallel executor
func (pe *ParallelExecutor) Stop() {
	close(pe.stopCh)
	pe.wg.Wait()
}

// Submit submits a job for parallel execution
func (pe *ParallelExecutor) Submit(job Job) {
	select {
	case pe.jobQueue <- job:
	case <-pe.stopCh:
		// Executor is stopped, ignore job
	}
}

// worker is the worker goroutine that processes jobs
func (pe *ParallelExecutor) worker() {
	defer pe.wg.Done()

	for {
		select {
		case job := <-pe.jobQueue:
			pe.pool.ExecuteWithPool(func() {
				if err := job.Function(); err != nil {
					// Log error but continue processing
					// In real implementation, you might want to collect errors
				}
			})
		case <-pe.stopCh:
			return
		}
	}
}

// GetPool returns the underlying resource pool
func (pe *ParallelExecutor) GetPool() *ResourcePool {
	return pe.pool
}

// CPU-aware settings
var (
	// OptimalParallelism calculates optimal parallelism based on CPU and memory
	OptimalParallelism = func() int {
		cpus := runtime.GOMAXPROCS(0)

		// For test execution, use 75% of CPUs with reasonable limits
		optimal := cpus * 3 / 4
		if optimal < 1 {
			optimal = 1
		}
		if optimal > 8 {
			optimal = 8 // Reasonable limit for test environments
		}

		return optimal
	}()

	// FastTestParallelism for quick unit tests
	FastTestParallelism = func() int {
		// Can be more aggressive for fast tests
		cpus := runtime.GOMAXPROCS(0)
		if cpus > 4 {
			return cpus
		}
		return cpus + 1 // Slightly oversubscribe for I/O bound tests
	}()

	// SlowTestParallelism for integration/E2E tests
	SlowTestParallelism = func() int {
		// More conservative for resource-intensive tests
		cpus := runtime.GOMAXPROCS(0)
		parallel := cpus / 2
		if parallel < 1 {
			parallel = 1
		}
		if parallel > 4 {
			parallel = 4 // Conservative limit
		}
		return parallel
	}()
)

// Global resource pools for different test categories
var (
	// UnitTestPool for fast unit tests
	UnitTestPool = NewResourcePoolWithLimit(FastTestParallelism)

	// IntegrationTestPool for integration tests
	IntegrationTestPool = NewResourcePoolWithLimit(OptimalParallelism)

	// E2ETestPool for end-to-end tests
	E2ETestPool = NewResourcePoolWithLimit(SlowTestParallelism)
)

// GetRecommendedParallelism returns recommended parallelism for test category
func GetRecommendedParallelism(category string) int {
	switch category {
	case "unit", "fast":
		return FastTestParallelism
	case "integration", "ci":
		return OptimalParallelism
	case "e2e", "slow":
		return SlowTestParallelism
	default:
		return OptimalParallelism
	}
}

// GetPoolForCategory returns the appropriate pool for test category
func GetPoolForCategory(category string) *ResourcePool {
	switch category {
	case "unit", "fast":
		return UnitTestPool
	case "integration", "ci":
		return IntegrationTestPool
	case "e2e", "slow":
		return E2ETestPool
	default:
		return IntegrationTestPool
	}
}
