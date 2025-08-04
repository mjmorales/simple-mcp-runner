// Package errors provides enhanced error types for the MCP server
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeValidation indicates a validation error
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeConfiguration indicates a configuration error
	ErrorTypeConfiguration ErrorType = "configuration"
	// ErrorTypeExecution indicates a command execution error
	ErrorTypeExecution ErrorType = "execution"
	// ErrorTypeTimeout indicates a timeout error
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypePermission indicates a permission error
	ErrorTypePermission ErrorType = "permission"
	// ErrorTypeNotFound indicates a not found error
	ErrorTypeNotFound ErrorType = "not_found"
	// ErrorTypeInternal indicates an internal server error
	ErrorTypeInternal ErrorType = "internal"
)

// Error represents an enhanced error with additional context
type Error struct {
	Type    ErrorType
	Message string
	Err     error
	Context map[string]any
	Stack   []string
}

// New creates a new error
func New(errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Context: make(map[string]any),
		Stack:   captureStack(),
	}
}

// Wrap wraps an existing error
func Wrap(err error, errType ErrorType, message string) *Error {
	if err == nil {
		return nil
	}
	
	// If it's already our error type, preserve the original
	if e, ok := err.(*Error); ok {
		e.Message = fmt.Sprintf("%s: %s", message, e.Message)
		return e
	}
	
	return &Error{
		Type:    errType,
		Message: message,
		Err:     err,
		Context: make(map[string]any),
		Stack:   captureStack(),
	}
}

// WithContext adds context to the error
func (e *Error) WithContext(key string, value any) *Error {
	if e == nil {
		return nil
	}
	e.Context[key] = value
	return e
}

// WithContextMap adds multiple context values
func (e *Error) WithContextMap(ctx map[string]any) *Error {
	if e == nil {
		return nil
	}
	for k, v := range ctx {
		e.Context[k] = v
	}
	return e
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *Error) Unwrap() error {
	return e.Err
}

// Is checks if the error is of a specific type
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}
	
	if err, ok := target.(*Error); ok {
		return e.Type == err.Type
	}
	
	return errors.Is(e.Err, target)
}

// GetContext returns a context value
func (e *Error) GetContext(key string) (any, bool) {
	if e == nil || e.Context == nil {
		return nil, false
	}
	val, ok := e.Context[key]
	return val, ok
}

// StackTrace returns the stack trace as a string
func (e *Error) StackTrace() string {
	if e == nil || len(e.Stack) == 0 {
		return ""
	}
	return strings.Join(e.Stack, "\n")
}

// captureStack captures the current stack trace
func captureStack() []string {
	var stack []string
	for i := 2; i < 10; i++ { // Skip this function and the caller
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			break
		}
		
		// Skip runtime functions
		fnName := fn.Name()
		if strings.Contains(fnName, "runtime.") {
			continue
		}
		
		stack = append(stack, fmt.Sprintf("%s:%d %s", trimPath(file), line, fnName))
	}
	return stack
}

// trimPath removes the full path for readability
func trimPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return path
}

// Helper functions for common error types

// ValidationError creates a validation error
func ValidationError(message string, field string) *Error {
	return New(ErrorTypeValidation, message).WithContext("field", field)
}

// ConfigurationError creates a configuration error
func ConfigurationError(message string) *Error {
	return New(ErrorTypeConfiguration, message)
}

// ExecutionError creates an execution error
func ExecutionError(message string, command string) *Error {
	return New(ErrorTypeExecution, message).WithContext("command", command)
}

// TimeoutError creates a timeout error
func TimeoutError(message string, duration string) *Error {
	return New(ErrorTypeTimeout, message).WithContext("duration", duration)
}

// PermissionError creates a permission error
func PermissionError(message string, resource string) *Error {
	return New(ErrorTypePermission, message).WithContext("resource", resource)
}

// NotFoundError creates a not found error
func NotFoundError(message string, resource string) *Error {
	return New(ErrorTypeNotFound, message).WithContext("resource", resource)
}

// InternalError creates an internal error
func InternalError(message string) *Error {
	return New(ErrorTypeInternal, message)
}