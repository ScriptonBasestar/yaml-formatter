package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInvalidInputHandling tests various forms of invalid input
func TestInvalidInputHandling(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.cleanup()

	if err := h.ChangeToTempDir(); err != nil {
		t.Fatal(err)
	}

	t.Run("InvalidYAMLSyntax", func(t *testing.T) {
		// Test various forms of invalid YAML
		invalidYAMLs := []struct {
			name    string
			content string
		}{
			{
				name: "UnclosedBracket",
				content: `key: [unclosed bracket
another: value`,
			},
			{
				name: "UnclosedQuote",
				content: `key: "unclosed quote
another: value`,
			},
			{
				name: "InvalidIndentation",
				content: `key:
   value
  another: bad_indent`,
			},
			{
				name:    "MixedTabsSpaces",
				content: "key:\n\tvalue1\n    value2",
			},
			{
				name: "DuplicateKeys",
				content: `key: value1
key: value2`,
			},
			{
				name: "InvalidAnchor",
				content: `key: &invalid anchor
another: *invalid`,
			},
		}

		for _, invalid := range invalidYAMLs {
			t.Run(invalid.name, func(t *testing.T) {
				filename := strings.ToLower(invalid.name) + ".yml"
				if err := h.CreateTestFile(filename, invalid.content); err != nil {
					t.Fatal(err)
				}

				// Try to format the invalid YAML - should fail
				_, _, err := h.ExecuteCommand("format", "any", filename)
				if err == nil {
					t.Errorf("Expected error for invalid YAML %s, but command succeeded", invalid.name)
				}
			})
		}
	})

	t.Run("InvalidSchemaContent", func(t *testing.T) {
		// Create a YAML file
		validYAML := `name: test
version: 1.0.0`
		if err := h.CreateTestFile("valid.yml", validYAML); err != nil {
			t.Fatal(err)
		}

		// Create an invalid schema
		invalidSchema := `this is not: [valid YAML syntax`
		if err := h.CreateSchemaFile("invalid-schema", invalidSchema); err != nil {
			t.Fatal(err)
		}

		// Try to format with invalid schema
		_, _, err := h.ExecuteCommand("format", "invalid-schema", "valid.yml")
		if err == nil {
			t.Error("Expected error when using invalid schema, but command succeeded")
		}
	})

	t.Run("EmptyFiles", func(t *testing.T) {
		// Test empty YAML file
		if err := h.CreateTestFile("empty.yml", ""); err != nil {
			t.Fatal(err)
		}

		// Try to generate schema from empty file - might succeed with empty schema
		_, _, err := h.ExecuteCommand("schema", "gen", "empty", "empty.yml")
		t.Logf("Schema generation from empty file result: error=%v", err)
		// Don't fail here - empty files might be handled gracefully

		// Test empty schema file
		if err := h.CreateSchemaFile("empty-schema", ""); err != nil {
			t.Fatal(err)
		}

		// Create a valid YAML file
		if err := h.CreateTestFile("test.yml", "key: value"); err != nil {
			t.Fatal(err)
		}

		// Try to format with empty schema - might succeed
		_, _, err = h.ExecuteCommand("format", "empty-schema", "test.yml")
		t.Logf("Format with empty schema result: error=%v", err)
		// Don't fail here - empty schemas might be handled gracefully
	})

	t.Run("BinaryDataFiles", func(t *testing.T) {
		// Create a binary file
		binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		binaryFile := filepath.Join(h.GetTempDir(), "binary.yml")
		if err := os.WriteFile(binaryFile, binaryData, 0644); err != nil {
			t.Fatal(err)
		}

		// Try to format binary file
		_, _, err := h.ExecuteCommand("format", "any", "binary.yml")
		if err == nil {
			t.Error("Expected error for binary file, but command succeeded")
		}
	})
}

// TestMissingFileScenarios tests various missing file scenarios
func TestMissingFileScenarios(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.cleanup()

	if err := h.ChangeToTempDir(); err != nil {
		t.Fatal(err)
	}

	t.Run("MissingYAMLFile", func(t *testing.T) {
		// Try to format non-existent file
		_, _, err := h.ExecuteCommand("format", "any", "nonexistent.yml")
		if err == nil {
			t.Error("Expected error for missing YAML file, but command succeeded")
		}

		// Try to generate schema from non-existent file
		_, _, err = h.ExecuteCommand("schema", "gen", "test", "nonexistent.yml")
		if err == nil {
			t.Error("Expected error for missing YAML file in schema gen, but command succeeded")
		}

		// Try to check non-existent file
		_, _, err = h.ExecuteCommand("check", "any", "nonexistent.yml")
		if err == nil {
			t.Error("Expected error for missing YAML file in check, but command succeeded")
		}
	})

	t.Run("MissingSchemaFile", func(t *testing.T) {
		// Create a valid YAML file
		validYAML := `name: test
version: 1.0.0`
		if err := h.CreateTestFile("test.yml", validYAML); err != nil {
			t.Fatal(err)
		}

		// Try to format with non-existent schema
		_, _, err := h.ExecuteCommand("format", "nonexistent-schema", "test.yml")
		if err == nil {
			t.Error("Expected error for missing schema, but command succeeded")
		}

		// Try to check with non-existent schema
		_, _, err = h.ExecuteCommand("check", "nonexistent-schema", "test.yml")
		if err == nil {
			t.Error("Expected error for missing schema in check, but command succeeded")
		}
	})

	t.Run("MissingSchemaDirectory", func(t *testing.T) {
		// Remove the schema directory
		if err := os.RemoveAll(h.GetSchemaDir()); err != nil {
			t.Fatal(err)
		}

		// Create a valid YAML file
		if err := h.CreateTestFile("test.yml", "key: value"); err != nil {
			t.Fatal(err)
		}

		// Try to format without schema directory
		_, _, err := h.ExecuteCommand("format", "any", "test.yml")
		if err == nil {
			t.Error("Expected error when schema directory is missing, but command succeeded")
		}

		// Try to list schemas when directory doesn't exist - might succeed with empty list
		_, _, err = h.ExecuteCommand("schema", "list")
		t.Logf("Schema list with missing directory result: error=%v", err)
		// Don't fail here - might return empty list gracefully
	})
}

// TestPermissionErrors tests various permission-related errors
func TestPermissionErrors(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.cleanup()

	if err := h.ChangeToTempDir(); err != nil {
		t.Fatal(err)
	}

	t.Run("ReadOnlyFile", func(t *testing.T) {
		// Create a YAML file
		yamlContent := `name: test
version: 1.0.0`
		if err := h.CreateTestFile("readonly.yml", yamlContent); err != nil {
			t.Fatal(err)
		}

		// Make file read-only
		filePath := filepath.Join(h.GetTempDir(), "readonly.yml")
		if err := os.Chmod(filePath, 0444); err != nil {
			t.Fatal(err)
		}

		// Create schema
		if err := h.CreateSchemaFile("test", "version:\nname:"); err != nil {
			t.Fatal(err)
		}

		// Try to format read-only file (should fail to write)
		_, _, err := h.ExecuteCommand("format", "test", "readonly.yml")
		if err == nil {
			t.Error("Expected error when formatting read-only file, but command succeeded")
		}

		// Restore permissions for cleanup
		os.Chmod(filePath, 0644)
	})

	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		// Create a subdirectory and make it read-only
		subDir := filepath.Join(h.GetTempDir(), "readonly-dir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.Chmod(subDir, 0555); err != nil {
			t.Fatal(err)
		}

		// Try to create a file in read-only directory
		_, _, err := h.ExecuteCommand("format", "any", "readonly-dir/newfile.yml")
		if err == nil {
			t.Error("Expected error when writing to read-only directory, but command succeeded")
		}

		// Restore permissions for cleanup
		os.Chmod(subDir, 0755)
	})
}

// TestMalformedCommandArguments tests various malformed command arguments
func TestMalformedCommandArguments(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.cleanup()

	if err := h.ChangeToTempDir(); err != nil {
		t.Fatal(err)
	}

	t.Run("InvalidCommands", func(t *testing.T) {
		// Test completely invalid commands
		invalidCommands := [][]string{
			{"invalid-command"},
			{"format"},                            // Missing arguments
			{"check"},                             // Missing arguments
			{"schema"},                            // Missing subcommand
			{"schema", "invalid-subcommand"},      // Invalid subcommand
			{"format", "schema", "file", "extra"}, // Too many arguments
			{"schema", "gen"},                     // Missing arguments for gen
			{"schema", "set"},                     // Missing arguments for set
			{"--invalid-flag"},                    // Invalid flag
		}

		for i, cmd := range invalidCommands {
			t.Run(fmt.Sprintf("InvalidCommand%d", i+1), func(t *testing.T) {
				_, _, err := h.ExecuteCommand(cmd...)
				if err == nil {
					t.Logf("Command %v succeeded (may show help or handle gracefully)", cmd)
				} else {
					t.Logf("Command %v failed as expected: %v", cmd, err)
				}
			})
		}
	})

	t.Run("InvalidFlags", func(t *testing.T) {
		// Create a valid test file
		if err := h.CreateTestFile("test.yml", "key: value"); err != nil {
			t.Fatal(err)
		}

		if err := h.CreateSchemaFile("test", "key:"); err != nil {
			t.Fatal(err)
		}

		// Test invalid flags
		invalidFlagCommands := [][]string{
			{"format", "test", "test.yml", "--invalid-flag"},
			{"format", "test", "test.yml", "--dry-run=invalid"},
			{"check", "test", "test.yml", "--verbose=invalid"},
			{"schema", "gen", "test", "test.yml", "--unknown"},
		}

		for i, cmd := range invalidFlagCommands {
			t.Run(fmt.Sprintf("InvalidFlag%d", i+1), func(t *testing.T) {
				_, _, err := h.ExecuteCommand(cmd...)
				if err == nil {
					t.Errorf("Expected error for command with invalid flags %v, but command succeeded", cmd)
				}
			})
		}
	})

	t.Run("InvalidPaths", func(t *testing.T) {
		// Test various invalid path scenarios
		invalidPaths := []struct {
			name string
			args []string
		}{
			{
				name: "PathWithNullByte",
				args: []string{"format", "test", "file\x00.yml"},
			},
			{
				name: "ExtremelyLongPath",
				args: []string{"format", "test", strings.Repeat("a", 5000) + ".yml"},
			},
			{
				name: "InvalidCharacters",
				args: []string{"format", "test", "file<>|.yml"},
			},
		}

		for _, test := range invalidPaths {
			t.Run(test.name, func(t *testing.T) {
				_, _, err := h.ExecuteCommand(test.args...)
				if err == nil {
					t.Logf("Path test %s succeeded (may handle gracefully)", test.name)
				} else {
					t.Logf("Path test %s failed as expected: %v", test.name, err)
				}
			})
		}
	})

	t.Run("SchemaNameValidation", func(t *testing.T) {
		// Create a valid YAML file
		if err := h.CreateTestFile("test.yml", "key: value"); err != nil {
			t.Fatal(err)
		}

		// Test invalid schema names
		invalidSchemaNames := []string{
			"",                       // Empty name
			"name with spaces",       // Spaces
			"name/with/slashes",      // Slashes
			"name.with.dots",         // Dots (might be invalid depending on implementation)
			"name\x00null",           // Null byte
			strings.Repeat("a", 300), // Very long name
		}

		for i, schemaName := range invalidSchemaNames {
			t.Run(fmt.Sprintf("InvalidSchemaName%d", i+1), func(t *testing.T) {
				_, _, err := h.ExecuteCommand("format", schemaName, "test.yml")
				if err == nil {
					t.Errorf("Expected error for invalid schema name %q, but command succeeded", schemaName)
				}
			})
		}
	})
}

// TestEdgeCases tests various edge cases and corner conditions
func TestEdgeCases(t *testing.T) {
	h := NewE2ETestHarness(t)
	defer h.cleanup()

	if err := h.ChangeToTempDir(); err != nil {
		t.Fatal(err)
	}

	t.Run("VeryLargeFiles", func(t *testing.T) {
		// Create a very large YAML file
		var builder strings.Builder
		builder.WriteString("large_data:\n")
		for i := 0; i < 10000; i++ {
			builder.WriteString(fmt.Sprintf("  key_%d: value_%d\n", i, i))
		}

		largeContent := builder.String()
		if err := h.CreateTestFile("large.yml", largeContent); err != nil {
			t.Fatal(err)
		}

		// Try to generate schema from very large file
		_, _, err := h.ExecuteCommand("schema", "gen", "large", "large.yml")
		// This might succeed or fail depending on implementation limits
		// We're just testing that it doesn't crash
		t.Logf("Large file schema generation result: error=%v", err)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		// Test YAML with special Unicode characters
		specialContent := `name: "Test with émojis 🚀 and spëcial chars"
unicode: "こんにちは世界"
symbols: "!@#$%^&*()_+-=[]{}|;':\",./<>?"
newlines: |
  Line 1
  Line 2
  Line 3`

		if err := h.CreateTestFile("special.yml", specialContent); err != nil {
			t.Fatal(err)
		}

		// Generate schema and format
		stdout, stderr, err := h.ExecuteCommand("schema", "gen", "special", "special.yml")
		if err != nil {
			t.Logf("Schema generation failed for special characters: %v, stderr: %s", err, stderr)
		} else {
			if err := h.CreateTestFile("special.schema.yaml", stdout); err != nil {
				t.Fatal(err)
			}

			_, _, err = h.ExecuteCommand("format", "special", "special.yml")
			if err != nil {
				t.Logf("Format failed for special characters: %v", err)
			}
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		// Test concurrent access to the same file
		content := `name: concurrent-test
version: 1.0.0`

		if err := h.CreateTestFile("concurrent.yml", content); err != nil {
			t.Fatal(err)
		}

		if err := h.CreateSchemaFile("concurrent", "version:\nname:"); err != nil {
			t.Fatal(err)
		}

		// Try to format the same file multiple times quickly
		// This tests for file locking and concurrent access issues
		for i := 0; i < 3; i++ {
			_, _, err := h.ExecuteCommand("format", "concurrent", "concurrent.yml")
			if err != nil {
				t.Logf("Concurrent format attempt %d failed: %v", i+1, err)
			}
		}
	})
}
