package testdata

import (
	"embed"
	"path/filepath"
)

//go:embed valid/*.yml invalid/*.yml edge-cases/*.yml formatting/*/*.yml schema-validation/*.yml schema-validation/*.yaml multi-document/*.yml
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