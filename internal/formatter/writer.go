package formatter

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Writer handles writing formatted YAML content
type Writer struct {
	indent           int
	lineWidth        int
	preserveComments bool
}

// NewWriter creates a new YAML writer
func NewWriter() *Writer {
	return &Writer{
		indent:           2,
		lineWidth:        80,
		preserveComments: true,
	}
}

// SetIndent sets the indentation size
func (w *Writer) SetIndent(indent int) *Writer {
	w.indent = indent
	return w
}

// SetLineWidth sets the maximum line width
func (w *Writer) SetLineWidth(width int) *Writer {
	w.lineWidth = width
	return w
}

// SetPreserveComments sets whether to preserve comments
func (w *Writer) SetPreserveComments(preserve bool) *Writer {
	w.preserveComments = preserve
	return w
}

// WriteNode writes a single YAML node to the provided writer
func (w *Writer) WriteNode(writer io.Writer, node *yaml.Node) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()

	// Configure encoder options
	encoder.SetIndent(w.indent)

	if err := encoder.Encode(node); err != nil {
		return fmt.Errorf("failed to encode YAML node: %w", err)
	}

	return nil
}

// WriteNodes writes multiple YAML nodes (documents) to the provided writer
func (w *Writer) WriteNodes(writer io.Writer, nodes []*yaml.Node) error {
	for i, node := range nodes {
		if i > 0 {
			// Add document separator for multiple documents
			if _, err := writer.Write([]byte("---\n")); err != nil {
				return fmt.Errorf("failed to write document separator: %w", err)
			}
		}

		if err := w.WriteNode(writer, node); err != nil {
			return fmt.Errorf("failed to write document %d: %w", i, err)
		}
	}

	return nil
}

// FormatToString formats a YAML node and returns it as a string
func (w *Writer) FormatToString(node *yaml.Node) (string, error) {
	var buf bytes.Buffer

	if err := w.WriteNode(&buf, node); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// FormatNodesToString formats multiple YAML nodes and returns them as a string
func (w *Writer) FormatNodesToString(nodes []*yaml.Node) (string, error) {
	var buf bytes.Buffer

	if err := w.WriteNodes(&buf, nodes); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// FormatBytes formats YAML content provided as bytes
func (w *Writer) FormatBytes(content []byte) ([]byte, error) {
	parser := NewParser(w.preserveComments)

	// Check if it's a multi-document YAML
	if parser.IsMultiDocument(content) {
		nodes, err := parser.ParseMultiDocument(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse multi-document YAML: %w", err)
		}

		result, err := w.FormatNodesToString(nodes)
		if err != nil {
			return nil, err
		}

		return []byte(result), nil
	} else {
		node, err := parser.ParseYAML(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		result, err := w.FormatToString(node)
		if err != nil {
			return nil, err
		}

		return []byte(result), nil
	}
}

// CompareFormatted compares original content with formatted content
func (w *Writer) CompareFormatted(original, formatted []byte) bool {
	// Normalize whitespace for comparison
	normalizedOriginal := w.normalizeYAML(string(original))
	normalizedFormatted := w.normalizeYAML(string(formatted))

	return normalizedOriginal == normalizedFormatted
}

// normalizeYAML normalizes YAML content for comparison
func (w *Writer) normalizeYAML(content string) string {
	lines := strings.Split(content, "\n")
	var normalized []string

	for _, line := range lines {
		// Trim trailing whitespace but preserve structure
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != "" || len(normalized) == 0 {
			normalized = append(normalized, trimmed)
		}
	}

	// Remove trailing empty lines
	for len(normalized) > 0 && normalized[len(normalized)-1] == "" {
		normalized = normalized[:len(normalized)-1]
	}

	return strings.Join(normalized, "\n")
}

// ValidateFormattedOutput validates that the formatted output is still valid YAML
func (w *Writer) ValidateFormattedOutput(content []byte) error {
	var temp interface{}
	if err := yaml.Unmarshal(content, &temp); err != nil {
		return fmt.Errorf("formatted output is not valid YAML: %w", err)
	}
	return nil
}

// PreserveComments returns whether comments are being preserved
func (w *Writer) PreserveComments() bool {
	return w.preserveComments
}

// GetIndent returns the current indentation size
func (w *Writer) GetIndent() int {
	return w.indent
}

// GetLineWidth returns the current line width
func (w *Writer) GetLineWidth() int {
	return w.lineWidth
}

// WriteToFile writes formatted content to a file path using the provided filesystem
func (w *Writer) WriteToFile(content []byte, filePath string) error {
	// This is a placeholder - in practice, this would use afero.Fs
	// For now, we'll leave this as a simple interface
	return fmt.Errorf("WriteToFile not implemented - use external file operations")
}

// CalculateStats calculates statistics about the formatting changes
func (w *Writer) CalculateStats(original, formatted []byte) *FormatStats {
	originalLines := strings.Split(string(original), "\n")
	formattedLines := strings.Split(string(formatted), "\n")

	stats := &FormatStats{
		OriginalLines:  len(originalLines),
		FormattedLines: len(formattedLines),
		OriginalBytes:  len(original),
		FormattedBytes: len(formatted),
		Changed:        !bytes.Equal(original, formatted),
	}

	// Calculate line differences
	stats.LinesChanged = w.countChangedLines(originalLines, formattedLines)

	return stats
}

// countChangedLines counts how many lines were changed
func (w *Writer) countChangedLines(original, formatted []string) int {
	maxLen := len(original)
	if len(formatted) > maxLen {
		maxLen = len(formatted)
	}

	changed := 0
	for i := 0; i < maxLen; i++ {
		origLine := ""
		formattedLine := ""

		if i < len(original) {
			origLine = original[i]
		}
		if i < len(formatted) {
			formattedLine = formatted[i]
		}

		if origLine != formattedLine {
			changed++
		}
	}

	return changed
}

// FormatStats contains statistics about formatting changes
type FormatStats struct {
	OriginalLines  int
	FormattedLines int
	OriginalBytes  int
	FormattedBytes int
	LinesChanged   int
	Changed        bool
}

// String returns a string representation of the format statistics
func (fs *FormatStats) String() string {
	if !fs.Changed {
		return "No changes needed"
	}

	return fmt.Sprintf("Lines: %d→%d, Bytes: %d→%d, Changed: %d lines",
		fs.OriginalLines, fs.FormattedLines,
		fs.OriginalBytes, fs.FormattedBytes,
		fs.LinesChanged)
}
