package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"yaml-formatter/tests/e2e"
)

func TestFormatCommand(t *testing.T) {
	h := testing_utils.NewCLITestHarness(t)
	h.Chdir() // Change to temp directory for test execution

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
		t.Errorf("Format command failed: %v
Stderr: %s", err, stderr)
	}

		testing_utils.AssertOutputContains(t, stdout, "Formatting")

	// Check file was modified
	expectedContent := `version: '3.8'
services:
  web:
    image: nginx
`
	e2e.AssertFileContentEquals(t, h, file, expectedContent)
}

func TestFormatDryRun(t *testing.T) {
	h := testing_utils.NewCLITestHarness(t)
	h.Chdir()

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
		t.Errorf("Format dry-run command failed: %v
Stderr: %s", err, stderr)
	}

	testing_utils.AssertOutputContains(t, stdout, "DRY RUN")

	// Check file was NOT modified
			testing_utils.AssertFileContentEquals(t, h, "test.yml", expectedContent)
}

func TestCheckCommand(t *testing.T) {
	h := testing_utils.NewCLITestHarness(t)
	h.Chdir()

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
		t.Errorf("Check command was expected to fail but didn't.
Stdout: %s
Stderr: %s", stdout, stderr)
	}

		testing_utils.AssertOutputContains(t, stdout, "needs formatting")
}

func TestFormatMultipleFiles(t *testing.T) {
	h := testing_utils.NewCLITestHarness(t)
	h.Chdir()

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
		t.Errorf("Format multiple files failed: %v
Stderr: %s", err, stderr)
	}

	testing_utils.AssertOutputContains(t, stdout, "3 file(s)")

	// Verify files were modified
	expectedContent := `version: 1.0
name: test
`
		for _, file := range files {
		testing_utils.e2e.AssertFileContentEquals(t, h, file, expectedContent)
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
			h := testing_utils.NewCLITestHarness(t)
			h.Chdir()
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
	h := testing_utils.NewCLITestHarness(t)
	h.Chdir()

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

	stdout, stderr, err := h.ExecuteCommand("check", "compose", "formatted.yml")
	testing_utils.AssertOutputContains(t, stdout, "✓")
}
