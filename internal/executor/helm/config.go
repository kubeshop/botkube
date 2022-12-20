package helm

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/executor"
)

// Config holds Helm plugin configuration parameters.
type Config struct {
	HelmDriver string
}

// Validate validates the Helm configuration parameters.
func (c *Config) Validate() error {
	switch c.HelmDriver {
	case "configmap", "secret", "memory", "":
	default:
		return fmt.Errorf("The %s is invalid. Allowed values are configmap, secret, memory.", c.HelmDriver)
	}
	return nil
}

// MergeConfigs merges the Helm configuration.
func MergeConfigs(configs []*executor.Config) (Config, error) {
	var out Config
	for _, rawCfg := range configs {
		var cfg Config
		err := yaml.Unmarshal(rawCfg.RawYAML, &cfg)
		if err != nil {
			return Config{}, err
		}

		if cfg.HelmDriver != "" {
			out.HelmDriver = cfg.HelmDriver
		}
	}

	if err := out.Validate(); err != nil {
		return Config{}, fmt.Errorf("while validating merged configuration: %w", err)
	}
	return out, nil
}
