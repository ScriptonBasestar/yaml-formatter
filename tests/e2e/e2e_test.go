package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EFullWorkflow(t *testing.T) {
	// Skip if binary not built
	binPath := "../../sb-yaml"
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binary not built, run 'go build' first")
	}
	
	tempDir := t.TempDir()
	
	// Step 1: Create a YAML file
	yamlPath := filepath.Join(tempDir, "app.yml")
	yamlContent := `database:
  host: localhost
  port: 5432
name: MyApp
version: 1.0.0
services:
  - name: api
    port: 8080
  - name: worker
    port: 9090`
	
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Step 2: Generate schema from YAML
	cmd := exec.Command(binPath, "schema", "gen", "app", yamlPath)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Schema gen failed: %v", err)
	}
	
	// Save generated schema
	schemaPath := filepath.Join(tempDir, "app.schema.yaml")
	if err := os.WriteFile(schemaPath, output, 0644); err != nil {
		t.Fatalf("Failed to save schema: %v", err)
	}
	
	// Step 3: Set the schema
	schemaDir := filepath.Join(tempDir, "schemas")
	cmd = exec.Command(binPath, "schema", "set", "app", schemaPath)
	cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Schema set failed: %v", err)
	}
	
	// Step 4: List schemas
	cmd = exec.Command(binPath, "schema", "list")
	cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Schema list failed: %v", err)
	}
	
	if !strings.Contains(string(output), "app") {
		t.Error("Schema list doesn't contain 'app' schema")
	}
	
	// Step 5: Check formatting
	cmd = exec.Command(binPath, "check", "app", yamlPath)
	cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
	err = cmd.Run()
	// Should fail because file is not formatted
	if err == nil {
		t.Error("Check command should have failed for unformatted file")
	}
	
	// Step 6: Format the file
	cmd = exec.Command(binPath, "format", "app", yamlPath)
	cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	
	// Step 7: Check formatting again
	cmd = exec.Command(binPath, "check", "app", yamlPath)
	cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
	if err := cmd.Run(); err != nil {
		t.Error("Check command failed for formatted file")
	}
	
	// Verify file content
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read formatted file: %v", err)
	}
	
	// Should start with 'name:' based on our schema
	if !strings.HasPrefix(string(content), "name:") {
		t.Error("Formatted file doesn't start with 'name:'")
	}
}

func TestE2EMultiDocument(t *testing.T) {
	binPath := "../../sb-yaml"
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binary not built, run 'go build' first")
	}
	
	tempDir := t.TempDir()
	
	// Create multi-document YAML
	yamlPath := filepath.Join(tempDir, "k8s.yml")
	yamlContent := `---
metadata:
  name: my-app
  namespace: default
apiVersion: v1
kind: Namespace
---
spec:
  replicas: 3
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app`
	
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create K8s schema
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}
	
	k8sSchema := `apiVersion:
kind:
metadata:
  name:
  namespace:
spec:
  replicas:`
	
	schemaPath := filepath.Join(schemaDir, "k8s.yaml")
	if err := os.WriteFile(schemaPath, []byte(k8sSchema), 0644); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	
	// Format the multi-document file
	cmd := exec.Command(binPath, "format", "k8s", yamlPath)
	cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Format multi-document failed: %v", err)
	}
	
	// Read and verify
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read formatted file: %v", err)
	}
	
	// Should have document separators
	if !strings.Contains(string(content), "---") {
		t.Error("Multi-document format lost document separators")
	}
	
	// Each document should start with apiVersion
	docs := strings.Split(string(content), "---")
	for i, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc != "" && !strings.HasPrefix(doc, "apiVersion:") {
			t.Errorf("Document %d doesn't start with apiVersion", i)
		}
	}
}

func TestE2EDryRun(t *testing.T) {
	binPath := "../../sb-yaml"
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binary not built, run 'go build' first")
	}
	
	tempDir := t.TempDir()
	
	// Create test file
	yamlPath := filepath.Join(tempDir, "test.yml")
	originalContent := `services:
  web:
    image: nginx
version: '3.8'`
	
	if err := os.WriteFile(yamlPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create schema
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}
	
	schemaContent := `version:
services:`
	schemaPath := filepath.Join(schemaDir, "compose.yaml")
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	
	// Run format with dry-run
	cmd := exec.Command(binPath, "format", "compose", yamlPath, "--dry-run")
	cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}
	
	// Check output mentions dry run
	if !strings.Contains(string(output), "DRY RUN") {
		t.Error("Dry run output doesn't mention DRY RUN")
	}
	
	// Verify file wasn't changed
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	
	if string(content) != originalContent {
		t.Error("File was modified during dry run")
	}
}

func TestE2EGitHookConfig(t *testing.T) {
	binPath := "../../sb-yaml"
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binary not built, run 'go build' first")
	}
	
	// Test show command
	cmd := exec.Command(binPath, "show", "pre-commit")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Show pre-commit failed: %v", err)
	}
	
	// Should contain pre-commit configuration
	if !strings.Contains(string(output), "repos:") {
		t.Error("Pre-commit output doesn't contain 'repos:'")
	}
	
	if !strings.Contains(string(output), "sb-yaml") {
		t.Error("Pre-commit output doesn't mention sb-yaml")
	}
}

func TestE2EWithTestData(t *testing.T) {
	binPath := "../../sb-yaml"
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binary not built, run 'go build' first")
	}
	
	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "schemas")
	
	// Copy test data files
	testFiles := []string{
		"../../tests/testdata/valid/simple.yml",
		"../../tests/testdata/valid/complex-nested.yml",
		"../../tests/testdata/valid/with-comments.yml",
	}
	
	for _, srcPath := range testFiles {
		content, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("Failed to read test file %s: %v", srcPath, err)
		}
		
		destPath := filepath.Join(tempDir, filepath.Base(srcPath))
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			t.Fatalf("Failed to copy test file: %v", err)
		}
		
		// Generate and set schema
		cmd := exec.Command(binPath, "schema", "gen", "test", destPath)
		schemaOutput, err := cmd.Output()
		if err != nil {
			t.Fatalf("Schema gen failed for %s: %v", destPath, err)
		}
		
		// Save schema
		schemaName := strings.TrimSuffix(filepath.Base(destPath), ".yml")
		schemaPath := filepath.Join(tempDir, schemaName+".schema.yaml")
		if err := os.WriteFile(schemaPath, schemaOutput, 0644); err != nil {
			t.Fatalf("Failed to save schema: %v", err)
		}
		
		// Set schema
		cmd = exec.Command(binPath, "schema", "set", schemaName, schemaPath)
		cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Schema set failed: %v", err)
		}
		
		// Format file
		cmd = exec.Command(binPath, "format", schemaName, destPath)
		cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Format failed for %s: %v", destPath, err)
		}
		
		// Verify file is valid YAML
		cmd = exec.Command(binPath, "check", schemaName, destPath)
		cmd.Env = append(os.Environ(), "SB_YAML_SCHEMA_DIR="+schemaDir)
		if err := cmd.Run(); err != nil {
			t.Errorf("Check failed for formatted %s: %v", destPath, err)
		}
	}
}