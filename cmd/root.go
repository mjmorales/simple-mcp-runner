// Package cmd implements the CLI commands
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information (set by build flags)
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "simple-mcp-runner",
	Short: "A Model Context Protocol (MCP) server for local command execution",
	Long: `simple-mcp-runner is a production-ready MCP server that provides Language Learning Models (LLMs)
with a safe interface to discover and execute system commands on the local machine.

The server communicates over stdio using JSON-RPC messages and can be configured with a YAML file
to define custom commands, security policies, and execution limits.

Features:
  - Command discovery with pattern matching
  - Safe command execution with timeouts and resource limits
  - Configurable security policies
  - Structured logging
  - Graceful shutdown handling

For more information, visit: https://github.com/mjmorales/simple-mcp-runner`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildTime),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.simple-mcp-runner.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// Remove unused flag
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


