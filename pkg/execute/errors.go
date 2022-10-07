package execute

import (
	"errors"
	"fmt"
)

var (
	errInvalidCommand     = errors.New("invalid command")
	errUnsupportedCommand = errors.New("unsupported command")
)

// ExecutionCommandError defines error occurred during command execution.
// Use it only if you want to print the error message details to end-users.
type ExecutionCommandError struct {
	msg string
}

// NewExecutionCommandError creates a new ExecutionCommandError instance. Messages should be suitable to be printed to the end user.
func NewExecutionCommandError(format string, args ...any) *ExecutionCommandError {
	return &ExecutionCommandError{msg: fmt.Sprintf(format, args...)}
}

// Error returns error message
func (e *ExecutionCommandError) Error() string {
	return e.msg
}

// IsExecutionCommandError returns true if a given error is ExecutionCommandError.
func IsExecutionCommandError(err error) bool {
	_, ok := err.(*ExecutionCommandError)
	return ok
}
