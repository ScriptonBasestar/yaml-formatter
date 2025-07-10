package schema

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// Loader manages loading and saving schemas
type Loader struct {
	fs        afero.Fs
	schemaDir string
}

// NewLoader creates a new schema loader
func NewLoader(filesystem afero.Fs, schemaDir string) *Loader {
	if filesystem == nil {
		filesystem = afero.NewOsFs()
	}
	
	return &Loader{
		fs:        filesystem,
		schemaDir: schemaDir,
	}
}

// DefaultLoader creates a loader with default settings
func DefaultLoader() *Loader {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	
	schemaDir := filepath.Join(home, ".sb-yaml", "schemas")
	return NewLoader(afero.NewOsFs(), schemaDir)
}

// ensureSchemaDir creates the schema directory if it doesn't exist
func (l *Loader) ensureSchemaDir() error {
	return l.fs.MkdirAll(l.schemaDir, 0755)
}

// LoadSchema loads a schema by name
func (l *Loader) LoadSchema(name string) (*Schema, error) {
	schemaPath := l.getSchemaPath(name)
	
	data, err := afero.ReadFile(l.fs, schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}
	
	schema, err := LoadFromBytes(data, name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema file %s: %w", schemaPath, err)
	}
	
	return schema, nil
}

// SaveSchema saves a schema to the schema directory
func (l *Loader) SaveSchema(schema *Schema) error {
	if err := l.ensureSchemaDir(); err != nil {
		return fmt.Errorf("failed to create schema directory: %w", err)
	}
	
	if schema.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}
	
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	
	// Create a copy without the Order field for serialization
	schemaData := map[string]interface{}{}
	
	// Add Keys
	for k, v := range schema.Keys {
		schemaData[k] = v
	}
	
	// Add NonSort if present
	if schema.NonSort != nil && len(schema.NonSort) > 0 {
		schemaData["non_sort"] = schema.NonSort
	}
	
	data, err := yaml.Marshal(schemaData)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}
	
	schemaPath := l.getSchemaPath(schema.Name)
	if err := afero.WriteFile(l.fs, schemaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write schema file %s: %w", schemaPath, err)
	}
	
	return nil
}

// LoadSchemaFromFile loads a schema from a specific file path
func (l *Loader) LoadSchemaFromFile(filePath string) (*Schema, error) {
	data, err := afero.ReadFile(l.fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", filePath, err)
	}
	
	// Generate name from file path
	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	
	schema, err := LoadFromBytes(data, name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema file %s: %w", filePath, err)
	}
	
	return schema, nil
}

// GenerateAndSaveFromYAML creates a schema from a YAML file and saves it
func (l *Loader) GenerateAndSaveFromYAML(yamlPath, schemaName string) (*Schema, error) {
	yamlData, err := afero.ReadFile(l.fs, yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file %s: %w", yamlPath, err)
	}
	
	schema, err := GenerateFromYAML(yamlData, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema from YAML: %w", err)
	}
	
	if err := l.SaveSchema(schema); err != nil {
		return nil, fmt.Errorf("failed to save generated schema: %w", err)
	}
	
	return schema, nil
}

// ListSchemas returns a list of all available schemas
func (l *Loader) ListSchemas() ([]string, error) {
	exists, err := afero.DirExists(l.fs, l.schemaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to check schema directory: %w", err)
	}
	
	if !exists {
		return []string{}, nil
	}
	
	var schemas []string
	
	err = afero.Walk(l.fs, l.schemaDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			rel, err := filepath.Rel(l.schemaDir, path)
			if err != nil {
				return err
			}
			
			name := strings.TrimSuffix(rel, filepath.Ext(rel))
			schemas = append(schemas, name)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk schema directory: %w", err)
	}
	
	return schemas, nil
}

// SchemaExists checks if a schema with the given name exists
func (l *Loader) SchemaExists(name string) bool {
	schemaPath := l.getSchemaPath(name)
	exists, err := afero.Exists(l.fs, schemaPath)
	return err == nil && exists
}

// DeleteSchema removes a schema by name
func (l *Loader) DeleteSchema(name string) error {
	if !l.SchemaExists(name) {
		return fmt.Errorf("schema %s does not exist", name)
	}
	
	schemaPath := l.getSchemaPath(name)
	if err := l.fs.Remove(schemaPath); err != nil {
		return fmt.Errorf("failed to delete schema %s: %w", name, err)
	}
	
	return nil
}

// getSchemaPath returns the full path for a schema file
func (l *Loader) getSchemaPath(name string) string {
	// Ensure the filename has the correct extension
	if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
		name += ".yaml"
	}
	
	return filepath.Join(l.schemaDir, name)
}

// GetSchemaDir returns the schema directory path
func (l *Loader) GetSchemaDir() string {
	return l.schemaDir
}