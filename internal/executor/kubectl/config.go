package kubectl

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/kubeshop/botkube/internal/executor/kubectl/builder"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/plugin"
)

// Config holds Kubectl plugin configuration parameters.
type Config struct {
	Log                config.Logger  `yaml:"log"`
	DefaultNamespace   string         `yaml:"defaultNamespace,omitempty"`
	InteractiveBuilder builder.Config `yaml:"interactiveBuilder,omitempty"`
}

func (c Config) Validate() error {
	if len(c.InteractiveBuilder.Allowed.Namespaces) > 0 {
		found := slices.Contains(c.InteractiveBuilder.Allowed.Namespaces, c.DefaultNamespace)
		if !found {
			return fmt.Errorf("the %q namespace must be included under allowed namespaces property", c.DefaultNamespace)
		}
	}
	return nil
}

// MergeConfigs merges the Kubectl configuration.
func MergeConfigs(configs []*executor.Config) (Config, error) {
	defaults := Config{
		DefaultNamespace:   defaultNamespace,
		InteractiveBuilder: builder.DefaultConfig(),
	}

	var out Config
	if err := plugin.MergeExecutorConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	return out, nil
}
