package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	v               *viper.Viper
	SchemaDir       string `mapstructure:"schema_dir"`
	DefaultIndent   int    `mapstructure:"default_indent"`
	DefaultLineWidth int   `mapstructure:"default_line_width"`
	PreserveComments bool  `mapstructure:"preserve_comments"`
	Verbose         bool   `mapstructure:"verbose"`
}

// Default configuration values
const (
	DefaultIndent          = 2
	DefaultLineWidth       = 80
	DefaultPreserveComments = true
	DefaultSchemaDir       = ".sb-yaml/schemas"
)

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	v := viper.New()
	
	// Set defaults
	v.SetDefault("default_indent", DefaultIndent)
	v.SetDefault("default_line_width", DefaultLineWidth)
	v.SetDefault("preserve_comments", DefaultPreserveComments)
	v.SetDefault("verbose", false)
	
	// Set default schema directory
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	defaultSchemaDir := filepath.Join(home, DefaultSchemaDir)
	v.SetDefault("schema_dir", defaultSchemaDir)
	
	// Environment variables
	v.SetEnvPrefix("SB_YAML")
	v.AutomaticEnv()
	
	config := &Config{
		v:                v,
		SchemaDir:        v.GetString("schema_dir"),
		DefaultIndent:    v.GetInt("default_indent"),
		DefaultLineWidth: v.GetInt("default_line_width"),
		PreserveComments: v.GetBool("preserve_comments"),
		Verbose:          v.GetBool("verbose"),
	}
	
	return config
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("default_indent", DefaultIndent)
	viper.SetDefault("default_line_width", DefaultLineWidth)
	viper.SetDefault("preserve_comments", DefaultPreserveComments)
	viper.SetDefault("verbose", false)
	
	// Set default schema directory
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	defaultSchemaDir := filepath.Join(home, DefaultSchemaDir)
	viper.SetDefault("schema_dir", defaultSchemaDir)
	
	// Environment variables
	viper.SetEnvPrefix("SB_YAML")
	viper.AutomaticEnv()
	
	// Config file settings
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	
	// Add config paths
	viper.AddConfigPath(".")
	viper.AddConfigPath(filepath.Join(home, ".sb-yaml"))
	viper.AddConfigPath(home)
	viper.AddConfigPath("/etc/sb-yaml/")
	
	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is not an error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	
	config := &Config{
		v: viper.GetViper(),
	}
	
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}
	
	// Expand home directory in schema_dir if needed
	if len(config.SchemaDir) > 0 && config.SchemaDir[0] == '~' {
		config.SchemaDir = filepath.Join(home, config.SchemaDir[1:])
	}
	
	return config, nil
}

// Save saves the current configuration to a file
func (c *Config) Save() error {
	viper.Set("schema_dir", c.SchemaDir)
	viper.Set("default_indent", c.DefaultIndent)
	viper.Set("default_line_width", c.DefaultLineWidth)
	viper.Set("preserve_comments", c.PreserveComments)
	viper.Set("verbose", c.Verbose)
	
	return viper.WriteConfig()
}

// GetSchemaDir returns the schema directory path
func (c *Config) GetSchemaDir() string {
	return c.SchemaDir
}

// SetSchemaDir sets the schema directory path
func (c *Config) SetSchemaDir(dir string) {
	c.SchemaDir = dir
}

// GetDefaultIndent returns the default indentation
func (c *Config) GetDefaultIndent() int {
	return c.DefaultIndent
}

// SetDefaultIndent sets the default indentation
func (c *Config) SetDefaultIndent(indent int) {
	c.DefaultIndent = indent
}

// GetDefaultLineWidth returns the default line width
func (c *Config) GetDefaultLineWidth() int {
	return c.DefaultLineWidth
}

// SetDefaultLineWidth sets the default line width
func (c *Config) SetDefaultLineWidth(width int) {
	c.DefaultLineWidth = width
}

// GetPreserveComments returns whether comments should be preserved
func (c *Config) GetPreserveComments() bool {
	return c.PreserveComments
}

// SetPreserveComments sets whether comments should be preserved
func (c *Config) SetPreserveComments(preserve bool) {
	c.PreserveComments = preserve
}

// IsVerbose returns whether verbose output is enabled
func (c *Config) IsVerbose() bool {
	return c.Verbose
}

// SetVerbose sets whether verbose output is enabled
func (c *Config) SetVerbose(verbose bool) {
	c.Verbose = verbose
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Sync values from viper
	if c.v != nil {
		c.DefaultIndent = c.v.GetInt("default_indent")
		c.DefaultLineWidth = c.v.GetInt("line_width")
		if c.DefaultLineWidth == 0 {
			c.DefaultLineWidth = c.v.GetInt("default_line_width")
		}
		c.SchemaDir = c.v.GetString("schema_dir")
	}
	
	if c.DefaultIndent < 1 {
		return fmt.Errorf("default_indent must be at least 1")
	}
	
	if c.DefaultLineWidth < 0 {
		return fmt.Errorf("default_line_width cannot be negative")
	}
	
	if c.SchemaDir == "" {
		return fmt.Errorf("schema_dir cannot be empty")
	}
	
	return nil
}

// GetLineWidth returns the line width (alias for GetDefaultLineWidth)
func (c *Config) GetLineWidth() int {
	return c.GetDefaultLineWidth()
}

// SetLineWidth sets the line width (alias for SetDefaultLineWidth)
func (c *Config) SetLineWidth(width int) {
	c.SetDefaultLineWidth(width)
}

// LoadDefaults resets configuration to default values
func (c *Config) LoadDefaults() {
	c.DefaultIndent = DefaultIndent
	c.DefaultLineWidth = DefaultLineWidth
	c.PreserveComments = DefaultPreserveComments
	c.Verbose = false
	
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	c.SchemaDir = filepath.Join(home, DefaultSchemaDir)
}

// LoadFromFile loads configuration from a specific file
func (c *Config) LoadFromFile(path string) error {
	c.v.SetConfigFile(path)
	
	if err := c.v.ReadInConfig(); err != nil {
		return err
	}
	
	if err := c.v.Unmarshal(c); err != nil {
		return err
	}
	
	// Handle line_width alias
	if c.v.IsSet("line_width") {
		c.DefaultLineWidth = c.v.GetInt("line_width")
	}
	
	return nil
}

// SaveToFile saves configuration to a specific file
func (c *Config) SaveToFile(path string) error {
	c.v.Set("schema_dir", c.SchemaDir)
	c.v.Set("default_indent", c.DefaultIndent)
	c.v.Set("default_line_width", c.DefaultLineWidth)
	c.v.Set("preserve_comments", c.PreserveComments)
	c.v.Set("verbose", c.Verbose)
	
	return c.v.WriteConfigAs(path)
}

// GetConfigPath returns the path to the config file
func (c *Config) GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".sb-yaml", "config.yaml")
}

// GetSchemaPath returns the path to a specific schema
func (c *Config) GetSchemaPath(name string) string {
	return filepath.Join(c.SchemaDir, name+".yaml")
}

// GetViper returns the underlying viper instance
func (c *Config) GetViper() *viper.Viper {
	return c.v
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	return fmt.Sprintf("Config{SchemaDir:%s, DefaultIndent:%d, DefaultLineWidth:%d, PreserveComments:%v, Verbose:%v}",
		c.SchemaDir, c.DefaultIndent, c.DefaultLineWidth, c.PreserveComments, c.Verbose)
}