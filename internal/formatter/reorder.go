package formatter

import (
	"fmt"
	"sort"

	"yaml-formatter/internal/schema"
	"gopkg.in/yaml.v3"
)

// Reorderer handles reordering YAML nodes according to a schema
type Reorderer struct {
	schema *schema.Schema
	parser *Parser
}

// NewReorderer creates a new YAML reorderer
func NewReorderer(s *schema.Schema, p *Parser) *Reorderer {
	return &Reorderer{
		schema: s,
		parser: p,
	}
}

// ReorderNode reorders a YAML node according to the schema
func (r *Reorderer) ReorderNode(node *yaml.Node, path string) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}
	
	// Skip document nodes
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return r.ReorderNode(node.Content[0], path)
	}
	
	switch node.Kind {
	case yaml.MappingNode:
		return r.reorderMappingNode(node, path)
	case yaml.SequenceNode:
		return r.reorderSequenceNode(node, path)
	default:
		// Scalar nodes don't need reordering
		return nil
	}
}

// reorderMappingNode reorders the keys in a mapping node
func (r *Reorderer) reorderMappingNode(node *yaml.Node, path string) error {
	if len(node.Content)%2 != 0 {
		return fmt.Errorf("mapping node has odd number of children")
	}
	
	// Get the key order from schema
	keyOrder := r.schema.GetKeyOrder(path)
	if len(keyOrder) == 0 {
		// No specific order defined, keep existing order but still process children
		return r.processChildren(node, path)
	}
	
	// Create a map of key-value pairs for easier manipulation
	pairs := make(map[string]*KeyValuePair)
	var existingKeys []string
	
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		
		pair := &KeyValuePair{
			Key:   keyNode,
			Value: valueNode,
		}
		
		pairs[keyNode.Value] = pair
		existingKeys = append(existingKeys, keyNode.Value)
	}
	
	// Create new content array with reordered keys
	var newContent []*yaml.Node
	var processedKeys []string
	
	// First, add keys in schema order
	for _, key := range keyOrder {
		if pair, exists := pairs[key]; exists {
			newContent = append(newContent, pair.Key, pair.Value)
			processedKeys = append(processedKeys, key)
		}
	}
	
	// Then add any remaining keys that weren't in the schema
	for _, key := range existingKeys {
		if !contains(processedKeys, key) {
			pair := pairs[key]
			newContent = append(newContent, pair.Key, pair.Value)
			processedKeys = append(processedKeys, key)
		}
	}
	
	// Update the node's content
	node.Content = newContent
	
	// Recursively process child nodes
	return r.processChildren(node, path)
}

// reorderSequenceNode processes sequence nodes (arrays)
func (r *Reorderer) reorderSequenceNode(node *yaml.Node, path string) error {
	// For sequences, we don't reorder the items themselves,
	// but we might need to reorder keys within each item if they are mappings
	for i, child := range node.Content {
		childPath := fmt.Sprintf("%s[%d]", path, i)
		if err := r.ReorderNode(child, childPath); err != nil {
			return fmt.Errorf("failed to reorder sequence item %d: %w", i, err)
		}
	}
	
	return nil
}

// processChildren recursively processes child nodes
func (r *Reorderer) processChildren(node *yaml.Node, path string) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		
		// Check if this key should not be sorted
		if r.schema.IsNonSortKey(keyNode.Value) {
			continue
		}
		
		// Build child path
		childPath := path
		if childPath != "" {
			childPath += "."
		}
		childPath += keyNode.Value
		
		// Recursively process the value node
		if err := r.ReorderNode(valueNode, childPath); err != nil {
			return fmt.Errorf("failed to reorder child %s: %w", keyNode.Value, err)
		}
	}
	
	return nil
}

// KeyValuePair represents a key-value pair in a YAML mapping
type KeyValuePair struct {
	Key   *yaml.Node
	Value *yaml.Node
}

// SortBySchema sorts a slice of key-value pairs according to the schema order
func (r *Reorderer) SortBySchema(pairs []*KeyValuePair, keyOrder []string) {
	sort.Slice(pairs, func(i, j int) bool {
		keyI := pairs[i].Key.Value
		keyJ := pairs[j].Key.Value
		
		indexI := indexOf(keyOrder, keyI)
		indexJ := indexOf(keyOrder, keyJ)
		
		// If both keys are in the schema order
		if indexI != -1 && indexJ != -1 {
			return indexI < indexJ
		}
		
		// If only one key is in the schema order, prioritize it
		if indexI != -1 {
			return true
		}
		if indexJ != -1 {
			return false
		}
		
		// If neither key is in the schema order, maintain lexicographic order
		return keyI < keyJ
	})
}

// CheckOrder verifies if a node is already properly ordered according to the schema
func (r *Reorderer) CheckOrder(node *yaml.Node, path string) (bool, error) {
	if node == nil {
		return true, nil
	}
	
	// Skip document nodes
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return r.CheckOrder(node.Content[0], path)
	}
	
	switch node.Kind {
	case yaml.MappingNode:
		return r.checkMappingOrder(node, path)
	case yaml.SequenceNode:
		return r.checkSequenceOrder(node, path)
	default:
		return true, nil
	}
}

// checkMappingOrder checks if a mapping node is properly ordered
func (r *Reorderer) checkMappingOrder(node *yaml.Node, path string) (bool, error) {
	if len(node.Content)%2 != 0 {
		return false, fmt.Errorf("mapping node has odd number of children")
	}
	
	keyOrder := r.schema.GetKeyOrder(path)
	if len(keyOrder) == 0 {
		// No specific order defined, check children
		return r.checkChildrenOrder(node, path)
	}
	
	// Get current key order
	var currentKeys []string
	for i := 0; i < len(node.Content); i += 2 {
		currentKeys = append(currentKeys, node.Content[i].Value)
	}
	
	// Check if current order matches expected order
	expectedOrder := r.buildExpectedOrder(currentKeys, keyOrder)
	
	for i, key := range currentKeys {
		if i >= len(expectedOrder) || key != expectedOrder[i] {
			return false, nil
		}
	}
	
	// Check children recursively
	return r.checkChildrenOrder(node, path)
}

// checkSequenceOrder checks if a sequence node is properly ordered
func (r *Reorderer) checkSequenceOrder(node *yaml.Node, path string) (bool, error) {
	for i, child := range node.Content {
		childPath := fmt.Sprintf("%s[%d]", path, i)
		if ordered, err := r.CheckOrder(child, childPath); err != nil {
			return false, err
		} else if !ordered {
			return false, nil
		}
	}
	
	return true, nil
}

// checkChildrenOrder recursively checks child node order
func (r *Reorderer) checkChildrenOrder(node *yaml.Node, path string) (bool, error) {
	if node.Kind != yaml.MappingNode {
		return true, nil
	}
	
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		
		if r.schema.IsNonSortKey(keyNode.Value) {
			continue
		}
		
		childPath := path
		if childPath != "" {
			childPath += "."
		}
		childPath += keyNode.Value
		
		if ordered, err := r.CheckOrder(valueNode, childPath); err != nil {
			return false, err
		} else if !ordered {
			return false, nil
		}
	}
	
	return true, nil
}

// buildExpectedOrder builds the expected key order based on schema and existing keys
func (r *Reorderer) buildExpectedOrder(currentKeys, schemaOrder []string) []string {
	var expected []string
	processed := make(map[string]bool)
	
	// Add keys in schema order
	for _, key := range schemaOrder {
		if contains(currentKeys, key) {
			expected = append(expected, key)
			processed[key] = true
		}
	}
	
	// Add remaining keys that weren't in schema
	for _, key := range currentKeys {
		if !processed[key] {
			expected = append(expected, key)
		}
	}
	
	return expected
}

// Helper functions

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// indexOf returns the index of an item in a slice, or -1 if not found
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}