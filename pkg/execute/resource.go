package execute

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"

	"github.com/kubeshop/botkube/pkg/config"
)

// ResourceMapping contains helper maps for kubectl execution.
type ResourceMapping struct {
	KindResourceMap           map[string]string
	ShortnameResourceMap      map[string]string
	AllowedKubectlResourceMap map[string]bool
	AllowedKubectlVerbMap     map[string]bool
}

// LoadResourceMappingIfShould initializes helper maps to allow kubectl execution for required resources.
// If Kubectl support is disabled, it returns empty ResourceMapping without an error.
func LoadResourceMappingIfShould(log logrus.FieldLogger, conf *config.Config, discoveryCli discovery.DiscoveryInterface) (ResourceMapping, error) {
	if !conf.Executors[0].Kubectl.Enabled {
		log.Infof("Kubectl disabled. Finishing...")
		return ResourceMapping{}, nil
	}

	resMapping := ResourceMapping{
		KindResourceMap:           make(map[string]string),
		ShortnameResourceMap:      make(map[string]string),
		AllowedKubectlResourceMap: make(map[string]bool),
		AllowedKubectlVerbMap:     make(map[string]bool),
	}

	for _, r := range conf.Executors[0].Kubectl.Commands.Resources {
		resMapping.AllowedKubectlResourceMap[r] = true
	}
	for _, r := range conf.Executors[0].Kubectl.Commands.Verbs {
		resMapping.AllowedKubectlVerbMap[r] = true
	}

	_, resourceList, err := discoveryCli.ServerGroupsAndResources()
	if err != nil {
		return ResourceMapping{}, fmt.Errorf("while getting resource list from K8s cluster: %w", err)
	}
	for _, resource := range resourceList {
		for _, r := range resource.APIResources {
			// Ignore subresources
			if strings.Contains(r.Name, "/") {
				continue
			}
			resMapping.KindResourceMap[strings.ToLower(r.Kind)] = r.Name
			for _, sn := range r.ShortNames {
				resMapping.ShortnameResourceMap[sn] = r.Name
			}
		}
	}
	log.Infof("Loaded resource mapping: %+v", resMapping)
	return resMapping, nil
}
