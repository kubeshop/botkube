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

// MergeForNamespace returns kubectl configuration for a given set of bindings.
//
// It merges entries only if a given Namespace is matched.
//   - kubectl.commands.verbs     - strategy append
//   - kubectl.commands.resources - strategy append
//   - kubectl.defaultNamespace   - strategy override (if not empty)
//   - kubectl.restrictAccess     - strategy override (if not empty)
//
// The order of merging is the same as the order of items specified in the includeBindings list.
func (kc *Merger) MergeForNamespace(includeBindings []string, forNamespace string) EnabledKubectl {
	enabledInNs := func(executor config.Kubectl) bool {
		return executor.Enabled && executor.Namespaces.IsAllowed(forNamespace)
	}
	return kc.merge(kc.collect(includeBindings, enabledInNs))
}

// MergeAllEnabled returns kubectl configuration for all kubectl configs.
func (kc *Merger) MergeAllEnabled(includeBindings []string) EnabledKubectl {
	onlyEnabled := func(executor config.Kubectl) bool {
		return executor.Enabled
	}
	return kc.merge(kc.collect(includeBindings, onlyEnabled))
}

// IsAtLeastOneEnabled returns true if at least one kubectl executor is enabled.
func (kc *Merger) IsAtLeastOneEnabled() bool {
	for _, executor := range kc.executors {
		if executor.Kubectl.Enabled {
			return true
		}
	}
	return false
}

// MergeAllEnabledVerbs returns verbs collected from all enabled kubectl executors.
func (kc *Merger) MergeAllEnabledVerbs(bindings []string) map[string]struct{} {
	verbs := map[string]struct{}{}

	for _, name := range bindings {
		executor, found := kc.executors[name]
		if !found {
			continue
		}

		if !executor.Kubectl.Enabled {
			continue
		}

		for _, name := range executor.Kubectl.Commands.Verbs {
			verbs[name] = struct{}{}
		}
	}
	return verbs
}

func (kc *Merger) merge(collectedKubectls []config.Kubectl) EnabledKubectl {
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

type collectPredicateFunc func(executor config.Kubectl) bool

func (kc *Merger) collect(includeBindings []string, predicate collectPredicateFunc) []config.Kubectl {
	var out []config.Kubectl
	for _, name := range includeBindings {
		executor, found := kc.executors[name]
		if !found {
			continue
		}

		if !predicate(executor.Kubectl) {
			continue
		}

		out = append(out, executor.Kubectl)
	}

	return out
}
