package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
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
	viper.SetConfigName(".sb-yaml")
	viper.SetConfigType("yaml")
	
	// Add config paths
	viper.AddConfigPath(".")
	viper.AddConfigPath(home)
	viper.AddConfigPath("/etc/sb-yaml/")
	
	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is not an error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	
	// Expand home directory in schema_dir if needed
	if config.SchemaDir[0] == '~' {
		config.SchemaDir = filepath.Join(home, config.SchemaDir[1:])
	}
	
	return &config, nil
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
	if c.DefaultIndent < 1 {
		c.DefaultIndent = DefaultIndent
	}
	
	if c.DefaultLineWidth < 40 {
		c.DefaultLineWidth = DefaultLineWidth
	}
	
	if c.SchemaDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		c.SchemaDir = filepath.Join(home, DefaultSchemaDir)
	}
	
	return nil
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	return viper.AllSettings()[""].(string)
}