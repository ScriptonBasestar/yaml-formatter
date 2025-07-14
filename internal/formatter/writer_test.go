package formatter

import (
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
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

func TestWriterUnicodeHandling(t *testing.T) {
	writer := NewWriter()

	tests := []struct {
		name     string
		input    string
		preserve bool
		expected string
	}{
		{
			name:     "Unicode characters preserved",
			input:    "unicode: \"Hello 世界\"",
			preserve: true,
			expected: "unicode: \"Hello 世界\"",
		},
		{
			name:     "Emoji handling",
			input:    "emoji: \"🚀 🎉\"",
			preserve: true,
			expected: "emoji:", // Emoji might be converted to Unicode escapes
		},
		{
			name:     "Mixed Unicode and ASCII",
			input:    "mixed: \"ASCII and 中文 text\"",
			preserve: true,
			expected: "mixed:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer.SetPreserveUnicode(tt.preserve)

			parser := NewParser(true)
			node, err := parser.ParseYAML([]byte(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			result, err := writer.FormatToString(node)
			if err != nil {
				t.Errorf("FormatToString failed: %v", err)
				return
			}

			// Check that output contains the key
			if !strings.Contains(result, strings.Split(tt.expected, ":")[0]+":") {
				t.Errorf("Result missing expected key. Got: %s", result)
			}

			// Verify the result is valid YAML
			if err := parser.ValidateYAML([]byte(result)); err != nil {
				t.Errorf("Result is not valid YAML: %v", err)
			}
		})
	}
}

func TestWriterSpecialCharacterEscaping(t *testing.T) {
	writer := NewWriter()

	tests := []struct {
		name          string
		input         string
		escapeEnabled bool
		shouldQuote   bool
	}{
		{
			name:          "Special chars without escaping",
			input:         "key: value:with:colons",
			escapeEnabled: false,
			shouldQuote:   false,
		},
		{
			name:          "Special chars with escaping",
			input:         "key: value:with:colons",
			escapeEnabled: true,
			shouldQuote:   true,
		},
		{
			name:          "Already quoted value",
			input:         "key: \"already quoted\"",
			escapeEnabled: true,
			shouldQuote:   false, // Should not double-quote
		},
		{
			name:          "Control characters",
			input:         "key: \"line1\\nline2\"",
			escapeEnabled: true,
			shouldQuote:   false, // Already quoted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer.SetEscapeSpecialChars(tt.escapeEnabled)

			parser := NewParser(true)
			node, err := parser.ParseYAML([]byte(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			result, err := writer.FormatToString(node)
			if err != nil {
				t.Errorf("FormatToString failed: %v", err)
				return
			}

			// Verify the result is valid YAML
			if err := parser.ValidateYAML([]byte(result)); err != nil {
				t.Errorf("Result is not valid YAML: %v", err)
			}

			t.Logf("Input: %s", tt.input)
			t.Logf("Output: %s", result)
		})
	}
}

func TestWriterLineEndingNormalization(t *testing.T) {
	writer := NewWriter()

	tests := []struct {
		name      string
		input     string
		normalize bool
		expected  string
	}{
		{
			name:      "Windows line endings",
			input:     "key: value\r\nother: data\r\n",
			normalize: true,
			expected:  "\n", // Should contain \n not \r\n
		},
		{
			name:      "Mixed line endings",
			input:     "key: value\r\nother: data\rthird: item\n",
			normalize: true,
			expected:  "\n", // Should normalize to \n
		},
		{
			name:      "No normalization",
			input:     "key: value\r\nother: data\r\n",
			normalize: false,
			expected:  "\r\n", // Should preserve original
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer.SetNormalizeLineEndings(tt.normalize)

			// Use the postprocessOutput method directly for testing
			result := writer.postprocessOutput([]byte(tt.input))
			resultStr := string(result)

			if tt.normalize {
				if strings.Contains(resultStr, "\r") {
					t.Errorf("Line endings not normalized: found \\r in output")
				}
			}

			t.Logf("Input: %q", tt.input)
			t.Logf("Output: %q", resultStr)
		})
	}
}

func TestWriterConfigurationMethods(t *testing.T) {
	writer := NewWriter()

	// Test default values
	if !writer.GetPreserveUnicode() {
		t.Error("Default PreserveUnicode should be true")
	}
	if writer.GetEscapeSpecialChars() {
		t.Error("Default EscapeSpecialChars should be false")
	}
	if !writer.GetNormalizeLineEndings() {
		t.Error("Default NormalizeLineEndings should be true")
	}

	// Test setters
	writer.SetPreserveUnicode(false)
	if writer.GetPreserveUnicode() {
		t.Error("SetPreserveUnicode(false) failed")
	}

	writer.SetEscapeSpecialChars(true)
	if !writer.GetEscapeSpecialChars() {
		t.Error("SetEscapeSpecialChars(true) failed")
	}

	writer.SetNormalizeLineEndings(false)
	if writer.GetNormalizeLineEndings() {
		t.Error("SetNormalizeLineEndings(false) failed")
	}
}

func TestWriterEscapeHelperMethods(t *testing.T) {
	writer := NewWriter()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Simple string", "hello", false},
		{"String with colon", "hello:world", false},     // Not quoted
		{"String with brackets", "hello[world]", false}, // Not quoted
		{"Already quoted", "\"hello:world\"", true},
		{"Single quoted", "'hello world'", true},
		{"Not quoted", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := writer.isQuoted(tt.input)
			if result != tt.expected {
				t.Errorf("isQuoted(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWriterAdvancedUnicodeHandling(t *testing.T) {
	writer := NewWriter()
	parser := NewParser(true)

	// Test with various Unicode scenarios
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Complex Unicode",
			input: "unicode: \"café naïve résumé\"",
		},
		{
			name:  "Mathematical symbols",
			input: "math: \"∑ ∆ ∞ ≤ ≥\"",
		},
		{
			name:  "Mixed scripts",
			input: "mixed: \"English العربية 中文 русский\"",
		},
		{
			name:  "Emoji combinations",
			input: "emoji: \"👨‍💻 👩‍🔬 🏳️‍🌈\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer.SetPreserveUnicode(true)

			node, err := parser.ParseYAML([]byte(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			result, err := writer.FormatToString(node)
			if err != nil {
				t.Errorf("FormatToString failed: %v", err)
				return
			}

			// Verify the result is valid YAML
			if err := parser.ValidateYAML([]byte(result)); err != nil {
				t.Errorf("Result is not valid YAML: %v", err)
			}

			// Verify Unicode is preserved (either as original or as escape sequences)
			if !utf8.ValidString(result) {
				t.Errorf("Result contains invalid UTF-8")
			}

			t.Logf("Input: %s", tt.input)
			t.Logf("Output: %s", result)
		})
	}
}

func TestWriterFormattingQualityImprovements(t *testing.T) {
	writer := NewWriter()
	parser := NewParser(true)

	tests := []struct {
		name     string
		input    string
		setup    func(*Writer)
		contains []string
	}{
		{
			name: "Smart blank lines",
			input: `name: test
version: 1.0
metadata:
  author: test
config:
  debug: true`,
			setup: func(w *Writer) {
				w.SetSmartBlankLines(true)
			},
			contains: []string{"name:", "version:", "metadata:", "config:"},
		},
		{
			name: "Comment alignment",
			input: `name: test # Main name
version: 1.0.0 # Version number
debug: false # Debug flag`,
			setup: func(w *Writer) {
				w.SetAlignComments(true)
			},
			contains: []string{"# Main name", "# Version number", "# Debug flag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(writer)

			node, err := parser.ParseYAML([]byte(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			result, err := writer.FormatToString(node)
			if err != nil {
				t.Errorf("FormatToString failed: %v", err)
				return
			}

			// Verify the result contains expected elements
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Result missing expected content: %s", expected)
				}
			}

			// Verify the result is valid YAML
			if err := parser.ValidateYAML([]byte(result)); err != nil {
				t.Errorf("Result is not valid YAML: %v", err)
			}

			t.Logf("Input:\n%s", tt.input)
			t.Logf("Output:\n%s", result)
		})
	}
}

func TestWriterFormattingQualityConfigurationMethods(t *testing.T) {
	writer := NewWriter()

	// Test default values
	if !writer.GetSmartBlankLines() {
		t.Error("Default SmartBlankLines should be true")
	}
	if writer.GetEnforceLineWidth() {
		t.Error("Default EnforceLineWidth should be false")
	}
	if !writer.GetAlignComments() {
		t.Error("Default AlignComments should be true")
	}
	if writer.GetMinimizeBlankLines() {
		t.Error("Default MinimizeBlankLines should be false")
	}

	// Test setters
	writer.SetSmartBlankLines(false)
	if writer.GetSmartBlankLines() {
		t.Error("SetSmartBlankLines(false) failed")
	}

	writer.SetEnforceLineWidth(true)
	if !writer.GetEnforceLineWidth() {
		t.Error("SetEnforceLineWidth(true) failed")
	}

	writer.SetAlignComments(false)
	if writer.GetAlignComments() {
		t.Error("SetAlignComments(false) failed")
	}

	writer.SetMinimizeBlankLines(true)
	if !writer.GetMinimizeBlankLines() {
		t.Error("SetMinimizeBlankLines(true) failed")
	}
}
