package testdata

import (
	"fmt"
	"path/filepath"
	"sync"
)

// LazyLoader provides lazy loading functionality for test data
type LazyLoader struct {
	cache    *TestDataCache
	loaders  map[string]LoaderFunc
	mutex    sync.RWMutex
	loaded   map[string]bool
}

// LoaderFunc defines a function that loads test data on demand
type LoaderFunc func() ([]byte, error)

// NewLazyLoader creates a new lazy loader with optional cache
func NewLazyLoader(cache *TestDataCache) *LazyLoader {
	if cache == nil {
		cache = globalCache
	}
	
	return &LazyLoader{
		cache:   cache,
		loaders: make(map[string]LoaderFunc),
		loaded:  make(map[string]bool),
	}
}

// Register registers a loader function for a specific path
func (l *LazyLoader) Register(path string, loader LoaderFunc) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.loaders[path] = loader
	l.loaded[path] = false
}

// Load loads data for a path, using cache if available
func (l *LazyLoader) Load(path string) ([]byte, error) {
	// Check cache first
	if data, hit := l.cache.Get(path); hit {
		return data, nil
	}
	
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	// Check if we have a registered loader
	loader, exists := l.loaders[path]
	if !exists {
		return nil, fmt.Errorf("no loader registered for path: %s", path)
	}
	
	// Load data using the loader function
	data, err := loader()
	if err != nil {
		return nil, fmt.Errorf("failed to load data for %s: %v", path, err)
	}
	
	// Cache the loaded data
	category := l.getCategoryFromPath(path)
	l.cache.Set(path, data, category)
	l.loaded[path] = true
	
	return data, nil
}

// IsLoaded checks if data for a path has been loaded
func (l *LazyLoader) IsLoaded(path string) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	return l.loaded[path]
}

// LoadAll loads all registered data
func (l *LazyLoader) LoadAll() error {
	l.mutex.RLock()
	paths := make([]string, 0, len(l.loaders))
	for path := range l.loaders {
		paths = append(paths, path)
	}
	l.mutex.RUnlock()
	
	for _, path := range paths {
		if _, err := l.Load(path); err != nil {
			return err
		}
	}
	
	return nil
}

// UnloadAll clears all loaded data
func (l *LazyLoader) UnloadAll() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	for path := range l.loaded {
		l.loaded[path] = false
	}
	
	l.cache.Clear()
}

// GetLoadedCount returns the number of loaded datasets
func (l *LazyLoader) GetLoadedCount() int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	count := 0
	for _, loaded := range l.loaded {
		if loaded {
			count++
		}
	}
	
	return count
}

// getCategoryFromPath extracts category from file path
func (l *LazyLoader) getCategoryFromPath(path string) string {
	dir := filepath.Dir(path)
	if dir == "." {
		return "root"
	}
	return filepath.Base(dir)
}

// Global lazy loader instance
var globalLazyLoader = NewLazyLoader(globalCache)

// LazyTestCase represents a test case that can be loaded on demand
type LazyTestCase struct {
	Name     string
	Path     string
	Category string
	loader   *LazyLoader
}

// GetContent loads and returns the content of the lazy test case
func (ltc *LazyTestCase) GetContent() ([]byte, error) {
	return ltc.loader.Load(ltc.Path)
}

// IsLoaded checks if the test case content has been loaded
func (ltc *LazyTestCase) IsLoaded() bool {
	return ltc.loader.IsLoaded(ltc.Path)
}

// GetLazyTestCases returns lazy test cases for a category
func GetLazyTestCases(category string) ([]LazyTestCase, error) {
	var cases []LazyTestCase
	
	entries, err := TestFiles.ReadDir(category)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		path := filepath.Join(category, entry.Name())
		
		// Register loader for this path
		globalLazyLoader.Register(path, func() ([]byte, error) {
			return TestFiles.ReadFile(path)
		})
		
		cases = append(cases, LazyTestCase{
			Name:     entry.Name(),
			Path:     path,
			Category: category,
			loader:   globalLazyLoader,
		})
	}
	
	return cases, nil
}

// PreloadCategory preloads all test cases in a category
func PreloadCategory(category string) error {
	cases, err := GetLazyTestCases(category)
	if err != nil {
		return err
	}
	
	for _, tc := range cases {
		if _, err := tc.GetContent(); err != nil {
			return fmt.Errorf("failed to preload %s: %v", tc.Path, err)
		}
	}
	
	return nil
}

// GetLazyFormattingPairs returns lazy loading formatting pairs
func GetLazyFormattingPairs() (map[string]*LazyFormattingPair, error) {
	pairs := make(map[string]*LazyFormattingPair)
	
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
		
		// Register loaders
		globalLazyLoader.Register(inputPath, func() ([]byte, error) {
			return TestFiles.ReadFile(inputPath)
		})
		
		globalLazyLoader.Register(expectedPath, func() ([]byte, error) {
			return TestFiles.ReadFile(expectedPath)
		})
		
		pairs[name] = &LazyFormattingPair{
			InputPath:    inputPath,
			ExpectedPath: expectedPath,
			loader:       globalLazyLoader,
		}
	}
	
	return pairs, nil
}

// LazyFormattingPair represents a lazy loading formatting pair
type LazyFormattingPair struct {
	InputPath    string
	ExpectedPath string
	loader       *LazyLoader
}

// GetInput loads and returns the input data
func (lfp *LazyFormattingPair) GetInput() ([]byte, error) {
	return lfp.loader.Load(lfp.InputPath)
}

// GetExpected loads and returns the expected data
func (lfp *LazyFormattingPair) GetExpected() ([]byte, error) {
	return lfp.loader.Load(lfp.ExpectedPath)
}

// IsInputLoaded checks if input data is loaded
func (lfp *LazyFormattingPair) IsInputLoaded() bool {
	return lfp.loader.IsLoaded(lfp.InputPath)
}

// IsExpectedLoaded checks if expected data is loaded
func (lfp *LazyFormattingPair) IsExpectedLoaded() bool {
	return lfp.loader.IsLoaded(lfp.ExpectedPath)
}