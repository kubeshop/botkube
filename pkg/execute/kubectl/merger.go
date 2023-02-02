package kubectl

import (
	"github.com/kubeshop/botkube/pkg/config"
)

// EnabledKubectl configuration for executing commands inside cluster
type EnabledKubectl struct {
	AllowedKubectlVerb     map[string]struct{}
	AllowedKubectlResource map[string]struct{}

	AllowedNamespacesPerResource map[string]config.RegexConstraints

	DefaultNamespace string
	RestrictAccess   bool
}

// Merger provides functionality to merge multiple bindings
// associated with the kubectl executor.
type Merger struct {
	executors map[string]config.Executors
}

// NewMerger returns a new Merger instance.
func NewMerger(executors map[string]config.Executors) *Merger {
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
	return kc.merge(kc.collect(includeBindings, enabledInNs), includeBindings)
}

// MergeAllEnabled returns kubectl configuration for all kubectl configs.
func (kc *Merger) MergeAllEnabled(includeBindings []string) EnabledKubectl {
	return kc.merge(kc.GetAllEnabled(includeBindings), includeBindings)
}

// GetAllEnabled returns the collection of enabled kubectl executors for a given list of bindings without merging them.
func (kc *Merger) GetAllEnabled(includeBindings []string) map[string]config.Kubectl {
	onlyEnabled := func(executor config.Kubectl) bool {
		return executor.Enabled
	}
	return kc.collect(includeBindings, onlyEnabled)
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

func (kc *Merger) merge(collectedKubectls map[string]config.Kubectl, mapKeyOrder []string) EnabledKubectl {
	if len(collectedKubectls) == 0 {
		return EnabledKubectl{}
	}

	var (
		defaultNs      string
		restrictAccess bool

		allowedResources     = map[string]struct{}{}
		allowedVerbs         = map[string]struct{}{}
		allowedNSPerResource = map[string]config.RegexConstraints{}
	)
	for _, name := range mapKeyOrder {
		item, found := collectedKubectls[name]
		if !found {
			continue
		}

		for _, resourceName := range item.Commands.Resources {
			allowedResources[resourceName] = struct{}{}
			ns, found := allowedNSPerResource[resourceName]
			if !found {
				allowedNSPerResource[resourceName] = item.Namespaces
			}
			ns.Exclude = append(ns.Exclude, item.Namespaces.Exclude...)
			ns.Include = append(ns.Include, item.Namespaces.Include...)
			allowedNSPerResource[resourceName] = ns
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
		AllowedKubectlResource:       allowedResources,
		AllowedKubectlVerb:           allowedVerbs,
		AllowedNamespacesPerResource: allowedNSPerResource,
		DefaultNamespace:             defaultNs,
		RestrictAccess:               restrictAccess,
	}
}

type collectPredicateFunc func(executor config.Kubectl) bool

func (kc *Merger) collect(includeBindings []string, predicate collectPredicateFunc) map[string]config.Kubectl {
	if kc.executors == nil {
		return nil
	}
	out := map[string]config.Kubectl{}
	for _, name := range includeBindings {
		executor, found := kc.executors[name]
		if !found {
			continue
		}

		if !predicate(executor.Kubectl) {
			continue
		}

		out[name] = executor.Kubectl
	}

	return out
}
