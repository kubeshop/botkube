package keptn

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/plugin"
)

// Config prometheus configuration
type Config struct {
	URL     string        `yaml:"url,omitempty"`
	Token   string        `yaml:"token,omitempty"`
	Project string        `yaml:"project,omitempty"`
	Service string        `yaml:"service,omitempty"`
	Log     config.Logger `yaml:"log,omitempty"`
}

// MergeConfigs merges all input configuration.
func MergeConfigs(configs []*source.Config) (Config, error) {
	defaults := Config{}

	var out Config
	if err := plugin.MergeSourceConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	return out, nil
}
