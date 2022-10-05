package kubectl

import (
	"errors"
	"fmt"

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

// CommandGuard is responsible for getting allowed resources for a given command.
type CommandGuard struct {
	log          logrus.FieldLogger
	discoveryCli discovery.DiscoveryInterface
}

var (
	// ErrVerbNotFound is returned when the verb is not supported for the resource.
	ErrVerbNotFound = errors.New("verb not found")

	// ErrResourceNotFound is returned when the resource is not found on the server.
	ErrResourceNotFound = errors.New("resource not found")

	// additionalResourceVerbs contains map of per-resource verbs which are not returned by K8s API, but should be supported.
	additionalResourceVerbs = map[string][]string{
		"nodes": {"cordon", "uncordon", "drain"},
	}

	// additionalResourcelessVerbs contains map of per-resource verbs which are not returned by K8s API, but should be supported.
	// These verbs are resourceless, so they should use different kubectl syntax with slash separator.
	additionalVerbsWithSlash = map[string][]string{
		"pods":         {"logs"},
		"jobs":         {"logs"},
		"deployments":  {"logs"},
		"statefulsets": {"logs"},
	}

	// resourcelessVerbs contains verbs which are not resource-specific.
	resourcelessVerbs = map[string]struct{}{
		"auth":          {},
		"api-versions":  {},
		"api-resources": {},
		"cluster-info":  {},
	}

	// unsupportedGlobalVerbs contains verbs returned by K8s API which are not supported for event-related operations.
	unsupportedGlobalVerbs = []string{"create", "update", "list", "patch", "watch", "deletecollection"}
)

// NewCommandGuard creates a new CommandGuard instance.
func NewCommandGuard(log logrus.FieldLogger, discoveryCli discovery.DiscoveryInterface) *CommandGuard {
	return &CommandGuard{log: log, discoveryCli: discoveryCli}
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
			if err == ErrVerbNotFound {
				continue
			}

			return nil, fmt.Errorf("while getting resource details for %q: %w", configuredRes, err)
		}

		resources = append(resources, res)
	}

	return resources, nil
}

// GetResourceDetails returns a Resource struct for a given resource type and verb.
func (g *CommandGuard) GetResourceDetails(selectedVerb, resourceType string) (Resource, error) {
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
		return nil, fmt.Errorf("while getting server resources: %w", err)
	}

	resourceMap := make(map[string]v1.APIResource)
	for _, item := range resList {
		for _, res := range item.APIResources {
			// TODO: Resources should be provided with full group version to avoid collisions in names.
			// 	For example, "pods" and "nodes" are both in "v1" and "metrics.k8s.io/v1beta1".
			// 	Ignoring second occurrence for now.
			if _, exists := resourceMap[res.Name]; exists {
				g.log.Infof("Skipping resource with the same name %q (%q)...", res.Name, item.GroupVersion)
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

	return Resource{}, ErrVerbNotFound
}

func (g *CommandGuard) getAllSupportedVerbs(resourceType string, inVerbs v1.Verbs) v1.Verbs {
	verbs := inVerbs.DeepCopy()

	// filter out not supported verbs
	verbs = slices.Filter(nil, verbs, func(s string) bool {
		return !slices.Contains(unsupportedGlobalVerbs, s)
	})

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
