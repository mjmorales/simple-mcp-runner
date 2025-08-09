package main

import (
	"testing"

	"github.com/mjmorales/simple-mcp-runner/pkg/config"
	"github.com/mjmorales/simple-mcp-runner/pkg/discovery"
	"github.com/mjmorales/simple-mcp-runner/pkg/executor"
	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

// TestPublicAPIUsage demonstrates how external users can use the public API.
func TestPublicAPIUsage(t *testing.T) {
	// Create a configuration
	cfg := config.Default()
	cfg.App = "external-app"

	// Test configuration loading from bytes
	yamlConfig := `
app: test-app
transport: stdio
security:
  max_command_length: 500
execution:
  default_timeout: 10s
`
	cfgFromYAML, err := config.LoadFromBytes([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("Failed to load config from YAML: %v", err)
	}

	if cfgFromYAML.App != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", cfgFromYAML.App)
	}

	// Test command builder
	req := executor.NewCommandBuilder("echo").
		WithArgs("hello", "world").
		WithTimeout("5s").
		WithWorkDir("/tmp").
		Build()

	if req.Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", req.Command)
	}

	if len(req.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(req.Args))
	}

	// Test discovery builder
	discReq := discovery.NewDiscoveryBuilder().
		WithPattern("git*").
		WithMaxResults(10).
		WithDescriptions(true).
		Build()

	if discReq.Pattern != "git*" {
		t.Errorf("Expected pattern 'git*', got '%s'", discReq.Pattern)
	}

	if discReq.MaxResults != 10 {
		t.Errorf("Expected max results 10, got %d", discReq.MaxResults)
	}

	// Test filter
	filter := discovery.NewFilterChain(
		&discovery.PatternFilter{Patterns: []string{"git", "npm"}},
		&discovery.PathFilter{AllowedPaths: []string{"/usr/bin", "/usr/local/bin"}},
	)

	cmd := types.CommandInfo{
		Name: "git",
		Path: "/usr/bin/git",
	}

	if !filter.ShouldInclude(cmd) {
		t.Error("Filter should include git command")
	}

	cmd2 := types.CommandInfo{
		Name: "unknown",
		Path: "/usr/bin/unknown",
	}

	if filter.ShouldInclude(cmd2) {
		t.Error("Filter should not include unknown command")
	}

	// Test config validation
	invalidCfg := &config.Config{
		App:       "", // Invalid - empty app name
		Transport: "stdio",
	}

	if err := invalidCfg.Validate(); err == nil {
		t.Error("Expected validation error for empty app name")
	}

	// Test security checks
	if !cfg.IsCommandAllowed("ls") {
		t.Error("Expected ls command to be allowed")
	}

	if cfg.IsCommandAllowed("rm") {
		t.Error("Expected rm command to be blocked")
	}

	// Test path checking
	if !cfg.IsPathAllowed("/tmp") {
		t.Error("Expected /tmp to be allowed")
	}
}

// TestConfigurationTypes demonstrates configuration usage.
func TestConfigurationTypes(t *testing.T) {
	cfg := &config.Config{
		App:       "test-app",
		Transport: "stdio",
		Commands: []config.Command{
			{
				Name:        "test_cmd",
				Description: "A test command",
				Command:     "echo",
				Args:        []string{"test"},
				Timeout:     "30s",
				AllowArgs:   true,
			},
		},
		Security: config.SecurityConfig{
			MaxCommandLength:      1000,
			DisableShellExpansion: true,
			BlockedCommands:       []string{"rm", "dd"},
		},
		Execution: config.ExecutionConfig{
			DefaultTimeout: "30s",
			MaxTimeout:     "5m",
			MaxConcurrent:  5,
			MaxOutputSize:  1024 * 1024,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Discovery: config.DiscoveryConfig{
			MaxResults:      50,
			CommonCommands:  []string{"ls", "cat", "git"},
			AdditionalPaths: []string{"/opt/bin"},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should not fail validation: %v", err)
	}

	// Test command timeout retrieval
	cmd := cfg.Commands[0]
	timeout := cmd.GetTimeout(0)
	if timeout.Seconds() != 30 {
		t.Errorf("Expected 30 second timeout, got %v", timeout)
	}
}

func TestInterfaceUsage(t *testing.T) {
	// Example showing how external code can work with interfaces
	var exec executor.Executor
	var disc discovery.Discoverer

	// These would be implemented by the internal packages
	_ = exec
	_ = disc

	// Test that we can create builders without implementations
	cmdReq := executor.NewCommandBuilder("test").WithArgs("arg1").Build()
	if cmdReq.Command != "test" {
		t.Error("Command builder failed")
	}

	discReq := discovery.NewDiscoveryBuilder().WithPattern("*").Build()
	if discReq.Pattern != "*" {
		t.Error("Discovery builder failed")
	}
}