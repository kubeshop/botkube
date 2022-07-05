package bot

import (
	"context"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
)

// Bot connects to communication channels and reads/sends messages
type Bot interface {
	Start(ctx context.Context) error
}

// ExecutorFactory facilitates creation of execute.Executor instances.
type ExecutorFactory interface {
	NewDefault(platform config.BotPlatform, isAuthChannel bool, message string) execute.Executor
}
