package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/utils/strings/slices"
)

// Resource represents a Kubernetes resource.
type Resource struct {
	// Name is always plural, e.g. "pods".
	Name       string
	Namespaced bool

	// SlashSeparatedInCommand indicates if the resource name should be separated with a slash in the command.
	// So, instead of `kubectl logs pods <name>` it should be `kubectl logs pods/<name>`.
	SlashSeparatedInCommand bool
}

// K8sDiscoveryInterface describes an interface for getting K8s server resources.
type K8sDiscoveryInterface interface {
	ServerPreferredResources() ([]*v1.APIResourceList, error)
}

// CommandGuard is responsible for getting allowed resources for a given command.
type CommandGuard struct {
	log          logrus.FieldLogger
	discoveryCli K8sDiscoveryInterface
}

var (
	// ErrVerbNotSupported is returned when the verb is not supported for the resource.
	ErrVerbNotSupported = errors.New("verb not supported")

	// ErrResourceNotFound is returned when the resource is not found on the server.
	ErrResourceNotFound = errors.New("resource not found")

	// additionalResourceVerbs contains map of per-resource verbs which are not returned by K8s API, but should be supported.
	additionalResourceVerbs = map[string][]string{
		"nodes": {"cordon", "uncordon", "drain", "top"},
		"pods":  {"top"},
	}

	// additionalResourcelessVerbs contains map of per-resource verbs which are not returned by K8s API, but should be supported.
	// These verbs are resourceless, so they should use different kubectl syntax with slash separator.
	additionalVerbsWithSlash = map[string][]string{
		"pods":         {"logs"},
		"jobs":         {"logs"},
		"deployments":  {"logs"},
		"statefulsets": {"logs"},
		"replicasets":  {"logs"},
		"daemonsets":   {"logs"},
	}

	// resourcelessVerbs contains verbs which are not resource-specific.
	resourcelessVerbs = map[string]struct{}{
		"auth":          {},
		"api-versions":  {},
		"api-resources": {},
		"cluster-info":  {},
	}

	// unsupportedGlobalVerbs contains verbs returned by K8s API which are not supported for interactive operations.
	unsupportedGlobalVerbs = map[string]struct{}{
		// invalid kubectl verbs returned by K8s API
		"list":             {},
		"watch":            {},
		"deletecollection": {},

		// valid kubectl verbs, but not supported by interactive kubectl + events actions
		"create":       {},
		"cp":           {},
		"update":       {},
		"patch":        {},
		"diff":         {},
		"port-forward": {},
		"attach":       {},
		"apply":        {},
		"replace":      {},
		"auth":         {},
		"explain":      {},
		"autoscale":    {},
		"scale":        {},
		"wait":         {},
		"proxy":        {},
		"run":          {},
	}
)

// NewCommandGuard creates a new CommandGuard instance.
func NewCommandGuard(log logrus.FieldLogger, discoveryCli K8sDiscoveryInterface) *CommandGuard {
	return &CommandGuard{log: log, discoveryCli: discoveryCli}
}

// FilterSupportedVerbs filters out unsupported verbs by the interactive commands.
func (g *CommandGuard) FilterSupportedVerbs(allVerbs []string) []string {
	return slices.Filter(nil, allVerbs, func(s string) bool {
		_, exists := unsupportedGlobalVerbs[s]
		return !exists
	})
}

// GetAllowedResourcesForVerb returns a list of allowed resources for a given verb.
func (g *CommandGuard) GetAllowedResourcesForVerb(verb string, allConfiguredResources []string) ([]Resource, error) {
	_, found := resourcelessVerbs[verb]
	if found {
		return nil, nil
	}

	resMap, err := g.GetServerResourceMap()
	if err != nil {
		return nil, err
	}

	var resources []Resource
	for _, configuredRes := range allConfiguredResources {
		res, err := g.GetResourceDetailsFromMap(verb, configuredRes, resMap)
		if err != nil {
			if err == ErrVerbNotSupported {
				continue
			}

			return nil, fmt.Errorf("while getting resource details for %q: %w", configuredRes, err)
		}

		resources = append(resources, res)
	}

	if len(resources) == 0 {
		return nil, ErrVerbNotSupported
	}

	return resources, nil
}

// GetResourceDetails returns a Resource struct for a given resource type and verb.
func (g *CommandGuard) GetResourceDetails(selectedVerb, resourceType string) (Resource, error) {
	_, found := resourcelessVerbs[selectedVerb]
	if found {
		return Resource{}, nil
	}

	resMap, err := g.GetServerResourceMap()
	if err != nil {
		return Resource{}, err
	}

	res, err := g.GetResourceDetailsFromMap(selectedVerb, resourceType, resMap)
	if err != nil {
		return Resource{}, err
	}

	return res, nil
}

// GetServerResourceMap returns a map of all resources available on the server.
// LIMITATION: This method ignores second occurrences of the same resource name.
func (g *CommandGuard) GetServerResourceMap() (map[string]v1.APIResource, error) {
	resList, err := g.discoveryCli.ServerPreferredResources()
	if err != nil {
		if !shouldIgnoreResourceListError(err) {
			return nil, fmt.Errorf("while getting resource list from K8s cluster: %w", err)
		}

		g.log.Warnf("Ignoring error while getting resource list from K8s cluster: %s", err.Error())
	}

	resourceMap := make(map[string]v1.APIResource)
	for _, item := range resList {
		for _, res := range item.APIResources {
			// TODO: Resources should be provided with full group version to avoid collisions in names.
			// 	For example, "pods" and "nodes" are both in "v1" and "metrics.k8s.io/v1beta1".
			// 	Ignoring second occurrence for now.
			if _, exists := resourceMap[res.Name]; exists {
				g.log.Debugf("Skipping resource with the same name %q (%q)...", res.Name, item.GroupVersion)
				continue
			}

			resourceMap[res.Name] = res
		}
	}

	return resourceMap, nil
}

// GetResourceDetailsFromMap returns a Resource struct for a given resource type and verb based on the server resource map.
func (g *CommandGuard) GetResourceDetailsFromMap(selectedVerb, resourceType string, resMap map[string]v1.APIResource) (Resource, error) {
	res, exists := resMap[resourceType]
	if !exists {
		return Resource{}, ErrResourceNotFound
	}

	verbs := g.getAllSupportedVerbs(resourceType, res.Verbs)
	if slices.Contains(verbs, selectedVerb) {
		return Resource{
			Name:                    res.Name,
			Namespaced:              res.Namespaced,
			SlashSeparatedInCommand: false,
		}, nil
	}

	addVerbsWithSlash, exists := additionalVerbsWithSlash[resourceType]
	if exists && slices.Contains(addVerbsWithSlash, selectedVerb) {
		return Resource{
			Name:                    res.Name,
			Namespaced:              res.Namespaced,
			SlashSeparatedInCommand: true,
		}, nil
	}

	return Resource{}, ErrVerbNotSupported
}

func (g *CommandGuard) getAllSupportedVerbs(resourceType string, inVerbs v1.Verbs) v1.Verbs {
	// filter out not supported verbs
	verbs := g.FilterSupportedVerbs(inVerbs)

	// enrich with additional verbs
	addResVerbs, exists := additionalResourceVerbs[resourceType]
	if exists {
		verbs = append(verbs, addResVerbs...)
	}

	if slices.Contains(verbs, "get") {
		verbs = append(verbs, "describe")
	}

	return verbs
}

// shouldIgnoreResourceListError returns true if the error should be ignored. This is a workaround for client-go behavior,
// which reports error on empty resource lists. However, some components can register empty lists for their resources.
// See
// See: https://github.com/kyverno/kyverno/issues/2267
func shouldIgnoreResourceListError(err error) bool {
	groupDiscoFailedErr, ok := err.(*discovery.ErrGroupDiscoveryFailed)
	if !ok {
		return false
	}

	for _, currentErr := range groupDiscoFailedErr.Groups {
		// Unfortunately there isn't a nicer way to do this.
		// See https://github.com/kubernetes/client-go/blob/release-1.25/discovery/cached/memory/memcache.go#L228
		if strings.Contains(currentErr.Error(), "Got empty response for") {
			// ignore it as it isn't necessarily an error
			continue
		}

		return false
	}

	return true
}
