package builder

type (
	// Config holds kubectl builder configuration parameters.
	Config struct {
		Allowed AllowedResources `yaml:"allowed,omitempty"`
	}
	AllowedResources struct {
		// Namespaces if not specified, builder needs to have proper permissions to list all namespaces in the cluster.
		Namespaces []string `yaml:"namespaces,omitempty"`
		Verbs      []string `yaml:"verbs,omitempty"`
		Resources  []string `yaml:"resources,omitempty"`
	}
)

func DefaultConfig() Config {
	return Config{
		Allowed: AllowedResources{
			Verbs: []string{
				"api-resources", "api-versions", "cluster-info", "describe", "explain", "get", "logs", "top",
			},
			Resources: []string{
				"deployments", "pods", "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes", "configmaps", "services", "ingresses",
			},
		},
	}
}
