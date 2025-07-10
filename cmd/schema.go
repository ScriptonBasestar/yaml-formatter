package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"yaml-formatter/internal/config"
	"yaml-formatter/internal/schema"
	"yaml-formatter/internal/utils"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage YAML schemas",
	Long:  `Commands to generate, save, and manage YAML schemas for formatting`,
}

var schemaGenCmd = &cobra.Command{
	Use:   "gen [schema_name] [yaml_file]",
	Short: "Generate a schema from an existing YAML file",
	Long:  `Generate a schema that defines the key order based on an existing YAML file`,
	Args:  cobra.ExactArgs(2),
	Example: `  sb-yaml schema gen compose docker-compose.yml > compose.schema.yaml
  sb-yaml schema gen k8s deployment.yaml > k8s.schema.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		schemaName := args[0]
		yamlFile := args[1]
		
		if err := generateSchema(schemaName, yamlFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var schemaSetCmd = &cobra.Command{
	Use:   "set [schema_name] [schema_file]",
	Short: "Save a schema for later use",
	Long:  `Save a schema file to the local schema store for use in formatting commands`,
	Args:  cobra.ExactArgs(2),
	Example: `  sb-yaml schema set compose compose.schema.yaml
  sb-yaml schema set k8s --from-yaml deployment.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		schemaName := args[0]
		schemaFile := args[1]
		
		if err := setSchema(schemaName, schemaFile, fromYaml); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var schemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved schemas",
	Long:  `Display all schemas that have been saved and are available for formatting`,
	Example: `  sb-yaml schema list`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listSchemas(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var fromYaml bool

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.AddCommand(schemaGenCmd)
	schemaCmd.AddCommand(schemaSetCmd)
	schemaCmd.AddCommand(schemaListCmd)

	schemaSetCmd.Flags().BoolVar(&fromYaml, "from-yaml", false, "Generate schema from YAML file instead of using schema file")
}

// generateSchema generates a schema from a YAML file and outputs it to stdout
func generateSchema(schemaName, yamlFile string) error {
	fileHandler := utils.NewFileHandler(nil)
	
	// Read the YAML file
	content, err := fileHandler.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}
	
	// Generate schema
	s, err := schema.GenerateFromYAML(content, schemaName)
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}
	
	// Output schema to stdout
	fmt.Print(s.String())
	
	return nil
}

// setSchema saves a schema to the local schema store
func setSchema(schemaName, schemaFile string, fromYaml bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	loader := schema.NewLoader(nil, cfg.GetSchemaDir())
	
	if fromYaml {
		// Generate schema from YAML file and save it
		s, err := loader.GenerateAndSaveFromYAML(schemaFile, schemaName)
		if err != nil {
			return fmt.Errorf("failed to generate and save schema from YAML: %w", err)
		}
		
		fmt.Printf("Schema '%s' generated from '%s' and saved successfully\n", s.Name, schemaFile)
	} else {
		// Load schema from file and save it
		s, err := loader.LoadSchemaFromFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema from file: %w", err)
		}
		
		// Update the name if provided
		if s.Name != schemaName {
			s.Name = schemaName
		}
		
		if err := loader.SaveSchema(s); err != nil {
			return fmt.Errorf("failed to save schema: %w", err)
		}
		
		fmt.Printf("Schema '%s' saved successfully\n", s.Name)
	}
	
	return nil
}

// listSchemas lists all available schemas
func listSchemas() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	loader := schema.NewLoader(nil, cfg.GetSchemaDir())
	
	schemas, err := loader.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}
	
	if len(schemas) == 0 {
		fmt.Println("No schemas found.")
		fmt.Printf("Schema directory: %s\n", cfg.GetSchemaDir())
		return nil
	}
	
	fmt.Printf("Available schemas (%d):\n", len(schemas))
	for _, name := range schemas {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Printf("\nSchema directory: %s\n", cfg.GetSchemaDir())
	
	return nil
}