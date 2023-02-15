package builder

type (
	// Config holds kubectl builder configuration parameters.
	Config struct {
		Allowed AllowedResources `yaml:"allowed,omitempty"`
	}
	// AllowedResources describes interactive builder "building blocks". It's needed to populate dropdowns with proper values.
	AllowedResources struct {
		// Namespaces if not specified, builder needs to have proper permissions to list all namespaces in the cluster.
		Namespaces []string `yaml:"namespaces,omitempty"`
		// Verbs holds allowed verbs, at least one verbs MUST be specified.
		Verbs []string `yaml:"verbs,omitempty"`
		// Resources holds allowed resources.
		Resources []string `yaml:"resources,omitempty"`
	}
)

// DefaultConfig returns default configuration for command builder.
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
