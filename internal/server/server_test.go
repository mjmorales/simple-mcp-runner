package server

import (
	"context"
	"testing"
	"time"

	"github.com/mjmorales/simple-mcp-runner/pkg/config"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: Options{
				Config: config.Default(),
				Logger: nil, // Should use default
			},
			wantErr: false,
		},
		{
			name: "missing config",
			opts: Options{
				Config: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && srv == nil {
				t.Error("expected server instance")
			}
		})
	}
}

func TestServer_registerTools(t *testing.T) {
	cfg := config.Default()
	// Add a test command
	cfg.Commands = []config.Command{
		{
			Name:        "test_echo",
			Description: "Test echo command",
			Command:     "echo",
			Args:        []string{"test"},
		},
	}

	log, _ := logger.New(logger.DefaultOptions())
	srv, err := New(Options{
		Config: cfg,
		Logger: log,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Check that tools were registered
	// Note: We can't directly inspect MCP server tools, but we can verify no errors
	if srv.mcpServer == nil {
		t.Error("MCP server not initialized")
	}
}

func TestServer_Shutdown(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	srv, err := New(Options{
		Config: cfg,
		Logger: log,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Test shutdown when not running
	ctx := context.Background()
	err = srv.Shutdown(ctx)
	if err != nil {
		t.Errorf("shutdown when not running should not error: %v", err)
	}

	// Test shutdown timeout
	srv.running = true // Simulate running state
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// This should timeout since we're not actually running
	err = srv.Shutdown(ctx)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestServer_IsRunning(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	srv, err := New(Options{
		Config: cfg,
		Logger: log,
	})
	if err != nil {
		t.Fatal(err)
	}

	if srv.IsRunning() {
		t.Error("new server should not be running")
	}

	srv.running = true
	if !srv.IsRunning() {
		t.Error("server should report as running")
	}
}

func TestServer_GetStats(t *testing.T) {
	cfg := config.Default()
	log, _ := logger.New(logger.DefaultOptions())
	srv, err := New(Options{
		Config: cfg,
		Logger: log,
	})
	if err != nil {
		t.Fatal(err)
	}

	stats := srv.GetStats()
	if stats.Running {
		t.Error("new server should not be running")
	}
	if stats.ActiveCommands != 0 {
		t.Error("new server should have no active commands")
	}
}

func TestConfigCommandParams(t *testing.T) {
	// This test ensures the ConfigCommandParams struct is properly defined
	params := ConfigCommandParams{
		WorkDir: "/tmp",
		Args:    []string{"arg1", "arg2"},
	}

	if params.WorkDir != "/tmp" {
		t.Error("WorkDir not set correctly")
	}
	if len(params.Args) != 2 {
		t.Error("Args not set correctly")
	}
}