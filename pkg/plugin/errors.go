package plugin

import (
	"errors"
	"fmt"
)

// ErrNotStartedPluginManager is an error returned when Plugin Manager was not yet started and initialized successfully.
var ErrNotStartedPluginManager = errors.New("plugin manager is not started yet")

// NotFoundPluginError is an error returned when a given Plugin cannot be found in a given repository.
type NotFoundPluginError struct {
	msg string
}

// NewNotFoundPluginError return a new NotFoundPluginError instance.
func NewNotFoundPluginError(msg string, args ...any) *NotFoundPluginError {
	return &NotFoundPluginError{msg: fmt.Sprintf(msg, args...)}
}

// Error returns the error message.
func (n NotFoundPluginError) Error() string {
	return n.msg
}

// Is returns true if target is not found error.
func (n *NotFoundPluginError) Is(target error) bool {
	_, ok := target.(*NotFoundPluginError)
	return ok
}

// IsNotFoundError returns true if one of the error in the chain is the not found error instance.
func IsNotFoundError(err error) bool {
	return errors.Is(err, &NotFoundPluginError{})
}
