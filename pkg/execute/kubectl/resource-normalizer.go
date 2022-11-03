package kubectl

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
)

// ResourceNormalizer contains helper maps to normalize the resource name specified in the kubectl command.
type ResourceNormalizer struct {
	kindResourceMap      map[string]string
	shortnameResourceMap map[string]string
}

// NewResourceNormalizer returns new ResourceNormalizer instance.
func NewResourceNormalizer(log logrus.FieldLogger, discoveryCli discovery.DiscoveryInterface) (ResourceNormalizer, error) {
	resMapping := ResourceNormalizer{
		kindResourceMap:      make(map[string]string),
		shortnameResourceMap: make(map[string]string),
	}

	_, resourceList, err := discoveryCli.ServerGroupsAndResources()
	if err != nil {
		if !shouldIgnoreResourceListError(err) {
			return ResourceNormalizer{}, fmt.Errorf("while getting resource list from K8s cluster: %w", err)
		}

		log.Warnf("Ignoring error while getting resource list from K8s cluster: %s", err.Error())
	}

	for _, resource := range resourceList {
		for _, r := range resource.APIResources {
			// Exclude subresources
			if strings.Contains(r.Name, "/") {
				continue
			}
			resMapping.kindResourceMap[strings.ToLower(r.Kind)] = r.Name
			for _, sn := range r.ShortNames {
				resMapping.shortnameResourceMap[sn] = r.Name
			}
		}
	}
	log.Infof("Loaded resource mapping: %+v", resMapping)
	return resMapping, nil
}

// Normalize returns list with alternative names for a given input resource.
func (r ResourceNormalizer) Normalize(in string) []string {
	variants := []string{
		// normalized received name
		strings.ToLower(in),
		// normalized short name
		r.shortnameResourceMap[strings.ToLower(in)],
		// normalized kind name
		r.kindResourceMap[strings.ToLower(in)],
	}
	return variants
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
