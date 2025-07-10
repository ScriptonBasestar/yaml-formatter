package formatter

import (
	"os"
	"path/filepath"
	"strings"
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

func TestFormatterEdgeCaseHandling(t *testing.T) {
	// Create a basic schema for testing
	s := &schema.Schema{
		Name:  "test",
		Order: []string{"name", "version"},
	}

	formatter := NewFormatter(s)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Empty file",
			input:    "",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "Whitespace only",
			input:    "   \n\t  \n  ",
			expected: "   \n\t  \n  ",
			wantErr:  false,
		},
		{
			name:     "Comments only",
			input:    "# This is a comment\n# Another comment\n",
			expected: "# This is a comment\n# Another comment\n",
			wantErr:  false,
		},
		{
			name:     "Single scalar value",
			input:    "hello world",
			expected: "hello world\n",
			wantErr:  false,
		},
		{
			name:     "Single quoted scalar",
			input:    "\"hello world\"",
			expected: "\"hello world\"\n",
			wantErr:  false,
		},
		{
			name:     "Single number",
			input:    "42",
			expected: "42\n",
			wantErr:  false,
		},
		{
			name:     "Single boolean",
			input:    "true",
			expected: "true\n",
			wantErr:  false,
		},
		{
			name:     "Mixed comments and whitespace",
			input:    "  # Comment\n\n  # Another comment  \n",
			expected: "  # Comment\n\n  # Another comment  \n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.FormatContent([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(result) != tt.expected {
				t.Errorf("FormatContent() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestFormatterHelperMethods(t *testing.T) {
	s := &schema.Schema{Name: "test", Order: []string{"name"}}
	formatter := NewFormatter(s)

	tests := []struct {
		name   string
		method string
		input  string
		want   bool
	}{
		{"Empty content", "isWhitespaceOnly", "", false},
		{"Whitespace only", "isWhitespaceOnly", "   \n\t  ", true},
		{"Mixed content", "isWhitespaceOnly", "hello   ", false},

		{"Comments only", "isCommentsOnly", "# Comment\n# Another", true},
		{"Mixed comments", "isCommentsOnly", "# Comment\nkey: value", false},
		{"Empty file", "isCommentsOnly", "", false},

		{"Single scalar", "isSingleScalar", "hello", true},
		{"YAML mapping", "isSingleScalar", "key: value", false},
		{"YAML sequence", "isSingleScalar", "- item", false},
		{"Comment line", "isSingleScalar", "# comment", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			switch tt.method {
			case "isWhitespaceOnly":
				result = formatter.isWhitespaceOnly([]byte(tt.input))
			case "isCommentsOnly":
				result = formatter.isCommentsOnly([]byte(tt.input))
			case "isSingleScalar":
				result = formatter.isSingleScalar([]byte(tt.input))
			default:
				t.Fatalf("Unknown method: %s", tt.method)
			}

			if result != tt.want {
				t.Errorf("%s(%q) = %v, want %v", tt.method, tt.input, result, tt.want)
			}
		})
	}
}

func TestFormatterSpecialCharacterHandling(t *testing.T) {
	s := &schema.Schema{
		Name:  "test",
		Order: []string{"unicode", "emoji", "escapes"},
	}
	formatter := NewFormatter(s)

	// Test with the existing special characters file
	content, err := os.ReadFile("../../testdata/edge-cases/special-characters.yml")
	if err != nil {
		t.Fatalf("Failed to read special characters test file: %v", err)
	}

	result, err := formatter.FormatContent(content)
	if err != nil {
		t.Errorf("Failed to format special characters YAML: %v", err)
		return
	}

	// Verify the result contains expected Unicode and special characters
	resultStr := string(result)

	expectedChars := []string{
		"世界",  // Unicode characters
		"\\n", // Escape sequences
		"\\t", // Tab escape
		"\"",  // Quotes
	}

	// Check for emojis (either direct or as Unicode escapes)
	emojiTests := []struct {
		emoji  string
		escape string
	}{
		{"🌍", "\\U0001F30D"}, // Earth emoji
		{"🚀", "\\U0001F680"}, // Rocket emoji
	}

	for _, expected := range expectedChars {
		if !strings.Contains(resultStr, expected) {
			t.Errorf("Formatted output missing expected character/sequence: %s", expected)
		}
	}

	for _, emojiTest := range emojiTests {
		if !strings.Contains(resultStr, emojiTest.emoji) && !strings.Contains(resultStr, emojiTest.escape) {
			t.Errorf("Formatted output missing emoji %s (or its Unicode escape %s)", emojiTest.emoji, emojiTest.escape)
		}
	}

	// Ensure the result is valid YAML
	parser := NewParser(true)
	if err := parser.ValidateYAML(result); err != nil {
		t.Errorf("Formatted special characters YAML is invalid: %v", err)
	}
}
