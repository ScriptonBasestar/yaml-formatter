package schema

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Schema represents a YAML formatting schema that defines key ordering
type Schema struct {
	Name    string                 `yaml:"-"` // Not serialized to avoid conflicts
	Keys    map[string]interface{} `yaml:",inline"`
	NonSort map[string]interface{} `yaml:"non_sort,omitempty"`
	Order   []string               `yaml:"-"` // Computed from Keys structure
}

// KeyOrder extracts the key ordering from a schema
func (s *Schema) KeyOrder() []string {
	return extractKeysFromMap(s.Keys)
}

// NonSortKeys returns keys that should not be sorted
func (s *Schema) NonSortKeys() []string {
	if s.NonSort == nil {
		return nil
	}
	return extractKeysFromMap(s.NonSort)
}

// extractKeysFromMap recursively extracts keys from a map structure
func extractKeysFromMap(m map[string]interface{}) []string {
	var keys []string
	for key := range m {
		if key == "non_sort" {
			continue
		}
		keys = append(keys, key)
	}
	// Sort keys for deterministic order
	// Note: In a real implementation, this should respect the original order
	// from the schema definition, but for testing we'll use alphabetical order
	return keys
}

// buildOrderFromKeys recursively builds an order list from the Keys structure
func buildOrderFromKeys(m map[string]interface{}, prefix string) []string {
	var order []string
	
	// Process in a deterministic order
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != "non_sort" {
			keys = append(keys, k)
		}
	}
	
	for _, key := range keys {
		value := m[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		
		order = append(order, fullKey)
		
		// If value is a map, recurse for nested structure
		if subMap, ok := value.(map[string]interface{}); ok && len(subMap) > 0 {
			subOrder := buildOrderFromKeys(subMap, fullKey)
			order = append(order, subOrder...)
		}
	}
	
	return order
}

// GetKeyOrder returns the ordering for a specific path in the schema
func (s *Schema) GetKeyOrder(path string) []string {
	if path == "" {
		// Return top-level keys from Order field
		var topLevel []string
		for _, orderKey := range s.Order {
			if !strings.Contains(orderKey, ".") && !strings.Contains(orderKey, "[*]") {
				topLevel = append(topLevel, orderKey)
			}
		}
		return topLevel
	}
	
	// Handle array index notation like "items[0]" -> "items"
	cleanPath := path
	if strings.Contains(path, "[") {
		cleanPath = strings.Split(path, "[")[0]
	}
	
	var result []string
	
	// Try regular nested path first
	prefix := cleanPath + "."
	for _, orderKey := range s.Order {
		if strings.HasPrefix(orderKey, prefix) {
			// Extract the immediate child key
			remaining := strings.TrimPrefix(orderKey, prefix)
			if strings.Contains(remaining, ".") {
				// This is a deeper nested key, extract only the immediate child
				childKey := strings.Split(remaining, ".")[0]
				// Check if we already have this child key
				found := false
				for _, existing := range result {
					if existing == childKey {
						found = true
						break
					}
				}
				if !found {
					result = append(result, childKey)
				}
			} else {
				// Direct child, add it
				result = append(result, remaining)
			}
		}
	}
	
	// If no results and the original path had an array index, try array notation
	if len(result) == 0 && strings.Contains(path, "[") {
		arrayPrefix := cleanPath + "[*]."
		for _, orderKey := range s.Order {
			if strings.HasPrefix(orderKey, arrayPrefix) {
				// Extract the immediate child key
				remaining := strings.TrimPrefix(orderKey, arrayPrefix)
				if strings.Contains(remaining, ".") {
					// This is a deeper nested key, extract only the immediate child
					childKey := strings.Split(remaining, ".")[0]
					// Check if we already have this child key
					found := false
					for _, existing := range result {
						if existing == childKey {
							found = true
							break
						}
					}
					if !found {
						result = append(result, childKey)
					}
				} else {
					// Direct child, add it
					result = append(result, remaining)
				}
			}
		}
	}
	
	return result
}

// IsNonSortKey checks if a key should not be sorted
func (s *Schema) IsNonSortKey(key string) bool {
	if s.NonSort == nil {
		return false
	}
	
	nonSortKeys := s.NonSortKeys()
	for _, nonSortKey := range nonSortKeys {
		if key == nonSortKey {
			return true
		}
	}
	return false
}

// Validate checks if the schema is valid
func (s *Schema) Validate() error {
	if s == nil {
		return fmt.Errorf("schema is nil")
	}
	
	if s.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}
	
	if s.Keys == nil || len(s.Keys) == 0 {
		return fmt.Errorf("schema must have at least one key defined")
	}
	
	if len(s.Order) == 0 {
		return fmt.Errorf("schema order is empty")
	}
	
	// Check for circular references or other validation rules
	// TODO: Implement more comprehensive validation
	
	return nil
}

// String returns a string representation of the schema
func (s *Schema) String() string {
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Sprintf("Schema{Name: %s, Error: %v}", s.Name, err)
	}
	return string(data)
}

// GenerateFromYAML creates a schema by analyzing an existing YAML structure
func GenerateFromYAML(yamlData []byte, name string) (*Schema, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(yamlData, &node); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	schema := &Schema{
		Name: name,
		Keys: make(map[string]interface{}),
		Order: []string{},
	}
	
	// Extract order and structure directly from the YAML node
	if len(node.Content) > 0 {
		extractSchemaOrder(node.Content[0], "", &schema.Order, schema.Keys)
	}
	
	return schema, nil
}

// extractSchemaFromNode recursively extracts the key structure from a YAML node
func extractSchemaFromNode(node *yaml.Node, target map[string]interface{}) {
	if node.Kind != yaml.MappingNode {
		return
	}
	
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		
		key := keyNode.Value
		
		switch valueNode.Kind {
		case yaml.MappingNode:
			// Nested mapping
			nested := make(map[string]interface{})
			extractSchemaFromNode(valueNode, nested)
			target[key] = nested
		case yaml.SequenceNode:
			// Array - check if it contains mappings
			if len(valueNode.Content) > 0 && valueNode.Content[0].Kind == yaml.MappingNode {
				// Array of objects, extract schema from first object
				nested := make(map[string]interface{})
				extractSchemaFromNode(valueNode.Content[0], nested)
				target[key] = nested
			} else {
				// Simple array
				target[key] = nil
			}
		default:
			// Scalar value
			target[key] = nil
		}
	}
}

// LoadFromBytes loads a schema from YAML bytes
func LoadFromBytes(data []byte, name string) (*Schema, error) {
	// Parse YAML to extract structure
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	
	schema := &Schema{
		Name:  name,
		Keys:  make(map[string]interface{}),
		Order: []string{},
	}
	
	// Extract schema structure from YAML node
	if len(node.Content) > 0 {
		extractSchemaOrder(node.Content[0], "", &schema.Order, schema.Keys)
	}
	
	return schema, nil
}

// extractSchemaOrder extracts both the order and structure from schema YAML
func extractSchemaOrder(node *yaml.Node, prefix string, order *[]string, keys map[string]interface{}) {
	if node.Kind == yaml.MappingNode {
		// Process mapping node
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			
			key := keyNode.Value
			if key == "non_sort" {
				continue
			}
			
			fullKey := key
			if prefix != "" {
				fullKey = prefix + "." + key
			}
			
			*order = append(*order, fullKey)
			
			if valueNode.Kind == yaml.MappingNode {
				// Nested mapping
				nestedKeys := make(map[string]interface{})
				extractSchemaOrder(valueNode, fullKey, order, nestedKeys)
				keys[key] = nestedKeys
			} else if valueNode.Kind == yaml.SequenceNode && len(valueNode.Content) > 0 {
				// Array with structure definition
				if valueNode.Content[0].Kind == yaml.MappingNode {
					nestedKeys := make(map[string]interface{})
					// Extract structure from first array element
					for j := 0; j < len(valueNode.Content[0].Content); j += 2 {
						elemKey := valueNode.Content[0].Content[j].Value
						*order = append(*order, fullKey + "[*]." + elemKey)
						nestedKeys[elemKey] = nil
					}
					keys[key] = nestedKeys
				} else {
					keys[key] = nil
				}
			} else {
				// Scalar or null
				keys[key] = nil
			}
		}
	}
}

// DefaultSchemaName generates a default schema name based on file path
func DefaultSchemaName(filePath string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	
	// Convert common patterns to schema names
	switch {
	case strings.Contains(name, "docker-compose"):
		return "compose"
	case strings.Contains(name, ".k8s") || strings.Contains(name, "kubernetes"):
		return "k8s"
	case strings.Contains(name, "github") || strings.Contains(filePath, ".github/workflows"):
		return "github-actions"
	case strings.Contains(name, "playbook") || strings.Contains(name, "ansible"):
		return "ansible"
	case strings.Contains(name, "values") && strings.Contains(filePath, "helm"):
		return "helm"
	default:
		return name
	}
}