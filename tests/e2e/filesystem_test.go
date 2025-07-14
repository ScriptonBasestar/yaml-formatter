//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestFilesystemDependencies tests various filesystem-related scenarios
func TestFilesystemDependencies(t *testing.T) {
	h := NewE2ETestHarness(t)

	// Test 1: Path traversal protection
	t.Run("PathTraversalProtection", func(t *testing.T) {
		err := h.CreateTestFile("../../../etc/passwd", "malicious content")
		if err == nil || !strings.Contains(err.Error(), "attempted to write outside temp directory") {
			t.Errorf("Expected path traversal protection, got: %v", err)
		}
	})

	// Test 2: Nested directory creation
	t.Run("NestedDirectoryCreation", func(t *testing.T) {
		if err := h.ChangeToTempDir(); err != nil {
			t.Fatal(err)
		}

		nestedPath := filepath.Join("deep", "nested", "path", "file.yml")
		content := "key: value"

		if err := h.CreateTestFile(nestedPath, content); err != nil {
			t.Fatalf("Failed to create nested file: %v", err)
		}

		if !h.FileExists(nestedPath) {
			t.Error("Nested file was not created")
		}

		readContent, err := h.ReadTestFile(nestedPath)
		if err != nil {
			t.Fatalf("Failed to read nested file: %v", err)
		}

		if readContent != content {
			t.Errorf("Expected %q, got %q", content, readContent)
		}
	})

	// Test 3: File permissions
	t.Run("FilePermissions", func(t *testing.T) {
		if err := h.ChangeToTempDir(); err != nil {
			t.Fatal(err)
		}

		filename := "test-permissions.yml"
		content := "test: value"

		if err := h.CreateTestFile(filename, content); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Check file permissions
		filePath := filepath.Join(h.GetTempDir(), filename)
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		expectedMode := os.FileMode(0644)
		if info.Mode().Perm() != expectedMode {
			t.Errorf("Expected permissions %v, got %v", expectedMode, info.Mode().Perm())
		}
	})

	// Test 4: Large file handling
	t.Run("LargeFileHandling", func(t *testing.T) {
		if err := h.ChangeToTempDir(); err != nil {
			t.Fatal(err)
		}

		// Create a moderately large YAML file
		var builder strings.Builder
		builder.WriteString("large_data:\n")
		for i := 0; i < 1000; i++ {
			builder.WriteString("  key_")
			builder.WriteString(strings.Repeat("0", 4-len(strconv.Itoa(i))))
			builder.WriteString(strconv.Itoa(i))
			builder.WriteString(": value_")
			builder.WriteString(strconv.Itoa(i))
			builder.WriteString("\n")
		}

		largeContent := builder.String()
		filename := "large-file.yml"

		if err := h.CreateTestFile(filename, largeContent); err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		readContent, err := h.ReadTestFile(filename)
		if err != nil {
			t.Fatalf("Failed to read large file: %v", err)
		}

		if len(readContent) != len(largeContent) {
			t.Errorf("Content length mismatch: expected %d, got %d", len(largeContent), len(readContent))
		}
	})

	// Test 5: Special characters in filenames
	t.Run("SpecialCharacterFilenames", func(t *testing.T) {
		if err := h.ChangeToTempDir(); err != nil {
			t.Fatal(err)
		}

		specialFiles := []string{
			"file with spaces.yml",
			"file-with-dashes.yml",
			"file_with_underscores.yml",
			"file.with.dots.yml",
			"file@with@symbols.yml",
		}

		for _, filename := range specialFiles {
			content := "test: " + filename

			if err := h.CreateTestFile(filename, content); err != nil {
				t.Errorf("Failed to create file with special chars %q: %v", filename, err)
				continue
			}

			if !h.FileExists(filename) {
				t.Errorf("File %q was not created", filename)
				continue
			}

			readContent, err := h.ReadTestFile(filename)
			if err != nil {
				t.Errorf("Failed to read file %q: %v", filename, err)
				continue
			}

			if readContent != content {
				t.Errorf("Content mismatch for %q: expected %q, got %q", filename, content, readContent)
			}
		}
	})

	// Test 6: Concurrent file operations
	t.Run("ConcurrentFileOperations", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping concurrent operations test in short mode")
		}

		if err := h.ChangeToTempDir(); err != nil {
			t.Fatal(err)
		}

		// Create multiple files "concurrently" (sequentially for test simplicity)
		numFiles := 10
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("concurrent-file-%d.yml", i)
			content := fmt.Sprintf("file_id: %d\ndata: test_data_%d", i, i)

			if err := h.CreateTestFile(filename, content); err != nil {
				t.Errorf("Failed to create concurrent file %d: %v", i, err)
			}
		}

		// Verify all files exist and have correct content
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("concurrent-file-%d.yml", i)
			expectedContent := fmt.Sprintf("file_id: %d\ndata: test_data_%d", i, i)

			if !h.FileExists(filename) {
				t.Errorf("Concurrent file %d was not created", i)
				continue
			}

			content, err := h.ReadTestFile(filename)
			if err != nil {
				t.Errorf("Failed to read concurrent file %d: %v", i, err)
				continue
			}

			if content != expectedContent {
				t.Errorf("Content mismatch for concurrent file %d", i)
			}
		}
	})

	// Test 7: Directory listing
	t.Run("DirectoryListing", func(t *testing.T) {
		if err := h.ChangeToTempDir(); err != nil {
			t.Fatal(err)
		}

		// Create test files
		testFiles := []string{"file1.yml", "file2.yml", "subdir/file3.yml"}
		for _, filename := range testFiles {
			if err := h.CreateTestFile(filename, "test: content"); err != nil {
				t.Fatalf("Failed to create test file %s: %v", filename, err)
			}
		}

		// List files
		files, err := h.ListFiles()
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}

		// Check that all test files are listed
		for _, expectedFile := range testFiles {
			found := false
			for _, actualFile := range files {
				if actualFile == expectedFile {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file %s not found in listing", expectedFile)
			}
		}
	})
}
