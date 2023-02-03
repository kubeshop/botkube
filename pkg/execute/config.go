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
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter

	// Used for deprecated showControllerConfig function.
	cfg config.Config
}

// NewConfigExecutor returns a new ConfigExecutor instance
func NewConfigExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, config config.Config) *ConfigExecutor {
	return &ConfigExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfg:               config,
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
func (e *ConfigExecutor) Show(_ context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)
	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)

	cfg, err := e.renderBotkubeConfiguration()
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while rendering Botkube configuration: %w", err)
	}
	return respond(cfg, cmdCtx), nil
}

func (e *ConfigExecutor) reportCommand(cmdVerb, cmdRes string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	cmdToReport := fmt.Sprintf("%s %s", cmdVerb, cmdRes)
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting config command: %s", err.Error())
	}
}

const redactedSecretStr = "*** REDACTED ***"

func (e *ConfigExecutor) renderBotkubeConfiguration() (string, error) {
	cfg := e.cfg

	// hide sensitive info
	// TODO: avoid printing sensitive data without need to resetting them manually (which is an error-prone approach)
	for key, old := range cfg.Communications {
		old.Slack.Token = redactedSecretStr
		old.SocketSlack.AppToken = redactedSecretStr
		old.SocketSlack.BotToken = redactedSecretStr
		old.Elasticsearch.Password = redactedSecretStr
		old.Discord.Token = redactedSecretStr
		old.Mattermost.Token = redactedSecretStr
		old.Teams.AppPassword = redactedSecretStr

		// maps are not addressable: https://stackoverflow.com/questions/42605337/cannot-assign-to-struct-field-in-a-map
		cfg.Communications[key] = old
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
