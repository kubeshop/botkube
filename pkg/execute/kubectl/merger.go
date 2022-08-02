package kubectl

import (
	"github.com/kubeshop/botkube/pkg/config"
)

// EnabledKubectl configuration for executing commands inside cluster
type EnabledKubectl struct {
	AllowedKubectlVerb     map[string]struct{}
	AllowedKubectlResource map[string]struct{}

	DefaultNamespace string
	RestrictAccess   bool
}

// Merger provides functionality to merge multiple bindings
// associated with the kubectl executor.
type Merger struct {
	executors config.IndexableMap[config.Executors]
}

// NewMerger returns a new Merger instance.
func NewMerger(executors config.IndexableMap[config.Executors]) *Merger {
	return &Merger{
		executors: executors,
	}
}

// Merge returns kubectl configuration for a given set of bindings.
//  Only if the allowed Namespace is matched.
//    kubectl.commands.verbs
//    kubectl.commands.resources
func (kc *Merger) Merge(includeBindings []string, forNamespace string) EnabledKubectl {
	bindings := map[string]struct{}{}
	for _, name := range includeBindings {
		bindings[name] = struct{}{}
	}

	var collectedKubectls []config.Kubectl
	for name, executor := range kc.executors {
		if _, found := bindings[name]; !found {
			continue
		}
		if !executor.Kubectl.Enabled {
			continue
		}
		if forNamespace != config.AllNamespaceIndicator && !executor.Kubectl.Namespaces.IsAllowed(forNamespace) {
			continue
		}

		collectedKubectls = append(collectedKubectls, executor.Kubectl)
	}

	if len(collectedKubectls) == 0 {
		return EnabledKubectl{}
	}

	var (
		defaultNs      string
		restrictAccess bool

		allowedResources = map[string]struct{}{}
		allowedVerbs     = map[string]struct{}{}
	)
	for _, item := range collectedKubectls {
		for _, resourceName := range item.Commands.Resources {
			allowedResources[resourceName] = struct{}{}
		}
		for _, verbName := range item.Commands.Verbs {
			allowedVerbs[verbName] = struct{}{}
		}
		if item.DefaultNamespace != "" {
			defaultNs = item.DefaultNamespace
		}

		if item.RestrictAccess != nil {
			restrictAccess = *item.RestrictAccess
		}
	}

	return EnabledKubectl{
		AllowedKubectlResource: allowedResources,
		AllowedKubectlVerb:     allowedVerbs,
		DefaultNamespace:       defaultNs,
		RestrictAccess:         restrictAccess,
	}
}

// AllEnabledVerbs returns verbs collected from all enabled kubectl executors.
func (kc *Merger) AllEnabledVerbs() map[string]struct{} {
	verbs := map[string]struct{}{}

	for _, executor := range kc.executors {
		if !executor.Kubectl.Enabled {
			continue
		}
		for _, name := range executor.Kubectl.Commands.Verbs {
			verbs[name] = struct{}{}
		}
	}
	return verbs
}
