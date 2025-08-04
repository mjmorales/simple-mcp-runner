// Package executor handles safe command execution with timeouts and resource limits
package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mjmorales/simple-mcp-runner/internal/config"
	apperrors "github.com/mjmorales/simple-mcp-runner/internal/errors"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

// Executor manages command execution with safety features.
type Executor struct {
	config         *config.Config
	logger         *logger.Logger
	activeCommands int32
	semaphore      chan struct{}
}

// New creates a new executor instance.
func New(cfg *config.Config, log *logger.Logger) *Executor {
	maxConcurrent := cfg.Execution.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	return &Executor{
		config:    cfg,
		logger:    log,
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

// Execute runs a command with safety checks and resource limits.
func (e *Executor) Execute(ctx context.Context, req *types.CommandExecutionRequest) (*types.CommandExecutionResult, error) {
	e.logger.WithFields(map[string]any{
		"command": req.Command,
		"args":    req.Args,
		"workdir": req.WorkDir,
	}).Debug("executing command")

	// Validate request
	if err := e.validateRequest(req); err != nil {
		return nil, err
	}

	// Check security constraints
	if err := e.checkSecurity(req); err != nil {
		return nil, err
	}

	// Acquire semaphore
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		return nil, apperrors.TimeoutError("context cancelled while waiting for execution slot", "")
	}

	// Track active commands
	atomic.AddInt32(&e.activeCommands, 1)
	defer atomic.AddInt32(&e.activeCommands, -1)

	// Parse timeout
	timeout := e.getTimeout(req.Timeout)

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute the command
	result := e.executeCommand(execCtx, req)

	// Log execution
	e.logExecution(req, result)

	return result, nil
}

// ExecuteConfigCommand executes a pre-configured command.
func (e *Executor) ExecuteConfigCommand(ctx context.Context, cmd *config.Command, workDir string) (*types.CommandExecutionResult, error) {
	req := &types.CommandExecutionRequest{
		Command: cmd.Command,
		Args:    cmd.Args,
		WorkDir: workDir,
		Timeout: cmd.Timeout,
	}

	// Add environment variables
	if len(cmd.Env) > 0 {
		env := make([]string, 0, len(cmd.Env))
		for k, v := range cmd.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		req.Env = env
	}

	// Override workdir if specified in command config
	if cmd.WorkDir != "" {
		req.WorkDir = cmd.WorkDir
	}

	return e.Execute(ctx, req)
}

// GetActiveCount returns the number of active command executions.
func (e *Executor) GetActiveCount() int {
	return int(atomic.LoadInt32(&e.activeCommands))
}

// validateRequest validates the execution request.
func (e *Executor) validateRequest(req *types.CommandExecutionRequest) error {
	if req.Command == "" {
		return apperrors.ValidationError("command is required", "command")
	}

	// Check command length
	if e.config.Security.MaxCommandLength > 0 {
		cmdLen := len(req.Command) + len(strings.Join(req.Args, " "))
		if cmdLen > e.config.Security.MaxCommandLength {
			return apperrors.ValidationError(
				fmt.Sprintf("command too long: %d > %d", cmdLen, e.config.Security.MaxCommandLength),
				"command",
			)
		}
	}

	// Validate workdir if specified
	if req.WorkDir != "" {
		if !filepath.IsAbs(req.WorkDir) {
			return apperrors.ValidationError("workdir must be an absolute path", "workdir")
		}

		info, err := os.Stat(req.WorkDir)
		if err != nil {
			return apperrors.NotFoundError(fmt.Sprintf("workdir not found: %v", err), req.WorkDir)
		}

		if !info.IsDir() {
			return apperrors.ValidationError("workdir is not a directory", "workdir")
		}
	}

	return nil
}

// checkSecurity performs security checks on the command.
func (e *Executor) checkSecurity(req *types.CommandExecutionRequest) error {
	// Check if command is allowed
	if !e.config.IsCommandAllowed(req.Command) {
		return apperrors.PermissionError(
			fmt.Sprintf("command not allowed: %s", req.Command),
			req.Command,
		)
	}

	// Check if path is allowed
	if req.WorkDir != "" && !e.config.IsPathAllowed(req.WorkDir) {
		return apperrors.PermissionError(
			fmt.Sprintf("path not allowed: %s", req.WorkDir),
			req.WorkDir,
		)
	}

	// Check for shell injection attempts if shell expansion is disabled
	if e.config.Security.DisableShellExpansion {
		dangerous := []string{";", "&&", "||", "|", "`", "$", "(", ")", "{", "}", "<", ">", "&"}
		cmdStr := req.Command + " " + strings.Join(req.Args, " ")

		for _, char := range dangerous {
			if strings.Contains(cmdStr, char) {
				return apperrors.PermissionError(
					fmt.Sprintf("potentially dangerous character detected: %s", char),
					"command",
				)
			}
		}
	}

	return nil
}

// getTimeout determines the timeout for command execution.
func (e *Executor) getTimeout(requested string) time.Duration {
	// Parse requested timeout
	if requested != "" {
		if dur, err := time.ParseDuration(requested); err == nil {
			// Check against max timeout
			maxTimeout := e.parseTimeoutConfig(e.config.Execution.MaxTimeout, 5*time.Minute)
			if dur > maxTimeout {
				return maxTimeout
			}
			return dur
		}
	}

	// Use default timeout
	return e.parseTimeoutConfig(e.config.Execution.DefaultTimeout, 30*time.Second)
}

// parseTimeoutConfig parses a timeout configuration value.
func (e *Executor) parseTimeoutConfig(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}

	dur, err := time.ParseDuration(value)
	if err != nil {
		e.logger.WithFields(map[string]any{
			"value":         value,
			"error":         err,
			"using_default": defaultValue,
		}).Warn("invalid timeout configuration")
		return defaultValue
	}

	return dur
}

// executeCommand performs the actual command execution.
func (e *Executor) executeCommand(ctx context.Context, req *types.CommandExecutionRequest) *types.CommandExecutionResult {
	startTime := time.Now()
	result := &types.CommandExecutionResult{
		StartTime: startTime,
		ExitCode:  -1,
	}

	// Create command
	cmd := exec.CommandContext(ctx, req.Command, req.Args...)

	// Set working directory
	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}

	// Set environment
	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	}

	// Create buffers for output with size limits
	stdout := &limitedBuffer{limit: e.config.Execution.MaxOutputSize}
	stderr := &limitedBuffer{limit: e.config.Execution.MaxOutputSize}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Start the command
	err := cmd.Start()
	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		result.ErrorMessage = fmt.Sprintf("failed to start command: %v", err)
		return result
	}

	// Wait for completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		// Command completed
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()

		if err != nil {
			exitErr := &exec.ExitError{}
			if errors.As(err, &exitErr) {
				result.ExitCode = exitErr.ExitCode()
			} else {
				result.ErrorMessage = err.Error()
			}
		} else {
			result.ExitCode = 0
		}

	case <-ctx.Done():
		// Timeout or cancellation
		result.TimedOut = true
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)

		// Try graceful termination first
		if cmd.Process != nil {
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				// Process might have already exited, which is fine
				e.logger.Debug("failed to send interrupt signal", "error", err)
			}

			// Wait for kill timeout
			killTimeout := e.parseTimeoutConfig(e.config.Execution.KillTimeout, 5*time.Second)
			killTimer := time.NewTimer(killTimeout)

			select {
			case <-done:
				// Process terminated gracefully
				killTimer.Stop()
			case <-killTimer.C:
				// Force kill
				if err := cmd.Process.Kill(); err != nil {
					e.logger.Debug("failed to kill process", "error", err)
				}
				<-done
			}
		}

		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.ErrorMessage = "command timed out"
	}

	return result
}

// logExecution logs command execution details.
func (e *Executor) logExecution(req *types.CommandExecutionRequest, result *types.CommandExecutionResult) {
	fields := map[string]any{
		"command":   req.Command,
		"args":      req.Args,
		"workdir":   req.WorkDir,
		"exit_code": result.ExitCode,
		"duration":  result.Duration.Milliseconds(),
		"timed_out": result.TimedOut,
	}

	if result.ErrorMessage != "" {
		fields["error"] = result.ErrorMessage
	}

	// Log at appropriate level
	if result.ExitCode == 0 && !result.TimedOut {
		e.logger.WithFields(fields).Info("command executed successfully")
	} else {
		e.logger.WithFields(fields).Error("command execution failed")
	}

	// Log output at debug level
	if e.logger.IsDebugEnabled() {
		if result.Stdout != "" {
			e.logger.WithFields(map[string]any{
				"command": req.Command,
				"output":  truncateString(result.Stdout, 1000),
			}).Debug("command stdout")
		}
		if result.Stderr != "" {
			e.logger.WithFields(map[string]any{
				"command": req.Command,
				"output":  truncateString(result.Stderr, 1000),
			}).Debug("command stderr")
		}
	}
}

// limitedBuffer is a buffer that limits the amount of data stored.
type limitedBuffer struct {
	buf   bytes.Buffer
	limit int64
	size  int64
	mu    sync.Mutex
}

func (b *limitedBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.limit <= 0 {
		return b.buf.Write(p)
	}

	remaining := b.limit - b.size
	if remaining <= 0 {
		return len(p), nil // Discard extra data
	}

	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err = b.buf.Write(p)
	b.size += int64(n)
	return n, err
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
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

// Build returns the command execution request.
func (b *CommandBuilder) Build() *types.CommandExecutionRequest {
	return b.req
}
