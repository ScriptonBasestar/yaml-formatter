package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EFullWorkflow(t *testing.T) {
	h := NewCLITestHarness(t)
	h.Chdir()

	// Step 1: Create a YAML file
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
	h.CreateTestFile("app.yml", yamlContent)

	// Step 2: Generate schema from YAML
	stdout, stderr, err := h.ExecuteCommand("schema", "gen", "app", "app.yml")
	if err != nil {
		t.Fatalf("Schema gen failed: %v\nStderr: %s", err, stderr)
	}

	// Save generated schema
	h.CreateTestFile("app.schema.yaml", stdout)

	// Step 3: Set the schema
	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())
	_, stderr, err = h.ExecuteCommand("schema", "set", "app", "app.schema.yaml")
	if err != nil {
		t.Fatalf("Schema set failed: %v\nStderr: %s", err, stderr)
	}

	// Step 4: List schemas
	stdout, stderr, err = h.ExecuteCommand("schema", "list")
	if err != nil {
		t.Fatalf("Schema list failed: %v\nStderr: %s", err, stderr)
	}

	AssertOutputContains(t, stdout, "app")

	// Step 5: Check formatting
	stdout, stderr, err = h.ExecuteCommand("check", "app", "app.yml")
	// Should fail because file is not formatted
	if err == nil {
		t.Errorf("Check command should have failed for unformatted file.\nStdout: %s\nStderr: %s", stdout, stderr)
	}

	AssertOutputContains(t, stdout, "needs formatting")

	// Step 6: Format the file
	stdout, stderr, err = h.ExecuteCommand("format", "app", "app.yml")
	if err != nil {
		t.Fatalf("Format failed: %v\nStderr: %s", err, stderr)
	}

	// Step 7: Check formatting again
	stdout, stderr, err = h.ExecuteCommand("check", "app", "app.yml")
	if err != nil {
		t.Errorf("Check command failed for formatted file: %v\nStderr: %s", err, stderr)
	}

	// Verify file content
	expectedContent := `name: MyApp
version: 1.0.0
database:
  host: localhost
  port: 5432
services:
  - name: api
    port: 8080
  - name: worker
    port: 9090
`
	AssertFileContentEquals(t, h, "app.yml", expectedContent)
}

func TestE2EMultiDocument(t *testing.T) {
	h := NewCLITestHarness(t)
	h.Chdir()

	// Create multi-document YAML
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
	h.CreateTestFile("k8s.yml", yamlContent)

	// Create K8s schema
	k8sSchema := `apiVersion:
kind:
metadata:
  name:
  namespace:
spec:
  replicas:`
	h.CreateTestFile(filepath.Join("schemas", "k8s.yaml"), k8sSchema)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	// Format the multi-document file
	_, stderr, err := h.ExecuteCommand("format", "k8s", "k8s.yml")
	if err != nil {
		t.Fatalf("Format multi-document failed: %v\nStderr: %s", err, stderr)
	}

	// Read and verify
	content, err := h.ReadTestFile("k8s.yml")
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
	h := NewCLITestHarness(t)
	h.Chdir()

	// Create test file
	originalContent := `services:
  web:
    image: nginx
version: '3.8'`
	h.CreateTestFile("test.yml", originalContent)

	// Create schema
	schemaContent := `version:
services:`
	h.CreateTestFile(filepath.Join("schemas", "compose.yaml"), schemaContent)

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	// Run format with dry-run
	stdout, stderr, err := h.ExecuteCommand("format", "compose", "test.yml", "--dry-run")
	if err != nil {
		t.Fatalf("Dry run failed: %v\nStderr: %s", err, stderr)
	}

	// Check output mentions dry run
	AssertOutputContains(t, stdout, "DRY RUN")

	// Verify file wasn't changed
	AssertFileContentEquals(t, h, "test.yml", originalContent)
}

func TestE2EGitHookConfig(t *testing.T) {
	h := NewCLITestHarness(t)
	h.Chdir()

	// Test show command
	stdout, stderr, err := h.ExecuteCommand("show", "pre-commit")
	if err != nil {
		t.Fatalf("Show pre-commit failed: %v\nStderr: %s", err, stderr)
	}

	// Should contain pre-commit configuration
	AssertOutputContains(t, stdout, "repos:")
	AssertOutputContains(t, stdout, "sb-yaml")
}

func TestE2EWithTestData(t *testing.T) {
	h := NewCLITestHarness(t)
	h.Chdir()

	// Copy test data files
	testFiles := []struct {
		name    string
		content string
	}{
		{
			name:    "simple.yml",
			content: `key: value`,
		},
		{
			name: "complex-nested.yml",
			content: `parent:
  child:
    grandchild: value`,
		},
		{
			name: "with-comments.yml",
			content: `# This is a comment
key: value # Inline comment`,
		},
	}

	for _, tf := range testFiles {
		h.CreateTestFile(tf.name, tf.content)

		// Generate and set schema
		stdout, stderr, err := h.ExecuteCommand("schema", "gen", strings.TrimSuffix(tf.name, ".yml"), tf.name)
		if err != nil {
			t.Fatalf("Schema gen failed for %s: %v\nStderr: %s", tf.name, err, stderr)
		}

		// Save schema
		schemaName := strings.TrimSuffix(tf.name, ".yml")
		h.CreateTestFile(filepath.Join("schemas", schemaName+".schema.yaml"), stdout)
	}

	t.Setenv("SB_YAML_SCHEMA_DIR", h.GetSchemaDir())

	for _, tf := range testFiles {
		// Format file
		_, stderr, err := h.ExecuteCommand("format", strings.TrimSuffix(tf.name, ".yml"), tf.name)
		if err != nil {
			t.Fatalf("Format failed for %s: %v\nStderr: %s", tf.name, err, stderr)
		}

		// Verify file is valid YAML
		_, stderr, err = h.ExecuteCommand("check", strings.TrimSuffix(tf.name, ".yml"), tf.name)
		if err != nil {
			t.Errorf("Check failed for formatted %s: %v\nStderr: %s", tf.name, err, stderr)
		}
	}
}
