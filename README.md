# Simple MCP Runner

A production-ready Model Context Protocol (MCP) server that provides Language Learning Models (LLMs) with a safe interface to discover and execute system commands on the local machine.

## Features

- **Command Discovery**: Pattern-based discovery of available system commands
- **Safe Execution**: Configurable security policies, timeouts, and resource limits
- **Structured Configuration**: YAML-based configuration with validation
- **Production Logging**: Structured logging with multiple output formats
- **Graceful Shutdown**: Proper signal handling and cleanup
- **Comprehensive Testing**: Unit and integration tests for critical functionality
- **Clean Architecture**: Modular design with separation of concerns

## Installation

### From Source

```bash
go install github.com/mjmorales/simple-mcp-runner@latest
```

### Building Locally

```bash
git clone https://github.com/mjmorales/simple-mcp-runner.git
cd simple-mcp-runner
go build -o simple-mcp-runner
```

## Quick Start

1. Run with default configuration:
```bash
simple-mcp-runner run
```

2. Run with custom configuration:
```bash
simple-mcp-runner run --config config.yaml
```

3. Run with debug logging:
```bash
simple-mcp-runner run --log-level debug
```

## Configuration

Create a `config.yaml` file to customize the server behavior:

```yaml
app: my-mcp-server
transport: stdio

# Define custom commands
commands:
  - name: list_files
    description: List files in current directory
    command: ls
    args: ["-la"]
    
  - name: show_date
    description: Show current date and time
    command: date

# Security settings
security:
  # Maximum command length
  max_command_length: 1000
  
  # Disable shell expansion for safety
  disable_shell_expansion: true
  
  # Block dangerous commands
  blocked_commands:
    - rm
    - dd
    - mkfs
    - shutdown
    - reboot
    
  # Or use a whitelist approach
  # allowed_commands:
  #   - echo
  #   - ls
  #   - cat
  
  # Restrict execution to specific paths
  # allowed_paths:
  #   - /home/user/projects
  #   - /tmp

# Execution limits
execution:
  default_timeout: 30s
  max_timeout: 5m
  max_concurrent: 10
  max_output_size: 10485760  # 10MB
  kill_timeout: 5s

# Logging configuration
logging:
  level: info  # debug, info, warn, error
  format: text # text, json
  output: stderr
  include_source: false

# Command discovery settings
discovery:
  max_results: 100
  common_commands:
    - ls
    - cat
    - grep
    - find
    - git
    - npm
    - go
    - python
    - node
```

## Usage

### CLI Commands

#### Run the MCP Server
```bash
simple-mcp-runner run [flags]

Flags:
  -c, --config string       Path to configuration file
      --log-level string    Log level (debug, info, warn, error) (default "info")
      --log-format string   Log format (text, json) (default "text")
  -h, --help               Help for run
```

#### Validate Configuration
```bash
simple-mcp-runner validate --config config.yaml
```

#### Show Version
```bash
simple-mcp-runner version
```

### MCP Tools

The server exposes the following tools via the Model Context Protocol:

#### 1. Command Discovery
- **Name**: `discover_commands`
- **Description**: Discover available system commands
- **Parameters**:
  - `pattern` (optional): Filter pattern (e.g., "git*", "npm")
  - `max_results` (optional): Limit number of results
  - `include_desc` (optional): Include command descriptions

#### 2. Command Execution
- **Name**: `execute_command`
- **Description**: Execute a system command
- **Parameters**:
  - `command` (required): Command to execute
  - `args` (optional): Command arguments
  - `workdir` (optional): Working directory
  - `timeout` (optional): Execution timeout

#### 3. Configured Commands
Custom commands defined in the configuration file are exposed as individual tools.

## Security Considerations

This tool is designed for **local development use only**. Security features include:

1. **Command Blocking**: Dangerous commands are blocked by default
2. **Shell Expansion Protection**: Prevents shell injection attacks
3. **Path Restrictions**: Limit execution to specific directories
4. **Resource Limits**: Prevent resource exhaustion
5. **Timeout Protection**: Commands have configurable timeouts
6. **Output Limits**: Prevent memory exhaustion from large outputs

## Architecture

The project follows clean architecture principles:

```
.
├── cmd/                    # CLI commands
│   ├── root.go
│   ├── run.go
│   ├── validate.go
│   └── version.go
├── internal/              # Private packages
│   ├── config/           # Configuration management
│   ├── discovery/        # Command discovery
│   ├── errors/          # Error handling
│   ├── executor/        # Command execution
│   ├── logger/          # Structured logging
│   └── server/          # MCP server implementation
├── pkg/                  # Public packages
│   └── types/           # Shared types
├── config.yaml          # Example configuration
├── go.mod
├── go.sum
└── main.go
```

## Development

### Running Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/executor
```

### Building with Version Info
```bash
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')

go build -ldflags "-X 'github.com/mjmorales/simple-mcp-runner/cmd.Version=$VERSION' \
  -X 'github.com/mjmorales/simple-mcp-runner/cmd.Commit=$COMMIT' \
  -X 'github.com/mjmorales/simple-mcp-runner/cmd.BuildTime=$BUILD_TIME'" \
  -o simple-mcp-runner
```

### Code Quality

The codebase follows Go best practices:
- Comprehensive error handling with context
- Structured logging for debugging
- Proper resource cleanup and timeouts
- Thread-safe operations
- Extensive test coverage

## Contributing

Contributions are welcome! Please ensure:
1. Code follows Go conventions
2. Tests are included for new functionality
3. Documentation is updated as needed
4. Security implications are considered

## License

[MIT License](LICENSE)

## Acknowledgments

Built using:
- [Cobra](https://github.com/spf13/cobra) for CLI
- [Model Context Protocol SDK](https://github.com/modelcontextprotocol/go-sdk) for MCP support
- Go standard library for robust system interaction