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
		return ResourceNormalizer{}, fmt.Errorf("while getting resource list from K8s cluster: %w", err)
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
