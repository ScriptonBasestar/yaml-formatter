package formatter

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestParseValidYAML(t *testing.T) {
	testDir := "../../testdata/valid"
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read test directory: %v", err)
	}

	parser := NewParser(true)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(testDir, file.Name()))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			node, err := parser.ParseYAML(content)
			if err != nil {
				t.Errorf("Failed to parse valid YAML: %v", err)
			}

			if node == nil {
				t.Error("ParseYAML returned nil node for valid YAML")
			}
		})
	}
}

func TestParseInvalidYAML(t *testing.T) {
	testDir := "../../testdata/invalid"
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read test directory: %v", err)
	}

	parser := NewParser(true)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(testDir, file.Name()))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// Some files might parse but fail validation
			_, parseErr := parser.ParseYAML(content)
			validateErr := parser.ValidateYAML(content)

			if parseErr == nil && validateErr == nil {
				t.Errorf("Expected error for invalid YAML file %s, but got none", file.Name())
			}
		})
	}
}

func TestParseEdgeCases(t *testing.T) {
	testDir := "../../testdata/edge-cases"

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"Empty file", "empty.yml", false},
		{"Only comments", "only-comments.yml", false},
		{"Special characters", "special-characters.yml", false},
		{"Long lines", "long-lines.yml", false},
		{"Deep nesting", "very-deep-nesting.yml", false},
	}

	parser := NewParser(true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(testDir, tt.filename))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			_, err = parser.ParseYAML(content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiDocumentParsing(t *testing.T) {
	testDir := "../../testdata/multi-document"
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read test directory: %v", err)
	}

	parser := NewParser(true)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(testDir, file.Name()))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// Check if it's multi-document
			if !parser.IsMultiDocument(content) {
				t.Error("Expected file to be identified as multi-document")
			}

			// Parse multi-document
			nodes, err := parser.ParseMultiDocument(content)
			if err != nil {
				t.Errorf("Failed to parse multi-document YAML: %v", err)
			}

			if len(nodes) == 0 {
				t.Error("ParseMultiDocument returned empty nodes slice")
			}
		})
	}
}

func TestCommentPreservation(t *testing.T) {
	content, err := os.ReadFile("../../testdata/valid/with-comments.yml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	parser := NewParser(true)
	node, err := parser.ParseYAML(content)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Check that comments are preserved in the node
	if node.HeadComment == "" && node.LineComment == "" && node.FootComment == "" {
		// Walk through the node to find any comments
		hasComments := false
		var checkNode func(*yaml.Node)
		checkNode = func(n *yaml.Node) {
			if n.HeadComment != "" || n.LineComment != "" || n.FootComment != "" {
				hasComments = true
				return
			}
			for _, child := range n.Content {
				checkNode(child)
			}
		}
		checkNode(node)

		if !hasComments {
			t.Error("Comments were not preserved in the parsed node")
		}
	}
}
