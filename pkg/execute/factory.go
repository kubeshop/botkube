package execute

import (
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/filterengine"
)

// DefaultExecutorFactory facilitates creation of the Executor instances.
type DefaultExecutorFactory struct {
	log          logrus.FieldLogger
	runCmdFn     CommandRunnerFunc
	cfg          config.Config
	filterEngine filterengine.FilterEngine
	resMapping   ResourceMapping
}

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() string
}

// NewExecutorFactory creates new DefaultExecutorFactory.
func NewExecutorFactory(
	log logrus.FieldLogger,
	runCmdFn CommandRunnerFunc,
	cfg config.Config,
	filterEngine filterengine.FilterEngine,
	resMapping ResourceMapping,
) *DefaultExecutorFactory {
	return &DefaultExecutorFactory{
		log:          log,
		runCmdFn:     runCmdFn,
		cfg:          cfg,
		filterEngine: filterEngine,
		resMapping:   resMapping,
	}
}

// NewDefault creates new Default Executor.
func (f *DefaultExecutorFactory) NewDefault(platform config.BotPlatform, isAuthChannel bool, message string) Executor {
	return &DefaultExecutor{
		log:        f.log,
		runCmdFn:   f.runCmdFn,
		cfg:        f.cfg,
		resMapping: f.resMapping,

		filterEngine:  f.filterEngine,
		IsAuthChannel: isAuthChannel,
		Message:       message,
		Platform:      platform,
	}
}
