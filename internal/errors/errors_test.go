package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrorTypeValidation, "test error")

	if err.Type != ErrorTypeValidation {
		t.Errorf("expected type %s, got %s", ErrorTypeValidation, err.Type)
	}

	if err.Message != "test error" {
		t.Errorf("expected message 'test error', got %s", err.Message)
	}

	if err.Context == nil {
		t.Error("expected context to be initialized")
	}

	if len(err.Stack) == 0 {
		t.Error("expected stack trace to be captured")
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrapped := Wrap(originalErr, ErrorTypeExecution, "wrapped error")

	if wrapped.Type != ErrorTypeExecution {
		t.Errorf("expected type %s, got %s", ErrorTypeExecution, wrapped.Type)
	}

	if wrapped.Message != "wrapped error" {
		t.Errorf("expected message 'wrapped error', got %s", wrapped.Message)
	}

	if !errors.Is(wrapped.Err, originalErr) {
		t.Error("expected original error to be preserved")
	}

	// Test wrapping nil
	nilWrapped := Wrap(nil, ErrorTypeExecution, "wrapped nil")
	if nilWrapped != nil {
		t.Error("wrapping nil should return nil")
	}

	// Test wrapping our error type
	doubleWrapped := Wrap(wrapped, ErrorTypeInternal, "double wrapped")
	if doubleWrapped.Type != ErrorTypeExecution {
		t.Error("wrapping should preserve original error type")
	}
	if !strings.Contains(doubleWrapped.Message, "double wrapped") {
		t.Error("wrapping should prepend new message")
	}
}

func TestError_WithContext(t *testing.T) {
	err := New(ErrorTypeValidation, "test error")
	_ = err.WithContext("field", "username").
		WithContext("value", "invalid-user")

	if val, ok := err.GetContext("field"); !ok || val != "username" {
		t.Error("expected context 'field' to be 'username'")
	}

	if val, ok := err.GetContext("value"); !ok || val != "invalid-user" {
		t.Error("expected context 'value' to be 'invalid-user'")
	}

	// Test nil error
	var nilErr *Error
	result := nilErr.WithContext("key", "value")
	if result != nil {
		t.Error("WithContext on nil should return nil")
	}
}

func TestError_WithContextMap(t *testing.T) {
	err := New(ErrorTypeValidation, "test error")
	ctx := map[string]any{
		"field": "email",
		"value": "not-an-email",
		"line":  42,
	}
	_ = err.WithContextMap(ctx)

	for k, v := range ctx {
		if val, ok := err.GetContext(k); !ok || val != v {
			t.Errorf("expected context %s to be %v", k, v)
		}
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "error without wrapped error",
			err: &Error{
				Type:    ErrorTypeValidation,
				Message: "invalid input",
			},
			expected: "validation: invalid input",
		},
		{
			name: "error with wrapped error",
			err: &Error{
				Type:    ErrorTypeExecution,
				Message: "command failed",
				Err:     errors.New("exit code 1"),
			},
			expected: "execution: command failed: exit code 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestError_Is(t *testing.T) {
	baseErr := errors.New("base error")
	err1 := Wrap(baseErr, ErrorTypeExecution, "wrapped")
	err2 := New(ErrorTypeExecution, "same type")
	err3 := New(ErrorTypeValidation, "different type")

	// Test matching by type
	if !err1.Is(err2) {
		t.Error("expected errors with same type to match")
	}

	if err1.Is(err3) {
		t.Error("expected errors with different types not to match")
	}

	// Test unwrapping
	if !errors.Is(err1, baseErr) {
		t.Error("expected wrapped error to match base error")
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() *Error
		errType  ErrorType
		checkCtx func(t *testing.T, err *Error)
	}{
		{
			name:    "ValidationError",
			fn:      func() *Error { return ValidationError("invalid field", "username") },
			errType: ErrorTypeValidation,
			checkCtx: func(t *testing.T, err *Error) {
				if val, ok := err.GetContext("field"); !ok || val != "username" {
					t.Error("expected field context to be 'username'")
				}
			},
		},
		{
			name:    "ConfigurationError",
			fn:      func() *Error { return ConfigurationError("invalid config") },
			errType: ErrorTypeConfiguration,
		},
		{
			name:    "ExecutionError",
			fn:      func() *Error { return ExecutionError("command failed", "rm -rf /") },
			errType: ErrorTypeExecution,
			checkCtx: func(t *testing.T, err *Error) {
				if val, ok := err.GetContext("command"); !ok || val != "rm -rf /" {
					t.Error("expected command context")
				}
			},
		},
		{
			name:    "TimeoutError",
			fn:      func() *Error { return TimeoutError("operation timed out", "30s") },
			errType: ErrorTypeTimeout,
			checkCtx: func(t *testing.T, err *Error) {
				if val, ok := err.GetContext("duration"); !ok || val != "30s" {
					t.Error("expected duration context")
				}
			},
		},
		{
			name:    "PermissionError",
			fn:      func() *Error { return PermissionError("access denied", "/etc/passwd") },
			errType: ErrorTypePermission,
			checkCtx: func(t *testing.T, err *Error) {
				if val, ok := err.GetContext("resource"); !ok || val != "/etc/passwd" {
					t.Error("expected resource context")
				}
			},
		},
		{
			name:    "NotFoundError",
			fn:      func() *Error { return NotFoundError("file not found", "/tmp/missing") },
			errType: ErrorTypeNotFound,
			checkCtx: func(t *testing.T, err *Error) {
				if val, ok := err.GetContext("resource"); !ok || val != "/tmp/missing" {
					t.Error("expected resource context")
				}
			},
		},
		{
			name:    "InternalError",
			fn:      func() *Error { return InternalError("unexpected error") },
			errType: ErrorTypeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()

			if err.Type != tt.errType {
				t.Errorf("expected error type %s, got %s", tt.errType, err.Type)
			}

			if tt.checkCtx != nil {
				tt.checkCtx(t, err)
			}
		})
	}
}

func TestError_StackTrace(t *testing.T) {
	err := New(ErrorTypeInternal, "test error")

	stack := err.StackTrace()
	if stack == "" {
		t.Error("expected non-empty stack trace")
	}

	// Should contain function names and line numbers
	if !strings.Contains(stack, "TestError_StackTrace") {
		t.Error("expected stack trace to contain test function name")
	}

	// Test nil error
	var nilErr *Error
	if nilErr.StackTrace() != "" {
		t.Error("nil error should return empty stack trace")
	}
}
