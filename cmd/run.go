package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mjmorales/simple-mcp-runner/pkg/config"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
	"github.com/mjmorales/simple-mcp-runner/internal/server"
	"github.com/spf13/cobra"
)

var (
	logLevel  string
	logFormat string
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the MCP server",
	Long: `Start the Model Context Protocol (MCP) server that provides LLMs with an interface
to discover and execute system commands. The server communicates over stdio using 
JSON-RPC messages and can be configured with a YAML file to define custom commands,
security policies, and execution limits.

The server runs in the foreground and can be stopped with Ctrl+C (SIGINT) or SIGTERM.

Example:
  # Run with default configuration
  simple-mcp-runner run

  # Run with custom configuration
  simple-mcp-runner run --config config.yaml

  # Run with debug logging
  simple-mcp-runner run --log-level debug

  # Run with JSON logging
  simple-mcp-runner run --log-format json`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Logging flags
	runCmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	runCmd.Flags().StringVar(&logFormat, "log-format", "text", "log format (text, json)")
}

// runServer runs the MCP server.
func runServer(cmd *cobra.Command, args []string) error {
	// Setup logger
	logOpts := logger.Options{
		Level:      logLevel,
		JSONOutput: logFormat == "json",
		Output:     os.Stderr,
		AddSource:  logLevel == "debug",
	}

	log, err := logger.New(logOpts)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	logger.SetDefault(log)

	// Load configuration
	var cfg *config.Config
	if configFile != "" {
		cfg, err = config.LoadFromFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		log.Info("loaded configuration", "file", configFile)
	} else {
		// Try to load from default location
		defaultPath := GetDefaultConfigPath()
		if defaultPath != "" {
			if _, err := os.Stat(defaultPath); err == nil {
				cfg, err = config.LoadFromFile(defaultPath)
				if err != nil {
					return fmt.Errorf("failed to load default config: %w", err)
				}
				log.Info("loaded default configuration", "file", defaultPath)
			} else {
				cfg = config.Default()
				log.Info("using built-in default configuration")
			}
		} else {
			cfg = config.Default()
			log.Info("using built-in default configuration")
		}
	}

	// Override logging config from CLI flags if provided
	if cmd.Flags().Changed("log-level") {
		cfg.Logging.Level = logLevel
	}
	if cmd.Flags().Changed("log-format") {
		cfg.Logging.Format = logFormat
	}

	// Create and run server
	srv, err := server.New(server.Options{
		Config: cfg,
		Logger: log,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Run server with context
	ctx := context.Background()
	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
