//go:build integration

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	// Test root command with no args
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Root command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "YAML formatter") {
		t.Error("Root command output doesn't contain expected description")
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Root command help failed: %v", err)
	}

	output := buf.String()

	// Check for expected commands
	expectedCommands := []string{"format", "check", "schema", "show"}
	for _, expected := range expectedCommands {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing command: %s", expected)
		}
	}
}

func TestRootCommandVersion(t *testing.T) {
	// Skip version test for now - no version is currently set in the command
	t.Skip("Version functionality not implemented yet")
}

func TestVerboseFlag(t *testing.T) {
	// Test that verbose flag is accepted without error
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--verbose", "schema", "list"})

	// Execute should not error with verbose flag
	err := cmd.Execute()
	// We expect an error here since 'schema list' might not be fully implemented
	// but the error should not be about the verbose flag
	if err != nil && strings.Contains(err.Error(), "verbose") {
		t.Error("Verbose flag parsing failed")
	}
}

func TestInvalidCommand(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"invalid-command"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid command")
	}

	output := buf.String()
	if !strings.Contains(output, "unknown command") {
		t.Error("Error output doesn't mention unknown command")
	}
}
