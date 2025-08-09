package discovery

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mjmorales/simple-mcp-runner/pkg/config"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

func TestDiscoverer_Discover(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	disc := New(cfg, log)

	tests := []struct {
		name    string
		req     *types.CommandDiscoveryRequest
		wantErr bool
		check   func(t *testing.T, result *types.CommandDiscoveryResult)
	}{
		{
			name: "discover with wildcard pattern",
			req: &types.CommandDiscoveryRequest{
				Pattern: "*",
			},
			wantErr: false,
			check: func(t *testing.T, result *types.CommandDiscoveryResult) {
				if runtime.GOOS == "windows" {
					t.Skip("Skipping Unix command discovery test on Windows")
				}
				if len(result.Commands) == 0 {
					t.Error("expected to find at least one command")
				}
				// Should find common commands like echo, ls, etc.
				found := false
				for _, cmd := range result.Commands {
					if cmd.Name == "echo" || cmd.Name == "ls" {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected to find common commands like echo or ls")
				}
			},
		},
		{
			name: "discover with specific pattern",
			req: &types.CommandDiscoveryRequest{
				Pattern: "echo",
			},
			wantErr: false,
			check: func(t *testing.T, result *types.CommandDiscoveryResult) {
				if runtime.GOOS == "windows" {
					t.Skip("Skipping Unix command discovery test on Windows")
				}
				found := false
				for _, cmd := range result.Commands {
					if cmd.Name == "echo" {
						found = true
						if cmd.Path == "" {
							t.Error("expected path to be set for echo command")
						}
						break
					}
				}
				if !found {
					t.Error("expected to find echo command")
				}
			},
		},
		{
			name: "discover with max results",
			req: &types.CommandDiscoveryRequest{
				Pattern:    "*",
				MaxResults: 5,
			},
			wantErr: false,
			check: func(t *testing.T, result *types.CommandDiscoveryResult) {
				if len(result.Commands) > 5 {
					t.Errorf("expected at most 5 commands, got %d", len(result.Commands))
				}
				if result.TotalFound <= 5 {
					if result.Truncated {
						t.Error("should not be truncated when total <= max")
					}
				}
			},
		},
		{
			name: "discover with descriptions",
			req: &types.CommandDiscoveryRequest{
				Pattern:     "ls",
				IncludeDesc: true,
			},
			wantErr: false,
			check: func(t *testing.T, result *types.CommandDiscoveryResult) {
				if runtime.GOOS == "windows" {
					t.Skip("Skipping Unix command discovery test on Windows")
				}
				for _, cmd := range result.Commands {
					if cmd.Name == "ls" && cmd.Description == "" {
						t.Error("expected description for ls command")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := disc.Discover(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestDiscoverer_matchesPattern(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	disc := New(cfg, log)

	tests := []struct {
		name     string
		cmdName  string
		pattern  string
		expected bool
	}{
		{
			name:     "wildcard pattern with common command",
			cmdName:  "echo",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "wildcard pattern with uncommon command",
			cmdName:  "uncommon-cmd",
			pattern:  "*",
			expected: false,
		},
		{
			name:     "exact match",
			cmdName:  "echo",
			pattern:  "echo",
			expected: true,
		},
		{
			name:     "glob pattern",
			cmdName:  "git-status",
			pattern:  "git*",
			expected: true,
		},
		{
			name:     "substring match",
			cmdName:  "kubectl",
			pattern:  "kube",
			expected: true,
		},
		{
			name:     "no match",
			cmdName:  "echo",
			pattern:  "cat",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := disc.matchesPattern(tt.cmdName, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%s, %s) = %v, want %v", tt.cmdName, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestDiscoverer_isExecutable(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	disc := New(cfg, log)

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create executable file
	execFile := filepath.Join(tmpDir, "executable")
	if err := os.WriteFile(execFile, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create non-executable file
	nonExecFile := filepath.Join(tmpDir, "nonexecutable")
	if err := os.WriteFile(nonExecFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		file     string
		expected bool
	}{
		{
			name:     "executable file",
			file:     execFile,
			expected: runtime.GOOS != "windows", // On Windows, we check extensions
		},
		{
			name:     "non-executable file",
			file:     nonExecFile,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := os.Stat(tt.file)
			if err != nil {
				t.Fatal(err)
			}

			result := disc.isExecutable(info)
			if result != tt.expected {
				t.Errorf("isExecutable(%s) = %v, want %v", tt.file, result, tt.expected)
			}
		})
	}
}

func TestDiscoverer_getCommandDescription(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	disc := New(cfg, log)

	tests := []struct {
		name     string
		cmdName  string
		expected string
	}{
		{
			name:     "known command",
			cmdName:  "ls",
			expected: "List directory contents",
		},
		{
			name:     "known command with extension",
			cmdName:  "ls.exe",
			expected: "List directory contents",
		},
		{
			name:     "git subcommand",
			cmdName:  "git-status",
			expected: "Git subcommand",
		},
		{
			name:     "unknown command",
			cmdName:  "unknowncmd",
			expected: "System command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := disc.getCommandDescription(tt.cmdName)
			if result != tt.expected {
				t.Errorf("getCommandDescription(%s) = %v, want %v", tt.cmdName, result, tt.expected)
			}
		})
	}
}

func TestDiscoverer_Cache(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	disc := New(cfg, log)

	ctx := context.Background()
	req := &types.CommandDiscoveryRequest{
		Pattern: "echo",
	}

	// First call should populate cache
	result1, err := disc.Discover(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	// Second call should use cache
	result2, err := disc.Discover(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	// Results should be identical
	if len(result1.Commands) != len(result2.Commands) {
		t.Error("cached result differs from original")
	}

	// Clear cache
	disc.ClearCache()

	// Next call should not use cache
	result3, err := disc.Discover(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	// Should still find same commands (but fresh discovery)
	if len(result1.Commands) != len(result3.Commands) {
		t.Error("fresh discovery differs from original")
	}
}

func TestDeduplicateCommands(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	disc := New(cfg, log)

	commands := []types.CommandInfo{
		{Name: "echo", Path: "/bin/echo"},
		{Name: "ls", Path: "/bin/ls"},
		{Name: "echo", Path: "/usr/bin/echo"}, // Duplicate
		{Name: "cat", Path: "/bin/cat"},
		{Name: "ls", Path: "/usr/bin/ls"}, // Duplicate
	}

	result := disc.deduplicateCommands(commands)

	if len(result) != 3 {
		t.Errorf("expected 3 unique commands, got %d", len(result))
	}

	// Check that we kept the first occurrence
	for _, cmd := range result {
		if cmd.Name == "echo" && cmd.Path != "/bin/echo" {
			t.Error("deduplication should keep first occurrence")
		}
		if cmd.Name == "ls" && cmd.Path != "/bin/ls" {
			t.Error("deduplication should keep first occurrence")
		}
	}
}