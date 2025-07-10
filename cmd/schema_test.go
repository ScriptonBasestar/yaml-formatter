package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaGenCommand(t *testing.T) {
	// Create test YAML file
	tempDir := t.TempDir()
	yamlPath := filepath.Join(tempDir, "test.yml")
	yamlContent := `name: test
version: 1.0.0
metadata:
  author: tester`
	
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"schema", "gen", "test-schema", yamlPath})
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Schema gen command failed: %v", err)
	}
	
	output := buf.String()
	
	// Check output contains expected schema structure
	if !strings.Contains(output, "name:") {
		t.Error("Schema output doesn't contain 'name:' field")
	}
	
	if !strings.Contains(output, "version:") {
		t.Error("Schema output doesn't contain 'version:' field")
	}
	
	if !strings.Contains(output, "metadata:") {
		t.Error("Schema output doesn't contain 'metadata:' field")
	}
}

func TestSchemaSetCommand(t *testing.T) {
	// Create test schema file
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "test.schema.yaml")
	schemaContent := `name:
version:
description:`
	
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}
	
	// Create temporary schema directory
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema directory: %v", err)
	}
	
	// Override config to use temp schema dir
	os.Setenv("SB_YAML_SCHEMA_DIR", schemaDir)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"schema", "set", "test-schema", schemaPath})
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Schema set command failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "saved successfully") {
		t.Error("Schema set output doesn't indicate success")
	}
	
	// Verify schema was saved
	savedPath := filepath.Join(schemaDir, "test-schema.yaml")
	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Error("Schema file was not saved")
	}
}

func TestSchemaSetFromYAML(t *testing.T) {
	// Create test YAML file
	tempDir := t.TempDir()
	yamlPath := filepath.Join(tempDir, "source.yml")
	yamlContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test`
	
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create YAML file: %v", err)
	}
	
	// Create temporary schema directory
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema directory: %v", err)
	}
	
	os.Setenv("SB_YAML_SCHEMA_DIR", schemaDir)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"schema", "set", "k8s-config", yamlPath, "--from-yaml"})
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Schema set --from-yaml command failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "generated from") {
		t.Error("Schema set output doesn't indicate generation from YAML")
	}
}

func TestSchemaListCommand(t *testing.T) {
	// Create temporary schema directory with some schemas
	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema directory: %v", err)
	}
	
	// Create test schema files
	schemas := []string{"schema1.yaml", "schema2.yaml", "schema3.yaml"}
	for _, schema := range schemas {
		path := filepath.Join(schemaDir, schema)
		content := `name:
version:`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create schema file: %v", err)
		}
	}
	
	os.Setenv("SB_YAML_SCHEMA_DIR", schemaDir)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"schema", "list"})
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Schema list command failed: %v", err)
	}
	
	output := buf.String()
	
	// Check for expected schemas
	for _, schema := range []string{"schema1", "schema2", "schema3"} {
		if !strings.Contains(output, schema) {
			t.Errorf("Schema list output missing schema: %s", schema)
		}
	}
	
	if !strings.Contains(output, "Available schemas (3)") {
		t.Error("Schema list output doesn't show correct count")
	}
}

func TestSchemaCommandErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "Gen with missing args",
			args:    []string{"schema", "gen"},
			wantErr: true,
		},
		{
			name:    "Gen with non-existent file",
			args:    []string{"schema", "gen", "test", "/non/existent/file.yml"},
			wantErr: true,
		},
		{
			name:    "Set with missing args",
			args:    []string{"schema", "set"},
			wantErr: true,
		},
		{
			name:    "Set with non-existent file",
			args:    []string{"schema", "set", "test", "/non/existent/schema.yaml"},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)
			
			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}