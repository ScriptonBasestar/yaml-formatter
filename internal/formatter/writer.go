package formatter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Writer handles writing formatted YAML content
type Writer struct {
	indent               int
	lineWidth            int
	preserveComments     bool
	preserveUnicode      bool
	escapeSpecialChars   bool
	normalizeLineEndings bool
}

// NewWriter creates a new YAML writer
func NewWriter() *Writer {
	return &Writer{
		indent:               2,
		lineWidth:            80,
		preserveComments:     true,
		preserveUnicode:      true,
		escapeSpecialChars:   false,
		normalizeLineEndings: true,
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

// SetPreserveUnicode sets whether to preserve Unicode characters
func (w *Writer) SetPreserveUnicode(preserve bool) *Writer {
	w.preserveUnicode = preserve
	return w
}

// SetEscapeSpecialChars sets whether to escape special characters
func (w *Writer) SetEscapeSpecialChars(escape bool) *Writer {
	w.escapeSpecialChars = escape
	return w
}

// SetNormalizeLineEndings sets whether to normalize line endings
func (w *Writer) SetNormalizeLineEndings(normalize bool) *Writer {
	w.normalizeLineEndings = normalize
	return w
}

// WriteNode writes a single YAML node to the provided writer
func (w *Writer) WriteNode(writer io.Writer, node *yaml.Node) error {
	// Pre-process the node for special character handling
	processedNode := w.preprocessNode(node)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	defer encoder.Close()

	// Configure encoder options
	encoder.SetIndent(w.indent)

	if err := encoder.Encode(processedNode); err != nil {
		return fmt.Errorf("failed to encode YAML node: %w", err)
	}

	// Post-process the output for special character handling
	output := w.postprocessOutput(buf.Bytes())

	if _, err := writer.Write(output); err != nil {
		return fmt.Errorf("failed to write processed output: %w", err)
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

// preprocessNode processes a YAML node to handle special characters before encoding
func (w *Writer) preprocessNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	// Create a deep copy of the node
	processed := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       w.preprocessValue(node.Value),
		Anchor:      node.Anchor,
		Alias:       node.Alias,
		Content:     make([]*yaml.Node, len(node.Content)),
		HeadComment: w.preprocessComment(node.HeadComment),
		LineComment: w.preprocessComment(node.LineComment),
		FootComment: w.preprocessComment(node.FootComment),
		Line:        node.Line,
		Column:      node.Column,
	}

	// Recursively process child nodes
	for i, child := range node.Content {
		processed.Content[i] = w.preprocessNode(child)
	}

	return processed
}

// preprocessValue handles special character processing for YAML values
func (w *Writer) preprocessValue(value string) string {
	if value == "" {
		return value
	}

	// Handle Unicode preservation
	if w.preserveUnicode {
		// Ensure proper UTF-8 encoding
		if !utf8.ValidString(value) {
			// Convert invalid UTF-8 to replacement characters
			value = strings.ToValidUTF8(value, "�")
		}
	}

	// Handle special character escaping if enabled
	if w.escapeSpecialChars {
		value = w.escapeYAMLSpecialChars(value)
	}

	return value
}

// preprocessComment handles special character processing for YAML comments
func (w *Writer) preprocessComment(comment string) string {
	if comment == "" {
		return comment
	}

	// Ensure comments are valid UTF-8
	if !utf8.ValidString(comment) {
		comment = strings.ToValidUTF8(comment, "�")
	}

	return comment
}

// escapeYAMLSpecialChars escapes special characters in YAML values
func (w *Writer) escapeYAMLSpecialChars(value string) string {
	// Define characters that might need special handling in YAML
	needsQuoting := false

	// Check for characters that require quoting
	for _, r := range value {
		if r == ':' || r == '{' || r == '}' || r == '[' || r == ']' ||
			r == ',' || r == '#' || r == '&' || r == '*' || r == '!' ||
			r == '|' || r == '>' || r == '\'' || r == '"' ||
			r == '%' || r == '@' || r == '`' {
			needsQuoting = true
			break
		}

		// Check for control characters
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			needsQuoting = true
			break
		}
	}

	// If the value needs quoting and doesn't already have it
	if needsQuoting && !w.isQuoted(value) {
		// Use double quotes and escape internal quotes
		escaped := strings.ReplaceAll(value, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return "\"" + escaped + "\""
	}

	return value
}

// isQuoted checks if a string is already quoted
func (w *Writer) isQuoted(value string) bool {
	if len(value) < 2 {
		return false
	}

	return (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'"))
}

// postprocessOutput handles post-processing of the encoded YAML output
func (w *Writer) postprocessOutput(content []byte) []byte {
	output := string(content)

	// Normalize line endings if enabled
	if w.normalizeLineEndings {
		output = w.doNormalizeLineEndings(output)
	}

	// Enhance Unicode handling
	output = w.enhanceUnicodeOutput(output)

	// Handle emoji preservation
	output = w.preserveEmojis(output)

	return []byte(output)
}

// doNormalizeLineEndings normalizes line endings to \n
func (w *Writer) doNormalizeLineEndings(content string) string {
	// Replace Windows line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Replace old Mac line endings
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// enhanceUnicodeOutput enhances Unicode character handling in output
func (w *Writer) enhanceUnicodeOutput(content string) string {
	if !w.preserveUnicode {
		return content
	}

	// Ensure the content is valid UTF-8
	if !utf8.ValidString(content) {
		content = strings.ToValidUTF8(content, "�")
	}

	return content
}

// preserveEmojis ensures emojis are properly preserved in the output
func (w *Writer) preserveEmojis(content string) string {
	// Convert Unicode escape sequences back to actual emoji characters if desired
	if w.preserveUnicode {
		// This is a basic implementation - in practice, you might want more sophisticated handling
		// For now, we'll leave Unicode escapes as-is since they're valid YAML
		return content
	}

	return content
}

// GetPreserveUnicode returns whether Unicode preservation is enabled
func (w *Writer) GetPreserveUnicode() bool {
	return w.preserveUnicode
}

// GetEscapeSpecialChars returns whether special character escaping is enabled
func (w *Writer) GetEscapeSpecialChars() bool {
	return w.escapeSpecialChars
}

// GetNormalizeLineEndings returns whether line ending normalization is enabled
func (w *Writer) GetNormalizeLineEndings() bool {
	return w.normalizeLineEndings
}
