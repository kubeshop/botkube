package helm

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// Config holds Helm plugin configuration parameters.
type Config struct {
	HelmDriver    string `yaml:"helmDriver,omitempty"`
	HelmCacheDir  string `yaml:"helmCacheDir,omitempty"`
	HelmConfigDir string `yaml:"helmConfigDir,omitempty"`
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
	defaults := Config{
		HelmDriver:    "secret",
		HelmCacheDir:  "/tmp/helm/.cache",
		HelmConfigDir: "/tmp/helm/",
	}

	var out Config
	if err := pluginx.MergeExecutorConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	if err := out.Validate(); err != nil {
		return Config{}, fmt.Errorf("while validating merged configuration: %w", err)
	}
	return out, nil
}
