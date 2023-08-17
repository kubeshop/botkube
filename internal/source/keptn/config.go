package keptn

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// Config prometheus configuration
type Config struct {
	URL     string        `yaml:"url,omitempty"`
	Token   string        `yaml:"token,omitempty"`
	Project string        `yaml:"project,omitempty"`
	Service string        `yaml:"service,omitempty"`
	Log     config.Logger `yaml:"log,omitempty"`
}

// Log logging configuration
type Log struct {
	Level string `yaml:"level"`
}

// MergeConfigs merges all input configuration.
func MergeConfigs(userCfg *source.Config) (Config, error) {
	defaults := Config{}

	err, out := pluginx.MergeSourceConfigWithDefaults[Config](defaults, userCfg)
	if err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	return out, nil
}
