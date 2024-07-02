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
	cfg, err := e.renderBotkubeConfiguration()
	if err != nil {
		return interactive.CoreMessage{}, fmt.Errorf("while rendering Botkube configuration: %w", err)
	}
	return respond(cfg, cmdCtx), nil
}

const redactedSecretStr = "*** REDACTED ***"

func (e *ConfigExecutor) renderBotkubeConfiguration() (string, error) {
	cfg := e.cfg

	// hide sensitive info
	// TODO: avoid printing sensitive data without need to resetting them manually (which is an error-prone approach)
	for key, val := range cfg.Communications {
		val.SocketSlack.AppToken = redactedSecretStr
		val.SocketSlack.BotToken = redactedSecretStr
		val.Elasticsearch.Password = redactedSecretStr
		val.Discord.Token = redactedSecretStr
		val.Mattermost.Token = redactedSecretStr
		val.CloudSlack.Token = redactedSecretStr
		// To keep the printed config readable, we don't print the certificate bytes.
		val.CloudSlack.Server.TLS.CACertificate = nil
		val.CloudTeams.Server.TLS.CACertificate = nil

		// Replace private channel names with aliases
		cloudSlackChannels := make(config.IdentifiableMap[config.CloudSlackChannel])
		for _, channel := range val.CloudSlack.Channels {
			if channel.Alias == nil {
				cloudSlackChannels[channel.ChannelBindingsByName.Name] = channel
				continue
			}

			outChannel := channel
			outChannel.ChannelBindingsByName.Name = fmt.Sprintf("%s (public alias)", *channel.Alias)
			outChannel.Alias = nil
			cloudSlackChannels[*channel.Alias] = outChannel
		}
		val.CloudSlack.Channels = cloudSlackChannels

		// maps are not addressable: https://stackoverflow.com/questions/42605337/cannot-assign-to-struct-field-in-a-map
		cfg.Communications[key] = val
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
