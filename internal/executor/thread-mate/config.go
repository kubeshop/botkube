package thread_mate

import (
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

//go:embed jsonschema.json
var JSONSchema string

// Config holds the executor configuration.
type Config struct {
	RoundRobinGroupName string        `yaml:"roundRobinGroupName"`
	Assignees           []string      `yaml:"assignees"`
	Logger              config.Logger `yaml:"log"`
	DataSyncInterval    time.Duration `yaml:"dataSyncInterval"`
	ConfigMapNamespace  string        `yaml:"configMapNamespace"`
}

// Validate validates the configuration parameters.
func (c *Config) Validate() error {
	issues := multierror.New()
	if c.RoundRobinGroupName == "" {
		issues = multierror.Append(issues, errors.New("the round robin group name cannot be empty"))
	}
	if len(c.Assignees) == 0 {
		issues = multierror.Append(issues, errors.New("the assignees list cannot be empty"))
	}
	return issues.ErrorOrNil()
}

// MergeConfigs merges the configuration.
func MergeConfigs(configs []*executor.Config) (Config, error) {
	defaults := Config{
		RoundRobinGroupName: "default",
		DataSyncInterval:    5 * time.Second,
		ConfigMapNamespace:  "botkube",
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
