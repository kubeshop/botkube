package commander

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
)

// Command defines a command that is executed by the app.
type Command struct {
	Name string
	Cmd  string
}

// Commander is responsible for generating kubectl commands for the given event.
type Commander struct {
	log              logrus.FieldLogger
	guard            CmdGuard
	allowedVerbs     []string
	allowedResources []string
}

// unsupportedEventCommandVerbs contains list of verbs that are not supported for the actionable event notifications.
var unsupportedEventCommandVerbs = map[string]struct{}{
	"delete": {}, // See https://github.com/kubeshop/botkube/issues/824
}

// CmdGuard is responsible for guarding kubectl commands.
type CmdGuard interface {
	GetServerResourceMap() (map[string]metav1.APIResource, error)
	GetResourceDetailsFromMap(selectedVerb, resourceType string, resMap map[string]metav1.APIResource) (Resource, error)
}

// NewCommander creates a new Commander instance.
func NewCommander(log logrus.FieldLogger, guard CmdGuard, verbs, resources []string) *Commander {
	return &Commander{log: log, guard: guard, allowedVerbs: verbs, allowedResources: resources}
}

// GetCommandsForEvent returns a list of commands for the given event.
func (c *Commander) GetCommandsForEvent(event event.Event) ([]Command, error) {
	if event.Type == config.DeleteEvent {
		c.log.Debug("Skipping commands for the DELETE type of event for %q...", event.Kind)
		return nil, nil
	}

	resourceTypeParts := strings.Split(event.Resource, "/")
	resourceName := resourceTypeParts[len(resourceTypeParts)-1]

	var allowedVerbs []string
	for _, verb := range c.allowedVerbs {
		if _, ok := unsupportedEventCommandVerbs[verb]; ok {
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
				c.log.Debugf("Not supported verb %q for resource %q. Skipping...", verb, resourceName)
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
