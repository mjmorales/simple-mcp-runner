package cmd

import (
	"fmt"
	"os"

	"github.com/mjmorales/simple-mcp-runner/internal/config"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command.
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long: `Validate a configuration file to ensure it meets all requirements and constraints.
This command checks for:
  - Valid YAML syntax
  - Required fields
  - Valid field values and formats
  - Security policy consistency
  - Command definitions

Example:
  simple-mcp-runner validate --config config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if config file is specified
		cfgFile := configFile
		if cfgFile == "" {
			// Try default location
			cfgFile = GetDefaultConfigPath()
			if cfgFile == "" || !fileExists(cfgFile) {
				return fmt.Errorf("configuration file must be specified with --config flag")
			}
		}

		// Check if file exists
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			return fmt.Errorf("configuration file not found: %s", cfgFile)
		}

		// Load and validate configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		// Print validation results
		fmt.Printf("✓ Configuration file is valid: %s\n", cfgFile)
		fmt.Printf("\nConfiguration summary:\n")
		fmt.Printf("  Application: %s\n", cfg.App)
		fmt.Printf("  Transport: %s\n", cfg.Transport)
		fmt.Printf("  Commands: %d defined\n", len(cfg.Commands))

		if len(cfg.Commands) > 0 {
			fmt.Printf("\n  Configured commands:\n")
			for _, cmd := range cfg.Commands {
				fmt.Printf("    - %s: %s\n", cmd.Name, cmd.Description)
			}
		}

		fmt.Printf("\n  Security settings:\n")
		fmt.Printf("    Max command length: %d\n", cfg.Security.MaxCommandLength)
		fmt.Printf("    Shell expansion disabled: %v\n", cfg.Security.DisableShellExpansion)
		if len(cfg.Security.BlockedCommands) > 0 {
			fmt.Printf("    Blocked commands: %d\n", len(cfg.Security.BlockedCommands))
		}
		if len(cfg.Security.AllowedCommands) > 0 {
			fmt.Printf("    Allowed commands: %d\n", len(cfg.Security.AllowedCommands))
		}
		if len(cfg.Security.AllowedPaths) > 0 {
			fmt.Printf("    Allowed paths: %d\n", len(cfg.Security.AllowedPaths))
		}

		fmt.Printf("\n  Execution limits:\n")
		fmt.Printf("    Default timeout: %s\n", cfg.Execution.DefaultTimeout)
		fmt.Printf("    Max timeout: %s\n", cfg.Execution.MaxTimeout)
		fmt.Printf("    Max concurrent: %d\n", cfg.Execution.MaxConcurrent)
		fmt.Printf("    Max output size: %d bytes\n", cfg.Execution.MaxOutputSize)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
