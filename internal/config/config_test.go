package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	
	assert.Equal(t, "simple-mcp-runner", cfg.App)
	assert.Equal(t, "stdio", cfg.Transport)
	assert.Equal(t, "1.0", cfg.Version)
	
	// Security defaults
	assert.Equal(t, 1000, cfg.Security.MaxCommandLength)
	assert.True(t, cfg.Security.DisableShellExpansion)
	assert.NotEmpty(t, cfg.Security.BlockedCommands)
	
	// Execution defaults
	assert.Equal(t, "30s", cfg.Execution.DefaultTimeout)
	assert.Equal(t, "5m", cfg.Execution.MaxTimeout)
	assert.Equal(t, 10, cfg.Execution.MaxConcurrent)
	assert.Equal(t, int64(10*1024*1024), cfg.Execution.MaxOutputSize)
	
	// Logging defaults
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)
	assert.Equal(t, "stderr", cfg.Logging.Output)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				App:       "test-app",
				Transport: "stdio",
			},
			wantErr: false,
		},
		{
			name: "missing app name",
			config: &Config{
				App:       "",
				Transport: "stdio",
			},
			wantErr: true,
			errMsg:  "app name is required",
		},
		{
			name: "invalid transport",
			config: &Config{
				App:       "test-app",
				Transport: "http",
			},
			wantErr: true,
			errMsg:  "only 'stdio' transport is supported",
		},
		{
			name: "invalid command name",
			config: &Config{
				App:       "test-app",
				Transport: "stdio",
				Commands: []Command{
					{
						Name:        "123-invalid",
						Description: "test",
						Command:     "echo",
					},
				},
			},
			wantErr: true,
			errMsg:  "command name must be alphanumeric",
		},
		{
			name: "duplicate command names",
			config: &Config{
				App:       "test-app",
				Transport: "stdio",
				Commands: []Command{
					{
						Name:        "test",
						Description: "test 1",
						Command:     "echo",
					},
					{
						Name:        "test",
						Description: "test 2",
						Command:     "echo",
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate command name",
		},
		{
			name: "invalid timeout",
			config: &Config{
				App:       "test-app",
				Transport: "stdio",
				Commands: []Command{
					{
						Name:        "test",
						Description: "test",
						Command:     "echo",
						Timeout:     "invalid",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid timeout format",
		},
		{
			name: "relative workdir",
			config: &Config{
				App:       "test-app",
				Transport: "stdio",
				Commands: []Command{
					{
						Name:        "test",
						Description: "test",
						Command:     "echo",
						WorkDir:     "relative/path",
					},
				},
			},
			wantErr: true,
			errMsg:  "workdir must be an absolute path",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with defaults and override
			cfg := Default()
			cfg.App = tt.config.App
			cfg.Transport = tt.config.Transport
			cfg.Commands = tt.config.Commands
			
			err := cfg.Validate()
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	configContent := `
app: test-app
transport: stdio
commands:
  - name: hello
    description: Say hello
    command: echo
    args: ["Hello, World!"]
security:
  max_command_length: 500
execution:
  default_timeout: 10s
logging:
  level: debug
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	
	// Load config
	cfg, err := Load(configPath)
	require.NoError(t, err)
	
	// Verify loaded values
	assert.Equal(t, "test-app", cfg.App)
	assert.Equal(t, "stdio", cfg.Transport)
	assert.Len(t, cfg.Commands, 1)
	assert.Equal(t, "hello", cfg.Commands[0].Name)
	assert.Equal(t, []string{"Hello, World!"}, cfg.Commands[0].Args)
	assert.Equal(t, 500, cfg.Security.MaxCommandLength)
	assert.Equal(t, "10s", cfg.Execution.DefaultTimeout)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestIsCommandAllowed(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		command  string
		expected bool
	}{
		{
			name: "blocked command",
			config: &Config{
				Security: SecurityConfig{
					BlockedCommands: []string{"rm", "dd"},
				},
			},
			command:  "rm",
			expected: false,
		},
		{
			name: "blocked command with path",
			config: &Config{
				Security: SecurityConfig{
					BlockedCommands: []string{"rm", "dd"},
				},
			},
			command:  "rm/something",
			expected: false,
		},
		{
			name: "allowed command in whitelist",
			config: &Config{
				Security: SecurityConfig{
					AllowedCommands: []string{"echo", "ls"},
				},
			},
			command:  "echo",
			expected: true,
		},
		{
			name: "not in whitelist",
			config: &Config{
				Security: SecurityConfig{
					AllowedCommands: []string{"echo", "ls"},
				},
			},
			command:  "cat",
			expected: false,
		},
		{
			name: "no restrictions",
			config: &Config{
				Security: SecurityConfig{},
			},
			command:  "anything",
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsCommandAllowed(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPathAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix path tests on Windows")
	}
	
	tests := []struct {
		name     string
		config   *Config
		path     string
		expected bool
	}{
		{
			name: "no path restrictions",
			config: &Config{
				Security: SecurityConfig{},
			},
			path:     "/any/path",
			expected: true,
		},
		{
			name: "allowed path",
			config: &Config{
				Security: SecurityConfig{
					AllowedPaths: []string{"/home/user", "/tmp"},
				},
			},
			path:     "/home/user/file.txt",
			expected: true,
		},
		{
			name: "not in allowed paths",
			config: &Config{
				Security: SecurityConfig{
					AllowedPaths: []string{"/home/user", "/tmp"},
				},
			},
			path:     "/etc/passwd",
			expected: false,
		},
		{
			name: "relative path converted to absolute",
			config: &Config{
				Security: SecurityConfig{
					AllowedPaths: []string{"/tmp"},
				},
			},
			path:     "../../../tmp/file",
			expected: false, // Will likely resolve outside /tmp
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsPathAllowed(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidCommandName(t *testing.T) {
	tests := []struct {
		name     string
		cmdName  string
		expected bool
	}{
		{"valid simple", "test", true},
		{"valid with numbers", "test123", true},
		{"valid with underscore", "test_cmd", true},
		{"valid complex", "my_test_command_123", true},
		{"invalid starts with number", "123test", false},
		{"invalid special chars", "test-cmd", false},
		{"invalid spaces", "test cmd", false},
		{"empty", "", false},
		{"too long", "a_very_long_command_name_that_exceeds_fifty_characters", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCommandName(tt.cmdName)
			assert.Equal(t, tt.expected, result)
		})
	}
}