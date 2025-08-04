package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed config.example.yaml
var configExampleFS embed.FS

const (
	defaultConfigName = ".simple-mcp-runner.yaml"
	exampleConfigFile = "config.example.yaml"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long: `Initialize creates a default configuration file for simple-mcp-runner.

The configuration file will be created at ~/.simple-mcp-runner.yaml with
sensible defaults. You can then customize it to your needs.

If a configuration file already exists, init will not overwrite it unless
you use the --force flag.`,
	RunE: runInit,
}

var forceInit bool

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing configuration file")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Construct config path
	configPath := filepath.Join(homeDir, defaultConfigName)

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil && !forceInit {
		fmt.Printf("Configuration file already exists at %s\n", configPath)
		fmt.Println("Use --force to overwrite")
		return nil
	}

	// Read example config from embedded filesystem
	data, err := configExampleFS.ReadFile(exampleConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read example config: %w", err)
	}

	// Write config file
	// #nosec G306 - Configuration file needs to be readable by the user
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration file created at %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file to customize commands and settings")
	fmt.Println("2. Run 'simple-mcp-runner validate' to check your configuration")
	fmt.Println("3. Run 'simple-mcp-runner run' to start the MCP server")

	return nil
}

