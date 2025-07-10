package formatter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing YAML files while preserving structure and comments
type Parser struct {
	preserveComments bool
}

// NewParser creates a new YAML parser
func NewParser(preserveComments bool) *Parser {
	return &Parser{
		preserveComments: preserveComments,
	}
}

// ParseYAML parses YAML content and returns the root node
func (p *Parser) ParseYAML(content []byte) (*yaml.Node, error) {
	var node yaml.Node

	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &node, nil
}

// ParseMultiDocument parses YAML content that may contain multiple documents
func (p *Parser) ParseMultiDocument(content []byte) ([]*yaml.Node, error) {
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))

	var documents []*yaml.Node
	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to parse YAML document: %w", err)
		}
		documents = append(documents, &node)
	}

	return documents, nil
}

// IsMultiDocument checks if the YAML content contains multiple documents
func (p *Parser) IsMultiDocument(content []byte) bool {
	return strings.Contains(string(content), "\n---\n") || strings.HasPrefix(string(content), "---\n")
}

// GetNodeAtPath traverses the YAML node tree to find a node at a specific path
func (p *Parser) GetNodeAtPath(root *yaml.Node, path string) *yaml.Node {
	if path == "" {
		return root
	}

	parts := strings.Split(path, ".")
	current := root

	// Skip document node if present
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		current = p.findChildNode(current, part)
		if current == nil {
			return nil
		}
	}

	return current
}

// findChildNode finds a child node with the given key
func (p *Parser) findChildNode(parent *yaml.Node, key string) *yaml.Node {
	if parent.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(parent.Content); i += 2 {
		keyNode := parent.Content[i]
		valueNode := parent.Content[i+1]

		if keyNode.Value == key {
			return valueNode
		}
	}

	return nil
}

// GetKeys extracts all keys from a mapping node
func (p *Parser) GetKeys(node *yaml.Node) []string {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	var keys []string
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		keys = append(keys, keyNode.Value)
	}

	return keys
}

// CloneNode creates a deep copy of a YAML node
func (p *Parser) CloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	clone := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		Alias:       node.Alias,
		Content:     make([]*yaml.Node, len(node.Content)),
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}

	for i, child := range node.Content {
		clone.Content[i] = p.CloneNode(child)
	}

	return clone
}

// ValidateYAML checks if the YAML content is valid
func (p *Parser) ValidateYAML(content []byte) error {
	var temp interface{}
	if err := yaml.Unmarshal(content, &temp); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	return nil
}

// GetNodeType returns a human-readable description of the node type
func (p *Parser) GetNodeType(node *yaml.Node) string {
	switch node.Kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return "unknown"
	}
}

// HasComments checks if a node has any comments
func (p *Parser) HasComments(node *yaml.Node) bool {
	return node.HeadComment != "" || node.LineComment != "" || node.FootComment != ""
}

// PreserveComments returns whether comments should be preserved
func (p *Parser) PreserveComments() bool {
	return p.preserveComments
}

// SetPreserveComments sets whether comments should be preserved
func (p *Parser) SetPreserveComments(preserve bool) {
	p.preserveComments = preserve
}
