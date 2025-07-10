package formatter

import (
	"os"
	"path/filepath"
	"testing"
	"yaml-formatter/internal/schema"
)

func TestFormatterWithTestData(t *testing.T) {
	// Load Docker Compose schema for testing
	schemaContent, err := os.ReadFile("../../examples/docker-compose.schema.yaml")
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}
	
	s, err := schema.LoadFromBytes(schemaContent, "docker-compose")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}
	
	formatter := NewFormatter(s)
	
	// Test formatting pairs
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unordered-docker-compose.yml",
			input:    "../../testdata/formatting/input/unordered-docker-compose.yml",
			expected: "../../testdata/formatting/expected/unordered-docker-compose.yml",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := os.ReadFile(tc.input)
			if err != nil {
				t.Fatalf("Failed to read input file: %v", err)
			}
			
			expected, err := os.ReadFile(tc.expected)
			if err != nil {
				t.Fatalf("Failed to read expected file: %v", err)
			}
			
			result, err := formatter.FormatContent(input)
			if err != nil {
				t.Errorf("FormatContent failed: %v", err)
				return
			}
			
			if string(result) != string(expected) {
				t.Errorf("Formatted output doesn't match expected.\nGot:\n%s\nExpected:\n%s", result, expected)
			}
		})
	}
}

func TestCheckFormat(t *testing.T) {
	// Load test schema
	schemaContent, err := os.ReadFile("../../testdata/schema-validation/test.schema.yaml")
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}
	
	s, err := schema.LoadFromBytes(schemaContent, "test")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}
	
	formatter := NewFormatter(s)
	
	// Test properly formatted file
	properlyFormatted, err := os.ReadFile("../../testdata/schema-validation/matches-schema.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	
	isFormatted, err := formatter.CheckFormat(properlyFormatted)
	if err != nil {
		t.Errorf("CheckFormat failed: %v", err)
	}
	
	if !isFormatted {
		t.Error("CheckFormat returned false for properly formatted file")
	}
}

func TestFormatterEdgeCases(t *testing.T) {
	// Create a simple schema for testing
	s := &schema.Schema{
		Name: "test",
		Order: []string{
			"name",
			"version",
			"description",
		},
	}
	
	formatter := NewFormatter(s)
	
	testDir := "../../testdata/edge-cases"
	
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"Empty file", "empty.yml", false},
		{"Only comments", "only-comments.yml", false},
		{"Special characters", "special-characters.yml", false},
		{"Long lines", "long-lines.yml", false},
		{"Deep nesting", "very-deep-nesting.yml", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(testDir, tt.filename))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			
			_, err = formatter.FormatContent(content)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiDocumentFormatting(t *testing.T) {
	// Create a Kubernetes-like schema
	s := &schema.Schema{
		Name: "kubernetes",
		Order: []string{
			"apiVersion",
			"kind",
			"metadata",
			"spec",
			"data",
		},
	}
	
	formatter := NewFormatter(s)
	
	content, err := os.ReadFile("../../testdata/multi-document/kubernetes-multi.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	
	formatted, err := formatter.FormatContent(content)
	if err != nil {
		t.Errorf("Failed to format multi-document YAML: %v", err)
		return
	}
	
	// Verify the output is still valid multi-document YAML
	parser := NewParser(true)
	if !parser.IsMultiDocument(formatted) {
		t.Error("Formatted output is not multi-document YAML")
	}
	
	nodes, err := parser.ParseMultiDocument(formatted)
	if err != nil {
		t.Errorf("Failed to parse formatted multi-document YAML: %v", err)
	}
	
	if len(nodes) != 4 { // kubernetes-multi.yml has 4 documents
		t.Errorf("Expected 4 documents, got %d", len(nodes))
	}
}