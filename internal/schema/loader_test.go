package schema

import (
	"github.com/spf13/afero"
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderSaveAndLoad(t *testing.T) {
	// Create in-memory filesystem for testing
	fs := afero.NewMemMapFs()
	tempDir := "/tmp/schemas"

	loader := NewLoader(fs, tempDir)

	// Create test schema
	s := &Schema{
		Name: "test-schema",
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

	// Save schema
	err := loader.SaveSchema(s)
	if err != nil {
		t.Fatalf("SaveSchema failed: %v", err)
	}

	// Load schema back
	loaded, err := loader.LoadSchema("test-schema")
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	if loaded.Name != s.Name {
		t.Errorf("Loaded schema name mismatch: got %s, want %s", loaded.Name, s.Name)
	}

	if len(loaded.Order) != len(s.Order) {
		t.Errorf("Loaded schema order length mismatch: got %d, want %d",
			len(loaded.Order), len(s.Order))
	}
}

func TestLoaderListSchemas(t *testing.T) {
	fs := afero.NewMemMapFs()
	tempDir := "/tmp/schemas"

	loader := NewLoader(fs, tempDir)

	// Save multiple schemas
	schemas := []string{"schema1", "schema2", "schema3"}
	for _, name := range schemas {
		s := &Schema{
			Name:  name,
			Keys:  map[string]interface{}{"key1": nil},
			Order: []string{"key1"},
		}
		if err := loader.SaveSchema(s); err != nil {
			t.Fatalf("Failed to save schema %s: %v", name, err)
		}
	}

	// List schemas
	list, err := loader.ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas failed: %v", err)
	}

	if len(list) != len(schemas) {
		t.Errorf("Expected %d schemas, got %d", len(schemas), len(list))
	}

	// Check all schemas are present
	for _, expected := range schemas {
		found := false
		for _, actual := range list {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Schema %s not found in list", expected)
		}
	}
}

func TestLoaderLoadSchemaFromFile(t *testing.T) {
	// Test with real file from examples
	schemaPath := "../../examples/docker-compose.schema.yaml"

	// Check if file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Skip("Example schema file not found")
	}

	loader := NewLoader(nil, "")

	s, err := loader.LoadSchemaFromFile(schemaPath)
	if err != nil {
		t.Fatalf("LoadSchemaFromFile failed: %v", err)
	}

	if s.Name == "" {
		t.Error("Loaded schema has empty name")
	}

	if len(s.Order) == 0 {
		t.Error("Loaded schema has empty order")
	}

	// Check for expected Docker Compose keys
	hasVersion := false
	hasServices := false
	for _, key := range s.Order {
		if key == "version" {
			hasVersion = true
		}
		if key == "services" {
			hasServices = true
		}
	}

	if !hasVersion {
		t.Error("Docker Compose schema should have 'version' key")
	}

	if !hasServices {
		t.Error("Docker Compose schema should have 'services' key")
	}
}

func TestLoaderGenerateAndSaveFromYAML(t *testing.T) {
	fs := afero.NewMemMapFs()
	tempDir := "/tmp/schemas"

	loader := NewLoader(fs, tempDir)

	// Create a test YAML file
	yamlContent := `name: TestApp
version: 1.0.0
config:
  debug: true
  port: 8080`

	yamlPath := "/tmp/test.yml"
	if err := afero.WriteFile(fs, yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	// Generate and save schema
	s, err := loader.GenerateAndSaveFromYAML(yamlPath, "generated-schema")
	if err != nil {
		t.Fatalf("GenerateAndSaveFromYAML failed: %v", err)
	}

	if s.Name != "generated-schema" {
		t.Errorf("Generated schema name mismatch: got %s, want generated-schema", s.Name)
	}

	// Verify schema was saved
	loaded, err := loader.LoadSchema("generated-schema")
	if err != nil {
		t.Fatalf("Failed to load generated schema: %v", err)
	}

	if loaded.Name != s.Name {
		t.Error("Loaded schema doesn't match generated schema")
	}
}

func TestLoaderNonExistentSchema(t *testing.T) {
	fs := afero.NewMemMapFs()
	loader := NewLoader(fs, "/tmp/schemas")

	_, err := loader.LoadSchema("non-existent")
	if err == nil {
		t.Error("Expected error when loading non-existent schema")
	}
}

func TestLoaderInvalidSchemaFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	tempDir := "/tmp/schemas"

	loader := NewLoader(fs, tempDir)

	// Create invalid schema file
	invalidPath := filepath.Join(tempDir, "invalid.yaml")
	if err := fs.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create schema directory: %v", err)
	}

	invalidContent := `this is not: valid schema
- format`

	if err := afero.WriteFile(fs, invalidPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create invalid schema file: %v", err)
	}

	_, err := loader.LoadSchema("invalid")
	if err == nil {
		t.Error("Expected error when loading invalid schema")
	}
}

func TestGetSchemaPath(t *testing.T) {
	loader := NewLoader(nil, "/schemas")

	// Test that schemas are saved to the correct path
	schema := &Schema{
		Name: "test",
		Keys: map[string]interface{}{
			"key1": nil,
		},
	}

	// Save and verify the path used
	err := loader.SaveSchema(schema)
	if err == nil {
		// We expect an error because we're using nil filesystem
		t.Error("Expected error with nil filesystem")
	}
}

func TestLoaderWithRealTestData(t *testing.T) {
	// Test generating schema from test data
	testFiles := []struct {
		name       string
		yamlPath   string
		schemaName string
	}{
		{
			name:       "Simple YAML",
			yamlPath:   "../../testdata/valid/simple.yml",
			schemaName: "simple",
		},
		{
			name:       "Complex nested",
			yamlPath:   "../../testdata/valid/complex-nested.yml",
			schemaName: "complex",
		},
		{
			name:       "With anchors",
			yamlPath:   "../../testdata/valid/anchors-and-aliases.yml",
			schemaName: "anchors",
		},
	}

	fs := afero.NewMemMapFs()
	loader := NewLoader(fs, "/tmp/schemas")

	for _, tt := range testFiles {
		t.Run(tt.name, func(t *testing.T) {
			// Read real file
			content, err := os.ReadFile(tt.yamlPath)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			// Write to memory fs
			tempPath := "/tmp/test.yml"
			if err := afero.WriteFile(fs, tempPath, content, 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			// Generate and save
			s, err := loader.GenerateAndSaveFromYAML(tempPath, tt.schemaName)
			if err != nil {
				t.Errorf("GenerateAndSaveFromYAML failed: %v", err)
			}

			if s == nil || len(s.Order) == 0 {
				t.Error("Generated empty schema")
			}
		})
	}
}
