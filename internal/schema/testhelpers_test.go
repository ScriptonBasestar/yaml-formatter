package schema

import (
	"testing"
)

func TestCreateTestSchema(t *testing.T) {
	keys := []string{"name", "version", "description"}
	schema := CreateTestSchema("test", keys)

	if schema.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", schema.Name)
	}

	if len(schema.Order) != len(keys) {
		t.Errorf("Expected %d order entries, got %d", len(keys), len(schema.Order))
	}

	for i, key := range keys {
		if schema.Order[i] != key {
			t.Errorf("Expected order[%d] = '%s', got '%s'", i, key, schema.Order[i])
		}

		if _, exists := schema.Keys[key]; !exists {
			t.Errorf("Expected key '%s' to exist in Keys map", key)
		}
	}
}

func TestCreateNestedTestSchema(t *testing.T) {
	schema := CreateNestedTestSchema("nested")

	if schema.Name != "nested" {
		t.Errorf("Expected name 'nested', got '%s'", schema.Name)
	}

	// Check that we have nested structure
	if metadata, ok := schema.Keys["metadata"].(map[string]interface{}); ok {
		if _, exists := metadata["author"]; !exists {
			t.Error("Expected 'author' key in metadata")
		}
	} else {
		t.Error("Expected metadata to be a map")
	}

	// Check that order includes nested paths
	found := false
	for _, path := range schema.Order {
		if path == "metadata.author" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'metadata.author' in order")
	}
}

func TestCreateDockerComposeTestSchema(t *testing.T) {
	schema := CreateDockerComposeTestSchema()

	if schema.Name != "compose" {
		t.Errorf("Expected name 'compose', got '%s'", schema.Name)
	}

	// Check key structure
	if _, exists := schema.Keys["version"]; !exists {
		t.Error("Expected 'version' key")
	}

	if services, ok := schema.Keys["services"].(map[string]interface{}); ok {
		if _, exists := services["image"]; !exists {
			t.Error("Expected 'image' key in services")
		}
	} else {
		t.Error("Expected services to be a map")
	}

	// Check that order starts with version
	if len(schema.Order) == 0 || schema.Order[0] != "version" {
		t.Error("Expected first order entry to be 'version'")
	}
}

func TestGetTestData(t *testing.T) {
	data := GetTestData("minimal")
	if data == nil {
		t.Error("Expected to get test data for 'minimal'")
	}

	if len(data) == 0 {
		t.Error("Expected test data to not be empty")
	}

	// Test non-existent key
	data = GetTestData("non-existent")
	if data != nil {
		t.Error("Expected nil for non-existent key")
	}
}

func TestListTestDataKeys(t *testing.T) {
	keys := ListTestDataKeys()

	if len(keys) == 0 {
		t.Error("Expected to have some test data keys")
	}

	// Check that we have expected keys
	expectedKeys := []string{"minimal", "docker-compose", "kubernetes"}
	for _, expected := range expectedKeys {
		found := false
		for _, key := range keys {
			if key == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find key '%s'", expected)
		}
	}
}
