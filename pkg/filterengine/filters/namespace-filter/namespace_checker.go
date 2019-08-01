package filters

import (
	"fmt"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"
)

// NamespaceChecker ignore events from blocklisted namespaces
type NamespaceChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(NamespaceChecker{
		Description: "Checks if event belongs to blocklisted namespaces and filter them.",
	})
}

// Run filters and modifies event struct
func (f NamespaceChecker) Run(object interface{}, event *events.Event) {
	// load config.yaml
	botkubeConfig, err := config.New()
	if err != nil {
		log.Logger.Errorf(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
		log.Logger.Debug("Skipping ignore namespace filter.")
	}
	if botkubeConfig != nil {
		for _, resource := range botkubeConfig.Resources {
			if resource.Name == strings.ToLower(event.Kind) {
				// check if namespace to be ignored
				if isNamespaceIgnored(resource.Namespaces, event.Namespace) {
					event.Skip = true
				}
			}
		}

	}
	log.Logger.Debug("Ignore Namespaces filter successful!")
}

// Describe filter
func (f NamespaceChecker) Describe() string {
	return f.Description
}

// isNamespaceIgnored checks if a event to be ignored from user config
func isNamespaceIgnored(resourceNamespaces config.Namespaces, eventNamespace string) bool {
	if len(resourceNamespaces.Include) == 1 && resourceNamespaces.Include[0] == "all" {
		if len(resourceNamespaces.Ignore) > 0 {
			ignoredNamespaces := fmt.Sprintf("%#v", resourceNamespaces.Ignore)
			if strings.Contains(ignoredNamespaces, eventNamespace) {
				return true
			}
		}
	}
	return false
}
