// Package executor provides interfaces and types for command execution
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/mjmorales/simple-mcp-runner/pkg/config"
	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

// Executor interface defines the contract for command execution.
type Executor interface {
	// Execute runs a command with safety checks and resource limits.
	Execute(ctx context.Context, req *types.CommandExecutionRequest) (*types.CommandExecutionResult, error)

	// ExecuteConfigCommand executes a pre-configured command.
	ExecuteConfigCommand(ctx context.Context, cmd *config.Command, workDir string) (*types.CommandExecutionResult, error)

	// GetActiveCount returns the number of active command executions.
	GetActiveCount() int
}

// CommandBuilder helps build command execution requests.
type CommandBuilder struct {
	req *types.CommandExecutionRequest
}

// NewCommandBuilder creates a new command builder.
func NewCommandBuilder(command string) *CommandBuilder {
	return &CommandBuilder{
		req: &types.CommandExecutionRequest{
			Command: command,
		},
	}
}

// WithArgs sets the command arguments.
func (b *CommandBuilder) WithArgs(args ...string) *CommandBuilder {
	b.req.Args = args
	return b
}

// WithWorkDir sets the working directory.
func (b *CommandBuilder) WithWorkDir(dir string) *CommandBuilder {
	b.req.WorkDir = dir
	return b
}

// WithTimeout sets the timeout.
func (b *CommandBuilder) WithTimeout(timeout string) *CommandBuilder {
	b.req.Timeout = timeout
	return b
}

// WithEnv adds environment variables.
func (b *CommandBuilder) WithEnv(env map[string]string) *CommandBuilder {
	envList := make([]string, 0, len(env))
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	b.req.Env = envList
	return b
}

// WithEnvList sets environment variables from a list.
func (b *CommandBuilder) WithEnvList(env []string) *CommandBuilder {
	b.req.Env = env
	return b
}

// Build returns the command execution request.
func (b *CommandBuilder) Build() *types.CommandExecutionRequest {
	return b.req
}

// BuildAndExecute builds the request and executes it.
func (b *CommandBuilder) BuildAndExecute(ctx context.Context, executor Executor) (*types.CommandExecutionResult, error) {
	return executor.Execute(ctx, b.req)
}

// ExecutionStats provides statistics about command execution.
type ExecutionStats struct {
	ActiveCommands int
	TotalExecuted  int64
	TotalErrors    int64
	AverageLatency time.Duration
}

// StatsCollector interface for collecting execution statistics.
type StatsCollector interface {
	GetStats() ExecutionStats
	RecordExecution(result *types.CommandExecutionResult)
}