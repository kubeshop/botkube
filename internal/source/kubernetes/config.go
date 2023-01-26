package kubernetes

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/source"
)

// Config Kubernetes configuration
type Config struct {
	KubeConfig           string        `yaml:"kubeConfig"`
	InformerReSyncPeriod time.Duration `yaml:"informerReSyncPeriod"`
	Log                  *Log          `yaml:"log"`
}

// Log logging configuration
type Log struct {
	Level string `yaml:"level"`
}

// MergeConfigs merges all input configuration.
func MergeConfigs(configs []*source.Config) (Config, error) {
	out := Config{
		Log: &Log{
			Level: "info",
		},
	}
	for _, rawCfg := range configs {
		var cfg Config
		err := yaml.Unmarshal(rawCfg.RawYAML, &cfg)
		if err != nil {
			return Config{}, fmt.Errorf("while unmarshalling YAML config: %w", err)
		}

		if cfg.Log != nil && cfg.Log.Level != "" {
			out.Log = &Log{Level: cfg.Log.Level}
		}
	}

	return out, nil
}
