//go:build smoke

package smoke

import (
	"os/exec"
	"strings"
	"testing"
)

// TestBinaryExists verifies the binary can be executed
func TestBinaryExists(t *testing.T) {
	cmd := exec.Command("../../yaml-formatter-test", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Binary execution failed: %v, output: %s", err, string(output))
	}
	
	if !strings.Contains(string(output), "sb-yaml") {
		t.Errorf("Expected help output to contain 'sb-yaml', got: %s", string(output))
	}
}

// TestBinaryVersion verifies help command works (version not implemented)
func TestBinaryVersion(t *testing.T) {
	cmd := exec.Command("../../yaml-formatter-test", "help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v, output: %s", err, string(output))
	}
	
	// Should contain help information
	if !strings.Contains(string(output), "Available Commands") {
		t.Error("Help command should contain 'Available Commands'")
	}
}

// TestBasicFormatOperation smoke test for basic formatting
func TestBasicFormatOperation(t *testing.T) {
	// This is a basic smoke test that would ideally test against
	// a live environment or a simple format operation
	t.Skip("Skipping basic format operation test - requires environment setup")
}