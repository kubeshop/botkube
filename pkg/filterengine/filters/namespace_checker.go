package filters

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/utils"
)

// NamespaceChecker ignore events from blocklisted namespaces
type NamespaceChecker struct {
	log                 logrus.FieldLogger
	configuredResources []config.Resource
}

// NewNamespaceChecker creates a new NamespaceChecker instance
func NewNamespaceChecker(log logrus.FieldLogger, configuredResources []config.Resource) *NamespaceChecker {
	return &NamespaceChecker{log: log, configuredResources: configuredResources}
}

// Run filters and modifies event struct
func (f *NamespaceChecker) Run(_ context.Context, _ interface{}, event *events.Event) error {
	// Skip filter for cluster scoped resource
	if len(event.Namespace) == 0 {
		return nil
	}

	for _, resource := range f.configuredResources {
		if event.Resource != resource.Name {
			continue
		}
		shouldSkipEvent := !utils.IsNamespaceAllowed(resource.Namespaces, event.Namespace)
		event.Skip = shouldSkipEvent
		break
	}
	f.log.Debug("Ignore Namespaces filter successful!")
	return nil
}

// Name returns the filter's name
func (f *NamespaceChecker) Name() string {
	return "NamespaceChecker"
}

// Describe describes the filter
func (f *NamespaceChecker) Describe() string {
	return "Checks if event belongs to blocklisted namespaces and filter them."
}
