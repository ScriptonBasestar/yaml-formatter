package formatter

import (
	"os"
	"testing"
	"yaml-formatter/internal/schema"
	"gopkg.in/yaml.v3"
)

func TestReorderNode(t *testing.T) {
	// Create test schema
	s := &schema.Schema{
		Name: "test",
		Order: []string{
			"name",
			"version",
			"metadata",
			"metadata.author",
			"metadata.created",
			"settings",
			"features",
		},
	}
	
	parser := NewParser(true)
	reorderer := NewReorderer(s, parser)
	
	// Test simple reordering
	content := `settings:
  debug: true
features:
  - feature1
  - feature2
name: Test App
metadata:
  created: 2024-01-01
  author: John Doe
version: 1.0.0`
	
	node, err := parser.ParseYAML([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	
	err = reorderer.ReorderNode(node, "")
	if err != nil {
		t.Errorf("ReorderNode failed: %v", err)
	}
	
	// Check order of keys - handle document node wrapper
	testNode := node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		testNode = node.Content[0]
	}
	
	if testNode.Kind != yaml.MappingNode {
		t.Fatalf("Expected mapping node, got %d", testNode.Kind)
	}
	
	expectedOrder := []string{"name", "version", "metadata", "settings", "features"}
	actualOrder := make([]string, 0)
	
	for i := 0; i < len(testNode.Content); i += 2 {
		if i < len(testNode.Content) {
			actualOrder = append(actualOrder, testNode.Content[i].Value)
		}
	}
	
	for i, expected := range expectedOrder {
		if i >= len(actualOrder) || actualOrder[i] != expected {
			t.Errorf("Key order mismatch at position %d: expected %s, got %s", 
				i, expected, actualOrder[i])
		}
	}
}

func TestReorderWithWildcards(t *testing.T) {
	// Schema with wildcards
	s := &schema.Schema{
		Name: "wildcard-test",
		Order: []string{
			"apiVersion",
			"kind",
			"metadata",
			"spec",
			"spec.containers",
			"spec.containers[*].name",
			"spec.containers[*].image",
			"spec.containers[*].ports",
		},
	}
	
	parser := NewParser(true)
	reorderer := NewReorderer(s, parser)
	
	content := `spec:
  containers:
    - ports:
        - containerPort: 8080
      image: myapp:latest
      name: app
    - image: sidecar:latest
      name: sidecar
kind: Deployment
apiVersion: apps/v1`
	
	node, err := parser.ParseYAML([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	
	err = reorderer.ReorderNode(node, "")
	if err != nil {
		t.Errorf("ReorderNode failed: %v", err)
	}
	
	// Verify the containers are properly ordered
	// This would require walking the tree to verify
}

func TestCheckOrder(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Keys: map[string]interface{}{
			"name":        nil,
			"version":     nil,
			"description": nil,
		},
		Order: []string{
			"name",
			"version",
			"description",
		},
	}
	
	parser := NewParser(true)
	reorderer := NewReorderer(s, parser)
	
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "Properly ordered",
			content: `name: Test
version: 1.0.0
description: A test`,
			expected: true,
		},
		{
			name: "Out of order",
			content: `version: 1.0.0
name: Test
description: A test`,
			expected: false,
		},
		{
			name: "Extra fields",
			content: `name: Test
extra: field
version: 1.0.0
description: A test`,
			expected: true, // Extra fields don't affect order check
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.ParseYAML([]byte(tt.content))
			if err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
			}
			
			result, err := reorderer.CheckOrder(node, "")
			if err != nil {
				t.Errorf("CheckOrder failed: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("CheckOrder returned %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestReorderComplexTestData(t *testing.T) {
	// Load Kubernetes schema
	schemaContent, err := os.ReadFile("../../examples/kubernetes.schema.yaml")
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}
	
	s, err := schema.LoadFromBytes(schemaContent, "kubernetes")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}
	
	parser := NewParser(true)
	reorderer := NewReorderer(s, parser)
	
	// Test with unordered Kubernetes YAML
	content, err := os.ReadFile("../../testdata/formatting/input/unordered-kubernetes.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	
	node, err := parser.ParseYAML(content)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	
	err = reorderer.ReorderNode(node, "")
	if err != nil {
		t.Errorf("ReorderNode failed: %v", err)
	}
	
	// Handle document node wrapper
	testNode := node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		testNode = node.Content[0]
	}
	
	// Verify apiVersion and kind are first
	if len(testNode.Content) < 4 {
		t.Fatalf("Expected at least 2 key-value pairs, got %d items", len(testNode.Content))
	}
	
	if testNode.Content[0].Value != "apiVersion" {
		t.Errorf("Expected first key to be 'apiVersion', got '%s'", testNode.Content[0].Value)
	}
	
	if testNode.Content[2].Value != "kind" {
		t.Errorf("Expected second key to be 'kind', got '%s'", testNode.Content[2].Value)
	}
}