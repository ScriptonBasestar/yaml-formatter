package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestFiles(t *testing.T) (string, string) {
	tempDir := t.TempDir()
	
	// Create test YAML file
	yamlPath := filepath.Join(tempDir, "test.yml")
	yamlContent := `services:
  web:
    image: nginx
version: '3.8'`
	
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test YAML: %v", err)
	}
	
	// Create schema directory and file
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}
	
	schemaPath := filepath.Join(schemaDir, "compose.yaml")
	schemaContent := `version:
services:`
	
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	
	// Set schema dir env
	os.Setenv("SB_YAML_SCHEMA_DIR", schemaDir)
	
	return yamlPath, schemaDir
}

func TestFormatCommand(t *testing.T) {
	yamlPath, _ := setupTestFiles(t)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"format", "compose", yamlPath})
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Format command failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "Formatting") {
		t.Error("Format output doesn't indicate formatting")
	}
	
	// Check file was modified
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read formatted file: %v", err)
	}
	
	// Should have version first now
	if !strings.HasPrefix(string(content), "version:") {
		t.Error("Formatted file doesn't have version first")
	}
}

func TestFormatDryRun(t *testing.T) {
	yamlPath, _ := setupTestFiles(t)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	// Read original content
	originalContent, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"format", "compose", yamlPath, "--dry-run"})
	
	err = cmd.Execute()
	if err != nil {
		t.Errorf("Format dry-run command failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Error("Dry run output doesn't indicate dry run mode")
	}
	
	// Check file was NOT modified
	afterContent, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read file after dry run: %v", err)
	}
	
	if string(originalContent) != string(afterContent) {
		t.Error("File was modified during dry run")
	}
}

func TestCheckCommand(t *testing.T) {
	yamlPath, _ := setupTestFiles(t)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"check", "compose", yamlPath})
	
	// File is not properly formatted, should exit with error
	_ = cmd.Execute()
	
	output := buf.String()
	if !strings.Contains(output, "needs formatting") {
		t.Error("Check output doesn't indicate file needs formatting")
	}
}

func TestFormatMultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}
	
	// Create multiple test files
	files := []string{"file1.yml", "file2.yml", "file3.yml"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		content := `name: test
version: 1.0`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	// Create schema
	schemaPath := filepath.Join(schemaDir, "test.yaml")
	schemaContent := `version:
name:`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	
	os.Setenv("SB_YAML_SCHEMA_DIR", schemaDir)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	
	pattern := filepath.Join(tempDir, "*.yml")
	cmd.SetArgs([]string{"format", "test", pattern})
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Format multiple files failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "3 file(s)") {
		t.Error("Format output doesn't show correct file count")
	}
}

func TestFormatCommandErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "Missing args",
			args:    []string{"format"},
			wantErr: true,
		},
		{
			name:    "Non-existent schema",
			args:    []string{"format", "non-existent", "file.yml"},
			wantErr: true,
		},
		{
			name:    "Non-existent file",
			args:    []string{"format", "test", "/non/existent/file.yml"},
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

func TestCheckCommandProperlyFormatted(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create properly formatted file
	yamlPath := filepath.Join(tempDir, "formatted.yml")
	yamlContent := `version: '3.8'
services:
  web:
    image: nginx`
	
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test YAML: %v", err)
	}
	
	// Create schema
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}
	
	schemaPath := filepath.Join(schemaDir, "compose.yaml")
	schemaContent := `version:
services:`
	
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	
	os.Setenv("SB_YAML_SCHEMA_DIR", schemaDir)
	defer os.Unsetenv("SB_YAML_SCHEMA_DIR")
	
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"check", "compose", yamlPath})
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Check command failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Error("Check output doesn't indicate file is properly formatted")
	}
}