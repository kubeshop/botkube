package kubectl

import "github.com/kubeshop/botkube/internal/command"

// FakeCommandGuard provides functionality to resolve correlations between kubectl verbs and resource types.
// It's used for test purposes.
type FakeCommandGuard struct{}

func NewFakeCommandGuard() *FakeCommandGuard {
	return &FakeCommandGuard{}
}

// FilterSupportedVerbs filters out unsupported verbs by the interactive commands.
func (f *FakeCommandGuard) FilterSupportedVerbs(allVerbs []string) []string {
	return allVerbs
}

// GetAllowedResourcesForVerb returns allowed resources types for a given verb.
func (f *FakeCommandGuard) GetAllowedResourcesForVerb(selectedVerb string, allConfiguredResources []string) ([]command.Resource, error) {
	_, found := f.resourcelessVerbs()[selectedVerb]
	if found {
		return nil, nil
	}

	// special case for 'logs'
	if selectedVerb == "logs" {
		return []command.Resource{
			f.staticResourceMapping()["deployments"],
			f.staticResourceMapping()["pods"],
		}, nil
	}

	var out []command.Resource
	for _, name := range allConfiguredResources {
		res, found := f.staticResourceMapping()[name]
		if !found {
			continue
		}
		out = append(out, res)
	}
	return out, nil
}

// GetResourceDetails returns resource details.
func (f *FakeCommandGuard) GetResourceDetails(verb, resourceType string) (command.Resource, error) {
	if verb == "logs" {
		return command.Resource{
			Name:                    resourceType,
			Namespaced:              true,
			SlashSeparatedInCommand: true,
		}, nil
	}

	res, found := f.staticResourceMapping()[resourceType]
	if found {
		return res, nil
	}

	// fake data about resource
	return command.Resource{
		Name:       resourceType,
		Namespaced: true,
	}, nil
}

func (f *FakeCommandGuard) resourcelessVerbs() map[string]struct{} {
	return map[string]struct{}{
		"auth":          {},
		"api-versions":  {},
		"api-resources": {},
		"cluster-info":  {},
	}
}

func (f *FakeCommandGuard) staticResourceMapping() map[string]command.Resource {
	return map[string]command.Resource{
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
}
