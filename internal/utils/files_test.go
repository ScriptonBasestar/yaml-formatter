package utils

import (
	"github.com/spf13/afero"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	fh := NewFileHandler(fs)

	// Create test file
	testPath := "/test/file.yml"
	testContent := []byte("name: test\nversion: 1.0.0")

	if err := fs.MkdirAll("/test", 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := afero.WriteFile(fs, testPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read file
	content, err := fh.ReadFile(testPath)
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Read content mismatch: got %s, want %s", content, testContent)
	}
}

func TestWriteFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	fh := NewFileHandler(fs)

	testPath := "/test/output.yml"
	testContent := []byte("name: output\nversion: 2.0.0")

	// Write file
	err := fh.WriteFile(testPath, testContent)
	if err != nil {
		t.Errorf("WriteFile failed: %v", err)
	}

	// Verify file was written
	exists, err := afero.Exists(fs, testPath)
	if err != nil {
		t.Fatalf("Failed to check file existence: %v", err)
	}

	if !exists {
		t.Error("File was not created")
	}

	// Read back and verify
	content, err := afero.ReadFile(fs, testPath)
	if err != nil {
		t.Fatalf("Failed to read back file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Written content mismatch: got %s, want %s", content, testContent)
	}
}

func TestFileExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	fh := NewFileHandler(fs)

	// Create test file
	testPath := "/test/exists.yml"
	if err := fs.MkdirAll("/test", 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := afero.WriteFile(fs, testPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	exists, err := fh.FileExists(testPath)
	if err != nil {
		t.Errorf("FileExists failed: %v", err)
	}

	if !exists {
		t.Error("FileExists returned false for existing file")
	}

	// Test non-existing file
	exists, err = fh.FileExists("/test/not-exists.yml")
	if err != nil {
		t.Errorf("FileExists failed: %v", err)
	}

	if exists {
		t.Error("FileExists returned true for non-existing file")
	}
}

func TestIsYAMLFile(t *testing.T) {
	fh := NewFileHandler(nil)

	tests := []struct {
		path     string
		expected bool
	}{
		{"file.yml", true},
		{"file.yaml", true},
		{"file.YML", true},
		{"file.YAML", true},
		{"complex.test.yml", true},
		{"file.json", false},
		{"file.txt", false},
		{"noextension", false},
		{".yml", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := fh.IsYAMLFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsYAMLFile(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExpandGlob(t *testing.T) {
	fs := afero.NewMemMapFs()
	fh := NewFileHandler(fs)

	// Create test directory structure
	files := []string{
		"/project/docker-compose.yml",
		"/project/docker-compose.prod.yml",
		"/project/k8s/deployment.yaml",
		"/project/k8s/service.yaml",
		"/project/k8s/configmap.yml",
		"/project/config/app.yml",
		"/project/README.md",
		"/project/.hidden.yml",
	}

	for _, file := range files {
		dir := filepath.Dir(file)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := afero.WriteFile(fs, file, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	tests := []struct {
		name     string
		patterns []string
		expected int
	}{
		{
			name:     "Single file",
			patterns: []string{"/project/docker-compose.yml"},
			expected: 1,
		},
		{
			name:     "Wildcard pattern",
			patterns: []string{"/project/*.yml"},
			expected: 3, // docker-compose.yml, docker-compose.prod.yml, .hidden.yml
		},
		{
			name:     "Recursive pattern",
			patterns: []string{"/project/**/*.yml"},
			expected: 5, // All .yml files including subdirectories
		},
		{
			name:     "Multiple patterns",
			patterns: []string{"/project/*.yml", "/project/k8s/*.yaml"},
			expected: 5, // 3 yml in root + 2 yaml in k8s
		},
		{
			name:     "No matches",
			patterns: []string{"/project/*.json"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := fh.ExpandGlob(tt.patterns)
			if err != nil {
				t.Errorf("ExpandGlob failed: %v", err)
			}

			if len(files) != tt.expected {
				t.Errorf("ExpandGlob returned %d files, expected %d", len(files), tt.expected)
				t.Logf("Files: %v", files)
			}
		})
	}
}

func TestGetAbsolutePath(t *testing.T) {
	fh := NewFileHandler(nil)

	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{
			name:     "Relative path",
			input:    "./file.yml",
			hasError: false,
		},
		{
			name:     "Absolute path",
			input:    "/absolute/path/file.yml",
			hasError: false,
		},
		{
			name:     "Home directory",
			input:    "~/file.yml",
			hasError: false,
		},
		{
			name:     "Parent directory",
			input:    "../file.yml",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fh.GetAbsolutePath(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("GetAbsolutePath(%s) error = %v, wantErr %v", tt.input, err, tt.hasError)
			}

			if !tt.hasError && result == "" {
				t.Errorf("GetAbsolutePath(%s) returned empty string", tt.input)
			}
		})
	}
}

func TestBackupFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	fh := NewFileHandler(fs)

	// Create original file
	originalPath := "/test/original.yml"
	originalContent := []byte("original content")

	if err := fs.MkdirAll("/test", 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := afero.WriteFile(fs, originalPath, originalContent, 0644); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Create backup
	backupPath, err := fh.BackupFile(originalPath)
	if err != nil {
		t.Errorf("BackupFile failed: %v", err)
	}

	if backupPath == "" {
		t.Error("BackupFile returned empty path")
	}

	// Verify backup exists
	exists, err := afero.Exists(fs, backupPath)
	if err != nil {
		t.Fatalf("Failed to check backup existence: %v", err)
	}

	if !exists {
		t.Error("Backup file was not created")
	}

	// Verify backup content
	backupContent, err := afero.ReadFile(fs, backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != string(originalContent) {
		t.Error("Backup content doesn't match original")
	}
}

func TestListYAMLFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	fh := NewFileHandler(fs)

	// Create test files
	testDir := "/test/yaml"
	files := []string{
		"file1.yml",
		"file2.yaml",
		"file3.json",
		"file4.txt",
		".hidden.yml",
		"nested/file5.yml",
	}

	if err := fs.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := fs.MkdirAll(filepath.Join(testDir, "nested"), 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	for _, file := range files {
		path := filepath.Join(testDir, file)
		if err := afero.WriteFile(fs, path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	// List YAML files (non-recursive)
	yamlFiles, err := fh.ListYAMLFiles(testDir, false)
	if err != nil {
		t.Errorf("ListYAMLFiles failed: %v", err)
	}

	expectedCount := 3 // file1.yml, file2.yaml, .hidden.yml
	if len(yamlFiles) != expectedCount {
		t.Errorf("Expected %d YAML files, got %d", expectedCount, len(yamlFiles))
	}

	// List YAML files (recursive)
	yamlFiles, err = fh.ListYAMLFiles(testDir, true)
	if err != nil {
		t.Errorf("ListYAMLFiles (recursive) failed: %v", err)
	}

	expectedCountRecursive := 4 // Including nested/file5.yml
	if len(yamlFiles) != expectedCountRecursive {
		t.Errorf("Expected %d YAML files (recursive), got %d", expectedCountRecursive, len(yamlFiles))
	}
}
