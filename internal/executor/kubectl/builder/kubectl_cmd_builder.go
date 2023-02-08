package builder

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api"
)

const (
	kubectlMissingCommandMsg    = "Please specify the kubectl command"
	interactiveBuilderIndicator = "@builder"
)

// KubectlCmdBuilder provides functionality to handle interactive kubectl command selection.
type KubectlCmdBuilder struct {
}

// NewKubectlCmdBuilder returns a new KubectlCmdBuilder instance.
func NewKubectlCmdBuilder() *KubectlCmdBuilder {
	return &KubectlCmdBuilder{}
}

// ShouldHandle returns true if it's a valid command for interactive builder.
func (e *KubectlCmdBuilder) ShouldHandle(cmd string) bool {
	if cmd == "" || strings.HasPrefix(cmd, interactiveBuilderIndicator) {
		return true
	}
	return false
}

// Handle constructs the interactive command builder messages.
func (e *KubectlCmdBuilder) Handle(_ context.Context, log logrus.FieldLogger, isInteractivitySupported bool) (api.Message, error) {
	if !isInteractivitySupported {
		log.Debug("Interactive kubectl command builder is not supported. Requesting a full kubectl command.")
		return e.message(kubectlMissingCommandMsg)
	}

	return e.message("interactivity not yet implemented")
}

func (e *KubectlCmdBuilder) message(msg string) (api.Message, error) {
	return api.NewPlaintextMessage(msg, true), nil
}
