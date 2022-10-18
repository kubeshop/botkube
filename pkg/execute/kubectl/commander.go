package kubectl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

// Command defines a command that is executed by the app.
type Command struct {
	Name string
	Cmd  string
}

// Commander is responsible for generating kubectl commands for the given event.
type Commander struct {
	log    logrus.FieldLogger
	merger EnabledKubectlMerger
	guard  CmdGuard
}

// unsupportedEventCommandVerbs contains list of verbs that are not supported for the actionable event notifications.
var unsupportedEventCommandVerbs = map[string]struct{}{
	"delete": {}, // See https://github.com/kubeshop/botkube/issues/824
}

// EnabledKubectlMerger is responsible for merging enabled kubectl commands for the given namespace.
type EnabledKubectlMerger interface {
	MergeForNamespace(includeBindings []string, forNamespace string) EnabledKubectl
}

// CmdGuard is responsible for guarding kubectl commands.
type CmdGuard interface {
	GetServerResourceMap() (map[string]metav1.APIResource, error)
	GetResourceDetailsFromMap(selectedVerb, resourceType string, resMap map[string]metav1.APIResource) (Resource, error)
}

// NewCommander creates a new Commander instance.
func NewCommander(log logrus.FieldLogger, merger EnabledKubectlMerger, guard CmdGuard) *Commander {
	return &Commander{log: log, merger: merger, guard: guard}
}

// GetCommandsForEvent returns a list of commands for the given event based on the executor bindings.
func (c *Commander) GetCommandsForEvent(event events.Event, executorBindings []string) ([]Command, error) {
	if event.Type == config.DeleteEvent {
		c.log.Debug("Skipping commands for the DELETE type of event for %q...", event.Kind)
		return nil, nil
	}

	enabledKubectls := c.merger.MergeForNamespace(executorBindings, event.Namespace)

	resourceTypeParts := strings.Split(event.Resource, "/")
	resourceName := resourceTypeParts[len(resourceTypeParts)-1]

	if _, exists := enabledKubectls.AllowedKubectlResource[resourceName]; !exists {
		// resource not allowed
		return nil, nil
	}

	var allowedVerbs []string
	for key := range enabledKubectls.AllowedKubectlVerb {
		verb := key
		if _, exists := unsupportedEventCommandVerbs[verb]; exists {
			c.log.Debug("Skipping unsupported verb %q for event notification %q...", verb, event.Kind)
			continue
		}

		allowedVerbs = append(allowedVerbs, verb)
	}
	sort.Strings(allowedVerbs)

	resMap, err := c.guard.GetServerResourceMap()
	if err != nil {
		return nil, err
	}

	var commands []Command
	for _, verb := range allowedVerbs {
		res, err := c.guard.GetResourceDetailsFromMap(verb, resourceName, resMap)
		if err != nil {
			if err == ErrVerbNotSupported {
				c.log.Warnf("Not supported verb %q for resource %q. Skipping...", verb, resourceName)
				continue
			}

			return nil, fmt.Errorf("while getting resource details: %w", err)
		}

		var resourceSubstr string
		if res.SlashSeparatedInCommand {
			resourceSubstr = fmt.Sprintf("%s/%s", resourceName, event.Name)
		} else {
			resourceSubstr = fmt.Sprintf("%s %s", resourceName, event.Name)
		}

		var namespaceSubstr string
		if res.Namespaced {
			namespaceSubstr = fmt.Sprintf(" --namespace %s", event.Namespace)
		}

		commands = append(commands, Command{
			Name: verb,
			Cmd:  fmt.Sprintf("%s %s%s", verb, resourceSubstr, namespaceSubstr),
		})
	}

	return commands, nil
}
