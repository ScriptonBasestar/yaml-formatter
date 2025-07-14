package testdata

import (
	"testing"
	"time"
)

func TestTestDataCache(t *testing.T) {
	cache := NewTestDataCache(1 * time.Second)
	
	// Test cache set/get
	testData := []byte("test: data")
	cache.Set("test.yml", testData, "test")
	
	retrieved, hit := cache.Get("test.yml")
	if !hit {
		t.Error("Expected cache hit")
	}
	
	if string(retrieved) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(retrieved))
	}
	
	// Test cache size
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}
	
	// Test expiration
	time.Sleep(1100 * time.Millisecond)
	_, hit = cache.Get("test.yml")
	if hit {
		t.Error("Expected cache miss due to expiration")
	}
}

func TestLazyLoader(t *testing.T) {
	loader := NewLazyLoader(nil)
	
	// Register a test loader
	testData := []byte("lazy: data")
	loader.Register("test-lazy.yml", func() ([]byte, error) {
		return testData, nil
	})
	
	// Test that data is not loaded initially
	if loader.IsLoaded("test-lazy.yml") {
		t.Error("Data should not be loaded initially")
	}
	
	// Load data
	retrieved, err := loader.Load("test-lazy.yml")
	if err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}
	
	if string(retrieved) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(retrieved))
	}
	
	// Test that data is now loaded
	if !loader.IsLoaded("test-lazy.yml") {
		t.Error("Data should be loaded now")
	}
	
	// Test loaded count
	if loader.GetLoadedCount() != 1 {
		t.Errorf("Expected loaded count 1, got %d", loader.GetLoadedCount())
	}
}

func TestOptimizedTestSuite(t *testing.T) {
	suite := NewOptimizedTestSuite()
	
	// Test getting minimal test set
	minimalSet, err := suite.GetMinimalTestSet()
	if err != nil {
		t.Fatalf("Failed to get minimal test set: %v", err)
	}
	
	// Should have at least some test cases
	if len(minimalSet) == 0 {
		t.Error("Expected non-empty minimal test set")
	}
	
	// Test stats
	stats := suite.GetTestDataStats()
	if stats.TotalFiles == 0 {
		t.Error("Expected some total files")
	}
}

func TestOptimizedTestData(t *testing.T) {
	// Test that optimized test files exist and can be loaded
	optimizedFiles := []string{
		"minimal-valid.yml",
		"minimal-complex.yml",
		"minimal-multi.yml",
		"minimal-kubernetes.yml",
		"minimal-docker.yml",
	}
	
	suite := GlobalOptimizedSuite
	
	for _, file := range optimizedFiles {
		data, err := suite.GetOptimizedTestData(file)
		if err != nil {
			t.Errorf("Failed to load optimized test data %s: %v", file, err)
			continue
		}
		
		if len(data) == 0 {
			t.Errorf("Optimized test data %s is empty", file)
		}
		
		// Verify it's valid YAML-like content
		if len(data) > 0 && data[0] == '{' {
			t.Errorf("Optimized test data %s appears to be JSON, expected YAML", file)
		}
	}
}

func TestGetTestDataBySize(t *testing.T) {
	suite := GlobalOptimizedSuite
	
	sizeCategories, err := suite.GetTestDataBySize()
	if err != nil {
		t.Fatalf("Failed to get test data by size: %v", err)
	}
	
	// Should have all three size categories
	expectedCategories := []string{"small", "medium", "large"}
	for _, category := range expectedCategories {
		if _, exists := sizeCategories[category]; !exists {
			t.Errorf("Missing size category: %s", category)
		}
	}
	
	// Verify size constraints
	for category, cases := range sizeCategories {
		for _, testCase := range cases {
			size := len(testCase.Content)
			switch category {
			case "small":
				if size >= 100 {
					t.Errorf("Small category case %s has size %d, expected < 100", testCase.Name, size)
				}
			case "medium":
				if size < 100 || size >= 500 {
					t.Errorf("Medium category case %s has size %d, expected 100-499", testCase.Name, size)
				}
			case "large":
				if size < 500 {
					t.Errorf("Large category case %s has size %d, expected >= 500", testCase.Name, size)
				}
			}
		}
	}
}

func BenchmarkCachedVsDirectAccess(b *testing.B) {
	suite := GlobalOptimizedSuite
	
	// Warm up cache
	suite.GetOptimizedTestData("minimal-valid.yml")
	
	b.Run("CachedAccess", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := suite.GetOptimizedTestData("minimal-valid.yml")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("DirectAccess", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := TestFiles.ReadFile("optimized/minimal-valid.yml")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}