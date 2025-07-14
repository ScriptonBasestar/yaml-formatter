package testdata

import (
	"embed"
	"path/filepath"
	"sync"
)

//go:embed valid/*.yml invalid/*.yml edge-cases/*.yml formatting/*/*.yml schema-validation/*.yml schema-validation/*.yaml multi-document/*.yml optimized/*.yml
var TestFiles embed.FS

// TestCase represents a test file
type TestCase struct {
	Name     string
	Path     string
	Category string
	Content  []byte
}

// GetTestCases returns all test cases from a category
func GetTestCases(category string) ([]TestCase, error) {
	var cases []TestCase
	
	entries, err := TestFiles.ReadDir(category)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		path := filepath.Join(category, entry.Name())
		content, err := TestFiles.ReadFile(path)
		if err != nil {
			return nil, err
		}
		
		cases = append(cases, TestCase{
			Name:     entry.Name(),
			Path:     path,
			Category: category,
			Content:  content,
		})
	}
	
	return cases, nil
}

// GetFormattingPairs returns input/expected pairs for formatting tests
func GetFormattingPairs() (map[string]FormattingPair, error) {
	pairs := make(map[string]FormattingPair)
	
	inputFiles, err := TestFiles.ReadDir("formatting/input")
	if err != nil {
		return nil, err
	}
	
	for _, file := range inputFiles {
		if file.IsDir() {
			continue
		}
		
		name := file.Name()
		inputPath := filepath.Join("formatting/input", name)
		expectedPath := filepath.Join("formatting/expected", name)
		
		input, err := TestFiles.ReadFile(inputPath)
		if err != nil {
			return nil, err
		}
		
		expected, err := TestFiles.ReadFile(expectedPath)
		if err != nil {
			return nil, err
		}
		
		pairs[name] = FormattingPair{
			Input:    input,
			Expected: expected,
		}
	}
	
	return pairs, nil
}

// FormattingPair represents an input/expected pair
type FormattingPair struct {
	Input    []byte
	Expected []byte
}

// OptimizedTestSuite provides efficient access to test data
type OptimizedTestSuite struct {
	cache  *TestDataCache
	loader *LazyLoader
}

// NewOptimizedTestSuite creates a new optimized test suite
func NewOptimizedTestSuite() *OptimizedTestSuite {
	return &OptimizedTestSuite{
		cache:  globalCache,
		loader: globalLazyLoader,
	}
}

// GetOptimizedTestData returns test data from optimized directory
func (ots *OptimizedTestSuite) GetOptimizedTestData(name string) ([]byte, error) {
	path := filepath.Join("optimized", name)
	return GetCachedTestData(path, "optimized")
}

// GetTestDataBySize returns test data categorized by size
func (ots *OptimizedTestSuite) GetTestDataBySize() (map[string][]TestCase, error) {
	categories := map[string][]TestCase{
		"small":  {},  // < 100 bytes
		"medium": {},  // 100-500 bytes
		"large":  {},  // > 500 bytes
	}
	
	allCategories := []string{"valid", "invalid", "edge-cases", "multi-document", "optimized"}
	
	for _, category := range allCategories {
		cases, err := GetTestCases(category)
		if err != nil {
			continue // Skip if category doesn't exist
		}
		
		for _, testCase := range cases {
			size := len(testCase.Content)
			switch {
			case size < 100:
				categories["small"] = append(categories["small"], testCase)
			case size < 500:
				categories["medium"] = append(categories["medium"], testCase)
			default:
				categories["large"] = append(categories["large"], testCase)
			}
		}
	}
	
	return categories, nil
}

// GetMinimalTestSet returns a minimal set of test cases for quick testing
func (ots *OptimizedTestSuite) GetMinimalTestSet() ([]TestCase, error) {
	minimalPaths := []string{
		"optimized/minimal-valid.yml",
		"optimized/minimal-complex.yml", 
		"optimized/minimal-multi.yml",
		"edge-cases/empty.yml",
		"invalid/bad-indentation.yml",
	}
	
	var cases []TestCase
	for _, path := range minimalPaths {
		content, err := GetCachedTestData(path, "minimal")
		if err != nil {
			continue // Skip if file doesn't exist
		}
		
		cases = append(cases, TestCase{
			Name:     filepath.Base(path),
			Path:     path,
			Category: "minimal",
			Content:  content,
		})
	}
	
	return cases, nil
}

// PreloadEssentialData preloads the most commonly used test data
func (ots *OptimizedTestSuite) PreloadEssentialData() error {
	essentialCategories := []string{"optimized", "edge-cases"}
	
	for _, category := range essentialCategories {
		if err := PreloadCategory(category); err != nil {
			return err
		}
	}
	
	return nil
}

// GetTestDataStats returns statistics about test data usage
func (ots *OptimizedTestSuite) GetTestDataStats() TestDataStats {
	cacheSize, categories := GetCacheStats()
	loadedCount := ots.loader.GetLoadedCount()
	
	// Calculate total test files
	totalFiles := 0
	allCategories := []string{"valid", "invalid", "edge-cases", "formatting/input", "formatting/expected", "multi-document", "schema-validation", "optimized"}
	
	for _, category := range allCategories {
		if entries, err := TestFiles.ReadDir(category); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					totalFiles++
				}
			}
		}
	}
	
	return TestDataStats{
		TotalFiles:       totalFiles,
		CachedFiles:      cacheSize,
		LoadedFiles:      loadedCount,
		CacheByCategory:  categories,
		CacheHitRate:     calculateCacheHitRate(),
	}
}

// TestDataStats represents statistics about test data usage
type TestDataStats struct {
	TotalFiles      int
	CachedFiles     int
	LoadedFiles     int
	CacheByCategory map[string]int
	CacheHitRate    float64
}

// Cache hit tracking
var (
	cacheHits   int64
	cacheMisses int64
	statsMutex  sync.RWMutex
)

func calculateCacheHitRate() float64 {
	statsMutex.RLock()
	defer statsMutex.RUnlock()
	
	total := cacheHits + cacheMisses
	if total == 0 {
		return 0
	}
	
	return float64(cacheHits) / float64(total) * 100
}

// TrackCacheHit increments cache hit counter
func TrackCacheHit() {
	statsMutex.Lock()
	defer statsMutex.Unlock()
	cacheHits++
}

// TrackCacheMiss increments cache miss counter
func TrackCacheMiss() {
	statsMutex.Lock()
	defer statsMutex.Unlock()
	cacheMisses++
}

// Global optimized test suite instance
var GlobalOptimizedSuite = NewOptimizedTestSuite()