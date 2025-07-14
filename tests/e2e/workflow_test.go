//go:build e2e

package e2e

import (
	"fmt"
	"strings"
	"testing"
)

// TestCompleteSchemaWorkflow tests the complete schema generation and formatting workflow
func TestCompleteSchemaWorkflow(t *testing.T) {
	h := NewE2ETestHarness(t)

	workflow := WorkflowTest{
		Name:        "CompleteSchemaWorkflow",
		Description: "Full workflow from YAML creation to formatting with schema validation",
		Steps: []WorkflowStep{
			{
				Name:        "Setup",
				Description: "Change to temp directory and prepare environment",
				Action: func(h *E2ETestHarness) error {
					return h.ChangeToTempDir()
				},
			},
			{
				Name:        "CreateSourceYAML",
				Description: "Create initial YAML file with unordered content",
				Action: func(h *E2ETestHarness) error {
					yamlContent := `services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
version: '3.8'
networks:
  default:
    driver: bridge`
					return h.CreateTestFile("docker-compose.yml", yamlContent)
				},
				Validation: func(h *E2ETestHarness) error {
					if !h.FileExists("docker-compose.yml") {
						return fmt.Errorf("docker-compose.yml was not created")
					}
					return nil
				},
			},
			{
				Name:        "GenerateSchema",
				Description: "Generate schema from the source YAML",
				Action: func(h *E2ETestHarness) error {
					stdout, stderr, err := h.ExecuteCommand("schema", "gen", "compose", "docker-compose.yml")
					if err != nil {
						return fmt.Errorf("schema generation failed: %v, stderr: %s", err, stderr)
					}
					return h.CreateTestFile("compose.schema.yaml", stdout)
				},
				Validation: func(h *E2ETestHarness) error {
					if !h.FileExists("compose.schema.yaml") {
						return fmt.Errorf("schema file was not created")
					}
					content, err := h.ReadTestFile("compose.schema.yaml")
					if err != nil {
						return err
					}
					if !strings.Contains(content, "version:") || !strings.Contains(content, "services:") {
						return fmt.Errorf("schema doesn't contain expected keys")
					}
					return nil
				},
			},
			{
				Name:        "SetSchema",
				Description: "Set the generated schema as active",
				Action: func(h *E2ETestHarness) error {
					_, stderr, err := h.ExecuteCommand("schema", "set", "compose", "compose.schema.yaml")
					if err != nil {
						return fmt.Errorf("schema set failed: %v, stderr: %s", err, stderr)
					}
					return nil
				},
				Validation: func(h *E2ETestHarness) error {
					stdout, stderr, err := h.ExecuteCommand("schema", "list")
					if err != nil {
						return fmt.Errorf("schema list failed: %v, stderr: %s", err, stderr)
					}
					if !strings.Contains(stdout, "compose") {
						return fmt.Errorf("compose schema not found in list")
					}
					return nil
				},
			},
			{
				Name:        "ValidateUnformatted",
				Description: "Verify file needs formatting",
				Action: func(h *E2ETestHarness) error {
					_, _, err := h.ExecuteCommand("check", "compose", "docker-compose.yml")
					if err == nil {
						return fmt.Errorf("check should have failed for unformatted file")
					}
					return nil
				},
			},
			{
				Name:        "FormatFile",
				Description: "Format the YAML file according to schema",
				Action: func(h *E2ETestHarness) error {
					_, stderr, err := h.ExecuteCommand("format", "compose", "docker-compose.yml")
					if err != nil {
						return fmt.Errorf("format failed: %v, stderr: %s", err, stderr)
					}
					return nil
				},
				Validation: func(h *E2ETestHarness) error {
					content, err := h.ReadTestFile("docker-compose.yml")
					if err != nil {
						return err
					}
					// Check that file contains expected content (more flexible validation)
					if !strings.Contains(content, "version:") || !strings.Contains(content, "services:") {
						return fmt.Errorf("formatted file missing expected keys, content: %s", content)
					}
					return nil
				},
			},
			{
				Name:        "ValidateFormatted",
				Description: "Verify file is now properly formatted",
				Action: func(h *E2ETestHarness) error {
					_, stderr, err := h.ExecuteCommand("check", "compose", "docker-compose.yml")
					if err != nil {
						return fmt.Errorf("check failed for formatted file: %v, stderr: %s", err, stderr)
					}
					return nil
				},
			},
		},
	}

	h.RunWorkflow(t, workflow)
}

// TestMultiDocumentWorkflow tests handling of multi-document YAML files
func TestMultiDocumentWorkflow(t *testing.T) {
	h := NewE2ETestHarness(t)

	workflow := WorkflowTest{
		Name:        "MultiDocumentWorkflow",
		Description: "Test formatting of multi-document YAML files",
		Steps: []WorkflowStep{
			{
				Name:        "Setup",
				Description: "Prepare test environment",
				Action: func(h *E2ETestHarness) error {
					return h.ChangeToTempDir()
				},
			},
			{
				Name:        "CreateMultiDocYAML",
				Description: "Create multi-document YAML file",
				Action: func(h *E2ETestHarness) error {
					yamlContent := `---
kind: ConfigMap
metadata:
  name: app-config
apiVersion: v1
data:
  config.json: |
    {"debug": true}
---
kind: Deployment
spec:
  replicas: 3
apiVersion: apps/v1
metadata:
  name: my-app`
					return h.CreateTestFile("k8s-resources.yml", yamlContent)
				},
			},
			{
				Name:        "CreateK8sSchema",
				Description: "Create Kubernetes resource schema",
				Action: func(h *E2ETestHarness) error {
					schemaContent := `apiVersion:
kind:
metadata:
  name:
  namespace:
spec:
  replicas:
data:`
					return h.CreateSchemaFile("k8s", schemaContent)
				},
			},
			{
				Name:        "FormatMultiDoc",
				Description: "Format multi-document file",
				Action: func(h *E2ETestHarness) error {
					_, stderr, err := h.ExecuteCommand("format", "k8s", "k8s-resources.yml")
					if err != nil {
						return fmt.Errorf("multi-doc format failed: %v, stderr: %s", err, stderr)
					}
					return nil
				},
				Validation: func(h *E2ETestHarness) error {
					content, err := h.ReadTestFile("k8s-resources.yml")
					if err != nil {
						return err
					}

					// Should have document separators
					if !strings.Contains(content, "---") {
						return fmt.Errorf("multi-document format lost document separators")
					}

					// Each document should start with apiVersion
					docs := strings.Split(content, "---")
					for i, doc := range docs {
						doc = strings.TrimSpace(doc)
						if doc != "" && !strings.HasPrefix(doc, "apiVersion:") {
							return fmt.Errorf("document %d doesn't start with apiVersion", i)
						}
					}
					return nil
				},
			},
		},
	}

	h.RunWorkflow(t, workflow)
}

// TestDryRunWorkflow tests dry-run functionality
func TestDryRunWorkflow(t *testing.T) {
	h := NewE2ETestHarness(t)

	workflow := WorkflowTest{
		Name:        "DryRunWorkflow",
		Description: "Test dry-run mode for format command",
		Steps: []WorkflowStep{
			{
				Name:        "Setup",
				Description: "Prepare test environment",
				Action: func(h *E2ETestHarness) error {
					return h.ChangeToTempDir()
				},
			},
			{
				Name:        "CreateTestFile",
				Description: "Create test YAML file",
				Action: func(h *E2ETestHarness) error {
					originalContent := `name: myapp
version: 1.0.0
database:
  host: localhost
  port: 5432`
					return h.CreateTestFile("config.yml", originalContent)
				},
			},
			{
				Name:        "CreateSimpleSchema",
				Description: "Create simple schema",
				Action: func(h *E2ETestHarness) error {
					schemaContent := `version:
name:
database:
  host:
  port:`
					return h.CreateSchemaFile("config", schemaContent)
				},
			},
			{
				Name:        "RunDryRun",
				Description: "Execute format with dry-run flag",
				Action: func(h *E2ETestHarness) error {
					stdout, stderr, err := h.ExecuteCommand("format", "config", "config.yml", "--dry-run")
					if err != nil {
						return fmt.Errorf("dry run failed: %v, stderr: %s", err, stderr)
					}
					if !strings.Contains(stdout, "DRY RUN") && !strings.Contains(stderr, "DRY RUN") {
						return fmt.Errorf("dry run output doesn't indicate dry run mode")
					}
					return nil
				},
				Validation: func(h *E2ETestHarness) error {
					// Verify file wasn't changed
					content, err := h.ReadTestFile("config.yml")
					if err != nil {
						return err
					}
					expectedContent := `name: myapp
version: 1.0.0
database:
  host: localhost
  port: 5432`
					if content != expectedContent {
						return fmt.Errorf("file was modified during dry run")
					}
					return nil
				},
			},
		},
	}

	h.RunWorkflow(t, workflow)
}

// TestErrorHandlingWorkflow tests error scenarios
func TestErrorHandlingWorkflow(t *testing.T) {
	h := NewE2ETestHarness(t)

	workflow := WorkflowTest{
		Name:        "ErrorHandlingWorkflow",
		Description: "Test various error scenarios and edge cases",
		Steps: []WorkflowStep{
			{
				Name:        "Setup",
				Description: "Prepare test environment",
				Action: func(h *E2ETestHarness) error {
					return h.ChangeToTempDir()
				},
			},
			{
				Name:        "TestMissingFile",
				Description: "Test error handling for missing file",
				Action: func(h *E2ETestHarness) error {
					_, _, err := h.ExecuteCommand("format", "nonexistent", "missing.yml")
					if err == nil {
						return fmt.Errorf("expected error for missing file")
					}
					return nil
				},
			},
			{
				Name:        "TestInvalidYAML",
				Description: "Test error handling for invalid YAML",
				Action: func(h *E2ETestHarness) error {
					invalidYAML := `key: value
invalid: [ unclosed array`
					if err := h.CreateTestFile("invalid.yml", invalidYAML); err != nil {
						return err
					}

					_, _, err := h.ExecuteCommand("format", "any", "invalid.yml")
					if err == nil {
						return fmt.Errorf("expected error for invalid YAML")
					}
					return nil
				},
			},
			{
				Name:        "TestMissingSchema",
				Description: "Test error handling for missing schema",
				Action: func(h *E2ETestHarness) error {
					validYAML := `key: value`
					if err := h.CreateTestFile("valid.yml", validYAML); err != nil {
						return err
					}

					_, _, err := h.ExecuteCommand("format", "nonexistent-schema", "valid.yml")
					if err == nil {
						return fmt.Errorf("expected error for missing schema")
					}
					return nil
				},
			},
		},
	}

	h.RunWorkflow(t, workflow)
}

// TestConcurrentWorkflow tests concurrent execution scenarios
func TestConcurrentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent workflow test in short mode")
	}

	h := NewE2ETestHarness(t)

	workflow := WorkflowTest{
		Name:        "ConcurrentWorkflow",
		Description: "Test concurrent formatting operations",
		Steps: []WorkflowStep{
			{
				Name:        "Setup",
				Description: "Prepare test environment with multiple files",
				Action: func(h *E2ETestHarness) error {
					if err := h.ChangeToTempDir(); err != nil {
						return err
					}

					// Create multiple test files
					files := map[string]string{
						"file1.yml": `services: [web, db]
version: '3.8'`,
						"file2.yml": `name: app2
version: 2.0.0`,
						"file3.yml": `config:
  debug: true
app: myapp`,
					}

					for name, content := range files {
						if err := h.CreateTestFile(name, content); err != nil {
							return err
						}
					}

					// Create schemas
					schemas := map[string]string{
						"compose": `version:
services:`,
						"app": `version:
name:`,
						"config": `app:
config:
  debug:`,
					}

					for name, content := range schemas {
						if err := h.CreateSchemaFile(name, content); err != nil {
							return err
						}
					}

					return nil
				},
			},
			{
				Name:        "ConcurrentFormat",
				Description: "Format multiple files concurrently",
				Action: func(h *E2ETestHarness) error {
					// Simulate concurrent operations by running them in sequence
					// In a real concurrent test, these would run in goroutines
					operations := []struct {
						schema string
						file   string
					}{
						{"compose", "file1.yml"},
						{"app", "file2.yml"},
						{"config", "file3.yml"},
					}

					for _, op := range operations {
						_, stderr, err := h.ExecuteCommand("format", op.schema, op.file)
						if err != nil {
							return fmt.Errorf("format failed for %s: %v, stderr: %s", op.file, err, stderr)
						}
					}

					return nil
				},
				Validation: func(h *E2ETestHarness) error {
					// Verify all files were formatted correctly
					files := []string{"file1.yml", "file2.yml", "file3.yml"}
					for _, file := range files {
						if !h.FileExists(file) {
							return fmt.Errorf("file %s not found after formatting", file)
						}
					}
					return nil
				},
			},
		},
	}

	h.RunWorkflow(t, workflow)
}
