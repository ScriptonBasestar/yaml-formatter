package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "sb-yaml",
	Short: "A YAML formatter that reorders keys according to predefined schemas",
	Long: `sb-yaml is a CLI tool that formats YAML files by reordering keys according to predefined schemas.
It helps maintain consistent YAML structure across teams and projects, especially useful for:
- Docker Compose files
- Kubernetes manifests  
- GitHub Actions workflows
- Ansible playbooks
- Helm values files`,
	Example: `  sb-yaml format compose docker-compose.yml
  sb-yaml check k8s *.k8s.yaml
  sb-yaml schema gen compose docker-compose.yml`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sb-yaml.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".sb-yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
