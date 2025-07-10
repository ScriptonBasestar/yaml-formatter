package formatter

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"

	"yaml-formatter/internal/schema"
)

// Formatter provides high-level YAML formatting functionality
type Formatter struct {
	parser    *Parser
	reorderer *Reorderer
	writer    *Writer
	schema    *schema.Schema
}

// NewFormatter creates a new YAML formatter with the given schema
func NewFormatter(s *schema.Schema) *Formatter {
	parser := NewParser(true) // Preserve comments by default
	writer := NewWriter()
	reorderer := NewReorderer(s, parser)

	return &Formatter{
		parser:    parser,
		reorderer: reorderer,
		writer:    writer,
		schema:    s,
	}
}

// handleEdgeCases handles special edge cases that don't require full parsing
func (f *Formatter) handleEdgeCases(content []byte) ([]byte, bool) {
	// Handle empty files
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return content, true
	}

	// Handle whitespace-only files (preserve original whitespace)
	if f.isWhitespaceOnly(content) {
		return content, true
	}

	// Handle comments-only files
	if f.isCommentsOnly(content) {
		return content, true
	}

	// Handle single scalar value files
	if f.isSingleScalar(content) {
		return f.formatSingleScalar(content), true
	}

	return nil, false
}

// isWhitespaceOnly checks if content contains only whitespace characters
func (f *Formatter) isWhitespaceOnly(content []byte) bool {
	for _, b := range content {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return len(content) > 0
}

// isCommentsOnly checks if content contains only YAML comments
func (f *Formatter) isCommentsOnly(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	hasNonEmptyLine := false

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		hasNonEmptyLine = true

		// If line doesn't start with #, it's not a comment-only file
		if line[0] != '#' {
			return false
		}
	}

	return hasNonEmptyLine && scanner.Err() == nil
}

// isSingleScalar checks if content contains only a single scalar value
func (f *Formatter) isSingleScalar(content []byte) bool {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}

	// Simple heuristic: if it doesn't contain YAML structure characters
	// and doesn't start with comment, it might be a single scalar
	if !bytes.Contains(trimmed, []byte(":")) &&
		!bytes.Contains(trimmed, []byte("-")) &&
		!bytes.Contains(trimmed, []byte("[")) &&
		!bytes.Contains(trimmed, []byte("{")) &&
		!bytes.HasPrefix(trimmed, []byte("#")) &&
		!bytes.Contains(trimmed, []byte("\n---")) {

		// Additional check: try to parse as single value
		return f.validateSingleScalar(content)
	}

	return false
}

// validateSingleScalar validates that content is a valid single scalar value
func (f *Formatter) validateSingleScalar(content []byte) bool {
	// Use regex to match simple scalar patterns
	scalarPattern := regexp.MustCompile(`^[\s]*[^:\-\[\{\#\n][^\n]*[\s]*$`)
	return scalarPattern.Match(content)
}

// formatSingleScalar formats a single scalar value
func (f *Formatter) formatSingleScalar(content []byte) []byte {
	// For single scalars, just ensure consistent whitespace
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return content
	}

	// Add newline if not present
	if !bytes.HasSuffix(trimmed, []byte("\n")) {
		return append(trimmed, '\n')
	}

	return trimmed
}

// FormatContent formats YAML content according to the schema
func (f *Formatter) FormatContent(content []byte) ([]byte, error) {
	// Handle edge cases first
	if result, handled := f.handleEdgeCases(content); handled {
		return result, nil
	}

	// Validate input
	if err := f.parser.ValidateYAML(content); err != nil {
		return nil, fmt.Errorf("invalid input YAML: %w", err)
	}

	// Handle multi-document YAML
	if f.parser.IsMultiDocument(content) {
		return f.formatMultiDocument(content)
	}

	// Handle single document
	return f.formatSingleDocument(content)
}

// formatSingleDocument formats a single YAML document
func (f *Formatter) formatSingleDocument(content []byte) ([]byte, error) {
	// Parse the YAML
	node, err := f.parser.ParseYAML(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Reorder according to schema
	if err := f.reorderer.ReorderNode(node, ""); err != nil {
		return nil, fmt.Errorf("failed to reorder YAML: %w", err)
	}

	// Format and return
	formatted, err := f.writer.FormatToString(node)
	if err != nil {
		return nil, fmt.Errorf("failed to format YAML: %w", err)
	}

	formattedBytes := []byte(formatted)

	// Validate output
	if err := f.writer.ValidateFormattedOutput(formattedBytes); err != nil {
		return nil, fmt.Errorf("formatted output validation failed: %w", err)
	}

	return formattedBytes, nil
}

// formatMultiDocument formats multiple YAML documents
func (f *Formatter) formatMultiDocument(content []byte) ([]byte, error) {
	// Parse all documents
	nodes, err := f.parser.ParseMultiDocument(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multi-document YAML: %w", err)
	}

	// Reorder each document
	for i, node := range nodes {
		if err := f.reorderer.ReorderNode(node, ""); err != nil {
			return nil, fmt.Errorf("failed to reorder document %d: %w", i, err)
		}
	}

	// Format all documents
	formatted, err := f.writer.FormatNodesToString(nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to format multi-document YAML: %w", err)
	}

	formattedBytes := []byte(formatted)

	// Validate each document in the output
	outputNodes, err := f.parser.ParseMultiDocument(formattedBytes)
	if err != nil {
		return nil, fmt.Errorf("formatted multi-document output validation failed: %w", err)
	}

	if len(outputNodes) != len(nodes) {
		return nil, fmt.Errorf("document count mismatch after formatting: expected %d, got %d", len(nodes), len(outputNodes))
	}

	return formattedBytes, nil
}

// CheckFormat checks if the content is already properly formatted
func (f *Formatter) CheckFormat(content []byte) (bool, error) {
	// Validate input
	if err := f.parser.ValidateYAML(content); err != nil {
		return false, fmt.Errorf("invalid input YAML: %w", err)
	}

	// Handle multi-document YAML
	if f.parser.IsMultiDocument(content) {
		return f.checkMultiDocumentFormat(content)
	}

	// Handle single document
	return f.checkSingleDocumentFormat(content)
}

// checkSingleDocumentFormat checks if a single document is properly formatted
func (f *Formatter) checkSingleDocumentFormat(content []byte) (bool, error) {
	node, err := f.parser.ParseYAML(content)
	if err != nil {
		return false, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return f.reorderer.CheckOrder(node, "")
}

// checkMultiDocumentFormat checks if multiple documents are properly formatted
func (f *Formatter) checkMultiDocumentFormat(content []byte) (bool, error) {
	nodes, err := f.parser.ParseMultiDocument(content)
	if err != nil {
		return false, fmt.Errorf("failed to parse multi-document YAML: %w", err)
	}

	for i, node := range nodes {
		if ordered, err := f.reorderer.CheckOrder(node, ""); err != nil {
			return false, fmt.Errorf("failed to check order for document %d: %w", i, err)
		} else if !ordered {
			return false, nil
		}
	}

	return true, nil
}

// GetStats returns formatting statistics for the given content
func (f *Formatter) GetStats(original []byte) (*FormatStats, error) {
	formatted, err := f.FormatContent(original)
	if err != nil {
		return nil, fmt.Errorf("failed to format content for stats: %w", err)
	}

	return f.writer.CalculateStats(original, formatted), nil
}

// SetPreserveComments sets whether comments should be preserved
func (f *Formatter) SetPreserveComments(preserve bool) {
	f.parser.SetPreserveComments(preserve)
	f.writer.SetPreserveComments(preserve)
}

// SetIndent sets the indentation size for output
func (f *Formatter) SetIndent(indent int) {
	f.writer.SetIndent(indent)
}

// SetLineWidth sets the maximum line width for output
func (f *Formatter) SetLineWidth(width int) {
	f.writer.SetLineWidth(width)
}

// GetSchema returns the current schema
func (f *Formatter) GetSchema() *schema.Schema {
	return f.schema
}

// SetSchema updates the schema and recreates the reorderer
func (f *Formatter) SetSchema(s *schema.Schema) {
	f.schema = s
	f.reorderer = NewReorderer(s, f.parser)
}

// ValidateSchema validates that the current schema is valid
func (f *Formatter) ValidateSchema() error {
	if f.schema == nil {
		return fmt.Errorf("no schema set")
	}

	return f.schema.Validate()
}

// GenerateSchemaFromContent generates a schema from the provided YAML content
func (f *Formatter) GenerateSchemaFromContent(content []byte, name string) (*schema.Schema, error) {
	return schema.GenerateFromYAML(content, name)
}

// Clone creates a copy of the formatter with the same configuration
func (f *Formatter) Clone() *Formatter {
	newFormatter := NewFormatter(f.schema)
	newFormatter.SetPreserveComments(f.parser.PreserveComments())
	newFormatter.SetIndent(f.writer.GetIndent())
	newFormatter.SetLineWidth(f.writer.GetLineWidth())
	return newFormatter
}
