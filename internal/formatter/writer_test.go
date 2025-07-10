package formatter

import (
	"os"
	"strings"
	"testing"
	"gopkg.in/yaml.v3"
)

func TestFormatToString(t *testing.T) {
	writer := NewWriter()
	parser := NewParser(true)
	
	// Test simple formatting
	content := `name: Test
version: 1.0.0
items:
  - item1
  - item2`
	
	node, err := parser.ParseYAML([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	
	result, err := writer.FormatToString(node)
	if err != nil {
		t.Errorf("FormatToString failed: %v", err)
	}
	
	if result == "" {
		t.Error("FormatToString returned empty string")
	}
	
	// Verify the output is valid YAML
	var test interface{}
	if err := yaml.Unmarshal([]byte(result), &test); err != nil {
		t.Errorf("Output is not valid YAML: %v", err)
	}
}

func TestFormatWithComments(t *testing.T) {
	writer := NewWriter()
	writer.SetPreserveComments(true)
	parser := NewParser(true)
	
	content, err := os.ReadFile("../../testdata/valid/with-comments.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	
	node, err := parser.ParseYAML(content)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	
	result, err := writer.FormatToString(node)
	if err != nil {
		t.Errorf("FormatToString failed: %v", err)
	}
	
	// Check that comments are preserved
	if !strings.Contains(result, "#") {
		t.Error("Comments were not preserved in output")
	}
}

func TestFormatNodesToString(t *testing.T) {
	writer := NewWriter()
	parser := NewParser(true)
	
	content, err := os.ReadFile("../../testdata/multi-document/simple-multi.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	
	nodes, err := parser.ParseMultiDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse multi-document YAML: %v", err)
	}
	
	result, err := writer.FormatNodesToString(nodes)
	if err != nil {
		t.Errorf("FormatNodesToString failed: %v", err)
	}
	
	// Check that document separators are present
	separatorCount := strings.Count(result, "---")
	if separatorCount < len(nodes)-1 {
		t.Errorf("Expected at least %d document separators, found %d", len(nodes)-1, separatorCount)
	}
}

func TestIndentSettings(t *testing.T) {
	writer := NewWriter()
	parser := NewParser(true)
	
	content := `root:
  child1:
    grandchild1: value1
    grandchild2: value2
  child2:
    - item1
    - item2`
	
	node, err := parser.ParseYAML([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	
	// Test with 2-space indent (default)
	writer.SetIndent(2)
	result2, err := writer.FormatToString(node)
	if err != nil {
		t.Errorf("FormatToString failed: %v", err)
	}
	
	// Test with 4-space indent
	writer.SetIndent(4)
	result4, err := writer.FormatToString(node)
	if err != nil {
		t.Errorf("FormatToString failed: %v", err)
	}
	
	// Verify different indentation by checking the first indented line
	lines2 := strings.Split(result2, "\n")
	lines4 := strings.Split(result4, "\n")
	
	// Find first indented line
	var indent2, indent4 int
	for _, line := range lines2 {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") {
			indent2 = 2
			break
		}
	}
	for _, line := range lines4 {
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     ") {
			indent4 = 4
			break
		}
	}
	
	if indent4 != 4 || indent2 != 2 {
		t.Errorf("Expected 2-space and 4-space indentation, got %d and %d", indent2, indent4)
	}
}

func TestValidateFormattedOutput(t *testing.T) {
	writer := NewWriter()
	
	tests := []struct {
		name    string
		content []byte
		wantErr bool
	}{
		{
			name:    "Valid YAML",
			content: []byte("name: test\nversion: 1.0.0"),
			wantErr: false,
		},
		{
			name:    "Invalid YAML",
			content: []byte("name: test\nversion: [unclosed"),
			wantErr: true,
		},
		{
			name:    "Empty content",
			content: []byte(""),
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writer.ValidateFormattedOutput(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormattedOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateStats(t *testing.T) {
	writer := NewWriter()
	
	original := []byte(`name: test
version: 1.0.0
items:
  - item1
  - item2`)
	
	formatted := []byte(`name: test
version: 1.0.0
items:
  - item1
  - item2
  - item3`)
	
	stats := writer.CalculateStats(original, formatted)
	
	if stats == nil {
		t.Fatal("CalculateStats returned nil")
	}
	
	if stats.OriginalLines == 0 {
		t.Error("OriginalLines should not be 0")
	}
	
	if stats.FormattedLines == 0 {
		t.Error("FormattedLines should not be 0")
	}
	
	if stats.OriginalBytes == 0 {
		t.Error("OriginalBytes should not be 0")
	}
	
	if stats.FormattedBytes == 0 {
		t.Error("FormattedBytes should not be 0")
	}
}

func TestSpecialCharacterHandling(t *testing.T) {
	writer := NewWriter()
	parser := NewParser(true)
	
	content, err := os.ReadFile("../../testdata/edge-cases/special-characters.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	
	node, err := parser.ParseYAML(content)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	
	result, err := writer.FormatToString(node)
	if err != nil {
		t.Errorf("FormatToString failed: %v", err)
	}
	
	// Verify special characters are preserved
	if !strings.Contains(result, "世界") {
		t.Error("Unicode characters were not preserved")
	}
	
	if !strings.Contains(result, "🌍") && !strings.Contains(result, "\\U0001F30D") {
		t.Error("Emoji characters were not preserved")
	}
}