package kubectl

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"golang.org/x/exp/slices"

	"github.com/kubeshop/botkube/internal/executor/kubectl/builder"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/pluginx"
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
	if err := pluginx.MergeExecutorConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, fmt.Errorf("while merging configuration: %w", err)
	}

	return out, nil
}

func jsonSchema(description string) api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
		  "$schema": "http://json-schema.org/draft-07/schema#",
		  "title": "Kubectl",
		  "description": "%s",
		  "type": "object",
		  "additionalProperties": false,
		  "uiSchema": {
			"interactiveBuilder": {
			  "allowed": {
				"verbs": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				},
				"resources": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				},
				"namespaces": {
				  "ui:classNames": "non-orderable",
				  "ui:options": {
					"orderable": false
				  },
				  "items": {
					"ui:options": {
					  "label": false
					}
				  }
				}
			  }
			}
		  },
		  "properties": {
			"defaultNamespace": {
			  "description": "Namespace used if not explicitly specified during command execution.",
			  "title": "Default Kubernetes Namespace",
			  "type": "string",
			  "default": "default"
			},
			"interactiveBuilder": {
			  "title": "Interactive command builder",
			  "description": "Configuration of the interactive Kubectl command builder.",
			  "type": "object",
			  "properties": {
				"allowed": {
				  "title": "",
				  "type": "object",
				  "description": "",
				  "properties": {
					"verbs": {
					  "type": "array",
					  "title": "Verbs",
					  "description": "Kubectl verbs enabled for interactive Kubectl builder. At least one verb must be specified.",
					  "default": [
						"api-resources",
						"api-versions",
						"cluster-info",
						"describe",
						"explain",
						"get",
						"logs",
						"top"
					  ],
					  "items": {
						"title": "Verb",
						"type": "string"
					  },
					  "minItems": 1
					},
					"resources": {
					  "type": "array",
					  "title": "Resources",
					  "description": "List of allowed resources. Each resource must be provided as a plural noun, such as \"deployments\", \"services\" or \"pods\".",
					  "default": [
						"deployments",
						"pods",
						"namespaces",
						"daemonsets",
						"statefulsets",
						"storageclasses",
						"nodes",
						"configmaps",
						"services",
						"ingresses"
					  ],
					  "minItems": 1,
					  "items": {
						"type": "string",
						"title": "Resource"
					  }
					},
					"namespaces": {
					  "type": "array",
					  "title": "Namespaces",
					  "description": "List of allowed namespaces. If not specified, builder needs to have proper permissions to list all namespaces in the cluster",
					  "default": [],
					  "minItems": 0,
					  "items": {
						"type": "string",
						"title": "Namespace"
					  }
					}
				  }
				}
			  }
			},
			"log": {
			  "title": "Logging",
			  "description": "Logging configuration for the plugin.",
			  "type": "object",
			  "properties": {
				"level": {
				  "title": "Log Level",
				  "description": "Define log level for the plugin. Ensure that Botkube has plugin logging enabled for standard output.",
				  "type": "string",
				  "default": "info",
				  "oneOf": [
					{
					  "const": "panic",
					  "title": "Panic"
					},
					{
					  "const": "fatal",
					  "title": "Fatal"
					},
					{
					  "const": "error",
					  "title": "Error"
					},
					{
					  "const": "warn",
					  "title": "Warning"
					},
					{
					  "const": "info",
					  "title": "Info"
					},
					{
					  "const": "debug",
					  "title": "Debug"
					},
					{
					  "const": "trace",
					  "title": "Trace"
					}
				  ]
				},
				"disableColors": {
				  "type": "boolean",
				  "default": false,
				  "description": "If enabled, disables color logging output.",
				  "title": "Disable Colors"
				}
			  }
			}
		  }
		}`, description),
	}
}
