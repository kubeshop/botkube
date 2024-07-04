package execute

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	configFeatureName = FeatureName{
		Name:    "config",
		Aliases: []string{"cfg", "configuration"},
	}
)

// ConfigExecutor executes all commands that are related to config
type ConfigExecutor struct {
	log logrus.FieldLogger
	// Used for deprecated showControllerConfig function.
	cfg config.Config
}

// NewConfigExecutor returns a new ConfigExecutor instance
func NewConfigExecutor(log logrus.FieldLogger, config config.Config) *ConfigExecutor {
	return &ConfigExecutor{
		log: log,
		cfg: config,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *ConfigExecutor) FeatureName() FeatureName {
	return configFeatureName
}

// Commands returns slice of commands the executor supports
func (e *ConfigExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.ShowVerb: e.Show,
	}
}

// Show returns Config in yaml format
func (e *ConfigExecutor) Show(_ context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	redactedCfg := config.HideSensitiveInfo(e.cfg)
	bytes, err := yaml.Marshal(redactedCfg)
	if err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while rendering Botkube configuration: %w", err)
	}

	if err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while rendering Botkube configuration: %w", err)
	}
	return respond(string(bytes), cmdCtx), nil
}
