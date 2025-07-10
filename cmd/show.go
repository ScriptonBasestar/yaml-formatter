package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show configuration snippets and examples",
	Long:  `Display configuration snippets for integrating sb-yaml with various tools`,
}

var showGitHookCmd = &cobra.Command{
	Use:   "git-pre-commit-hook",
	Short: "Show Git pre-commit hook configuration",
	Long:  `Display a pre-commit hook configuration that can be used to automatically format YAML files`,
	Example: `  sb-yaml show git-pre-commit-hook > .pre-commit-config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		hookConfig := `# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: sb-yaml-format
        name: sb-yaml format YAML files
        entry: sb-yaml check
        language: system
        files: \.(yml|yaml)$
        args: [compose]  # Change this to your schema name
        pass_filenames: true
        
  # Example for multiple schemas
  - repo: local
    hooks:
      - id: sb-yaml-format-compose
        name: sb-yaml format Docker Compose files
        entry: sb-yaml check compose
        language: system
        files: docker-compose.*\.(yml|yaml)$
        pass_filenames: true
        
      - id: sb-yaml-format-k8s
        name: sb-yaml format Kubernetes files
        entry: sb-yaml check k8s
        language: system
        files: .*\.k8s\.(yml|yaml)$
        pass_filenames: true

      - id: sb-yaml-format-github-actions
        name: sb-yaml format GitHub Actions workflows
        entry: sb-yaml check github-actions
        language: system
        files: \.github/workflows/.*\.(yml|yaml)$
        pass_filenames: true`

		fmt.Println(hookConfig)
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showGitHookCmd)
}