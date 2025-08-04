.PHONY: build test lint clean install run help

# Build variables
BINARY_NAME=simple-mcp-runner
VERSION?=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/mjmorales/simple-mcp-runner/cmd.version=$(VERSION) -X github.com/mjmorales/simple-mcp-runner/cmd.commit=$(COMMIT) -X github.com/mjmorales/simple-mcp-runner/cmd.date=$(DATE)"

## help: Display this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  %-20s %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## test: Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

## test-coverage: Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linters
lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run ./...

## fmt: Format code
fmt:
	go fmt ./...
	goimports -w .

## clean: Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	go clean -cache

## install: Install the binary
install: build
	go install $(LDFLAGS)

## run: Run the server with example config
run: build
	./$(BINARY_NAME) run --config config.example.yaml

## validate: Validate configuration
validate: build
	./$(BINARY_NAME) validate --config config.example.yaml

## deps: Download dependencies
deps:
	go mod download
	go mod tidy

## update-deps: Update dependencies
update-deps:
	go get -u ./...
	go mod tidy

## security: Run security checks
security:
	@if ! command -v gosec &> /dev/null; then \
		echo "gosec not found. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@if ! command -v govulncheck &> /dev/null; then \
		echo "govulncheck not found. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	gosec ./...
	govulncheck ./...

## dev: Run server in development mode with hot reload
dev:
	@if ! command -v air &> /dev/null; then \
		echo "air not found. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	air

## benchmark: Run benchmarks
benchmark:
	go test -bench=. -benchmem ./...

## release-dry-run: Test release process
release-dry-run:
	@if ! command -v goreleaser &> /dev/null; then \
		echo "goreleaser not found. Please install from https://goreleaser.com"; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

.DEFAULT_GOAL := help