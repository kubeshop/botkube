package prometheus

import (
	"fmt"

	promApi "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/pluginx"
	"github.com/kubeshop/botkube/pkg/ptr"
)

// Config prometheus configuration
type Config struct {
	URL             string               `yaml:"url,omitempty"`
	AlertStates     []promApi.AlertState `yaml:"alertStates,omitempty"`
	IgnoreOldAlerts *bool                `yaml:"ignoreOldAlerts,omitempty"`
	Log             Log                  `yaml:"log,omitempty"`
}

// Log logging configuration
type Log struct {
	Level string `yaml:"level"`
}

// MergeConfigs merges all input configuration.
func MergeConfigs(configs []*source.Config) (Config, error) {
	defaults := Config{
		URL:             "http://localhost:9090",
		AlertStates:     []promApi.AlertState{promApi.AlertStateFiring, promApi.AlertStatePending, promApi.AlertStateInactive},
		IgnoreOldAlerts: ptr.Bool(true),
		Log: Log{
			Level: "info",
		},
	}

	var out Config
	if err := pluginx.MergeSourceConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	return out, nil
}
