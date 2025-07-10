package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/spf13/afero"
)

// FileHandler provides file system operations
type FileHandler struct {
	fs afero.Fs
}

// NewFileHandler creates a new file handler
func NewFileHandler(filesystem afero.Fs) *FileHandler {
	if filesystem == nil {
		filesystem = afero.NewOsFs()
	}
	return &FileHandler{fs: filesystem}
}

// ExpandGlob expands glob patterns to actual file paths
func (fh *FileHandler) ExpandGlob(patterns []string) ([]string, error) {
	var files []string
	
	for _, pattern := range patterns {
		matches, err := fh.expandSinglePattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to expand pattern %s: %w", pattern, err)
		}
		files = append(files, matches...)
	}
	
	// Remove duplicates
	return removeDuplicates(files), nil
}

// expandSinglePattern expands a single glob pattern
func (fh *FileHandler) expandSinglePattern(pattern string) ([]string, error) {
	// Use doublestar for advanced glob patterns
	fsys := afero.NewIOFS(fh.fs)
	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return nil, err
	}
	
	// Filter out directories and non-YAML files
	var yamlFiles []string
	for _, match := range matches {
		info, err := fh.fs.Stat(match)
		if err != nil {
			continue
		}
		
		if !info.IsDir() && isYAMLFile(match) {
			yamlFiles = append(yamlFiles, match)
		}
	}
	
	return yamlFiles, nil
}

// ReadFile reads a file and returns its content
func (fh *FileHandler) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(fh.fs, path)
}

// WriteFile writes content to a file
func (fh *FileHandler) WriteFile(path string, content []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := fh.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	return afero.WriteFile(fh.fs, path, content, 0644)
}

// BackupFile creates a backup of a file
func (fh *FileHandler) BackupFile(path string) (string, error) {
	backupPath := path + ".bak"
	
	content, err := fh.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}
	
	if err := fh.WriteFile(backupPath, content); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}
	
	return backupPath, nil
}

// FileExists checks if a file exists
func (fh *FileHandler) FileExists(path string) (bool, error) {
	exists, err := afero.Exists(fh.fs, path)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// IsDirectory checks if a path is a directory
func (fh *FileHandler) IsDirectory(path string) bool {
	info, err := fh.fs.Stat(path)
	return err == nil && info.IsDir()
}

// GetFileInfo returns file information
func (fh *FileHandler) GetFileInfo(path string) (FileInfo, error) {
	info, err := fh.fs.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	
	return FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
		Path:    path,
	}, nil
}

// FilterYAMLFiles filters a list of files to only include YAML files
func (fh *FileHandler) FilterYAMLFiles(files []string) []string {
	var yamlFiles []string
	for _, file := range files {
		if isYAMLFile(file) {
			yamlFiles = append(yamlFiles, file)
		}
	}
	return yamlFiles
}

// FileInfo contains information about a file
type FileInfo struct {
	Name    string
	Size    int64
	ModTime interface{}
	IsDir   bool
	Path    string
}

// Helper functions

// isYAMLFile checks if a file is a YAML file based on its extension
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// SanitizePath sanitizes a file path to prevent directory traversal attacks
func SanitizePath(path string) string {
	return filepath.Clean(path)
}

// IsHiddenFile checks if a file is hidden (starts with a dot)
func IsHiddenFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".")
}

// GetRelativePath returns the relative path from base to target
func GetRelativePath(base, target string) (string, error) {
	return filepath.Rel(base, target)
}

// EnsureDir ensures that a directory exists, creating it if necessary
func EnsureDir(fs afero.Fs, dir string) error {
	return fs.MkdirAll(dir, 0755)
}

// IsYAMLFile checks if a file is a YAML file (exported method)
func (fh *FileHandler) IsYAMLFile(path string) bool {
	return isYAMLFile(path)
}

// GetAbsolutePath returns the absolute path for a given path
func (fh *FileHandler) GetAbsolutePath(path string) (string, error) {
	return filepath.Abs(path)
}

// ListYAMLFiles lists all YAML files in a directory
func (fh *FileHandler) ListYAMLFiles(dir string, recursive bool) ([]string, error) {
	var yamlFiles []string
	
	if recursive {
		pattern := filepath.Join(dir, "**", "*.{yml,yaml}")
		matches, err := fh.ExpandGlob([]string{pattern})
		if err != nil {
			return nil, err
		}
		return matches, nil
	}
	
	// Non-recursive
	entries, err := afero.ReadDir(fh.fs, dir)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && isYAMLFile(entry.Name()) {
			yamlFiles = append(yamlFiles, filepath.Join(dir, entry.Name()))
		}
	}
	
	return yamlFiles, nil
}