package kubectl

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/kubeshop/botkube/pkg/api"

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

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"title": "kubectl",
			"description": "%s",
			"type": "object",
			"properties": {
				"defaultNamespace": {
					"description": "The default Kubernetes Namespace to use when not directly specified in the kubectl command.",
					"type": "string",
					"default": "default",
				}
			},
		}`, description),
	}
}
