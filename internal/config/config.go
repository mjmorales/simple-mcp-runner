// Package config handles configuration loading and validation
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/mjmorales/simple-mcp-runner/internal/errors"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	// App name for identification
	App string `yaml:"app" validate:"required,min=1,max=100"`

	// Version of the configuration schema
	Version string `yaml:"version,omitempty"`

	// Transport type (currently only stdio is supported)
	Transport string `yaml:"transport" validate:"required,oneof=stdio"`

	// Commands defines custom commands exposed by the server
	Commands []Command `yaml:"commands,omitempty"`

	// Security settings
	Security SecurityConfig `yaml:"security,omitempty"`

	// Execution settings
	Execution ExecutionConfig `yaml:"execution,omitempty"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging,omitempty"`

	// Discovery settings
	Discovery DiscoveryConfig `yaml:"discovery,omitempty"`
}

// Command represents a configured command.
type Command struct {
	// Name is the command identifier
	Name string `yaml:"name" validate:"required,min=1,max=50,alphanum"`

	// Description explains what the command does
	Description string `yaml:"description" validate:"required,min=1,max=500"`

	// Command is the actual command to execute
	Command string `yaml:"command" validate:"required"`

	// Args are the command arguments
	Args []string `yaml:"args,omitempty"`

	// WorkDir is the working directory for the command
	WorkDir string `yaml:"workdir,omitempty"`

	// Env are additional environment variables
	Env map[string]string `yaml:"env,omitempty"`

	// Timeout for command execution
	Timeout string `yaml:"timeout,omitempty"`

	// AllowArgs allows additional arguments from the client
	AllowArgs bool `yaml:"allow_args,omitempty"`
}

// SecurityConfig contains security settings.
type SecurityConfig struct {
	// AllowedCommands is a whitelist of commands that can be executed
	AllowedCommands []string `yaml:"allowed_commands,omitempty"`

	// BlockedCommands is a blacklist of commands that cannot be executed
	BlockedCommands []string `yaml:"blocked_commands,omitempty"`

	// AllowedPaths restricts execution to these paths
	AllowedPaths []string `yaml:"allowed_paths,omitempty"`

	// MaxCommandLength limits the command string length
	MaxCommandLength int `yaml:"max_command_length,omitempty"`

	// DisableShellExpansion prevents shell expansion in commands
	DisableShellExpansion bool `yaml:"disable_shell_expansion,omitempty"`
}

// ExecutionConfig contains execution settings.
type ExecutionConfig struct {
	// DefaultTimeout is the default command timeout
	DefaultTimeout string `yaml:"default_timeout,omitempty"`

	// MaxTimeout is the maximum allowed timeout
	MaxTimeout string `yaml:"max_timeout,omitempty"`

	// MaxConcurrent limits concurrent command executions
	MaxConcurrent int `yaml:"max_concurrent,omitempty"`

	// MaxOutputSize limits the output size in bytes
	MaxOutputSize int64 `yaml:"max_output_size,omitempty"`

	// KillTimeout is the time to wait after SIGTERM before SIGKILL
	KillTimeout string `yaml:"kill_timeout,omitempty"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `yaml:"level,omitempty"`

	// Format is the log format (text, json)
	Format string `yaml:"format,omitempty"`

	// Output is where to write logs (stderr, stdout, file path)
	Output string `yaml:"output,omitempty"`

	// IncludeSource includes source file information
	IncludeSource bool `yaml:"include_source,omitempty"`
}

// DiscoveryConfig contains command discovery settings.
type DiscoveryConfig struct {
	// AdditionalPaths to search for commands
	AdditionalPaths []string `yaml:"additional_paths,omitempty"`

	// ExcludePaths to exclude from search
	ExcludePaths []string `yaml:"exclude_paths,omitempty"`

	// MaxResults limits discovery results
	MaxResults int `yaml:"max_results,omitempty"`

	// CommonCommands to prioritize in discovery
	CommonCommands []string `yaml:"common_commands,omitempty"`
}

// Default returns a default configuration.
func Default() *Config {
	return &Config{
		App:       "simple-mcp-runner",
		Version:   "1.0",
		Transport: "stdio",
		Security: SecurityConfig{
			MaxCommandLength:      1000,
			DisableShellExpansion: true,
			BlockedCommands: []string{
				"rm", "dd", "mkfs", "fdisk", "shutdown", "reboot",
				"systemctl", "service", "kill", "killall", "pkill",
			},
		},
		Execution: ExecutionConfig{
			DefaultTimeout: "30s",
			MaxTimeout:     "5m",
			MaxConcurrent:  10,
			MaxOutputSize:  10 * 1024 * 1024, // 10MB
			KillTimeout:    "5s",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
		Discovery: DiscoveryConfig{
			MaxResults: 100,
			CommonCommands: []string{
				"ls", "cat", "grep", "find", "git", "npm", "go",
				"python", "node", "curl", "wget", "echo", "pwd",
			},
		},
	}
}

// Load loads configuration from a file.
func Load(filename string) (*Config, error) {
	// Start with defaults
	cfg := Default()

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, apperrors.ConfigurationError(fmt.Sprintf("config file not found: %s", filename))
	}

	// Read file
	// #nosec G304 - Configuration files are loaded from user-specified paths
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorTypeConfiguration, "failed to read config file")
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorTypeConfiguration, "failed to parse YAML")
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Validate app name
	if c.App == "" {
		return apperrors.ValidationError("app name is required", "app")
	}

	if len(c.App) > 100 {
		return apperrors.ValidationError("app name too long (max 100 chars)", "app")
	}

	// Validate transport
	if c.Transport != "stdio" {
		return apperrors.ValidationError("only 'stdio' transport is supported", "transport")
	}

	// Validate commands
	seen := make(map[string]bool)
	for i, cmd := range c.Commands {
		if err := c.validateCommand(cmd, i); err != nil {
			return err
		}

		if seen[cmd.Name] {
			return apperrors.ValidationError(fmt.Sprintf("duplicate command name: %s", cmd.Name), "commands")
		}
		seen[cmd.Name] = true
	}

	// Validate security config
	if err := c.validateSecurity(); err != nil {
		return err
	}

	// Validate execution config
	if err := c.validateExecution(); err != nil {
		return err
	}

	// Validate logging config
	if err := c.validateLogging(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateCommand(cmd Command, index int) error {
	field := fmt.Sprintf("commands[%d]", index)

	// Validate name
	if cmd.Name == "" {
		return apperrors.ValidationError("command name is required", field+".name")
	}

	if !isValidCommandName(cmd.Name) {
		return apperrors.ValidationError(
			"command name must be alphanumeric with underscores (1-50 chars)",
			field+".name",
		)
	}

	// Validate description
	if cmd.Description == "" {
		return apperrors.ValidationError("command description is required", field+".description")
	}

	if len(cmd.Description) > 500 {
		return apperrors.ValidationError("command description too long (max 500 chars)", field+".description")
	}

	// Validate command
	if cmd.Command == "" {
		return apperrors.ValidationError("command is required", field+".command")
	}

	// Validate timeout if specified
	if cmd.Timeout != "" {
		if _, err := time.ParseDuration(cmd.Timeout); err != nil {
			return apperrors.ValidationError(
				fmt.Sprintf("invalid timeout format: %v", err),
				field+".timeout",
			)
		}
	}

	// Validate workdir if specified
	if cmd.WorkDir != "" {
		if !filepath.IsAbs(cmd.WorkDir) {
			return apperrors.ValidationError("workdir must be an absolute path", field+".workdir")
		}
	}

	return nil
}

func (c *Config) validateSecurity() error {
	// Validate max command length
	if c.Security.MaxCommandLength < 0 {
		return apperrors.ValidationError("max_command_length cannot be negative", "security.max_command_length")
	}

	// Validate allowed paths
	for i, path := range c.Security.AllowedPaths {
		if !filepath.IsAbs(path) {
			return apperrors.ValidationError(
				fmt.Sprintf("allowed_path must be absolute: %s", path),
				fmt.Sprintf("security.allowed_paths[%d]", i),
			)
		}
	}

	return nil
}

func (c *Config) validateExecution() error {
	// Validate timeouts
	if c.Execution.DefaultTimeout != "" {
		if _, err := time.ParseDuration(c.Execution.DefaultTimeout); err != nil {
			return apperrors.ValidationError(
				fmt.Sprintf("invalid default_timeout: %v", err),
				"execution.default_timeout",
			)
		}
	}

	if c.Execution.MaxTimeout != "" {
		maxDur, err := time.ParseDuration(c.Execution.MaxTimeout)
		if err != nil {
			return apperrors.ValidationError(
				fmt.Sprintf("invalid max_timeout: %v", err),
				"execution.max_timeout",
			)
		}

		// Ensure max timeout is reasonable
		if maxDur > 1*time.Hour {
			return apperrors.ValidationError(
				"max_timeout cannot exceed 1 hour",
				"execution.max_timeout",
			)
		}
	}

	// Validate max concurrent
	if c.Execution.MaxConcurrent < 0 {
		return apperrors.ValidationError("max_concurrent cannot be negative", "execution.max_concurrent")
	}

	// Validate max output size
	if c.Execution.MaxOutputSize < 0 {
		return apperrors.ValidationError("max_output_size cannot be negative", "execution.max_output_size")
	}

	return nil
}

func (c *Config) validateLogging() error {
	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error", ""}
	valid := false
	for _, level := range validLevels {
		if c.Logging.Level == level {
			valid = true
			break
		}
	}
	if !valid {
		return apperrors.ValidationError(
			"invalid log level (must be: debug, info, warn, error)",
			"logging.level",
		)
	}

	// Validate log format
	validFormats := []string{"text", "json", ""}
	valid = false
	for _, format := range validFormats {
		if c.Logging.Format == format {
			valid = true
			break
		}
	}
	if !valid {
		return apperrors.ValidationError(
			"invalid log format (must be: text, json)",
			"logging.format",
		)
	}

	return nil
}

// isValidCommandName checks if a command name is valid.
func isValidCommandName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}

	// Must start with letter, can contain letters, numbers, underscores
	match, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9_]*$", name)
	return match
}

// GetTimeout returns the timeout duration for a command.
func (c *Command) GetTimeout(defaultTimeout time.Duration) time.Duration {
	if c.Timeout == "" {
		return defaultTimeout
	}

	dur, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return defaultTimeout
	}

	return dur
}

// IsCommandAllowed checks if a command is allowed by security settings.
func (c *Config) IsCommandAllowed(command string) bool {
	// Check blocked commands
	for _, blocked := range c.Security.BlockedCommands {
		if command == blocked || strings.HasPrefix(command, blocked+"/") {
			return false
		}
	}

	// If allowed list is specified, check it
	if len(c.Security.AllowedCommands) > 0 {
		for _, allowed := range c.Security.AllowedCommands {
			if command == allowed || strings.HasPrefix(command, allowed+"/") {
				return true
			}
		}
		return false
	}

	return true
}

// IsPathAllowed checks if a path is allowed by security settings.
func (c *Config) IsPathAllowed(path string) bool {
	if len(c.Security.AllowedPaths) == 0 {
		return true
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowed := range c.Security.AllowedPaths {
		if strings.HasPrefix(absPath, allowed) {
			return true
		}
	}

	return false
}
