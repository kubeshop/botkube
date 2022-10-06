package execute

// FakeCommandGuard provides functionality to resolve correlations between kubectl verbs and resource types.
// TODO: Currently, we use dumb implementation that handles only a happy path, just for test purposes.
//
//	This will be replaced by https://github.com/kubeshop/botkube/issues/786.
type FakeCommandGuard struct{}

// Resource holds additional details about the K8s resource type.
type Resource struct {
	Name                    string
	Namespaced              bool
	SlashSeparatedInCommand bool
}

// GetAllowedResourcesForVerb returns allowed resources types for a given verb.
func (f *FakeCommandGuard) GetAllowedResourcesForVerb(selectedVerb string, allConfiguredResources []string) ([]Resource, error) {
	_, found := resourcelessVerbs[selectedVerb]
	if found {
		return nil, nil
	}

	// special case for 'logs'
	if selectedVerb == "logs" {
		return []Resource{
			staticResourceMapping["deployments"],
			staticResourceMapping["pods"],
		}, nil
	}

	var out []Resource
	for _, name := range allConfiguredResources {
		out = append(out, staticResourceMapping[name])
	}
	return out, nil
}

// GetResourceDetails returns resource details.
func (f *FakeCommandGuard) GetResourceDetails(verb, resourceType string) Resource {
	if verb == "logs" {
		return Resource{
			Name:                    resourceType,
			Namespaced:              true,
			SlashSeparatedInCommand: true,
		}
	}

	return staticResourceMapping[resourceType]
}

var resourcelessVerbs = map[string]struct{}{
	"auth":          {},
	"api-versions":  {},
	"api-resources": {},
	"cluster-info":  {},
}

var staticResourceMapping = map[string]Resource{
	// namespace-scoped:
	"deployments":  {Name: "deployments", Namespaced: true},
	"pods":         {Name: "pods", Namespaced: true},
	"daemonsets":   {Name: "daemonsets", Namespaced: true},
	"statefulsets": {Name: "statefulsets", Namespaced: true},
	"configmaps":   {Name: "configmaps", Namespaced: true},
	"services":     {Name: "services", Namespaced: true},

	// cluster wide:
	"namespaces":     {Name: "namespaces", Namespaced: false},
	"storageclasses": {Name: "storageclasses", Namespaced: false},
	"nodes":          {Name: "nodes", Namespaced: false},
}
