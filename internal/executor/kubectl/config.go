package kubectl

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// Config holds Kubectl plugin configuration parameters.
type Config struct {
	DefaultNamespace string `yaml:"defaultNamespace,omitempty"`
}

// MergeConfigs merges the Kubectl configuration.
func MergeConfigs(configs []*executor.Config) (Config, error) {
	defaults := Config{
		DefaultNamespace: defaultNamespace,
	}

	var out Config
	if err := pluginx.MergeExecutorConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	return out, nil
}
