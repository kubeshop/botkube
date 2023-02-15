package kubectl

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"k8s.io/utils/strings/slices"

	"github.com/kubeshop/botkube/internal/executor/kubectl/builder"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// Config holds Kubectl plugin configuration parameters.
type Config struct {
	Log                loggerx.Config `yaml:"log"`
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
	if err := pluginx.MergeExecutorConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	return out, nil
}

func jsonSchema(description string) api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"title": "kubectl",
			"description": "%s",
			"type": "object",
			"additionalProperties": false,
			"properties": {
				"defaultNamespace": {
					"description": "The default Kubernetes Namespace to use when not directly specified in the kubectl command.",
					"title": "Default Kubernetes Namespace",
					"type": "string",
					"default": "default"
				},
				"allowed": {
					"type": "object",
					"properties": {
						"verbs": {
							"type": "array",
							"default": ["api-resources", "api-versions", "cluster-info", "describe", "explain", "get", "logs", "top"],
							"items": {
								"type": "string"
							}
						},
						"resources": {
							"type": "array",
							"default": ["deployments", "pods", "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes", "configmaps", "services", "ingresses"],
							"items": {
								"type": "string"
							}
						}
					},
					"title": "Allowed Resources"
				}
			}
		}`, description),
	}
}
