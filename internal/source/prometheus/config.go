package prometheus

import (
	"fmt"

	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/ptr"
)

// Config prometheus configuration
type Config struct {
	URL             string               `yaml:"url"`
	AlertStates     []promApi.AlertState `yaml:"alertStates"`
	IgnoreOldAlerts *bool                `yaml:"ignoreOldAlerts"`
	Log             *Log                 `yaml:"log"`
}

// Log logging configuration
type Log struct {
	Level string `yaml:"level"`
}

// MergeConfigs merges all input configuration.
func MergeConfigs(configs []*source.Config) (Config, error) {
	out := Config{
		URL:             "http://localhost:9090",
		AlertStates:     []promApi.AlertState{promApi.AlertStateFiring, promApi.AlertStatePending, promApi.AlertStateInactive},
		IgnoreOldAlerts: ptr.Bool(true),
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

		if cfg.URL != "" {
			out.URL = cfg.URL
		}
		if len(cfg.AlertStates) > 0 {
			out.AlertStates = cfg.AlertStates
		}
		if cfg.IgnoreOldAlerts != nil {
			out.IgnoreOldAlerts = cfg.IgnoreOldAlerts
		}

		if cfg.Log != nil && cfg.Log.Level != "" {
			out.Log = &Log{Level: cfg.Log.Level}
		}
	}

	return out, nil
}
