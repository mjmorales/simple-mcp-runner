// Package types contains shared types for the MCP server
package types

import "time"

// CommandInfo represents information about a discovered command
type CommandInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	Executable  bool   `json:"executable"`
}

// CommandExecutionRequest represents a request to execute a command
type CommandExecutionRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	WorkDir string   `json:"workdir,omitempty"`
	Env     []string `json:"env,omitempty"`
	Timeout string   `json:"timeout,omitempty"` // Duration string like "30s"
}

// CommandExecutionResult represents the result of command execution
type CommandExecutionResult struct {
	Stdout       string        `json:"stdout"`
	Stderr       string        `json:"stderr"`
	ExitCode     int           `json:"exit_code"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration_ms"`
	TimedOut     bool          `json:"timed_out"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// CommandDiscoveryRequest represents a request to discover commands
type CommandDiscoveryRequest struct {
	Pattern     string   `json:"pattern,omitempty"`
	Paths       []string `json:"paths,omitempty"`      // Additional paths to search
	MaxResults  int      `json:"max_results,omitempty"` // Limit number of results
	IncludeDesc bool     `json:"include_desc,omitempty"` // Include descriptions
}

// CommandDiscoveryResult represents the result of command discovery
type CommandDiscoveryResult struct {
	Commands    []CommandInfo `json:"commands"`
	TotalFound  int           `json:"total_found"`
	Truncated   bool          `json:"truncated"`
	SearchPaths []string      `json:"search_paths"`
}