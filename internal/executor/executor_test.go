package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mjmorales/simple-mcp-runner/pkg/config"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

type testCase struct {
	name    string
	req     *types.CommandExecutionRequest
	wantErr bool
	check   func(t *testing.T, result *types.CommandExecutionResult)
}

func TestExecutor_Execute(t *testing.T) {
	// Setup
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	exec := New(cfg, log)

	tests := []testCase{
		{
			name: "simple echo command",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				Args:    []string{"hello", "world"},
			},
			wantErr: false,
			check: func(t *testing.T, result *types.CommandExecutionResult) {
				if result.ExitCode != 0 {
					t.Errorf("expected exit code 0, got %d", result.ExitCode)
				}
				if !strings.Contains(result.Stdout, "hello world") {
					t.Errorf("expected stdout to contain 'hello world', got %s", result.Stdout)
				}
			},
		},
		getTimeoutTestCase(),
		{
			name: "command not found",
			req: &types.CommandExecutionRequest{
				Command: "nonexistentcommand123",
			},
			wantErr: false,
			check: func(t *testing.T, result *types.CommandExecutionResult) {
				if result.ExitCode != -1 {
					t.Errorf("expected exit code -1, got %d", result.ExitCode)
				}
				if result.ErrorMessage == "" {
					t.Error("expected error message for command not found")
				}
			},
		},
		{
			name: "empty command",
			req: &types.CommandExecutionRequest{
				Command: "",
			},
			wantErr: true,
		},
		{
			name: "blocked command",
			req: &types.CommandExecutionRequest{
				Command: "rm",
				Args:    []string{"-rf", "/"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := exec.Execute(ctx, tt.req)

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

func TestExecutor_validateRequest(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	exec := New(cfg, log)

	tests := []struct {
		name    string
		req     *types.CommandExecutionRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				Args:    []string{"test"},
			},
			wantErr: false,
		},
		{
			name: "empty command",
			req: &types.CommandExecutionRequest{
				Command: "",
			},
			wantErr: true,
		},
		{
			name: "command too long",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				Args:    []string{strings.Repeat("a", 2000)},
			},
			wantErr: true,
		},
		{
			name: "relative workdir",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				WorkDir: "relative/path",
			},
			wantErr: true,
		},
		{
			name: "nonexistent workdir",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				WorkDir: "/nonexistent/directory/path",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.validateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecutor_checkSecurity(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	exec := New(cfg, log)

	tests := []struct {
		name    string
		req     *types.CommandExecutionRequest
		wantErr bool
	}{
		{
			name: "allowed command",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				Args:    []string{"test"},
			},
			wantErr: false,
		},
		{
			name: "blocked command",
			req: &types.CommandExecutionRequest{
				Command: "rm",
			},
			wantErr: true,
		},
		{
			name: "shell injection attempt",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				Args:    []string{"test; rm -rf /"},
			},
			wantErr: true,
		},
		{
			name: "command with pipe",
			req: &types.CommandExecutionRequest{
				Command: "echo",
				Args:    []string{"test | grep something"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.checkSecurity(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSecurity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecutor_getTimeout(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	exec := New(cfg, log)

	tests := []struct {
		name      string
		requested string
		expected  time.Duration
	}{
		{
			name:      "empty uses default",
			requested: "",
			expected:  30 * time.Second,
		},
		{
			name:      "valid duration",
			requested: "5s",
			expected:  5 * time.Second,
		},
		{
			name:      "exceeds max timeout",
			requested: "10m",
			expected:  5 * time.Minute,
		},
		{
			name:      "invalid duration uses default",
			requested: "invalid",
			expected:  30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exec.getTimeout(tt.requested)
			if result != tt.expected {
				t.Errorf("getTimeout() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommandBuilder(t *testing.T) {
	builder := NewCommandBuilder("echo").
		WithArgs("hello", "world").
		WithWorkDir("/tmp").
		WithTimeout("5s").
		WithEnv(map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		})

	req := builder.Build()

	if req.Command != "echo" {
		t.Errorf("expected command 'echo', got %s", req.Command)
	}

	if len(req.Args) != 2 || req.Args[0] != "hello" || req.Args[1] != "world" {
		t.Errorf("expected args [hello world], got %v", req.Args)
	}

	if req.WorkDir != "/tmp" {
		t.Errorf("expected workdir '/tmp', got %s", req.WorkDir)
	}

	if req.Timeout != "5s" {
		t.Errorf("expected timeout '5s', got %s", req.Timeout)
	}

	if len(req.Env) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(req.Env))
	}
}

func TestLimitedBuffer(t *testing.T) {
	buf := &limitedBuffer{limit: 10}

	// Write within limit
	n, err := buf.Write([]byte("hello"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	// Write more data
	n, err = buf.Write([]byte("world!"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 5 { // Should only write 5 bytes due to limit
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	result := buf.String()
	if result != "helloworld" {
		t.Errorf("expected 'helloworld', got %s", result)
	}

	// Try to write more (should be discarded)
	n, err = buf.Write([]byte("extra"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 5 { // Reports bytes "written" even if discarded
		t.Errorf("expected 5 bytes reported, got %d", n)
	}

	result = buf.String()
	if result != "helloworld" {
		t.Errorf("buffer should not change after limit, got %s", result)
	}
}
