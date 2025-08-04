// Package server implements the MCP server functionality
package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mjmorales/simple-mcp-runner/internal/config"
	"github.com/mjmorales/simple-mcp-runner/internal/discovery"
	apperrors "github.com/mjmorales/simple-mcp-runner/internal/errors"
	"github.com/mjmorales/simple-mcp-runner/internal/executor"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
	"github.com/mjmorales/simple-mcp-runner/pkg/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server represents the MCP server.
type Server struct {
	config     *config.Config
	logger     *logger.Logger
	executor   *executor.Executor
	discoverer *discovery.Discoverer
	mcpServer  *mcp.Server

	mu       sync.RWMutex
	running  bool
	shutdown chan struct{}
}

// Options for creating a new server.
type Options struct {
	Config *config.Config
	Logger *logger.Logger
}

// New creates a new MCP server instance.
func New(opts Options) (*Server, error) {
	if opts.Config == nil {
		return nil, apperrors.ConfigurationError("config is required")
	}

	if opts.Logger == nil {
		opts.Logger = logger.Default()
	}

	// Create executor
	exec := executor.New(opts.Config, opts.Logger)

	// Create discoverer
	disc := discovery.New(opts.Config, opts.Logger)

	// Create MCP implementation
	impl := &mcp.Implementation{
		Name:    opts.Config.App,
		Version: "1.0.0",
	}

	// Create MCP server
	mcpServer := mcp.NewServer(impl, nil)

	s := &Server{
		config:     opts.Config,
		logger:     opts.Logger,
		executor:   exec,
		discoverer: disc,
		mcpServer:  mcpServer,
		shutdown:   make(chan struct{}),
	}

	// Register tools
	if err := s.registerTools(); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorTypeConfiguration, "failed to register tools")
	}

	return s, nil
}

// Run starts the MCP server.
func (s *Server) Run(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return apperrors.InternalError("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	s.logger.Info("starting MCP server",
		"app", s.config.App,
		"transport", s.config.Transport,
	)

	// Create transport based on config
	transport, err := s.createTransport()
	if err != nil {
		return err
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.mcpServer.Run(ctx, transport)
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		s.logger.Info("received shutdown signal", "signal", sig)
		s.shutdown <- struct{}{}
		cancel()

		// Wait for graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		select {
		case err := <-errChan:
			if err != nil && !errors.Is(err, context.Canceled) {
				return apperrors.Wrap(err, apperrors.ErrorTypeInternal, "server error during shutdown")
			}
		case <-shutdownCtx.Done():
			s.logger.Warn("shutdown timeout exceeded")
		}

	case err := <-errChan:
		if err != nil {
			return apperrors.Wrap(err, apperrors.ErrorTypeInternal, "server error")
		}

	case <-ctx.Done():
		s.logger.Info("context cancelled")
		return ctx.Err()
	}

	s.logger.Info("MCP server stopped")
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return nil
	}

	s.logger.Info("shutting down MCP server")

	// Signal shutdown
	select {
	case s.shutdown <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for server to stop
	deadline := time.Now().Add(10 * time.Second)
	for {
		s.mu.RLock()
		running = s.running
		s.mu.RUnlock()

		if !running {
			break
		}

		if time.Now().After(deadline) {
			return apperrors.TimeoutError("shutdown timeout", "10s")
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// createTransport creates the appropriate transport based on configuration.
func (s *Server) createTransport() (mcp.Transport, error) {
	switch s.config.Transport {
	case "stdio":
		return mcp.NewStdioTransport(), nil
	default:
		return nil, apperrors.ConfigurationError(fmt.Sprintf("unsupported transport: %s", s.config.Transport))
	}
}

// registerTools registers all MCP tools.
func (s *Server) registerTools() error {
	// Register configured commands
	for _, cmd := range s.config.Commands {
		if err := s.registerConfigCommand(cmd); err != nil {
			return err
		}
	}

	// Register discovery tool
	if err := s.registerDiscoveryTool(); err != nil {
		return err
	}

	// Register execution tool
	if err := s.registerExecutionTool(); err != nil {
		return err
	}

	return nil
}

// registerConfigCommand registers a configured command as a tool.
func (s *Server) registerConfigCommand(cmd config.Command) error {
	// Create a copy of cmd for the closure
	cmdCopy := cmd

	tool := &mcp.Tool{
		Name:        cmd.Name,
		Description: cmd.Description,
	}

	handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ConfigCommandParams]) (*mcp.CallToolResultFor[types.CommandExecutionResult], error) {
		// Execute the configured command
		result, err := s.executor.ExecuteConfigCommand(ctx, &cmdCopy, params.Arguments.WorkDir)
		if err != nil {
			s.logger.WithError(err).Error("config command execution failed",
				"command", cmdCopy.Name,
			)

			// Return error result instead of failing
			return &mcp.CallToolResultFor[types.CommandExecutionResult]{
				StructuredContent: types.CommandExecutionResult{
					ExitCode:     -1,
					ErrorMessage: err.Error(),
					StartTime:    time.Now(),
					EndTime:      time.Now(),
				},
			}, nil
		}

		return &mcp.CallToolResultFor[types.CommandExecutionResult]{
			StructuredContent: *result,
		}, nil
	}

	mcp.AddTool(s.mcpServer, tool, handler)

	s.logger.Debug("registered config command tool",
		"name", cmd.Name,
		"command", cmd.Command,
	)

	return nil
}

// registerDiscoveryTool registers the command discovery tool.
func (s *Server) registerDiscoveryTool() error {
	tool := &mcp.Tool{
		Name:        "discover_commands",
		Description: "Discover available system commands. Use pattern parameter to filter commands (e.g., 'git*', 'npm'). Returns command names, paths, and descriptions.",
	}

	handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[types.CommandDiscoveryRequest]) (*mcp.CallToolResultFor[types.CommandDiscoveryResult], error) {
		result, err := s.discoverer.Discover(ctx, &params.Arguments)
		if err != nil {
			s.logger.WithError(err).Error("command discovery failed")
			return nil, err
		}

		return &mcp.CallToolResultFor[types.CommandDiscoveryResult]{
			StructuredContent: *result,
		}, nil
	}

	mcp.AddTool(s.mcpServer, tool, handler)

	s.logger.Debug("registered discovery tool")

	return nil
}

// registerExecutionTool registers the command execution tool.
func (s *Server) registerExecutionTool() error {
	tool := &mcp.Tool{
		Name:        "execute_command",
		Description: "Execute a system command with optional arguments and working directory. Returns stdout, stderr, and exit code.",
	}

	handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[types.CommandExecutionRequest]) (*mcp.CallToolResultFor[types.CommandExecutionResult], error) {
		// Log the request
		s.logger.Info("executing command",
			"command", params.Arguments.Command,
			"args", params.Arguments.Args,
			"workdir", params.Arguments.WorkDir,
		)

		result, err := s.executor.Execute(ctx, &params.Arguments)
		if err != nil {
			s.logger.WithError(err).Error("command execution failed")

			// Return error result instead of failing
			return &mcp.CallToolResultFor[types.CommandExecutionResult]{
				StructuredContent: types.CommandExecutionResult{
					ExitCode:     -1,
					ErrorMessage: err.Error(),
					StartTime:    time.Now(),
					EndTime:      time.Now(),
				},
			}, nil
		}

		return &mcp.CallToolResultFor[types.CommandExecutionResult]{
			StructuredContent: *result,
		}, nil
	}

	mcp.AddTool(s.mcpServer, tool, handler)

	s.logger.Debug("registered execution tool")

	return nil
}

// GetStats returns server statistics.
func (s *Server) GetStats() ServerStats {
	return ServerStats{
		Running:        s.IsRunning(),
		ActiveCommands: s.executor.GetActiveCount(),
	}
}

// IsRunning returns true if the server is running.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ServerStats contains server statistics.
type ServerStats struct {
	Running        bool
	ActiveCommands int
}

// ConfigCommandParams represents parameters for configured commands.
type ConfigCommandParams struct {
	WorkDir string   `json:"workdir,omitempty"`
	Args    []string `json:"args,omitempty"` // Only if AllowArgs is true
}
