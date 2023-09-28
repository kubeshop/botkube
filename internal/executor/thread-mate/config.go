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

//go:embed messages/round-robin.yaml
var defaultRoundRobinMessage string

// Config holds the executor configuration.
type Config struct {
	RoundRobin RoundRobinConfig `yaml:"roundRobin"`
	Logger     config.Logger    `yaml:"log"`
	Pick       PickConfig       `yaml:"pick"`

	Persistence PersistenceConfig `yaml:"persistence"`
}

type PersistenceConfig struct {
	SyncInterval       time.Duration `yaml:"syncInterval"`
	ConfigMapNamespace string        `yaml:"configMapNamespace"`
}
type RoundRobinConfig struct {
	Assignees []string `yaml:"assignees"`
	GroupName string   `yaml:"groupName"`
}

type PickConfig struct {
	UserCooldownTime time.Duration `yaml:"userCooldownTime"`
	MessagesTemplate string        `yaml:"messagesTemplate"`
}

// Validate validates the configuration parameters.
func (c *Config) Validate() error {
	issues := multierror.New()
	if c.RoundRobin.GroupName == "" {
		issues = multierror.Append(issues, errors.New("the round robin group name cannot be empty"))
	}
	if len(c.RoundRobin.Assignees) == 0 {
		issues = multierror.Append(issues, errors.New("the assignees list cannot be empty"))
	}
	return issues.ErrorOrNil()
}

// MergeConfigs merges the configuration.
func MergeConfigs(configs []*executor.Config) (Config, error) {
	defaults := Config{
		RoundRobin: RoundRobinConfig{
			GroupName: "default",
		},
		Persistence: PersistenceConfig{
			SyncInterval:       5 * time.Second,
			ConfigMapNamespace: "botkube",
		},
		Pick: PickConfig{
			MessagesTemplate: defaultRoundRobinMessage,
			UserCooldownTime: 3 * time.Minute,
		},
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
