package schema

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Schema represents a YAML formatting schema that defines key ordering
type Schema struct {
	Name    string                 `yaml:"name,omitempty"`
	Keys    map[string]interface{} `yaml:",inline"`
	NonSort map[string]interface{} `yaml:"non_sort,omitempty"`
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
	return keys
}

// GetKeyOrder returns the ordering for a specific path in the schema
func (s *Schema) GetKeyOrder(path string) []string {
	parts := strings.Split(path, ".")
	current := s.Keys
	
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		if val, exists := current[part]; exists {
			if mapVal, ok := val.(map[string]interface{}); ok {
				current = mapVal
			} else {
				// Leaf node reached
				return nil
			}
		} else {
			// Path not found in schema
			return nil
		}
	}
	
	return extractKeysFromMap(current)
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
	if s.Keys == nil {
		return fmt.Errorf("schema must have at least one key defined")
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
	}
	
	// Extract key structure from the YAML node
	if len(node.Content) > 0 {
		extractSchemaFromNode(node.Content[0], schema.Keys)
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