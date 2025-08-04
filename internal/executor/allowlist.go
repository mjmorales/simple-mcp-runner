package executor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// AllowlistConfig defines allowed commands and their constraints.
type AllowlistConfig struct {
	// Commands maps command names to their allowed configurations
	Commands map[string]CommandConfig `yaml:"commands"`

	// DefaultPolicy defines what happens when command is not in allowlist
	DefaultPolicy Policy `yaml:"default_policy"` // "allow", "deny", "prompt"

	// MaxArguments limits the number of arguments per command
	MaxArguments int `yaml:"max_arguments"`

	// AllowedWorkDirs restricts working directories
	AllowedWorkDirs []string `yaml:"allowed_work_dirs"`

	// ForbiddenPatterns are regex patterns that are never allowed
	ForbiddenPatterns []string `yaml:"forbidden_patterns"`
}

// CommandConfig defines constraints for a specific command.
type CommandConfig struct {
	// Enabled controls if the command is allowed
	Enabled bool `yaml:"enabled"`

	// AllowedArgs constrains which arguments are permitted
	AllowedArgs []string `yaml:"allowed_args,omitempty"`

	// ForbiddenArgs lists arguments that are never allowed
	ForbiddenArgs []string `yaml:"forbidden_args,omitempty"`

	// ArgPatterns are regex patterns for validating arguments
	ArgPatterns []string `yaml:"arg_patterns,omitempty"`

	// MaxArgs limits argument count for this command
	MaxArgs int `yaml:"max_args,omitempty"`

	// RequiresAuth indicates if command needs authentication
	RequiresAuth bool `yaml:"requires_auth"`

	// AllowedUsers restricts which users can run this command
	AllowedUsers []string `yaml:"allowed_users,omitempty"`
}

type Policy string

const (
	PolicyAllow  Policy = "allow"
	PolicyDeny   Policy = "deny"
	PolicyPrompt Policy = "prompt"
)

// DefaultAllowlistConfig returns a secure default configuration.
func DefaultAllowlistConfig() *AllowlistConfig {
	return &AllowlistConfig{
		Commands: map[string]CommandConfig{
			// Safe read-only commands
			"ls": {
				Enabled:       true,
				AllowedArgs:   []string{"-l", "-a", "-la", "-lt", "-lh", "--help"},
				ForbiddenArgs: []string{"--color=always"}, // Prevent terminal escape sequences
				MaxArgs:       5,
				RequiresAuth:  false,
			},
			"cat": {
				Enabled:      true,
				MaxArgs:      3,
				RequiresAuth: false,
				ArgPatterns:  []string{`^[a-zA-Z0-9._/-]+$`}, // Alphanumeric paths only
			},
			"pwd": {
				Enabled:      true,
				MaxArgs:      1,
				RequiresAuth: false,
			},
			"echo": {
				Enabled:       true,
				MaxArgs:       10,
				RequiresAuth:  false,
				ForbiddenArgs: []string{"-e", "-E"}, // Prevent escape sequence interpretation
			},
			"grep": {
				Enabled:       true,
				AllowedArgs:   []string{"-n", "-i", "-r", "-l", "--help"},
				ForbiddenArgs: []string{"-P"}, // Prevent Perl regex
				MaxArgs:       10,
				RequiresAuth:  false,
			},
			"find": {
				Enabled:      true,
				AllowedArgs:  []string{"-name", "-type", "-maxdepth", "-mindepth", "-size", "--help"},
				MaxArgs:      15,
				RequiresAuth: false,
			},
			// Version control (read-only operations)
			"git": {
				Enabled:      true,
				AllowedArgs:  []string{"status", "log", "diff", "show", "branch", "remote", "--help"},
				MaxArgs:      8,
				RequiresAuth: false,
			},
			// Development tools (restricted)
			"go": {
				Enabled:      true,
				AllowedArgs:  []string{"version", "env", "list", "help"},
				MaxArgs:      5,
				RequiresAuth: true,
			},
			"npm": {
				Enabled:      true,
				AllowedArgs:  []string{"list", "version", "help", "--version"},
				MaxArgs:      5,
				RequiresAuth: true,
			},
		},
		DefaultPolicy:   PolicyDeny,
		MaxArguments:    20,
		AllowedWorkDirs: []string{"/tmp", "/home", "/Users"},
		ForbiddenPatterns: []string{
			`.*[;&|><$` + "`" + `].*`, // Shell metacharacters
			`.*\.\./.*`,               // Path traversal
			`.*--exec.*`,              // Exec flags
			`.*-e\s+.*`,               // Execute flags
			`.*/dev/.*`,               // Device files
			`.*/proc/.*`,              // Process files
			`.*/sys/.*`,               // System files
		},
	}
}

// AllowlistValidator implements command allowlist validation.
type AllowlistValidator struct {
	config           *AllowlistConfig
	forbiddenRegexes []*regexp.Regexp
}

// NewAllowlistValidator creates a new allowlist validator.
func NewAllowlistValidator(config *AllowlistConfig) (*AllowlistValidator, error) {
	if config == nil {
		config = DefaultAllowlistConfig()
	}

	validator := &AllowlistValidator{
		config: config,
	}

	// Compile forbidden patterns
	for _, pattern := range config.ForbiddenPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid forbidden pattern %q: %w", pattern, err)
		}
		validator.forbiddenRegexes = append(validator.forbiddenRegexes, regex)
	}

	return validator, nil
}

// ValidateCommand validates if a command execution is allowed.
func (v *AllowlistValidator) ValidateCommand(command string, args []string) error {
	// Check forbidden patterns first
	fullCommand := command + " " + strings.Join(args, " ")
	for _, regex := range v.forbiddenRegexes {
		if regex.MatchString(fullCommand) {
			return fmt.Errorf("command matches forbidden pattern: %s", command)
		}
	}

	// Check global argument limit
	if len(args) > v.config.MaxArguments {
		return fmt.Errorf("too many arguments: %d > %d", len(args), v.config.MaxArguments)
	}

	// Get command configuration
	cmdConfig, exists := v.config.Commands[command]
	if !exists {
		switch v.config.DefaultPolicy {
		case PolicyDeny:
			return fmt.Errorf("command %q not in allowlist", command)
		case PolicyAllow:
			return nil // Allow by default
		case PolicyPrompt:
			return fmt.Errorf("command %q requires manual approval", command)
		default:
			return fmt.Errorf("unknown default policy: %s", v.config.DefaultPolicy)
		}
	}

	// Check if command is enabled
	if !cmdConfig.Enabled {
		return fmt.Errorf("command %q is disabled", command)
	}

	// Check command-specific argument limits
	if cmdConfig.MaxArgs > 0 && len(args) > cmdConfig.MaxArgs {
		return fmt.Errorf("too many arguments for %q: %d > %d", command, len(args), cmdConfig.MaxArgs)
	}

	// Validate arguments
	return v.validateArguments(command, args, &cmdConfig)
}

// ValidatePath validates if a working directory path is allowed.
func (v *AllowlistValidator) ValidatePath(path string) error {
	if path == "" {
		return nil // Empty path is okay (uses current directory)
	}

	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Check against allowed directories
	if len(v.config.AllowedWorkDirs) == 0 {
		return nil // No restrictions configured
	}

	for _, allowedDir := range v.config.AllowedWorkDirs {
		if strings.HasPrefix(cleanPath, allowedDir) {
			return nil
		}
	}

	return fmt.Errorf("working directory not allowed: %s", path)
}

// SanitizeArgs removes potentially dangerous arguments.
func (v *AllowlistValidator) SanitizeArgs(args []string) ([]string, error) {
	sanitized := make([]string, 0, len(args))

	for _, arg := range args {
		// Remove null bytes
		if strings.Contains(arg, "\x00") {
			return nil, fmt.Errorf("null bytes not allowed in arguments")
		}

		// Remove arguments that look like shell injection
		if strings.ContainsAny(arg, ";&|><$`") {
			return nil, fmt.Errorf("shell metacharacters not allowed: %s", arg)
		}

		sanitized = append(sanitized, arg)
	}

	return sanitized, nil
}

// validateArguments checks command-specific argument validation.
func (v *AllowlistValidator) validateArguments(command string, args []string, config *CommandConfig) error {
	for _, arg := range args {
		// Check forbidden arguments
		for _, forbidden := range config.ForbiddenArgs {
			if arg == forbidden {
				return fmt.Errorf("forbidden argument for %q: %s", command, arg)
			}
		}

		// Check allowed arguments (if specified)
		if len(config.AllowedArgs) > 0 {
			allowed := false
			for _, allowedArg := range config.AllowedArgs {
				if arg == allowedArg {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("argument not allowed for %q: %s", command, arg)
			}
		}

		// Check argument patterns
		for _, pattern := range config.ArgPatterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				continue // Skip invalid patterns
			}
			if !regex.MatchString(arg) {
				return fmt.Errorf("argument %q doesn't match required pattern for %q", arg, command)
			}
		}
	}

	return nil
}
