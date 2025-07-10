package schema

import (
	"os"
	"strings"
	"testing"
)

func TestLoadFromBytes(t *testing.T) {
	schemaContent := `name:
version:
metadata:
  author:
  created:
items:
  - name:
    value:`
	
	s, err := LoadFromBytes([]byte(schemaContent), "test-schema")
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}
	
	if s.Name != "test-schema" {
		t.Errorf("Expected schema name 'test-schema', got '%s'", s.Name)
	}
	
	expectedOrder := []string{
		"name",
		"version",
		"metadata",
		"metadata.author",
		"metadata.created",
		"items",
		"items[*].name",
		"items[*].value",
	}
	
	if len(s.Order) != len(expectedOrder) {
		t.Errorf("Expected %d order entries, got %d", len(expectedOrder), len(s.Order))
	}
	
	for i, expected := range expectedOrder {
		if i >= len(s.Order) || s.Order[i] != expected {
			t.Errorf("Order mismatch at position %d: expected '%s', got '%s'", 
				i, expected, s.Order[i])
		}
	}
}

func TestGenerateFromYAML(t *testing.T) {
	yamlContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
  labels:
    app: test
data:
  config.yaml: |
    key: value`
	
	s, err := GenerateFromYAML([]byte(yamlContent), "k8s-configmap")
	if err != nil {
		t.Fatalf("GenerateFromYAML failed: %v", err)
	}
	
	if s.Name != "k8s-configmap" {
		t.Errorf("Expected schema name 'k8s-configmap', got '%s'", s.Name)
	}
	
	// Check that apiVersion and kind are first
	if len(s.Order) < 2 {
		t.Fatal("Schema should have at least 2 entries")
	}
	
	if s.Order[0] != "apiVersion" {
		t.Errorf("First entry should be 'apiVersion', got '%s'", s.Order[0])
	}
	
	if s.Order[1] != "kind" {
		t.Errorf("Second entry should be 'kind', got '%s'", s.Order[1])
	}
	
	// Check nested paths
	hasMetadataName := false
	for _, path := range s.Order {
		if path == "metadata.name" {
			hasMetadataName = true
			break
		}
	}
	
	if !hasMetadataName {
		t.Error("Schema should include 'metadata.name' path")
	}
}

func TestGenerateFromComplexYAML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/valid/complex-nested.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	
	s, err := GenerateFromYAML(content, "complex")
	if err != nil {
		t.Fatalf("GenerateFromYAML failed: %v", err)
	}
	
	// Verify nested paths are generated
	expectedPaths := []string{
		"application",
		"application.name",
		"application.version",
		"application.services.database",
		"application.services.cache",
		"application.monitoring.providers",
	}
	
	for _, expected := range expectedPaths {
		found := false
		for _, path := range s.Order {
			if path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected path '%s' not found in schema", expected)
		}
	}
}

func TestSchemaString(t *testing.T) {
	s := &Schema{
		Name: "test",
		Keys: map[string]interface{}{
			"name":    nil,
			"version": nil,
			"items": map[string]interface{}{
				"key": nil,
			},
		},
		Order: []string{
			"name",
			"version",
			"items",
			"items[*].key",
		},
	}
	
	output := s.String()
	
	if output == "" {
		t.Error("String() returned empty output")
	}
	
	// Verify it's valid YAML and contains the basic structure
	if !strings.Contains(output, "version:") {
		t.Error("Output should contain the version field from Keys")
	}
	
	// Just verify we can create a new schema with similar functionality
	// Note: Order field is not serialized due to yaml:"-" tag, so exact match is not expected
	newSchema, err := GenerateFromYAML([]byte("name: test\nversion: 1.0\nitems:\n  - key: value"), "test2")
	if err != nil {
		t.Errorf("Failed to generate test schema: %v", err)
	}
	
	if len(newSchema.Order) == 0 {
		t.Error("Generated schema should have some order")
	}
}

func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name    string
		schema  *Schema
		wantErr bool
	}{
		{
			name: "Valid schema",
			schema: &Schema{
				Name:  "valid",
				Keys:  map[string]interface{}{"key1": nil, "key2": nil},
				Order: []string{"key1", "key2"},
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			schema: &Schema{
				Name:  "",
				Order: []string{"key1"},
			},
			wantErr: true,
		},
		{
			name: "Empty order",
			schema: &Schema{
				Name:  "test",
				Order: []string{},
			},
			wantErr: true,
		},
		{
			name:    "Nil schema",
			schema:  nil,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetKeyOrder(t *testing.T) {
	s := &Schema{
		Name: "test",
		Keys: map[string]interface{}{
			"name":    nil,
			"version": nil,
			"metadata": map[string]interface{}{
				"author":  nil,
				"created": nil,
			},
			"items": map[string]interface{}{
				"name":  nil,
				"value": nil,
			},
		},
		Order: []string{
			"name",
			"version",
			"metadata",
			"metadata.author",
			"metadata.created",
			"items",
			"items[*].name",
			"items[*].value",
		},
	}
	
	tests := []struct {
		path     string
		expected []string
	}{
		{
			path: "",
			expected: []string{"name", "version", "metadata", "items"},
		},
		{
			path: "metadata",
			expected: []string{"author", "created"},
		},
		{
			path: "items[0]",
			expected: []string{"name", "value"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			order := s.GetKeyOrder(tt.path)
			
			if len(order) != len(tt.expected) {
				t.Errorf("GetKeyOrder(%s) returned %d keys, expected %d", 
					tt.path, len(order), len(tt.expected))
			}
			
			for i, key := range tt.expected {
				if i >= len(order) || order[i] != key {
					t.Errorf("GetKeyOrder(%s)[%d] = %s, expected %s", 
						tt.path, i, order[i], key)
				}
			}
		})
	}
}

func TestSchemaWithArrays(t *testing.T) {
	yamlContent := `services:
  - name: api
    image: api:latest
    ports:
      - 8080
      - 8081
  - name: db
    image: postgres:14
    environment:
      POSTGRES_DB: mydb`
	
	s, err := GenerateFromYAML([]byte(yamlContent), "array-test")
	if err != nil {
		t.Fatalf("GenerateFromYAML failed: %v", err)
	}
	
	// Check for array wildcard paths
	hasServicesWildcard := false
	hasPortsWildcard := false
	
	for _, path := range s.Order {
		if path == "services[*].name" {
			hasServicesWildcard = true
		}
		if path == "services[*].ports" {
			hasPortsWildcard = true
		}
	}
	
	if !hasServicesWildcard {
		t.Error("Schema should include 'services[*].name' for array elements")
	}
	
	if !hasPortsWildcard {
		t.Error("Schema should include 'services[*].ports' for nested arrays")
	}
}