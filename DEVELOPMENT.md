# Development Guide

This guide provides information for developers who want to contribute to or extend the simple-mcp-runner project.

## Project Structure

```
.
├── cmd/                    # CLI commands
│   ├── root.go            # Root command and global flags
│   ├── run.go             # Main server run command
│   ├── validate.go        # Configuration validation
│   └── version.go         # Version information
├── internal/              # Private packages (not importable)
│   ├── config/           # Configuration management
│   │   ├── config.go     # Config structs and validation
│   │   └── config_test.go
│   ├── discovery/        # Command discovery logic
│   │   ├── discovery.go  # Discovery implementation
│   │   └── discovery_test.go
│   ├── errors/          # Custom error types
│   │   ├── errors.go    # Error definitions
│   │   └── errors_test.go
│   ├── executor/        # Command execution engine
│   │   ├── executor.go  # Execution with safety features
│   │   └── executor_test.go
│   ├── logger/          # Structured logging
│   │   └── logger.go    # Logger implementation
│   └── server/          # MCP server core
│       ├── server.go    # Server implementation
│       └── server_test.go
├── pkg/                  # Public packages (importable)
│   └── types/           # Shared type definitions
│       └── types.go
├── examples/            # Example scripts and configs
├── config.example.yaml  # Example configuration
├── Makefile            # Build and development tasks
├── go.mod              # Go module definition
├── go.sum              # Dependency checksums
└── main.go             # Entry point
```

## Development Setup

1. **Prerequisites**
   - Go 1.21 or later
   - Make (optional, for using Makefile)
   - golangci-lint (for linting)

2. **Clone and Setup**
   ```bash
   git clone https://github.com/mjmorales/simple-mcp-runner.git
   cd simple-mcp-runner
   go mod download
   ```

3. **Build**
   ```bash
   make build
   # or
   go build -o simple-mcp-runner .
   ```

## Code Organization

### Package Responsibilities

- **cmd**: CLI interface using Cobra
  - Handles command-line parsing
  - Initializes configuration
  - Sets up logging
  - Creates and runs the server

- **internal/config**: Configuration management
  - YAML parsing and validation
  - Default configuration
  - Security policy enforcement
  - Configuration schema versioning

- **internal/executor**: Safe command execution
  - Timeout management
  - Resource limits (output size, concurrency)
  - Security checks
  - Process lifecycle management

- **internal/discovery**: Command discovery
  - PATH scanning
  - Pattern matching
  - Command metadata
  - Caching for performance

- **internal/server**: MCP protocol implementation
  - Tool registration
  - Request handling
  - Transport management
  - Graceful shutdown

- **internal/errors**: Error handling
  - Typed errors for better handling
  - Error context and metadata
  - Stack trace capture
  - Error wrapping

- **internal/logger**: Structured logging
  - Multiple output formats (text, JSON)
  - Log levels
  - Context propagation
  - Performance optimization

## Testing

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# With race detection
make test-race

# Specific package
go test ./internal/executor
```

### Writing Tests

1. **Unit Tests**: Test individual functions and methods
   ```go
   func TestExecutor_validateRequest(t *testing.T) {
       // Test validation logic
   }
   ```

2. **Integration Tests**: Test component interactions
   ```go
   func TestServer_Run(t *testing.T) {
       // Test server lifecycle
   }
   ```

3. **Test Utilities**: Use testify for assertions
   ```go
   assert.Equal(t, expected, actual)
   require.NoError(t, err)
   ```

## Adding New Features

### Adding a New MCP Tool

1. Define the tool parameters in `pkg/types/types.go`:
   ```go
   type MyToolRequest struct {
       Field1 string `json:"field1"`
       Field2 int    `json:"field2,omitempty"`
   }
   ```

2. Implement the handler in `internal/server/server.go`:
   ```go
   func (s *Server) registerMyTool() error {
       tool := &mcp.Tool{
           Name:        "my_tool",
           Description: "Description of what the tool does",
       }
       
       handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[MyToolRequest]) (*mcp.CallToolResultFor[MyToolResult], error) {
           // Implementation
       }
       
       mcp.AddTool(s.mcpServer, tool, handler)
       return nil
   }
   ```

3. Register the tool in `registerTools()`:
   ```go
   if err := s.registerMyTool(); err != nil {
       return err
   }
   ```

### Adding Configuration Options

1. Update the config struct in `internal/config/config.go`
2. Add validation logic in `Validate()`
3. Update the default configuration in `Default()`
4. Document in `config.example.yaml`

## Security Considerations

When adding features, consider:

1. **Input Validation**: Always validate user input
2. **Path Traversal**: Use `filepath.Clean()` and check against allowed paths
3. **Command Injection**: Avoid shell expansion, use exec.Command directly
4. **Resource Limits**: Implement timeouts and size limits
5. **Error Messages**: Don't leak sensitive information

## Performance Guidelines

1. **Concurrency**: Use goroutines with proper synchronization
2. **Buffering**: Limit buffer sizes to prevent memory exhaustion
3. **Caching**: Cache expensive operations (like command discovery)
4. **Context**: Use context for cancellation and timeouts

## Code Style

1. **Formatting**: Use `gofmt` (enforced by `make fmt`)
2. **Linting**: Pass `golangci-lint` checks
3. **Comments**: Document exported types and functions
4. **Errors**: Return wrapped errors with context
5. **Testing**: Aim for >80% coverage on critical paths

## Debugging

### Enable Debug Logging
```bash
./simple-mcp-runner run --log-level debug
```

### Common Issues

1. **Permission Denied**
   - Check security configuration
   - Verify allowed paths and commands

2. **Command Not Found**
   - Check PATH environment
   - Verify command discovery settings

3. **Timeout Errors**
   - Increase timeout in configuration
   - Check for blocking operations

## Release Process

1. Update version in code
2. Run all tests: `make test`
3. Build release binaries: `make build`
4. Create git tag: `git tag v1.0.0`
5. Push tags: `git push --tags`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make fmt` and `make lint`
6. Submit a pull request

### Commit Message Format
```
type: short description

Longer explanation if needed.

Fixes #123
```

Types: feat, fix, docs, style, refactor, test, chore