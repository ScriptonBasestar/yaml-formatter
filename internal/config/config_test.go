package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	
	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}
	
	// Check default values
	if cfg.GetDefaultIndent() != 2 {
		t.Errorf("Default indent = %d, want 2", cfg.GetDefaultIndent())
	}
	
	if !cfg.GetPreserveComments() {
		t.Error("Preserve comments should be true by default")
	}
	
	if cfg.GetSchemaDir() == "" {
		t.Error("Schema dir should have a default value")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	configContent := `default_indent: 4
preserve_comments: false
schema_dir: /custom/schemas
line_width: 100`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Load config
	cfg := NewConfig()
	err := cfg.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	
	// Verify loaded values
	if cfg.GetDefaultIndent() != 4 {
		t.Errorf("Loaded indent = %d, want 4", cfg.GetDefaultIndent())
	}
	
	if cfg.GetPreserveComments() {
		t.Error("Preserve comments should be false")
	}
	
	if cfg.GetSchemaDir() != "/custom/schemas" {
		t.Errorf("Schema dir = %s, want /custom/schemas", cfg.GetSchemaDir())
	}
	
	if cfg.GetLineWidth() != 100 {
		t.Errorf("Line width = %d, want 100", cfg.GetLineWidth())
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg := NewConfig()
	cfg.LoadDefaults()
	
	// Check all default values
	if cfg.GetDefaultIndent() != 2 {
		t.Errorf("Default indent = %d, want 2", cfg.GetDefaultIndent())
	}
	
	if !cfg.GetPreserveComments() {
		t.Error("Default preserve comments should be true")
	}
	
	if cfg.GetLineWidth() != 80 {
		t.Errorf("Default line width = %d, want 80", cfg.GetLineWidth())
	}
	
	schemaDir := cfg.GetSchemaDir()
	if schemaDir == "" {
		t.Error("Default schema dir should not be empty")
	}
	
	// Should contain .sb-yaml/schemas
	if !filepath.IsAbs(schemaDir) {
		t.Error("Default schema dir should be absolute")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Config)
		wantErr bool
	}{
		{
			name:    "Valid config",
			setup:   func(c *Config) {},
			wantErr: false,
		},
		{
			name: "Invalid indent",
			setup: func(c *Config) {
				c.v.Set("default_indent", 0)
			},
			wantErr: true,
		},
		{
			name: "Invalid line width",
			setup: func(c *Config) {
				c.v.Set("line_width", -1)
			},
			wantErr: true,
		},
		{
			name: "Empty schema dir",
			setup: func(c *Config) {
				c.v.Set("schema_dir", "")
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			tt.setup(cfg)
			
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveToFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "saved-config.yaml")
	
	cfg := NewConfig()
	cfg.SetDefaultIndent(4)
	cfg.SetPreserveComments(false)
	cfg.SetLineWidth(120)
	cfg.SetSchemaDir("/test/schemas")
	
	err := cfg.SaveToFile(configPath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}
	
	// Load it back
	newCfg := NewConfig()
	err = newCfg.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}
	
	// Verify values
	if newCfg.GetDefaultIndent() != 4 {
		t.Errorf("Saved indent = %d, want 4", newCfg.GetDefaultIndent())
	}
	
	if newCfg.GetPreserveComments() {
		t.Error("Saved preserve comments should be false")
	}
	
	if newCfg.GetLineWidth() != 120 {
		t.Errorf("Saved line width = %d, want 120", newCfg.GetLineWidth())
	}
	
	if newCfg.GetSchemaDir() != "/test/schemas" {
		t.Errorf("Saved schema dir = %s, want /test/schemas", newCfg.GetSchemaDir())
	}
}

func TestGettersAndSetters(t *testing.T) {
	cfg := NewConfig()
	
	// Test SetDefaultIndent and GetDefaultIndent
	cfg.SetDefaultIndent(8)
	if cfg.GetDefaultIndent() != 8 {
		t.Errorf("GetDefaultIndent() = %d, want 8", cfg.GetDefaultIndent())
	}
	
	// Test SetPreserveComments and GetPreserveComments
	cfg.SetPreserveComments(false)
	if cfg.GetPreserveComments() {
		t.Error("GetPreserveComments() = true, want false")
	}
	
	// Test SetLineWidth and GetLineWidth
	cfg.SetLineWidth(100)
	if cfg.GetLineWidth() != 100 {
		t.Errorf("GetLineWidth() = %d, want 100", cfg.GetLineWidth())
	}
	
	// Test SetSchemaDir and GetSchemaDir
	cfg.SetSchemaDir("/new/schema/dir")
	if cfg.GetSchemaDir() != "/new/schema/dir" {
		t.Errorf("GetSchemaDir() = %s, want /new/schema/dir", cfg.GetSchemaDir())
	}
	
	// Test IsVerbose
	cfg.SetVerbose(true)
	if !cfg.IsVerbose() {
		t.Error("IsVerbose() = false, want true")
	}
}

func TestGetConfigPath(t *testing.T) {
	cfg := NewConfig()
	
	path := cfg.GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath() returned empty string")
	}
	
	// Should end with .sb-yaml/config.yaml
	if !filepath.IsAbs(path) {
		t.Error("Config path should be absolute")
	}
}

func TestGetSchemaPath(t *testing.T) {
	cfg := NewConfig()
	
	schemaPath := cfg.GetSchemaPath("test-schema")
	
	if schemaPath == "" {
		t.Error("GetSchemaPath() returned empty string")
	}
	
	// Should end with test-schema.yaml
	if !filepath.IsAbs(schemaPath) {
		t.Error("Schema path should be absolute")
	}
	
	dir := filepath.Dir(schemaPath)
	base := filepath.Base(schemaPath)
	
	if base != "test-schema.yaml" {
		t.Errorf("Schema file name = %s, want test-schema.yaml", base)
	}
	
	// Directory should be the schema dir
	if dir != cfg.GetSchemaDir() {
		t.Errorf("Schema directory mismatch")
	}
}

func TestLoad(t *testing.T) {
	// Save original home
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	
	// Set temporary home
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	
	// Create config directory and file
	configDir := filepath.Join(tempHome, ".sb-yaml")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `default_indent: 3
preserve_comments: false`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	
	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	
	// Verify loaded values
	if cfg.GetDefaultIndent() != 3 {
		t.Errorf("Loaded indent = %d, want 3", cfg.GetDefaultIndent())
	}
	
	if cfg.GetPreserveComments() {
		t.Error("Loaded preserve comments should be false")
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Save original env vars
	originalIndent := os.Getenv("SB_YAML_DEFAULT_INDENT")
	originalPreserve := os.Getenv("SB_YAML_PRESERVE_COMMENTS")
	defer func() {
		os.Setenv("SB_YAML_DEFAULT_INDENT", originalIndent)
		os.Setenv("SB_YAML_PRESERVE_COMMENTS", originalPreserve)
	}()
	
	// Set env vars
	os.Setenv("SB_YAML_DEFAULT_INDENT", "6")
	os.Setenv("SB_YAML_PRESERVE_COMMENTS", "false")
	
	// Create new config
	cfg := NewConfig()
	
	// Env vars should override defaults
	if cfg.GetDefaultIndent() != 6 {
		t.Errorf("Env var indent = %d, want 6", cfg.GetDefaultIndent())
	}
	
	if cfg.GetPreserveComments() {
		t.Error("Env var preserve comments should be false")
	}
}

func TestGetViper(t *testing.T) {
	cfg := NewConfig()
	v := cfg.GetViper()
	
	if v == nil {
		t.Fatal("GetViper() returned nil")
	}
	
	// Should be the same viper instance
	if v != cfg.v {
		t.Error("GetViper() returned different viper instance")
	}
}