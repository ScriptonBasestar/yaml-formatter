//go:build integration

package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"yaml-formatter/tests/e2e"
)

func TestSchemaGenCommand(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	yamlContent := `name: test
version: 1.0.0
metadata:
  author: tester`
	h.CreateTestFile("test.yml", yamlContent)

	stdout, stderr, err := h.ExecuteCommand("schema", "gen", "test-schema", "test.yml")
	if err != nil {
		t.Errorf("Schema gen command failed: %v\nStderr: %s", err, stderr)
	}

	// Check output contains expected schema structure
	if !strings.Contains(stdout, "name:") {
		t.Errorf("Expected stdout to contain 'name:', got: %s", stdout)
	}
	if !strings.Contains(stdout, "version:") {
		t.Errorf("Expected stdout to contain 'version:', got: %s", stdout)
	}
	if !strings.Contains(stdout, "metadata:") {
		t.Errorf("Expected stdout to contain 'metadata:', got: %s", stdout)
	}
}

func TestSchemaSetCommand(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	schemaContent := `name:
version:
description:`
	h.CreateTestFile("test.schema.yaml", schemaContent)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, stderr, err := h.ExecuteCommand("schema", "set", "test-schema", "test.schema.yaml")
	if err != nil {
		t.Errorf("Schema set command failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "saved successfully") {
		t.Errorf("Expected stdout to contain 'saved successfully', got: %s", stdout)
	}

	// Verify schema was saved
	actualContent, err := h.ReadTestFile(filepath.Join("schemas", "test-schema.yaml"))
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}
	if actualContent != schemaContent {
		t.Errorf("Expected schema content:\n%s\nGot:\n%s", schemaContent, actualContent)
	}
}

func TestSchemaSetFromYAML(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	yamlContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test`
	h.CreateTestFile("source.yml", yamlContent)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, stderr, err := h.ExecuteCommand("schema", "set", "k8s-config", "source.yml", "--from-yaml")
	if err != nil {
		t.Errorf("Schema set --from-yaml command failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "generated from") {
		t.Errorf("Expected stdout to contain 'generated from', got: %s", stdout)
	}
}

func TestSchemaListCommand(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	// Create test schema files
	schemas := []string{"schema1.yaml", "schema2.yaml", "schema3.yaml"}
	for _, schema := range schemas {
		content := `name:
version:`
		h.CreateTestFile(filepath.Join("schemas", schema), content)
	}

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, stderr, err := h.ExecuteCommand("schema", "list")
	if err != nil {
		t.Errorf("Schema list command failed: %v\nStderr: %s", err, stderr)
	}

	// Check for expected schemas
	if !strings.Contains(stdout, "schema1") {
		t.Errorf("Expected stdout to contain 'schema1', got: %s", stdout)
	}
	if !strings.Contains(stdout, "schema2") {
		t.Errorf("Expected stdout to contain 'schema2', got: %s", stdout)
	}
	if !strings.Contains(stdout, "schema3") {
		t.Errorf("Expected stdout to contain 'schema3', got: %s", stdout)
	}
	if !strings.Contains(stdout, "Available schemas (3)") {
		t.Errorf("Expected stdout to contain 'Available schemas (3)', got: %s", stdout)
	}
}

func TestSchemaCommandErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		wantOut string
	}{
		{
			name:    "Gen with missing args",
			args:    []string{"schema", "gen"},
			wantErr: true,
			wantOut: "Error: accepts 2 arg(s), received 0",
		},
		{
			name:    "Gen with non-existent file",
			args:    []string{"schema", "gen", "test", "/non/existent/file.yml"},
			wantErr: true,
			wantOut: "Error: open /non/existent/file.yml: no such file or directory",
		},
		{
			name:    "Set with missing args",
			args:    []string{"schema", "set"},
			wantErr: true,
			wantOut: "Error: accepts 2 arg(s), received 0",
		},
		{
			name:    "Set with non-existent file",
			args:    []string{"schema", "set", "test", "/non/existent/schema.yaml"},
			wantErr: true,
			wantOut: "Error: open /non/existent/schema.yaml: no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := e2e.NewE2ETestHarness(t)
			h.ChangeToTempDir()
			t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

			_, stderr, err := h.ExecuteCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(stderr, tt.wantOut) {
				t.Errorf("Expected stderr to contain '%s', but got: %s", tt.wantOut, stderr)
			}
		})
	}
}
