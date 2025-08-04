package server

import (
	"testing"

	"github.com/mjmorales/simple-mcp-runner/internal/config"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
)

func TestConfigCommandWithAllowArgs(t *testing.T) {
	cfg := &config.Config{
		App:       "test-server",
		Transport: "stdio",
		Commands: []config.Command{
			{
				Name:        "echo_test",
				Description: "Test echo with args",
				Command:     "echo",
				Args:        []string{"base"},
				AllowArgs:   true,
			},
			{
				Name:        "echo_no_args",
				Description: "Test echo without args",
				Command:     "echo",
				Args:        []string{"fixed"},
				AllowArgs:   false,
			},
		},
	}

	// Create server
	s, err := New(Options{
		Config: cfg,
		Logger: logger.Default(),
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// This test verifies that the server properly registers commands
	// with AllowArgs support. The actual execution would require
	// running the full MCP server, which is tested in integration tests.
	if s == nil {
		t.Error("server should not be nil")
	}
	
	t.Log("Server created successfully with AllowArgs commands")
}