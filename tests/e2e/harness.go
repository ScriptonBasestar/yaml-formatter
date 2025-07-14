package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// E2ETestHarness provides utilities for end-to-end testing
type E2ETestHarness struct {
	tempDir    string
	schemaDir  string
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
	originalWD string
	binaryPath string
	envVars    map[string]string
}

// NewE2ETestHarness creates a new E2E test harness with isolated environment
func NewE2ETestHarness(t *testing.T) *E2ETestHarness {
	// Create isolated temporary directory
	tempDir, err := os.MkdirTemp("", "yaml-formatter-e2e-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create schema directory
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}

	// Save original working directory
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Get the pre-built binary path
	binaryPath, err := getBinaryPath()
	if err != nil {
		t.Fatalf("Failed to get binary path: %v", err)
	}

	h := &E2ETestHarness{
		tempDir:    tempDir,
		schemaDir:  schemaDir,
		stdout:     new(bytes.Buffer),
		stderr:     new(bytes.Buffer),
		originalWD: originalWD,
		binaryPath: binaryPath,
		envVars:    make(map[string]string),
	}

	// Set default environment variables
	h.setEnvVar("SB_YAML_SCHEMA_DIR", schemaDir)

	// Register cleanup
	t.Cleanup(func() {
		h.cleanup()
	})

	return h
}

// setEnvVar sets an environment variable for the test
func (h *E2ETestHarness) setEnvVar(key, value string) {
	h.envVars[key] = value
}

// GetEnvVar gets an environment variable value
func (h *E2ETestHarness) GetEnvVar(key string) string {
	return h.envVars[key]
}

// ChangeToTempDir changes the current working directory to the temp directory
func (h *E2ETestHarness) ChangeToTempDir() error {
	return os.Chdir(h.tempDir)
}

// ExecuteCommand executes the CLI command with the given arguments in isolated environment
func (h *E2ETestHarness) ExecuteCommand(args ...string) (string, string, error) {
	cmd := exec.Command(h.binaryPath, args...)
	cmd.Stdout = h.stdout
	cmd.Stderr = h.stderr
	cmd.Dir = h.tempDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range h.envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	err := cmd.Run()

	stdout := h.stdout.String()
	stderr := h.stderr.String()

	h.stdout.Reset()
	h.stderr.Reset()

	return stdout, stderr, err
}

// CreateTestFile creates a file in the temporary test directory
func (h *E2ETestHarness) CreateTestFile(filename string, content string) error {
	// Handle nested directories
	filePath := filepath.Join(h.tempDir, filename)
	dir := filepath.Dir(filePath)

	// Ensure we're still within our temp directory (security check)
	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, h.tempDir) {
		return fmt.Errorf("attempted to write outside temp directory: %s", filename)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// ReadTestFile reads a file from the temporary test directory
func (h *E2ETestHarness) ReadTestFile(filename string) (string, error) {
	filePath := filepath.Join(h.tempDir, filename)

	// Ensure we're still within our temp directory (security check)
	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, h.tempDir) {
		return "", fmt.Errorf("attempted to read outside temp directory: %s", filename)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// FileExists checks if a file exists in the temp directory
func (h *E2ETestHarness) FileExists(filename string) bool {
	filePath := filepath.Join(h.tempDir, filename)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// GetTempDir returns the temporary directory path
func (h *E2ETestHarness) GetTempDir() string {
	return h.tempDir
}

// GetSchemaDir returns the path to the temporary schema directory
func (h *E2ETestHarness) GetSchemaDir() string {
	return h.schemaDir
}

// CreateSchemaFile creates a schema file in the schema directory
func (h *E2ETestHarness) CreateSchemaFile(name string, content string) error {
	filename := fmt.Sprintf("%s.yaml", name)
	schemaPath := filepath.Join(h.schemaDir, filename)
	return os.WriteFile(schemaPath, []byte(content), 0644)
}

// ListFiles lists all files in the temp directory
func (h *E2ETestHarness) ListFiles() ([]string, error) {
	var files []string
	err := filepath.Walk(h.tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(h.tempDir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	return files, err
}

// cleanup removes temporary files and restores working directory
func (h *E2ETestHarness) cleanup() {
	os.RemoveAll(h.tempDir)
	os.Chdir(h.originalWD)
}

// WorkflowTest represents a complete E2E workflow test
type WorkflowTest struct {
	Name        string
	Description string
	Steps       []WorkflowStep
}

// WorkflowStep represents a single step in an E2E workflow
type WorkflowStep struct {
	Name        string
	Action      func(*E2ETestHarness) error
	Validation  func(*E2ETestHarness) error
	Description string
}

// RunWorkflow executes a complete workflow test
func (h *E2ETestHarness) RunWorkflow(t *testing.T, workflow WorkflowTest) {
	t.Logf("Starting workflow: %s - %s", workflow.Name, workflow.Description)

	for i, step := range workflow.Steps {
		t.Logf("Step %d: %s - %s", i+1, step.Name, step.Description)

		// Execute action
		if step.Action != nil {
			if err := step.Action(h); err != nil {
				t.Fatalf("Step %d (%s) action failed: %v", i+1, step.Name, err)
			}
		}

		// Run validation
		if step.Validation != nil {
			if err := step.Validation(h); err != nil {
				t.Fatalf("Step %d (%s) validation failed: %v", i+1, step.Name, err)
			}
		}

		t.Logf("Step %d completed successfully", i+1)
	}

	t.Logf("Workflow completed successfully: %s", workflow.Name)
}

// getBinaryPath returns the path to the test binary
func getBinaryPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	projectRoot := filepath.Join(wd, "../..")
	binaryPath := filepath.Join(projectRoot, "yaml-formatter-test")

	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return "", err
	}

	return binaryPath, nil
}
