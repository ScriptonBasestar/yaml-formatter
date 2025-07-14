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

// CLITestHarness provides utilities for testing CLI commands
type CLITestHarness struct {
	tempDir    string
	schemaDir  string
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
	originalWD string
	binaryPath string
}

// NewCLITestHarness creates a new test harness
func NewCLITestHarness(t *testing.T) *CLITestHarness {
	tempDir, err := os.MkdirTemp("", "yaml-formatter-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}

	// Save original working directory
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Build the CLI binary
	binaryPath := filepath.Join(tempDir, "yaml-formatter-test-binary")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = originalWD // Build from the project root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, output)
	}

	// Set executable permissions
	if err := os.Chmod(binaryPath, 0755); err != nil {
		t.Fatalf("Failed to set executable permissions: %v", err)
	}

	h := &CLITestHarness{
		tempDir:    tempDir,
		schemaDir:  schemaDir,
		stdout:     new(bytes.Buffer),
		stderr:     new(bytes.Buffer),
		originalWD: originalWD,
		binaryPath: binaryPath,
	}

	t.Cleanup(func() {
		os.RemoveAll(h.tempDir)
		os.Chdir(h.originalWD) // Restore original working directory
	})

	return h
}

// Chdir changes the current working directory to the temp directory
func (h *CLITestHarness) Chdir() {
	if err := os.Chdir(h.tempDir); err != nil {
		panic(fmt.Sprintf("Failed to change directory to %s: %v", h.tempDir, err))
	}
}

// ExecuteCommand executes the CLI command with the given arguments
func (h *CLITestHarness) ExecuteCommand(args ...string) (string, string, error) {
	cmd := exec.Command(h.binaryPath, args...)
	cmd.Stdout = h.stdout
	cmd.Stderr = h.stderr
	cmd.Dir = h.tempDir // Execute in the temp directory

	err := cmd.Run()

	stdout := h.stdout.String()
	stderr := h.stderr.String()

	h.stdout.Reset()
	h.stderr.Reset()

	return stdout, stderr, err
}

// CreateTestFile creates a file in the temporary test directory
func (h *CLITestHarness) CreateTestFile(filename string, content string) error {
	filePath := filepath.Join(h.tempDir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// ReadTestFile reads a file from the temporary test directory
func (h *CLITestHarness) ReadTestFile(filename string) (string, error) {
	filePath := filepath.Join(h.tempDir, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetSchemaDir returns the path to the temporary schema directory
func (h *CLITestHarness) GetSchemaDir() string {
	return h.schemaDir
}

// AssertErrorContains asserts that the error output contains the given string
func AssertErrorContains(t *testing.T, err error, expected string) {
	if err == nil {
		t.Fatalf("Expected an error containing '%s', but got no error", expected)
	}
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("Expected error to contain '%s', but got: %v", expected, err)
	}
}

// AssertOutputContains asserts that the stdout contains the given string
func AssertOutputContains(t *testing.T, output, expected string) {
	if !strings.Contains(output, expected) {
		t.Fatalf("Expected output to contain '%s', but got: %s", expected, output)
	}
}

// AssertOutputEquals asserts that the stdout equals the given string
func AssertOutputEquals(t *testing.T, output, expected string) {
	if output != expected {
		t.Fatalf("Expected output to be '%s', but got: %s", expected, output)
	}
}

// AssertFileContentEquals asserts that the file content equals the given string
func AssertFileContentEquals(t *testing.T, h *CLITestHarness, filename, expected string) {
	content, err := h.ReadTestFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	if content != expected {
		t.Fatalf("File content mismatch for %s.\nExpected:\n%s\nGot:\n%s", filename, expected, content)
	}
}
