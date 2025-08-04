//go:build windows

package executor

import (
	"testing"

	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

func getTimeoutTestCase() testCase {
	return testCase{
		name: "command with timeout",
		req: &types.CommandExecutionRequest{
			Command: "powershell",
			Args:    []string{"-Command", "Start-Sleep", "-Seconds", "10"},
			Timeout: "100ms",
		},
		wantErr: true,
		check: func(t *testing.T, result *types.CommandExecutionResult) {
			if !result.TimedOut {
				t.Error("expected command to timeout")
			}
			if result.ErrorMessage != "command timed out" {
				t.Errorf("expected timeout error message, got %s", result.ErrorMessage)
			}
		},
	}
}
