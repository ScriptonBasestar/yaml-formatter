package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"yaml-formatter/internal/config"
	"yaml-formatter/internal/formatter"
	"yaml-formatter/internal/schema"
	"yaml-formatter/internal/utils"
)

var formatCmd = &cobra.Command{
	Use:   "format [schema_name] [files...]",
	Short: "Format YAML files according to a schema",
	Long: `Format one or more YAML files by reordering keys according to the specified schema.
The original files will be modified in-place unless --dry-run is specified.`,
	Args: cobra.MinimumNArgs(2),
	Example: `  sb-yaml format compose docker-compose.yml
  sb-yaml format k8s *.k8s.yaml
  sb-yaml format compose --dry-run docker-compose.yml`,
	Run: func(cmd *cobra.Command, args []string) {
		schemaName := args[0]
		files := args[1:]

		if err := formatFiles(schemaName, files, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var checkCmd = &cobra.Command{
	Use:   "check [schema_name] [files...]",
	Short: "Check if YAML files are properly formatted",
	Long: `Check whether YAML files conform to the specified schema without modifying them.
Exit code 0 means all files are properly formatted, non-zero means some files need formatting.`,
	Args: cobra.MinimumNArgs(2),
	Example: `  sb-yaml check compose docker-compose.yml
  sb-yaml check k8s *.k8s.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		schemaName := args[0]
		files := args[1:]

		if err := checkFiles(schemaName, files); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var dryRun bool

func init() {
	rootCmd.AddCommand(formatCmd)
	rootCmd.AddCommand(checkCmd)

	formatCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without modifying files")
}

// formatFiles formats multiple files using the specified schema
func formatFiles(schemaName string, filePatterns []string, dryRun bool) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load schema
	loader := schema.NewLoader(nil, cfg.GetSchemaDir())
	s, err := loader.LoadSchema(schemaName)
	if err != nil {
		return fmt.Errorf("failed to load schema '%s': %w", schemaName, err)
	}

	// Expand file patterns
	fileHandler := utils.NewFileHandler(nil)
	files, err := fileHandler.ExpandGlob(filePatterns)
	if err != nil {
		return fmt.Errorf("failed to expand file patterns: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No YAML files found matching the patterns")
		return nil
	}

	// Create formatter
	f := formatter.NewFormatter(s)
	f.SetIndent(cfg.GetDefaultIndent())
	f.SetPreserveComments(cfg.GetPreserveComments())

	var errors []string
	var processed int
	var changed int

	if dryRun {
		fmt.Printf("DRY RUN: Would format %d file(s) using schema '%s'\n", len(files), schemaName)
	} else {
		fmt.Printf("Formatting %d file(s) using schema '%s'\n", len(files), schemaName)
	}

	for _, file := range files {
		if err := formatSingleFile(f, fileHandler, file, dryRun); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", file, err))
			continue
		}

		processed++

		// Check if file would change
		content, err := fileHandler.ReadFile(file)
		if err != nil {
			continue
		}

		formatted, err := f.FormatContent(content)
		if err != nil {
			continue
		}

		if string(content) != string(formatted) {
			changed++
		}
	}

	// Print summary
	if dryRun {
		fmt.Printf("\nDry run complete: %d files would be changed out of %d processed\n", changed, processed)
	} else {
		fmt.Printf("\nFormatting complete: %d files processed, %d files changed\n", processed, changed)
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors encountered:\n")
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
		return fmt.Errorf("%d files failed to format", len(errors))
	}

	return nil
}

// formatSingleFile formats a single file
func formatSingleFile(f *formatter.Formatter, fileHandler *utils.FileHandler, filePath string, dryRun bool) error {
	// Read original content
	content, err := fileHandler.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Format content
	formatted, err := f.FormatContent(content)
	if err != nil {
		return fmt.Errorf("failed to format content: %w", err)
	}

	// Check if content changed
	if string(content) == string(formatted) {
		fmt.Printf("  ✓ %s (no changes needed)\n", filePath)
		return nil
	}

	if dryRun {
		fmt.Printf("  ~ %s (would be formatted)\n", filePath)
		return nil
	}

	// Write formatted content
	if err := fileHandler.WriteFile(filePath, formatted); err != nil {
		return fmt.Errorf("failed to write formatted content: %w", err)
	}

	fmt.Printf("  ✓ %s (formatted)\n", filePath)
	return nil
}

// checkFiles checks if files are properly formatted
func checkFiles(schemaName string, filePatterns []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load schema
	loader := schema.NewLoader(nil, cfg.GetSchemaDir())
	s, err := loader.LoadSchema(schemaName)
	if err != nil {
		return fmt.Errorf("failed to load schema '%s': %w", schemaName, err)
	}

	// Expand file patterns
	fileHandler := utils.NewFileHandler(nil)
	files, err := fileHandler.ExpandGlob(filePatterns)
	if err != nil {
		return fmt.Errorf("failed to expand file patterns: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No YAML files found matching the patterns")
		return nil
	}

	// Create formatter
	f := formatter.NewFormatter(s)

	var needsFormatting []string
	var errors []string

	fmt.Printf("Checking %d file(s) against schema '%s'\n", len(files), schemaName)

	for _, file := range files {
		content, err := fileHandler.ReadFile(file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to read file: %v", file, err))
			continue
		}

		formatted, err := f.CheckFormat(content)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to check format: %v", file, err))
			continue
		}

		if formatted {
			fmt.Printf("  ✓ %s\n", file)
		} else {
			fmt.Printf("  ✗ %s (needs formatting)\n", file)
			needsFormatting = append(needsFormatting, file)
		}
	}

	// Print summary
	if len(needsFormatting) == 0 && len(errors) == 0 {
		fmt.Printf("\nAll files are properly formatted ✓\n")
		return nil
	}

	if len(needsFormatting) > 0 {
		fmt.Printf("\n%d file(s) need formatting:\n", len(needsFormatting))
		for _, file := range needsFormatting {
			fmt.Printf("  %s\n", file)
		}
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors encountered:\n")
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
	}

	// Exit with non-zero code if files need formatting
	if len(needsFormatting) > 0 {
		os.Exit(1)
	}

	return nil
}
