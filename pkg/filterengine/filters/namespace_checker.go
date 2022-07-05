package filters

import (
	"context"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
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
		shouldSkipEvent := isNamespaceIgnored(resource.Namespaces, event.Namespace)
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

// isNamespaceIgnored checks if an event to be ignored from user config
func isNamespaceIgnored(resourceNamespaces config.Namespaces, eventNamespace string) bool {
	if len(resourceNamespaces.Include) == 1 && resourceNamespaces.Include[0] == "all" {
		if len(resourceNamespaces.Ignore) > 0 {
			for _, ignoredNamespace := range resourceNamespaces.Ignore {
				// exact match
				if ignoredNamespace == eventNamespace {
					return true
				}

				// regexp
				if strings.Contains(ignoredNamespace, "*") {
					ns := strings.Replace(ignoredNamespace, "*", ".*", -1)
					matched, err := regexp.MatchString(ns, eventNamespace)
					if err == nil && matched {
						return true
					}
				}
			}
		}
	}
	return false
}
