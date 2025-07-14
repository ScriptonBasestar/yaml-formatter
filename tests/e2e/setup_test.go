//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestMain sets up test environment and builds binaries
func TestMain(m *testing.M) {
	// Build test binary
	err := buildTestBinary()
	if err != nil {
		panic("Failed to build test binary: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup (test binary cleanup handled by individual test cleanup)
	os.Exit(code)
}

// buildTestBinary builds the CLI binary for testing
func buildTestBinary() error {
	// Get the project root directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Move up to project root (from tests/e2e to root)
	projectRoot := filepath.Join(wd, "../..")

	// Build command
	cmd := exec.Command("go", "build", "-o", "yaml-formatter-test", ".")
	cmd.Dir = projectRoot

	// Run build
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}

	// Verify binary was created
	binaryPath := filepath.Join(projectRoot, "yaml-formatter-test")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return err
	}

	return nil
}
