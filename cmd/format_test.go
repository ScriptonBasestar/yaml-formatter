//go:build integration

package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"yaml-formatter/tests/e2e"
)

func TestFormatCommand(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir() // Change to temp directory for test execution

	yamlContent := `services:
  web:
    image: nginx
version: '3.8'`
	h.CreateTestFile("test.yml", yamlContent)

	schemaContent := `version:
services:`
	h.CreateTestFile(filepath.Join("schemas", "compose.yaml"), schemaContent)

	// Set schema dir env for the command execution
	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, stderr, err := h.ExecuteCommand("format", "compose", "test.yml")
	if err != nil {
		t.Errorf("Format command failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Formatting") {
		t.Errorf("Expected stdout to contain 'Formatting', got: %s", stdout)
	}

	// Check file was modified
	expectedContent := `version: '3.8'
services:
  web:
    image: nginx
`
	actualContent, err := h.ReadTestFile("test.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if actualContent != expectedContent {
		t.Errorf("Expected file content:\n%s\nGot:\n%s", expectedContent, actualContent)
	}
}

func TestFormatDryRun(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	yamlContent := `services:
  web:
    image: nginx
version: '3.8'`
	h.CreateTestFile("test.yml", yamlContent)

	schemaContent := `version:
services:`
	h.CreateTestFile(filepath.Join("schemas", "compose.yaml"), schemaContent)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, stderr, err := h.ExecuteCommand("format", "compose", "test.yml", "--dry-run")
	if err != nil {
		t.Errorf("Format dry-run command failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "DRY RUN") {
		t.Errorf("Expected stdout to contain 'DRY RUN', got: %s", stdout)
	}

	// Check file was NOT modified - should remain in original format
	originalContent := `services:
  web:
    image: nginx
version: '3.8'`
	actualContent, err := h.ReadTestFile("test.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if actualContent != originalContent {
		t.Errorf("File should not have been modified. Expected:\n%s\nGot:\n%s", originalContent, actualContent)
	}
}

func TestCheckCommand(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	yamlContent := `services:
  web:
    image: nginx
version: '3.8'`
	h.CreateTestFile("test.yml", yamlContent)

	schemaContent := `version:
services:`
	h.CreateTestFile(filepath.Join("schemas", "compose.yaml"), schemaContent)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, stderr, err := h.ExecuteCommand("check", "compose", "test.yml")
	if err == nil {
		t.Errorf("Check command was expected to fail but didn't.\nStdout: %s\nStderr: %s", stdout, stderr)
	}

	if !strings.Contains(stdout, "needs formatting") {
		t.Errorf("Expected stdout to contain 'needs formatting', got: %s", stdout)
	}
}

func TestFormatMultipleFiles(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	// Create multiple test files
	files := []string{"file1.yml", "file2.yml", "file3.yml"}
	for _, file := range files {
		content := `name: test
version: 1.0`
		h.CreateTestFile(file, content)
	}

	// Create schema
	schemaContent := `version:
name:`
	h.CreateTestFile(filepath.Join("schemas", "test.yaml"), schemaContent)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, stderr, err := h.ExecuteCommand("format", "test", "*.yml")
	if err != nil {
		t.Errorf("Format multiple files failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "3 file(s)") {
		t.Errorf("Expected stdout to contain '3 file(s)', got: %s", stdout)
	}

	// Verify files were modified
	expectedContent := `version: 1.0
name: test
`
	for _, file := range files {
		actualContent, err := h.ReadTestFile(file)
		if err != nil {
			t.Fatalf("Failed to read test file %s: %v", file, err)
		}
		if actualContent != expectedContent {
			t.Errorf("File %s: Expected content:\n%s\nGot:\n%s", file, expectedContent, actualContent)
		}
	}
}

func TestFormatCommandErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		wantOut string
	}{
		{
			name:    "Missing args",
			args:    []string{"format"},
			wantErr: true,
			wantOut: "Error: accepts 2 arg(s), received 0",
		},
		{
			name:    "Non-existent schema",
			args:    []string{"format", "non-existent", "file.yml"},
			wantErr: true,
			wantOut: "Error: schema 'non-existent' not found",
		},
		{
			name:    "Non-existent file",
			args:    []string{"format", "test", "/non/existent/file.yml"},
			wantErr: true,
			wantOut: "Error: no YAML files found matching pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := e2e.NewE2ETestHarness(t)
			h.ChangeToTempDir()
			t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir()) // Ensure schema dir is set for all tests

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

func TestCheckCommandProperlyFormatted(t *testing.T) {
	h := e2e.NewE2ETestHarness(t)
	h.ChangeToTempDir()

	yamlContent := `version: '3.8'
services:
  web:
    image: nginx
`
	h.CreateTestFile("formatted.yml", yamlContent)

	schemaContent := `version:
services:`
	h.CreateTestFile(filepath.Join("schemas", "compose.yaml"), schemaContent)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	stdout, _, _ := h.ExecuteCommand("check", "compose", "formatted.yml")
	if !strings.Contains(stdout, "✓") {
		t.Errorf("Expected stdout to contain '✓', got: %s", stdout)
	}
}
